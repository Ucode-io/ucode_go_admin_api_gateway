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
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fn "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/github"
	"ucode/ucode_go_api_gateway/pkg/gitlab"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func (h *HandlerV2) GithubGetBranches(c *gin.Context) {
	var (
		username = c.Query("username")
		repoName = c.Query("repo")
		token    = c.Query("token")

		url      = fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", username, repoName)
		response models.GithubBranch
	)

	resultByte, err := gitlab.MakeRequestV1("GET", url, token, map[string]interface{}{})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	if err := json.Unmarshal(resultByte, &response); err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

func (h *HandlerV2) GithubGetRepos(c *gin.Context) {
	var (
		username = c.Query("username")
		token    = c.Query("token")
		url      = fmt.Sprintf("https://api.github.com/users/%s/repos", username)
		response = models.GithubRepo{}
	)

	resultByte, err := gitlab.MakeRequestV1("GET", url, token, map[string]any{})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	if err := json.Unmarshal(resultByte, &response); err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

func (h *HandlerV2) GithubGetUser(c *gin.Context) {
	var (
		token      = c.Query("token")
		getUserUrl = "https://api.github.com/user"
		response   models.GithubUser
	)

	resultByte, err := gitlab.MakeRequestV1("GET", getUserUrl, token, map[string]interface{}{})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	if err := json.Unmarshal(resultByte, &response); err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	if response.Status == "401" {
		h.handleResponse(c, status_http.BadRequest, "can not find username wrong token format")
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

func (h *HandlerV2) GithubLogin(c *gin.Context) {
	var (
		code                  = c.Query("code")
		accessTokenUrl string = "https://github.com/login/oauth/access_token"
	)

	param := map[string]interface{}{
		"client_id":     h.baseConf.GithubClientId,
		"client_secret": h.baseConf.GithubClientSecret,
		"code":          code,
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
	var createWebhookRequest models.CreateWebhook

	if err := c.ShouldBindJSON(&createWebhookRequest); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
		return
	}

	githubResource, err := h.companyServices.Resource().GetSingleProjectResouece(c.Request.Context(), &pb.PrimaryKeyProjectResource{
		Id:            createWebhookRequest.Resource,
		EnvironmentId: environmentId.(string),
		ProjectId:     projectId.(string),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	createWebhookRequest.Username = githubResource.GetSettings().Github.Username
	createWebhookRequest.GithubToken = githubResource.GetSettings().Github.Token

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
		Name:          createWebhookRequest.Name,
		Resource:      createWebhookRequest.Resource,
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
		h.handleResponse(c, status_http.BadRequest, "Failed to unmarshal JSON inside handle webhook")
		return
	}

	h.log.Info("From Webhook", logger.Any("data", payload))

	if !(verifySignature(c.GetHeader("X-Hub-Signature"), body, []byte(h.baseConf.WebhookSecret))) {
		h.handleResponse(c, status_http.BadRequest, "Failed to verify signature")
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
		resourceType      = cast.ToString(config["resource_id"])
		name              = cast.ToString(config["name"])
		branchFromWebhook = cast.ToString(payload["ref"])
	)

	if branchFromWebhook != "" {
		parts := strings.Split(branchFromWebhook, "/")
		branch = parts[len(parts)-1]
	}

	resources, err := h.companyServices.IntegrationResource().GetByUsername(
		c.Request.Context(),
		&pb.GetByUsernameRequest{Username: username},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	for _, r := range resources.IntegrationResources {
		resource, err := h.companyServices.ServiceResource().GetSingle(
			c.Request.Context(),
			&pb.GetSingleServiceResourceReq{
				ProjectId:     r.ProjectId,
				EnvironmentId: r.EnvironmentId,
				ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}

		services, err := h.GetProjectSrvc(
			c.Request.Context(),
			resource.GetProjectId(),
			resource.NodeType,
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}

		function, functionErr := services.FunctionService().FunctionService().GetSingle(
			c.Request.Context(),
			&fn.FunctionPrimaryKey{
				ProjectId: resource.ResourceEnvironmentId,
				SourceUrl: htmlUrl,
				Branch:    branch,
			},
		)
		if function != nil {
			functionType = function.Type
		}

		if functionType == "FUNCTION" {
			url := fmt.Sprintf("https://%v.u-code.io", uuid.New())

			if functionErr != nil {
				function, err = services.FunctionService().FunctionService().Create(
					context.Background(),
					&fn.CreateFunctionRequest{
						Path:           repoName,
						Name:           name,
						Description:    repoDescription,
						ProjectId:      resource.ResourceEnvironmentId,
						EnvironmentId:  resource.EnvironmentId,
						Type:           "FUNCTION",
						Url:            url,
						SourceUrl:      htmlUrl,
						Branch:         branch,
						PipelineStatus: "running",
						Resource:       resourceType,
					},
				)
				if err != nil {
					h.handleResponse(c, status_http.InvalidArgument, err.Error())
					return
				}
			} else {
				_, _ = services.FunctionService().FunctionService().Update(
					context.Background(),
					&fn.Function{
						Id:             function.Id,
						Path:           function.Path,
						Name:           function.Name,
						Description:    function.Description,
						ProjectId:      function.ProjectId,
						EnvironmentId:  function.EnvironmentId,
						Url:            function.Url,
						Type:           function.Type,
						SourceUrl:      function.SourceUrl,
						Branch:         function.Branch,
						PipelineStatus: "running",
						Resource:       function.Resource,
						ProvidedName:   function.ProvidedName,
					},
				)
				function.PipelineStatus = "running"
			}
			go h.deployOpenfaas(services, r.Token, repoId, function)
		} else {
			repoHost := fmt.Sprintf("%s-%s", uuid.New(), h.baseConf.GitlabHostMicroFE)

			if functionErr != nil {
				function, err = services.FunctionService().FunctionService().Create(
					context.Background(),
					&fn.CreateFunctionRequest{
						Path:           fmt.Sprintf("%s_%s", repoName, uuid.New()),
						Name:           repoName,
						Description:    repoDescription,
						ProjectId:      resource.ResourceEnvironmentId,
						EnvironmentId:  resource.EnvironmentId,
						Type:           "MICRO_FRONTEND",
						Url:            repoHost,
						FrameworkType:  frameworkType,
						SourceUrl:      htmlUrl,
						Branch:         branch,
						PipelineStatus: "running",
						Resource:       resourceType,
						ProvidedName:   name,
					},
				)
				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			} else {
				services.FunctionService().FunctionService().Update(
					context.Background(),
					&fn.Function{
						Id:             function.Id,
						Path:           function.Path,
						Name:           function.Name,
						Description:    function.Description,
						ProjectId:      function.ProjectId,
						EnvironmentId:  function.EnvironmentId,
						Type:           function.Type,
						Url:            function.Url,
						FrameworkType:  function.FrameworkType,
						SourceUrl:      function.SourceUrl,
						Branch:         function.Branch,
						PipelineStatus: "running",
						Resource:       function.Resource,
						ProvidedName:   function.ProvidedName,
					},
				)
				function.PipelineStatus = "running"
			}

			importResponse, err := h.deployMicrofrontend(r.Token, repoId, function)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			go h.pipelineStatus(services, function, importResponse.ID)
		}
	}
}

func (h *HandlerV2) deployOpenfaas(services services.ServiceManagerI, githubToken, repoId string, function *fn.Function) (gitlab.ImportResponse, error) {
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

	time.Sleep(10 * time.Second)
	err = gitlab.AddCiFile(h.baseConf.GitlabIntegrationToken, h.baseConf.PathToClone, importResponse.ID, function.Branch, "openfaas_integration")
	if err != nil {
		err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, importResponse.ID)
		if err != nil {
			return gitlab.ImportResponse{}, err
		}
	}

	for {
		time.Sleep(60 * time.Second)
		pipeline, err := gitlab.GetLatestPipeline(h.baseConf.GitlabIntegrationToken, function.Branch, importResponse.ID)
		if err != nil {
			services.FunctionService().FunctionService().Update(
				context.Background(),
				&fn.Function{
					Id:             function.Id,
					Path:           function.Path,
					Name:           function.Name,
					Description:    function.Description,
					ProjectId:      function.ProjectId,
					EnvironmentId:  function.EnvironmentId,
					Type:           function.Type,
					Url:            function.Url,
					SourceUrl:      function.SourceUrl,
					Branch:         function.Branch,
					PipelineStatus: "failed",
					RepoId:         fmt.Sprintf("%v", importResponse.ID),
					ErrorMessage:   "Failed to get pipeline status",
					JobName:        "",
					Resource:       function.Resource,
					ProvidedName:   function.ProvidedName,
				},
			)
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, importResponse.ID)
			if err != nil {
				return gitlab.ImportResponse{}, err
			}
			return gitlab.ImportResponse{}, err
		}

		if pipeline.Status == "failed" {
			logResp, err := h.getPipelineLog(fmt.Sprintf("%v", importResponse.ID))
			if err != nil {
				return gitlab.ImportResponse{}, err
			}

			services.FunctionService().FunctionService().Update(
				context.Background(),
				&fn.Function{
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
					ErrorMessage:     logResp.Log,
					JobName:          logResp.JobName,
					Resource:         function.Resource,
					ProvidedName:     function.ProvidedName,
				},
			)

			err = gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, importResponse.ID)
			if err != nil {
				return gitlab.ImportResponse{}, err
			}
			return gitlab.ImportResponse{}, err
		}

		services.FunctionService().FunctionService().Update(
			context.Background(),
			&fn.Function{
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
				ErrorMessage:     "",
				JobName:          "",
				Resource:         function.Resource,
				ProvidedName:     function.ProvidedName,
			},
		)

		if pipeline.Status == "success" || pipeline.Status == "skipped" {
			err := gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, importResponse.ID)
			if err != nil {
				return gitlab.ImportResponse{}, err
			}
			return gitlab.ImportResponse{}, nil
		}
	}
}

func (h *HandlerV2) deployMicrofrontend(githubToken, repoId string, function *fn.Function) (gitlab.ImportResponse, error) {
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

func (h *HandlerV2) pipelineStatus(services services.ServiceManagerI, function *fn.Function, repoId int) error {
	time.Sleep(10 * time.Second)
	err := gitlab.AddCiFile(h.baseConf.GitlabIntegrationToken, h.baseConf.PathToClone, repoId, function.Branch, "github_integration")
	if err != nil {
		err = gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoId)
		if err != nil {
			return err
		}
		return err
	}

	for {
		time.Sleep(70 * time.Second)
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

			services.FunctionService().FunctionService().Update(
				context.Background(),
				&fn.Function{
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
					Resource:         function.Resource,
					ProvidedName:     function.ProvidedName,
				},
			)
			err = gitlab.DeleteRepository(h.baseConf.GitlabIntegrationToken, repoId)
			if err != nil {
				return err
			}

			return nil
		}

		_, err = services.FunctionService().FunctionService().Update(
			context.Background(),
			&fn.Function{
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
				Resource:         function.Resource,
				ProvidedName:     function.ProvidedName,
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
