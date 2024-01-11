package v2

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/github"
	"ucode/ucode_go_api_gateway/pkg/gitlab"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func (h *HandlerV2) GithubLogin(c *gin.Context) {
	var (
		githubLoginRequest models.GithubLogin

		accessTokenUrl string = "https://github.com/login/oauth/access_token"
	)

	err := c.ShouldBindJSON(&githubLoginRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	param := map[string]interface{}{
		"client_id":     h.baseConf.GithubClientId,
		"client_secret": h.baseConf.GithubClientSecret,
		"code":          githubLoginRequest.Code,
	}

	result, err := gitlab.MakeRequest("POST", accessTokenUrl, "", param)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	if _, ok := result["error"]; ok {
		h.handleResponse(c, status_http.InvalidArgument, result["error_description"])
		return
	}

	h.handleResponse(c, status_http.OK, result)
}

func (h *HandlerV2) CreateWebhook(c *gin.Context) {
	var (
		createWebhookRequest models.CreateWebhook
	)

	err := c.ShouldBindJSON(&createWebhookRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if createWebhookRequest.RepoName == "" || createWebhookRequest.Username == "" {
		h.handleResponse(c, status_http.BadRequest, "Username or RepoName is empty")
		return
	}

	exists, err := github.ListWebhooks(github.ListWebhookRequest{
		Username:    createWebhookRequest.Username,
		RepoName:    createWebhookRequest.RepoName,
		GithubToken: createWebhookRequest.GithubToken,
		ProjectUrl:  h.baseConf.ProjectUrl,
	})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	if exists {
		h.handleResponse(c, status_http.OK, nil)
		return
	}

	err = github.CreateWebhook(github.CreateWebhookRequest{
		Username:      createWebhookRequest.Username,
		RepoName:      createWebhookRequest.RepoName,
		WebhookSecret: h.baseConf.WebhookSecret,
		FrameworkType: createWebhookRequest.FrameworkType,
		Branch:        createWebhookRequest.Branch,
		FunctionType:  createWebhookRequest.FunctionType,
		GithubToken:   createWebhookRequest.GithubToken,
		ProjectUrl:    h.baseConf.ProjectUrl,
	})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	time.Sleep(2 * time.Second)
	h.handleResponse(c, status_http.Created, nil)
}

func (h *HandlerV2) HandleWebhook(c *gin.Context) {
	var payload map[string]interface{}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, "Failed to read request body")
		return
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, "Failed to unmarshal JSON")
		return
	}

	if !(verifySignature(c.GetHeader("X-Hub-Signature"), body, []byte(h.baseConf.WebhookSecret))) {
		h.handleResponse(c, status_http.BadRequest, "Failed to verify signature")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	var (
		repository = cast.ToStringMap(payload["repository"])
		owner      = cast.ToStringMap(repository["owner"])
		username   = cast.ToString(owner["login"])

		repoId          = cast.ToString(repository["id"])
		repoName        = cast.ToString(repository["name"])
		repoDescription = cast.ToString(repository["description"])
		htmlUrl         = cast.ToString(repository["html_url"])

		hook              = cast.ToStringMap(payload["hook"])
		config            = cast.ToStringMap(hook["config"])
		frameworkType     = cast.ToString(config["framework_type"])
		functionType      = cast.ToString(config["type"])
		branch            = cast.ToString(config["branch"])
		branchFronWebhook = cast.ToString(payload["ref"])
	)

	if branchFronWebhook != "" {
		parts := strings.Split(branchFronWebhook, "/")
		branch = parts[len(parts)-1]
	}

	resources, err := h.companyServices.IntegrationResource().GetByUsername(
		c.Request.Context(),
		&pb.GetByUsernameRequest{Username: username},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, resource := range resources.IntegrationResources {
		r, err := h.companyServices.ServiceResource().GetSingle(
			c.Request.Context(),
			&pb.GetSingleServiceResourceReq{
				ProjectId:     resource.ProjectId,
				EnvironmentId: resource.EnvironmentId,
				ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		function, functionErr := services.FunctionService().FunctionService().GetSingle(
			c.Request.Context(),
			&fc.FunctionPrimaryKey{
				ProjectId: r.ResourceEnvironmentId,
				SourceUrl: cast.ToString(htmlUrl),
				Branch:    branch,
			},
		)

		if function != nil {
			functionType = function.Type
		}

		if functionType == "FUNCTION" {
			url := "https://" + uuid.New().String() + ".u-code.io"

			if functionErr != nil {
				function, err = services.FunctionService().FunctionService().Create(
					context.Background(),
					&fc.CreateFunctionRequest{
						Path:           fmt.Sprintf("%s_%s", repoName, uuid.New()),
						Name:           repoName,
						Description:    repoDescription,
						ProjectId:      r.ResourceEnvironmentId,
						EnvironmentId:  resource.EnvironmentId,
						Type:           "FUNCTION",
						Url:            url,
						SourceUrl:      htmlUrl,
						Branch:         branch,
						PipelineStatus: "running",
					},
				)
				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			}

			go h.deployOpenfaas(services, resource.Token, repoId, function)
		} else {
			var (
				repoHost = fmt.Sprintf("%s-%s", uuid.New(), h.baseConf.GitlabHostMicroFE)
			)

			if functionErr != nil {
				function, err = services.FunctionService().FunctionService().Create(
					context.Background(),
					&fc.CreateFunctionRequest{
						Path:           fmt.Sprintf("%s_%s", repoName, uuid.New()),
						Name:           repoName,
						Description:    repoDescription,
						ProjectId:      r.ResourceEnvironmentId,
						EnvironmentId:  resource.EnvironmentId,
						Type:           "MICRO_FRONTEND",
						Url:            repoHost,
						FrameworkType:  frameworkType,
						SourceUrl:      htmlUrl,
						Branch:         branch,
						PipelineStatus: "running",
					},
				)
				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			}

			importResponse, err := h.deployMicrofrontend(resource.Token, repoId, function)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			go h.pipelineStatus(services, function, importResponse.ID)
		}
	}
}

func (h *HandlerV2) deployOpenfaas(services services.ServiceManagerI, githubToken, repoId string, function *fc.Function) (gitlab.ImportResponse, error) {
	importResponse, err := gitlab.ImportFromGithub(gitlab.ImportData{
		PersonalAccessToken: githubToken,
		RepoId:              repoId,
		TargetNamespace:     "ucode_functions_group",
		NewName:             function.Path,
		GitlabToken:         h.baseConf.GitlabIntegrationToken,
	})
	if err != nil {
		return gitlab.ImportResponse{}, err
	}

	for {
		time.Sleep(60 * time.Second)
		pipeline, err := gitlab.GetLatestPipeline(h.baseConf.GitlabIntegrationToken, function.Branch, importResponse.ID)
		if err != nil {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, importResponse.ID)
			if err != nil {
				return gitlab.ImportResponse{}, err
			}
			return gitlab.ImportResponse{}, err
		}

		if pipeline.Status == "failed" {
			logResponse, err := h.getPipelineLog(fmt.Sprintf("%v", importResponse.ID))
			if err != nil {
				return gitlab.ImportResponse{}, err
			}

			_, err = services.FunctionService().FunctionService().Update(
				context.Background(),
				&fc.Function{
					Id:               function.Id,
					Path:             function.Path,
					Name:             function.Name,
					Description:      function.Description,
					FunctionFolderId: function.FunctionFolderId,
					ProjectId:        function.ProjectId,
					EnvironmentId:    function.EnvironmentId,
					Type:             function.Type,
					Url:              function.Url,
					FrameworkType:    function.FrameworkType,
					SourceUrl:        function.SourceUrl,
					Branch:           function.Branch,
					PipelineStatus:   pipeline.Status,
					RepoId:           fmt.Sprintf("%v", importResponse.ID),
					ErrorMessage:     logResponse.Log,
					JobName:          logResponse.JobName,
				},
			)
			if err != nil {
				err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, importResponse.ID)
				if err != nil {
					return gitlab.ImportResponse{}, err
				}
				return gitlab.ImportResponse{}, err
			}

			return gitlab.ImportResponse{}, nil
		}

		_, err = services.FunctionService().FunctionService().Update(
			context.Background(),
			&fc.Function{
				Id:               function.Id,
				Path:             function.Path,
				Name:             function.Name,
				Description:      function.Description,
				FunctionFolderId: function.FunctionFolderId,
				ProjectId:        function.ProjectId,
				EnvironmentId:    function.EnvironmentId,
				Type:             function.Type,
				Url:              function.Url,
				FrameworkType:    function.FrameworkType,
				SourceUrl:        function.SourceUrl,
				Branch:           function.Branch,
				PipelineStatus:   pipeline.Status,
				RepoId:           fmt.Sprintf("%v", repoId),
				ErrorMessage:     "",
				JobName:          "",
			},
		)

		repoID, _ := strconv.Atoi(repoId)
		if err != nil {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoID)
			if err != nil {
				return gitlab.ImportResponse{}, err
			}
			return gitlab.ImportResponse{}, err
		}

		if pipeline.Status == "success" || pipeline.Status == "skipped" {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoID)
			if err != nil {
				return gitlab.ImportResponse{}, err
			}
			return gitlab.ImportResponse{}, nil
		}
	}
}

