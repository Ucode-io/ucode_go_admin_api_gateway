package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"
	servicepkg "ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"google.golang.org/protobuf/types/known/structpb"
)

// ─── CRUD ────────────────────────────────────────────────────────────────────

func (h *HandlerV1) CreateUgenTemplate(c *gin.Context) {
	var req pb.CreateUgenTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if err := h.enrichCreateUgenTemplateSource(c, &req); err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.UgenTemplate().Create(c.Request.Context(), &req)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, resp)
}

func (h *HandlerV1) enrichCreateUgenTemplateSource(c *gin.Context, req *pb.CreateUgenTemplateReq) error {
	ctx := c.Request.Context()

	if req.GetMcpProjectId() == "" {
		return fmt.Errorf("mcp_project_id is required")
	}
	if req.GetSourceResourceEnvId() == "" {
		projectID, ok := c.Get("project_id")
		if !ok || !util.IsValidUUID(projectID.(string)) {
			return config.ErrProjectIdValid
		}
		environmentID, ok := c.Get("environment_id")
		if !ok || !util.IsValidUUID(environmentID.(string)) {
			return config.ErrEnvironmentIdValid
		}
		resource, err := h.companyServices.ServiceResource().GetSingle(
			ctx,
			&pb.GetSingleServiceResourceReq{
				ProjectId:     projectID.(string),
				EnvironmentId: environmentID.(string),
				ServiceType:   pb.ServiceType_BUILDER_SERVICE,
			},
		)
		if err != nil {
			return fmt.Errorf("get source builder resource from context: %w", err)
		}
		if resource.GetResourceEnvironmentId() == "" {
			return fmt.Errorf("source_resource_env_id could not be resolved from current project")
		}
		req.SourceResourceEnvId = resource.GetResourceEnvironmentId()
	}
	if req.GetSourceFunctionId() == "" {
		return fmt.Errorf("source_function_id is required")
	}

	sourceMcpResourceEnvID, sourceMcpProjectID, sourceMcpNodeType, err := h.resolveTemplateSourceMcpResource(ctx, c, req.GetSourceMcpResourceEnvId())
	if err != nil {
		return err
	}
	req.SourceMcpResourceEnvId = sourceMcpResourceEnvID

	sourceMcpService, err := h.GetProjectSrvc(ctx, sourceMcpProjectID, sourceMcpNodeType)
	if err != nil {
		return fmt.Errorf("get source mcp project service: %w", err)
	}

	if _, err = sourceMcpService.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		ctx,
		&pbo.McpProjectId{
			ResourceEnvId: req.GetSourceResourceEnvId(),
			Id:            req.GetMcpProjectId(),
			WithoutFiles:  true,
		},
	); err != nil {
		return fmt.Errorf("get source mcp project: %w", err)
	}

	sourceDataResourceEnv, err := h.companyServices.Resource().GetResourceEnvironment(
		ctx,
		&pb.GetResourceEnvironmentReq{Id: req.GetSourceResourceEnvId()},
	)
	if err != nil {
		return fmt.Errorf("get source resource env: %w", err)
	}
	if sourceDataResourceEnv.GetProjectId() == "" || sourceDataResourceEnv.GetEnvironmentId() == "" {
		return fmt.Errorf("source_resource_env_id does not resolve project/environment")
	}
	if sourceDataResourceEnv.GetServiceType() != 0 && sourceDataResourceEnv.GetServiceType() != int32(pb.ServiceType_BUILDER_SERVICE) {
		return fmt.Errorf("source_resource_env_id must belong to builder service")
	}
	if sourceDataResourceEnv.GetResourceType() != 0 && sourceDataResourceEnv.GetResourceType() != int32(pb.ResourceType_POSTGRESQL) {
		return fmt.Errorf("source_resource_env_id must belong to postgres resource")
	}

	sourceDataResource, err := h.companyServices.ServiceResource().GetSingle(
		ctx,
		&pb.GetSingleServiceResourceReq{
			ProjectId:     sourceDataResourceEnv.GetProjectId(),
			EnvironmentId: sourceDataResourceEnv.GetEnvironmentId(),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		return fmt.Errorf("get source builder resource: %w", err)
	}
	if sourceDataResource.GetResourceEnvironmentId() != "" && sourceDataResource.GetResourceEnvironmentId() != req.GetSourceResourceEnvId() {
		return fmt.Errorf("source_resource_env_id does not match source project's builder resource")
	}

	sourceNodeType := sourceDataResource.GetNodeType()
	if sourceNodeType == "" {
		sourceNodeType = sourceDataResourceEnv.GetNodeType()
	}
	if sourceNodeType == "" {
		return fmt.Errorf("source_node_type could not be resolved")
	}

	req.SourceProjectId = sourceDataResourceEnv.GetProjectId()
	req.SourceEnvironmentId = sourceDataResourceEnv.GetEnvironmentId()
	req.SourceNodeType = sourceNodeType

	sourceDataService, err := h.GetProjectSrvc(ctx, req.GetSourceProjectId(), req.GetSourceNodeType())
	if err != nil {
		return fmt.Errorf("get source data project service: %w", err)
	}

	sourceFunction, err := sourceDataService.GoObjectBuilderService().Function().GetSingle(
		ctx,
		&pbo.FunctionPrimaryKey{
			Id:        req.GetSourceFunctionId(),
			ProjectId: req.GetSourceMcpResourceEnvId(),
		},
	)
	if err != nil {
		return fmt.Errorf("get source function: %w", err)
	}

	if req.GetSourceRepoId() == "" {
		req.SourceRepoId = sourceFunction.GetRepoId()
	} else if sourceFunction.GetRepoId() != "" && req.GetSourceRepoId() != sourceFunction.GetRepoId() {
		return fmt.Errorf("source_repo_id does not match source_function_id repo_id")
	}
	if req.GetSourceRepoId() == "" {
		return fmt.Errorf("source_repo_id is required")
	}

	if req.GetPreviewUrl() == "" && sourceFunction.GetUrl() != "" {
		req.PreviewUrl = normalizeUgenTemplatePreviewURL(sourceFunction.GetUrl())
	}

	return nil
}