func (h *HandlerV2) deployMicrofrontend(githubToken, repoId string, function *fc.Function) (gitlab.ImportResponse, error) {
	importResponse, err := gitlab.ImportFromGithub(gitlab.ImportData{
		PersonalAccessToken: githubToken,
		RepoId:              repoId,
		TargetNamespace:     "ucode/ucode_micro_frontend",
		NewName:             function.Path,
		GitlabToken:         h.baseConf.GitlabIntegrationToken,
	})
	if err != nil {
		return gitlab.ImportResponse{}, err
	}

	_, err = gitlab.UpdateProject(gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.baseConf.GitlabIntegrationURL,
		GitlabIntegrationToken: h.baseConf.GitlabIntegrationToken,
		GitlabProjectId:        importResponse.ID,
		GitlabGroupId:          h.baseConf.GitlabGroupIdMicroFE,
	}, map[string]interface{}{
		"ci_config_path": ".gitlab-ci.yml",
	})
	if err != nil {
		return gitlab.ImportResponse{}, err
	}

	host := make(map[string]interface{})
	host["key"] = "INGRESS_HOST"
	host["value"] = function.Url

	_, err = gitlab.CreateProjectVariable(gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.baseConf.GitlabIntegrationURL,
		GitlabIntegrationToken: h.baseConf.GitlabIntegrationToken,
		GitlabProjectId:        importResponse.ID,
		GitlabGroupId:          h.baseConf.GitlabGroupIdMicroFE,
	}, host)
	if err != nil {
		return gitlab.ImportResponse{}, err
	}

	time.Sleep(3 * time.Second)

	err = gitlab.AddFilesToRepo(h.baseConf.GitlabIntegrationToken, h.baseConf.PathToClone, importResponse.ID, function.Branch)
	if err != nil {
		return gitlab.ImportResponse{}, err
	}

	return importResponse, nil
}

func (h *HandlerV2) pipelineStatus(services services.ServiceManagerI, function *fc.Function, repoId int) error {
	time.Sleep(10 * time.Second)
	err := gitlab.AddCiFile(h.baseConf.GitlabIntegrationToken, h.baseConf.PathToClone, repoId, function.Branch)
	if err != nil {
		return err
	}

	for {
		time.Sleep(60 * time.Second)
		pipeline, err := gitlab.GetLatestPipeline(h.baseConf.GitlabIntegrationToken, function.Branch, repoId)
		if err != nil {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoId)
			if err != nil {
				return err
			}
			return err
		}

		if pipeline.Status == "failed" {
			logResponse, err := h.getPipelineLog(fmt.Sprintf("%v", repoId))
			if err != nil {
				return err
			}

			_, err = services.FunctionService().FunctionService().Update(
				context.Background(),
				&fc.Function{
					Id:               function.Id,
					Path:             function.Path,
					Name:             function.Name,
					Description:      function.Description,
					FunctionFolderId: function.FunctionFolderId,
					ProjectId:        function.ProjectId,
					EnvironmentId:    function.EnvironmentId,
					Type:             function.Type,
					Url:              function.Url,
					FrameworkType:    function.FrameworkType,
					SourceUrl:        function.SourceUrl,
					Branch:           function.Branch,
					PipelineStatus:   pipeline.Status,
					RepoId:           fmt.Sprintf("%v", repoId),
					ErrorMessage:     logResponse.Log,
					JobName:          logResponse.JobName,
				},
			)
			if err != nil {
				err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoId)
				if err != nil {
					return err
				}
				return err
			}

			return nil
		}

		_, err = services.FunctionService().FunctionService().Update(
			context.Background(),
			&fc.Function{
				Id:               function.Id,
				Path:             function.Path,
				Name:             function.Name,
				Description:      function.Description,
				FunctionFolderId: function.FunctionFolderId,
				ProjectId:        function.ProjectId,
				EnvironmentId:    function.EnvironmentId,
				Type:             function.Type,
				Url:              function.Url,
				FrameworkType:    function.FrameworkType,
				SourceUrl:        function.SourceUrl,
				Branch:           function.Branch,
				PipelineStatus:   pipeline.Status,
				RepoId:           fmt.Sprintf("%v", repoId),
				ErrorMessage:     "",
				JobName:          "",
			},
		)
		if err != nil {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoId)
			if err != nil {
				return err
			}
			return err
		}

		if pipeline.Status == "success" || pipeline.Status == "skipped" {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoId)
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func (h *HandlerV2) getPipelineLog(repoId string) (models.PipelineLogResponse, error) {

	url := fmt.Sprintf("%s/api/v4/projects/%v/jobs", h.baseConf.GitlabIntegrationURL, repoId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.PipelineLogResponse{}, err
	}

	req.Header.Set("PRIVATE-TOKEN", h.baseConf.GitlabIntegrationToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.PipelineLogResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.PipelineLogResponse{}, err
	}

	var jobs []models.Job
	err = json.Unmarshal(body, &jobs)
	if err != nil {
		return models.PipelineLogResponse{}, err
	}

	for _, job := range jobs {
		url := fmt.Sprintf("%s/api/v4/projects/%v/jobs/%v/trace", h.baseConf.GitlabIntegrationURL, repoId, job.Id)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return models.PipelineLogResponse{}, err
		}

		req.Header.Set("PRIVATE-TOKEN", h.baseConf.GitlabIntegrationToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return models.PipelineLogResponse{}, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return models.PipelineLogResponse{}, err
		}

		if job.Status == "failed" {
			pipelineResp := models.PipelineLogResponse{
				JobName: job.Name,
				Log:     string(body),
			}

			return pipelineResp, err
		}
	}

	return models.PipelineLogResponse{}, nil
}

func verifySignature(signatureHeader string, body []byte, secret []byte) bool {
	mac := hmac.New(sha1.New, secret)

	mac.Write(body)

	expectedMAC := mac.Sum(nil)

	signature := signatureHeader[len("sha1="):]

	receivedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(receivedSignature, expectedMAC)
}