func (h *HandlerV1) resolveTemplateSourceMcpResource(ctx context.Context, c *gin.Context, resourceEnvID string) (string, string, string, error) {
	if resourceEnvID != "" {
		resEnv, err := h.companyServices.Resource().GetResourceEnvironment(
			ctx,
			&pb.GetResourceEnvironmentReq{Id: resourceEnvID},
		)
		if err != nil {
			return "", "", "", fmt.Errorf("get source mcp resource env: %w", err)
		}
		if resEnv.GetProjectId() == "" || resEnv.GetEnvironmentId() == "" {
			return "", "", "", fmt.Errorf("source_mcp_resource_env_id does not resolve project/environment")
		}
		nodeType := resEnv.GetNodeType()
		if nodeType == "" {
			resource, err := h.companyServices.ServiceResource().GetSingle(
				ctx,
				&pb.GetSingleServiceResourceReq{
					ProjectId:     resEnv.GetProjectId(),
					EnvironmentId: resEnv.GetEnvironmentId(),
					ServiceType:   pb.ServiceType_BUILDER_SERVICE,
				},
			)
			if err != nil {
				return "", "", "", fmt.Errorf("get source mcp builder resource: %w", err)
			}
			nodeType = resource.GetNodeType()
		}
		if nodeType == "" {
			return "", "", "", fmt.Errorf("source mcp node_type could not be resolved")
		}
		return resourceEnvID, resEnv.GetProjectId(), nodeType, nil
	}

	projectID, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectID.(string)) {
		return "", "", "", config.ErrProjectIdValid
	}
	environmentID, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentID.(string)) {
		return "", "", "", config.ErrEnvironmentIdValid
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		ctx,
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectID.(string),
			EnvironmentId: environmentID.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		return "", "", "", fmt.Errorf("get current builder resource: %w", err)
	}
	if resource.GetResourceEnvironmentId() == "" || resource.GetNodeType() == "" {
		return "", "", "", fmt.Errorf("current builder resource is incomplete")
	}

	return resource.GetResourceEnvironmentId(), projectID.(string), resource.GetNodeType(), nil
}

func normalizeUgenTemplatePreviewURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	return "https://" + strings.TrimPrefix(rawURL, "//")
}

func (h *HandlerV1) GetUgenTemplateById(c *gin.Context) {
	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid id")
		return
	}

	resp, err := h.companyServices.UgenTemplate().GetById(
		c.Request.Context(),
		&pb.GetUgenTemplateByIdReq{Id: id},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, resp)
}

func (h *HandlerV1) GetUgenTemplateList(c *gin.Context) {
	limit := cast.ToInt32(c.Query("limit"))
	offset := cast.ToInt32(c.Query("offset"))
	if limit == 0 {
		limit = 10
	}

	resp, err := h.companyServices.UgenTemplate().List(
		c.Request.Context(),
		&pb.GetUgenTemplateListReq{Limit: limit, Offset: offset},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, resp)
}

func (h *HandlerV1) UpdateUgenTemplate(c *gin.Context) {
	var req pb.UpdateUgenTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.Id = c.Param("id")
	if !util.IsValidUUID(req.Id) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid id")
		return
	}

	resp, err := h.companyServices.UgenTemplate().Update(c.Request.Context(), &req)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, resp)
}

func (h *HandlerV1) DeleteUgenTemplate(c *gin.Context) {
	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid id")
		return
	}

	_, err := h.companyServices.UgenTemplate().Delete(
		c.Request.Context(),
		&pb.DeleteUgenTemplateReq{Id: id},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"message": "deleted"})
}

// ─── Create project from template ────────────────────────────────────────────

type CreateProjectFromTemplateReq struct {
	TemplateId  string `json:"template_id" binding:"required"`
	ProjectName string `json:"project_name"`
}

// CreateProjectFromTemplate provisions a new isolated ucode project from a
// Ugen template: creates a generated backend project, copies template schema,
// data and MCP files, then publishes the copied microfrontend to u-gen.
func (h *HandlerV1) CreateProjectFromTemplate(c *gin.Context) {
	var req CreateProjectFromTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	var (
		projectId     any
		environmentId any
		ok            bool
		ctx           = context.Background()
	)

	projectId, ok = c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return
	}

	environmentId, ok = c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.HandleResponse(c, status_http.Unauthorized, "unauthorized")
		return
	}

	tmpl, err := h.companyServices.UgenTemplate().GetById(
		ctx, &pb.GetUgenTemplateByIdReq{Id: req.TemplateId},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get template: %v", err))
		return
	}
	if tmpl.GetMcpProjectId() == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "template source mcp_project_id is required")
		return
	}

	projectName := req.ProjectName
	if projectName == "" {
		projectName = tmpl.GetName()
	}

	headProject, err := h.companyServices.Project().GetById(
		ctx, &pb.GetProjectByIdRequest{ProjectId: projectId.(string)},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get head project: %v", err))
		return
	}

	headResource, err := h.companyServices.ServiceResource().GetSingle(
		ctx,
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get head resource: %v", err))
		return
	}

	mainResourceEnvID := headResource.GetResourceEnvironmentId()
	mainService, err := h.GetProjectSrvc(ctx, projectId.(string), headResource.GetNodeType())
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get main project service: %v", err))
		return
	}

	sourceDataResourceEnvID := tmpl.GetSourceResourceEnvId()
	if sourceDataResourceEnvID == "" {
		sourceDataResourceEnvID = mainResourceEnvID
	}
	sourceMcpResourceEnvID := tmpl.GetSourceMcpResourceEnvId()
	if sourceMcpResourceEnvID == "" {
		sourceMcpResourceEnvID = sourceDataResourceEnvID
	}
	sourceProjectID := tmpl.GetSourceProjectId()
	if sourceProjectID == "" {
		sourceProjectID = projectId.(string)
	}
	sourceNodeType := tmpl.GetSourceNodeType()
	if sourceNodeType == "" {
		sourceNodeType = headResource.GetNodeType()
	}

	sourceService, err := h.GetProjectSrvc(ctx, sourceProjectID, sourceNodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get source project service: %v", err))
		return
	}

	sourceMcp, err := sourceService.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		ctx,
		&pbo.McpProjectId{
			ResourceEnvId: sourceMcpResourceEnvID,
			Id:            tmpl.GetMcpProjectId(),
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get source mcp project: %v", err))
		return
	}
	sourceMcpFiles := cloneMcpProjectFiles(sourceMcp.GetProjectFiles())
	if len(sourceMcpFiles) == 0 {
		sourceMcpFiles, err = h.getTemplateMicrofrontendFiles(ctx, sourceService, tmpl, sourceDataResourceEnvID, c.GetHeader("Authorization"))
		if err != nil {
			h.HandleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(sourceMcpFiles) == 0 {
		h.HandleResponse(c, status_http.InvalidArgument, "template source has no microfrontend files")
		return
	}

	//if err = billing.CheckProjectCountLimit(ctx, h.companyServices, mainService, mainResourceEnvID, headProject.GetFareId()); err != nil {
	//	if errors.Is(err, billing.ErrProjectLimitExceeded) {
	//		h.HandleResponse(c, status_http.PaymentRequired, models.PaymentProjectLimit)
	//	} else {
	//		h.HandleResponse(c, status_http.GRPCError, err.Error())
	//	}
	//	return
	//}

	targetProject, err := h.companyServices.Project().Create(
		ctx, &pb.CreateProjectRequest{
			Title:        sanitizeProjectNameForBackend(projectName),
			CompanyId:    headProject.GetCompanyId(),
			K8SNamespace: headProject.GetK8SNamespace(),
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("create target project: %v", err))
		return
	}

	targetEnv, err := h.companyServices.Environment().CreateV2(
		ctx, &pb.CreateEnvironmentRequest{
			CompanyId:    headProject.GetCompanyId(),
			ProjectId:    targetProject.GetProjectId(),
			UserId:       authInfo.GetUserIdAuth(),
			ClientTypeId: authInfo.GetClientTypeId(),
			RoleId:       authInfo.GetRoleId(),
			Name:         "Production",
			DisplayColor: "#00FF00",
			Description:  "Production Environment",
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("create target env: %v", err))
		return
	}

	targetResource, err := h.companyServices.ServiceResource().GetSingle(
		ctx,
		&pb.GetSingleServiceResourceReq{
			ProjectId:     targetProject.GetProjectId(),
			EnvironmentId: targetEnv.GetId(),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get target resource: %v", err))
		return
	}

	targetService, err := h.GetProjectSrvc(ctx, targetProject.GetProjectId(), targetResource.GetNodeType())
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get target project service: %v", err))
		return
	}

	apiKeys, err := h.authService.ApiKey().GetList(
		ctx, &as.GetListReq{
			EnvironmentId: targetEnv.GetId(),
			ProjectId:     targetProject.GetProjectId(),
			Limit:         1,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("get api keys: %v", err))
		return
	}
	var apiKey string
	if len(apiKeys.GetData()) > 0 {
		apiKey = apiKeys.GetData()[0].GetAppId()
	}

	newMcpProject, err := mainService.GoObjectBuilderService().McpProject().CreateMcpProject(
		ctx, &pbo.CreateMcpProjectReqeust{
			ResourceEnvId:  mainResourceEnvID,
			Title:          projectName,
			Description:    "Created from template: " + tmpl.GetName(),
			ProjectFiles:   sourceMcpFiles,
			ProjectEnv:     sourceMcp.GetProjectEnv(),
			UcodeProjectId: targetProject.GetProjectId(),
			ApiKey:         apiKey,
			EnvironmentId:  targetEnv.GetId(),
			Status:         "ready",
			AppVisibility:  sourceMcp.GetAppVisibility(),
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("create mcp project: %v", err))
		return
	}

	chat, err := mainService.GoObjectBuilderService().AiChat().CreateChat(
		ctx,
		&pbo.CreateChatRequest{
			ResourceEnvId: mainResourceEnvID,
			ProjectId:     newMcpProject.GetId(),
			Title:         projectName,
			Description:   "Created from template: " + tmpl.GetName(),
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("create template project chat: %v", err))
		return
	}

	targetProjectData := &models.ProjectData{
		McpProjectId:   newMcpProject.GetId(),
		UcodeProjectId: targetProject.GetProjectId(),
		ApiKey:         apiKey,
		EnvironmentId:  targetEnv.GetId(),
		ResourceEnvId:  targetResource.GetResourceEnvironmentId(),
		NodeType:       targetResource.GetNodeType(),
		ResourceType:   int32(targetResource.GetResourceType()),
	}

	if err = h.copyUgenTemplateData(ctx, sourceService, targetService, sourceDataResourceEnvID, targetProjectData.ResourceEnvId); err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("copy template data: %v", err))
		return
	}

	published, err := h.publishTemplateMicrofrontend(ctx, projectName, sourceMcpFiles, targetProjectData, mainResourceEnvID, c.GetHeader("Authorization"))
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("publish template microfrontend: %v", err))
		return
	}

	if _, err = mainService.GoObjectBuilderService().McpProject().UpdateMcpProject(ctx, &pbo.McpProject{
		ResourceEnvId:       mainResourceEnvID,
		Id:                  newMcpProject.GetId(),
		MicrofrontendId:     published.Data.ID,
		MicrofrontendRepoId: published.Data.RepoId,
		MicrofrontendBranch: published.Data.Branch,
		MicrofrontendUrl:    published.Data.Url,
	}); err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("save template microfrontend refs: %v", err))
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{
		"project_id":                 targetProject.GetProjectId(),
		"ucode_project_id":           targetProject.GetProjectId(),
		"environment_id":             targetEnv.GetId(),
		"mcp_project_id":             newMcpProject.GetId(),
		"chat_id":                    chat.GetId(),
		"api_key":                    apiKey,
		"resource_env_id":            targetProjectData.ResourceEnvId,
		"main_resource_env_id":       mainResourceEnvID,
		"microfrontend_id":           published.Data.ID,
		"microfrontend_repo_id":      published.Data.RepoId,
		"microfrontend_url":          published.Data.Url,
		"microfrontend_branch":       published.Data.Branch,
		"template_preview_url":       tmpl.GetPreviewUrl(),
		"source_mcp_project_id":      tmpl.GetMcpProjectId(),
		"source_resource_env_id":     sourceDataResourceEnvID,
		"source_mcp_resource_env_id": sourceMcpResourceEnvID,
		"source_repo_id":             tmpl.GetSourceRepoId(),
		"source_function_id":         tmpl.GetSourceFunctionId(),
	})
}

func cloneMcpProjectFiles(files []*pbo.McpProjectFiles) []*pbo.McpProjectFiles {
	copied := make([]*pbo.McpProjectFiles, 0, len(files))
	for _, file := range files {
		copied = append(copied, &pbo.McpProjectFiles{
			Path:      file.GetPath(),
			Content:   file.GetContent(),
			FileGraph: file.GetFileGraph(),
		})
	}
	return copied
}

type templateMicrofrontendFilesResponse struct {
	Data struct {
		Files []struct {
			Path     string `json:"path"`
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		} `json:"files"`
	} `json:"data"`
}

func (h *HandlerV1) getTemplateMicrofrontendFiles(ctx context.Context, sourceService servicepkg.ServiceManagerI, tmpl *pb.UgenTemplate, sourceDataResourceEnvID, authToken string) ([]*pbo.McpProjectFiles, error) {
	repoID := tmpl.GetSourceRepoId()
	if repoID == "" && tmpl.GetSourceFunctionId() != "" {
		function, err := sourceService.GoObjectBuilderService().Function().GetSingle(ctx, &pbo.FunctionPrimaryKey{
			Id:        tmpl.GetSourceFunctionId(),
			ProjectId: sourceDataResourceEnvID,
		})
		if err != nil {
			return nil, fmt.Errorf("get template source function: %w", err)
		}
		repoID = function.GetRepoId()
	}
	if repoID == "" {
		return nil, fmt.Errorf("template source has no project_files and no source_repo_id")
	}

	filesURL := h.baseConf.GoFunctionServiceHost + h.baseConf.GoFunctionServiceHTTPPort +
		"/v2/functions/micro-frontend/files?repo_id=" + url.QueryEscape(repoID)

	filesReq, err := http.NewRequestWithContext(ctx, http.MethodGet, filesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build template files request: %w", err)
	}
	filesReq.Header.Set("Authorization", authToken)

	httpClient := &http.Client{Timeout: 2 * time.Minute}
	filesResp, err := httpClient.Do(filesReq)
	if err != nil {
		return nil, fmt.Errorf("fetch template u-gen files: %w", err)
	}
	defer filesResp.Body.Close()

	filesRespBytes, err := io.ReadAll(filesResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read template u-gen files response: %w", err)
	}
	if filesResp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch template u-gen files returned %d: %s", filesResp.StatusCode, string(filesRespBytes))
	}

	var result templateMicrofrontendFilesResponse
	if err = json.Unmarshal(filesRespBytes, &result); err != nil {
		return nil, fmt.Errorf("parse template u-gen files response: %w", err)
	}

	files := make([]*pbo.McpProjectFiles, 0, len(result.Data.Files))
	for _, file := range result.Data.Files {
		filePath := file.Path
		if filePath == "" {
			filePath = file.FilePath
		}
		if filePath == "" {
			continue
		}
		files = append(files, &pbo.McpProjectFiles{
			Path:    filePath,
			Content: file.Content,
		})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("template source repo %s has no microfrontend files", repoID)
	}

	log.Printf("[ugen-template] loaded %d source files from repo_id=%s", len(files), repoID)
	return files, nil
}

func (h *HandlerV1) copyUgenTemplateData(ctx context.Context, sourceService, targetService servicepkg.ServiceManagerI, sourceResourceEnvID, targetResourceEnvID string) error {
	tablesResp, err := sourceService.GoObjectBuilderService().Table().GetAll(ctx, &pbo.GetAllTablesRequest{
		ProjectId: sourceResourceEnvID,
		Limit:     1000,
		Offset:    0,
	})
	if err != nil {
		return fmt.Errorf("get source tables: %w", err)
	}

	for _, table := range tablesResp.GetTables() {
		if skipUgenTemplateTable(table.GetSlug()) {
			continue
		}
		createTable, err := convert[*pbo.Table, *pbo.CreateTableRequest](table)
		if err != nil {
			return fmt.Errorf("convert table %s: %w", table.GetSlug(), err)
		}
		createTable.ProjectId = targetResourceEnvID
		createTable.EnvId = targetResourceEnvID
		if createTable.Id == "" {
			createTable.Id = table.GetId()
		}
		if _, err = targetService.GoObjectBuilderService().Table().Create(ctx, createTable); err != nil {
			return fmt.Errorf("create table %s: %w", table.GetSlug(), err)
		}
	}

	if err = h.copyTemplateMenus(ctx, sourceService, targetService, sourceResourceEnvID, targetResourceEnvID); err != nil {
		return err
	}

	for _, table := range tablesResp.GetTables() {
		if skipUgenTemplateTable(table.GetSlug()) {
			continue
		}
		if err = h.copyTemplateTableDetails(ctx, sourceService, targetService, sourceResourceEnvID, targetResourceEnvID, table); err != nil {
			return err
		}
	}

	return nil
}

func (h *HandlerV1) copyTemplateTableDetails(ctx context.Context, sourceService, targetService servicepkg.ServiceManagerI, sourceResourceEnvID, targetResourceEnvID string, table *pbo.Table) error {
	fieldsResp, err := sourceService.GoObjectBuilderService().Field().GetAll(ctx, &pbo.GetAllFieldsRequest{
		Limit:     1000,
		Offset:    0,
		TableId:   table.GetId(),
		TableSlug: table.GetSlug(),
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get fields for %s: %w", table.GetSlug(), err)
	}
	for _, field := range fieldsResp.GetFields() {
		if skipUgenTemplateField(field.GetSlug(), field.GetType(), table.GetIsLoginTable()) {
			continue
		}
		createField, err := convert[*pbo.Field, *pbo.CreateFieldRequest](field)
		if err != nil {
			return fmt.Errorf("convert field %s.%s: %w", table.GetSlug(), field.GetSlug(), err)
		}
		createField.ProjectId = targetResourceEnvID
		createField.EnvId = targetResourceEnvID
		createField.TableId = table.GetId()
		if _, err = targetService.GoObjectBuilderService().Field().Create(ctx, createField); err != nil {
			return fmt.Errorf("create field %s.%s: %w", table.GetSlug(), field.GetSlug(), err)
		}
	}

	relationsResp, err := sourceService.GoObjectBuilderService().Relation().GetRelationsByTableFrom(ctx, &pbo.GetRelationsByTableFromRequest{
		TableFrom: table.GetSlug(),
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get relations for %s: %w", table.GetSlug(), err)
	}
	for _, relation := range relationsResp.GetRelations() {
		if skipUgenTemplateRelation(table, relation) {
			continue
		}
		relation.ProjectId = targetResourceEnvID
		relation.EnvId = targetResourceEnvID
		relation.RelationFieldId = uuid.NewString()
		relation.RelationToFieldId = uuid.NewString()
		if relation.Attributes == nil {
			relation.Attributes, _ = helper.ConvertMapToStruct(map[string]any{})
		}
		if _, err = targetService.GoObjectBuilderService().Relation().Create(ctx, relation); err != nil {
			return fmt.Errorf("create relation %s -> %s: %w", relation.GetTableFrom(), relation.GetTableTo(), err)
		}
	}

	layoutsResp, err := sourceService.GoObjectBuilderService().Layout().GetLayoutByTableID(ctx, &pbo.GetLayoutByTableIDRequest{
		TableId:   table.GetId(),
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get layouts for %s: %w", table.GetSlug(), err)
	}
	for _, layout := range layoutsResp.GetLayouts() {
		layoutReq, err := convert[*pbo.LayoutResponse, *pbo.LayoutRequest](layout)
		if err != nil {
			return fmt.Errorf("convert layout %s: %w", layout.GetId(), err)
		}
		layoutReq.ProjectId = targetResourceEnvID
		layoutReq.EnvId = targetResourceEnvID
		layoutReq.TableId = table.GetId()
		layoutReq.WithoutResponse = true
		if _, err = targetService.GoObjectBuilderService().Layout().Update(ctx, layoutReq); err != nil {
			return fmt.Errorf("create layout %s: %w", layout.GetId(), err)
		}
	}

	viewsResp, err := sourceService.GoObjectBuilderService().View().GetList(ctx, &pbo.GetAllViewsRequest{
		TableSlug: table.GetSlug(),
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get views for %s: %w", table.GetSlug(), err)
	}
	for _, view := range viewsResp.GetViews() {
		viewReq, err := convert[*pbo.View, *pbo.CreateViewRequest](view)
		if err != nil {
			return fmt.Errorf("convert view %s: %w", view.GetId(), err)
		}
		viewReq.ProjectId = targetResourceEnvID
		viewReq.EnvId = targetResourceEnvID
		viewReq.Id = ""
		if _, err = targetService.GoObjectBuilderService().View().Create(ctx, viewReq); err != nil {
			return fmt.Errorf("create view %s: %w", view.GetId(), err)
		}
	}

	eventsResp, err := sourceService.GoObjectBuilderService().CustomEvent().GetList(ctx, &pbo.GetCustomEventsListRequest{
		TableSlug: table.GetSlug(),
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get custom events for %s: %w", table.GetSlug(), err)
	}
	for _, event := range eventsResp.GetCustomEvents() {
		_, err = targetService.GoObjectBuilderService().CustomEvent().Create(ctx, &pbo.CreateCustomEventRequest{
			TableSlug:  table.GetSlug(),
			Icon:       event.GetIcon(),
			EventPath:  event.GetEventPath(),
			Label:      event.GetLabel(),
			Url:        event.GetUrl(),
			Disable:    event.GetDisable(),
			ProjectId:  targetResourceEnvID,
			Method:     event.GetMethod(),
			ActionType: event.GetActionType(),
			Attributes: event.GetAttributes(),
			EnvId:      targetResourceEnvID,
			Path:       event.GetPath(),
		})
		if err != nil {
			return fmt.Errorf("create custom event %s: %w", event.GetLabel(), err)
		}
	}

	return h.copyTemplateRows(ctx, sourceService, targetService, sourceResourceEnvID, targetResourceEnvID, table)
}

func (h *HandlerV1) copyTemplateRows(ctx context.Context, sourceService, targetService servicepkg.ServiceManagerI, sourceResourceEnvID, targetResourceEnvID string, table *pbo.Table) error {
	if table.GetIsLoginTable() {
		return nil
	}

	tableSlug := table.GetSlug()
	listData, err := helper.ConvertMapToStruct(map[string]any{
		"limit":  1000,
		"offset": 0,
	})
	if err != nil {
		return err
	}
	rows, err := sourceService.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &pbo.CommonMessage{
		TableSlug: tableSlug,
		Data:      listData,
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get rows for %s: %w", tableSlug, err)
	}
	rowsMap, err := convertAnyStruct(rows.Data)
	if err != nil {
		return fmt.Errorf("convert rows for %s: %w", tableSlug, err)
	}
	rowList := cast.ToSlice(rowsMap["response"])
	if len(rowList) == 0 {
		return nil
	}
	objects := make([]map[string]any, 0, len(rowList))
	for _, row := range rowList {
		if rowMap, ok := row.(map[string]any); ok {
			objects = append(objects, rowMap)
		}
	}
	if len(objects) == 0 {
		return nil
	}
	fields := make([]string, 0, len(objects[0]))
	for field := range objects[0] {
		fields = append(fields, field)
	}
	upsertData, err := helper.ConvertMapToStruct(map[string]any{
		"objects":    objects,
		"field_slug": "guid",
		"fields":     fields,
	})
	if err != nil {
		return err
	}
	_, err = targetService.GoObjectBuilderService().Items().UpsertMany(ctx, &pbo.CommonMessage{
		TableSlug: tableSlug,
		Data:      upsertData,
		ProjectId: targetResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("upsert rows for %s: %w", tableSlug, err)
	}
	return nil
}

func convertAnyStruct(s *structpb.Struct) (map[string]any, error) {
	if s == nil {
		return map[string]any{}, nil
	}
	return convert[*structpb.Struct, map[string]any](s)
}

func (h *HandlerV1) copyTemplateMenus(ctx context.Context, sourceService, targetService servicepkg.ServiceManagerI, sourceResourceEnvID, targetResourceEnvID string) error {
	const rootMenuID = "c57eedc3-a954-4262-a0af-376c65b5a284"

	keptTargetMenuIDs, err := h.clearTargetTemplateMenus(ctx, targetService, targetResourceEnvID, rootMenuID)
	if err != nil {
		return err
	}

	tree, err := sourceService.GoObjectBuilderService().Menu().GetMenuTree(ctx, &pbo.MenuPrimaryKey{
		Id:        rootMenuID,
		ProjectId: sourceResourceEnvID,
	})
	if err != nil {
		return fmt.Errorf("get menu tree: %w", err)
	}

	for _, child := range tree.GetChildren() {
		if err = h.copyTemplateMenuTree(ctx, targetService, child, targetResourceEnvID, rootMenuID, keptTargetMenuIDs); err != nil {
			return err
		}
	}
	return nil
}

func (h *HandlerV1) clearTargetTemplateMenus(ctx context.Context, targetService servicepkg.ServiceManagerI, targetResourceEnvID, rootMenuID string) (map[string]bool, error) {
	tree, err := targetService.GoObjectBuilderService().Menu().GetMenuTree(ctx, &pbo.MenuPrimaryKey{
		Id:        rootMenuID,
		ProjectId: targetResourceEnvID,
	})
	if err != nil {
		return nil, fmt.Errorf("get target menu tree: %w", err)
	}

	keptMenuIDs := map[string]bool{
		rootMenuID: true,
	}
	for _, child := range tree.GetChildren() {
		if err = h.clearTargetTemplateMenuTree(ctx, targetService, child, targetResourceEnvID, keptMenuIDs); err != nil {
			return nil, err
		}
	}
	return keptMenuIDs, nil
}

func (h *HandlerV1) clearTargetTemplateMenuTree(ctx context.Context, targetService servicepkg.ServiceManagerI, menu *pbo.MenuTree, targetResourceEnvID string, keptMenuIDs map[string]bool) error {
	for _, child := range menu.GetChildren() {
		if err := h.clearTargetTemplateMenuTree(ctx, targetService, child, targetResourceEnvID, keptMenuIDs); err != nil {
			return err
		}
	}

	if isProtectedUgenTemplateMenu(menu.GetId()) {
		keptMenuIDs[menu.GetId()] = true
		return nil
	}

	if _, err := targetService.GoObjectBuilderService().Menu().Delete(ctx, &pbo.MenuPrimaryKey{
		Id:        menu.GetId(),
		ProjectId: targetResourceEnvID,
		EnvId:     targetResourceEnvID,
	}); err != nil {
		return fmt.Errorf("delete target menu %s: %w", menu.GetLabel(), err)
	}
	return nil
}

func (h *HandlerV1) copyTemplateMenuTree(ctx context.Context, targetService servicepkg.ServiceManagerI, menu *pbo.MenuTree, targetResourceEnvID, parentID string, keptTargetMenuIDs map[string]bool) error {
	if keptTargetMenuIDs[menu.GetId()] {
		for _, child := range menu.GetChildren() {
			if err := h.copyTemplateMenuTree(ctx, targetService, child, targetResourceEnvID, menu.GetId(), keptTargetMenuIDs); err != nil {
				return err
			}
		}
		return nil
	}

	created, err := targetService.GoObjectBuilderService().Menu().Create(ctx, &pbo.CreateMenuRequest{
		Label:           menu.GetLabel(),
		Icon:            menu.GetIcon(),
		Type:            menu.GetType(),
		ProjectId:       targetResourceEnvID,
		ParentId:        parentID,
		MicrofrontendId: "",
		WebpageId:       menu.GetWebpageId(),
		Attributes:      menu.GetAttributes(),
		WikiId:          menu.GetWikiId(),
		IsVisible:       menu.GetIsVisible(),
		EnvId:           targetResourceEnvID,
		TableId:         menu.GetTableId(),
		LayoutId:        menu.GetLayoutId(),
		Id:              menu.GetId(),
	})
	if err != nil {
		return fmt.Errorf("create menu %s: %w", menu.GetLabel(), err)
	}

	for _, child := range menu.GetChildren() {
		if err = h.copyTemplateMenuTree(ctx, targetService, child, targetResourceEnvID, created.GetId(), keptTargetMenuIDs); err != nil {
			return err
		}
	}
	return nil
}

func isProtectedUgenTemplateMenu(id string) bool {
	return protectedUgenTemplateMenuIDs[id]
}

var protectedUgenTemplateMenuIDs = map[string]bool{
	"c57eedc3-a954-4262-a0af-376c65b5a284": true, // root
	"c57eedc3-a954-4262-a0af-376c65b5a282": true, // content
	"c57eedc3-a954-4262-a0af-376c65b5a280": true, // settings
	"c57eedc3-a954-4262-a0af-376c65b5a278": true, // analytics
	"c57eedc3-a954-4262-a0af-376c65b5a276": true, // pivot
	"c57eedc3-a954-4262-a0af-376c65b5a274": true, // report setting
	"7c26b15e-2360-4f17-8539-449c8829003f": true, // saved pivot
	"e96b654a-1692-43ed-89a8-de4d2357d891": true, // history pivot
	"a8de4296-c8c3-48d6-bef0-ee17057733d6": true, // user and permission
	"d1b3b349-4200-4ba9-8d06-70299795d5e6": true, // data
	"f7d1fa7d-b857-4a24-a18c-402345f65df8": true, // code
	"f313614f-f018-4ddc-a0ce-10a1f5716401": true, // resource
	"db4ffda3-7696-4f56-9f1f-be128d82ae68": true, // api
	"3b74ee68-26e3-48c8-bc95-257ca7d6aa5c": true, // profile setting
	"8a6f913a-e3d4-4b73-9fc0-c942f343d0b9": true, // files
	"744d63e6-0ab7-4f16-a588-d9129cf959d1": true, // wiki
	"9e988322-cffd-484c-9ed6-460d8701551b": true, // users
}

func (h *HandlerV1) publishTemplateMicrofrontend(ctx context.Context, projectName string, files []*pbo.McpProjectFiles, target *models.ProjectData, mainResourceEnvID, authToken string) (models.PublishAiMicroFrontendResponse, error) {
	pubFiles := make([]models.GitlabFileChange, 0, len(files))
	for _, file := range files {
		if file.GetPath() == "" {
			continue
		}
		pubFiles = append(pubFiles, models.GitlabFileChange{
			FilePath: file.GetPath(),
			Content:  sanitizeFileContent(file.GetContent()),
		})
	}
	if len(pubFiles) == 0 {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("no microfrontend files to publish")
	}

	publishBody := models.PublishAiMicroFrontendRequest{
		ProjectId:        target.UcodeProjectId,
		EnvironmentId:    target.EnvironmentId,
		Name:             slugify(projectName),
		Path:             uniqueMFEPath(),
		FrameworkType:    "REACT",
		Files:            pubFiles,
		McpProjectId:     target.McpProjectId,
		McpResourceEnvId: mainResourceEnvID,
	}

	pubBytes, err := json.Marshal(publishBody)
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("marshal publish request: %w", err)
	}

	pubURL := h.baseConf.GoFunctionServiceHost + h.baseConf.GoFunctionServiceHTTPPort +
		"/v2/functions/micro-frontend/publish-ai"

	pubReq, err := http.NewRequestWithContext(ctx, http.MethodPost, pubURL, bytes.NewReader(pubBytes))
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("build publish request: %w", err)
	}
	pubReq.Header.Set("Content-Type", "application/json")
	pubReq.Header.Set("Authorization", authToken)

	httpClient := &http.Client{Timeout: 2 * time.Minute}
	pubResp, err := httpClient.Do(pubReq)
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("publish-ai call: %w", err)
	}
	defer pubResp.Body.Close()

	pubRespBytes, err := io.ReadAll(pubResp.Body)
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("read publish-ai response: %w", err)
	}
	if pubResp.StatusCode >= 400 {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("publish-ai returned %d: %s", pubResp.StatusCode, string(pubRespBytes))
	}

	var result models.PublishAiMicroFrontendResponse
	if err = json.Unmarshal(pubRespBytes, &result); err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("parse publish-ai response: %w", err)
	}
	log.Printf("[ugen-template] published %d files to project %s", len(pubFiles), target.UcodeProjectId)
	return result, nil
}

func skipUgenTemplateTable(slug string) bool {
	switch slug {
	case "role", "client_type", "person", "sms_template":
		return true
	default:
		return false
	}
}

func skipUgenTemplateField(slug, fieldType string, isLoginTable bool) bool {
	switch slug {
	case "guid", "created_at", "updated_at", "deleted_at", "folder_id", "user_id_auth":
		return true
	}
	if isLoginTable {
		switch slug {
		case "login", "password", "phone", "email", "tin", "last_activity", "client_type_id", "role_id":
			return true
		}
	}
	switch fieldType {
	case "LOOKUP", "LOOKUPS":
		return true
	default:
		return false
	}
}

func skipUgenTemplateRelation(table *pbo.Table, relation *pbo.CreateRelationRequest) bool {
	if !table.GetIsLoginTable() {
		return false
	}
	switch relation.GetTableTo() {
	case "client_type", "role":
		return true
	}
	switch relation.GetFieldFrom() {
	case "client_type_id", "role_id":
		return true
	default:
		return false
	}
}
