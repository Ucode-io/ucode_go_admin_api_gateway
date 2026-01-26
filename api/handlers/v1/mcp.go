package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// ====================  MCP HANDLERS  ====================

func (h *HandlerV1) MCPCall(c *gin.Context) {
	var (
		req           models.MCPRequest
		projectId     any
		environmentId any

		ok bool
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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

	apiKeys, err := h.authService.ApiKey().GetList(c.Request.Context(), &as.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     projectId.(string),
		Limit:         1,
		Offset:        0,
	})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if len(apiKeys.Data) < 1 {
		h.HandleResponse(c, status_http.InvalidArgument, "Api key not found")
		return
	}

	if req.Method == "" {
		req.Method = "project"
	}

	content, message, err := helper.BuildBackendPrompt(
		models.BackendPromptRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			Method:        req.Method,
			APIKey:        apiKeys.GetData()[0].GetAppId(),
			UserPrompt:    req.Prompt,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = h.sendAnthropicBackend(content)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, message)
}

func (h *HandlerV1) MCPGenerateFrontend(c *gin.Context) {
	var (
		req           models.MCPRequest
		projectId     any
		environmentId any

		ok bool
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if resource.ResourceType != pb.ResourceType_POSTGRESQL {
		h.HandleResponse(c, status_http.InvalidArgument, "resource type not supported")
		return
	}

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	apiKeys, err := h.authService.ApiKey().GetList(c.Request.Context(), &as.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     projectId.(string),
		Limit:         1,
		Offset:        0,
	})

	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if len(apiKeys.Data) < 1 {
		h.HandleResponse(c, status_http.InvalidArgument, "Api key not found")
		return
	}

	go func() {
		content, _, err := helper.BuildBackendPrompt(
			models.BackendPromptRequest{
				ProjectId:     projectId.(string),
				EnvironmentId: environmentId.(string),
				Method:        req.Method,
				APIKey:        apiKeys.GetData()[0].GetAppId(),
				UserPrompt:    req.Prompt,
			},
		)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		_, err = h.sendAnthropicBackend(content)
		if err != nil {
			log.Println("error in generating backend: " + err.Error())
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}()

	userPrompt := helper.BuildFrontendGeneratePrompt(
		models.FrontendPromptRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			APIKey:        apiKeys.GetData()[0].GetAppId(),
			UserPrompt:    req.Prompt,
			BaseURL:       "https://admin-api.ucode.run",
		},
	)

	project, err := h.generateFrontendProject(userPrompt)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "AI Generation Failed: "+err.Error())
		return
	}

	var (
		saveProject = pbo.CreateMcpProjectReqeust{
			ResourceEnvId: resource.ResourceEnvironmentId,
			Title:         project.ProjectName,
			Description:   "Generated by ucode with claude-sonnet-4-5",
		}

		projectFiles  []*pbo.McpProjectFiles
		projectEnv, _ = helperFunc.ConvertMapToStruct(project.Env)
	)

	for _, file := range project.Files {
		fileGraph, _ := project.FileGraph[file.Path].(map[string]any)
		fileGraphStruct, _ := helperFunc.ConvertMapToStruct(fileGraph)

		projectFiles = append(projectFiles, &pbo.McpProjectFiles{
			Path:      file.Path,
			Content:   file.Content,
			FileGraph: fileGraphStruct,
		})
	}

	saveProject.ProjectEnv = projectEnv
	saveProject.ProjectFiles = projectFiles

	response, err := services.GoObjectBuilderService().McpProject().CreateMcpProject(context.Background(), &saveProject)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) MCPUpdateFrontend(c *gin.Context) {
	var (
		request      models.MCPRequest
		mcpProjectId = c.Param("mcp_project_id")

		projectId     any
		environmentId any
		ok            bool
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if resource.ResourceType != pb.ResourceType_POSTGRESQL {
		h.HandleResponse(c, status_http.InvalidArgument, "resource type not supported")
		return
	}

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	projectFiles, err := services.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		c.Request.Context(),
		&pbo.McpProjectId{
			Id:            mcpProjectId,
			ResourceEnvId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		filesGraphMap = make(map[string]any)
		filesMap      = make(map[string]string)
	)

	for _, file := range projectFiles.ProjectFiles {
		filesGraphMap[file.Path] = file.FileGraph.AsMap()
		filesMap[file.Path] = file.Content
	}

	fmt.Println("========== STEP 1: ANALYSIS ==========")

	var (
		analyzeReq = models.AnalyzeFrontendPromptRequest{
			UserRequest: request.Prompt,
			FileGraph:   filesGraphMap,
			ProjectName: projectFiles.Title,
			Context:     request.Context,
		}

		filesToUpdate []models.ProjectFile
	)

	analysisPrompt, err := helper.BuildFrontendAnalyzePrompt(analyzeReq)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	analysis, err := h.analyzeProject(analysisPrompt)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	fmt.Printf("Analysis: modify=%d, create=%d, delete=%d\n",
		len(analysis.FilesToModify),
		len(analysis.NewFilesNeeded),
		len(analysis.FilesToDelete))

	if len(analysis.FilesToModify) == 0 && len(analysis.NewFilesNeeded) == 0 && len(analysis.FilesToDelete) == 0 {
		h.HandleResponse(c, status_http.OK, "Nothing to update")
		return
	}

	fmt.Println("========== STEP 2: PREPARING FILES ==========")

	for _, fileToMod := range analysis.FilesToModify {
		if content, exists := filesMap[fileToMod.Path]; exists {
			filesToUpdate = append(filesToUpdate, models.ProjectFile{
				Path:    fileToMod.Path,
				Content: content,
			})
			fmt.Printf("  - Prepared: %s\n", fileToMod.Path)
		}
	}

	if len(filesToUpdate) == 0 && len(analysis.NewFilesNeeded) == 0 {
		h.HandleResponse(c, status_http.OK, "Nothing to update")
		return
	}

	fmt.Println("========== STEP 3: UPDATE REQUEST ==========")

	var (
		updateReq = models.UpdateFrontendPromptRequest{
			UserRequest:    request.Prompt,
			FilesToUpdate:  filesToUpdate,
			AnalysisResult: *analysis,
			ProjectName:    projectFiles.Title,
			Context:        request.Context,
		}

		graphUpdate     any
		mcpProjectFiles []*pbo.McpProjectFiles
	)

	updatePrompt, err := helper.BuildFrontendUpdatePrompt(updateReq)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	update, err := h.updateProject(updatePrompt)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	fmt.Printf("Update: updated=%d, new=%d, deleted=%d\n",
		len(update.UpdatedFiles),
		len(update.NewFiles),
		len(update.DeletedFiles))

	for _, file := range update.UpdatedFiles {
		filesMap[file.Path] = file.Content
		if graphUpdate, ok = update.FileGraphUpdates[file.Path]; ok {
			filesGraphMap[file.Path] = graphUpdate
		}
	}

	for _, file := range update.NewFiles {
		filesMap[file.Path] = file.Content
		if graphUpdate, ok = update.FileGraphUpdates[file.Path]; ok {
			filesGraphMap[file.Path] = graphUpdate
		}
	}

	for path, content := range filesMap {
		fileGraph, _ := filesGraphMap[path].(map[string]any)
		fileGraphStruct, _ := helperFunc.ConvertMapToStruct(fileGraph)

		mcpProjectFiles = append(mcpProjectFiles, &pbo.McpProjectFiles{
			ProjectId: mcpProjectId,
			Path:      path,
			Content:   content,
			FileGraph: fileGraphStruct,
		})
	}

	var saveProject = pbo.McpProject{
		ResourceEnvId: resource.ResourceEnvironmentId,
		Id:            mcpProjectId,
		ProjectFiles:  mcpProjectFiles,
	}

	_, err = services.GoObjectBuilderService().McpProject().UpdateMcpProject(context.Background(), &saveProject)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, update)
}

// ==================== Send Anthropic Request methods ====================

func (h *HandlerV1) sendAnthropicBackend(content string) (string, error) {
	return h.callAnthropicAPI(
		models.AnthropicRequest{
			Model:     h.baseConf.ClaudeModel,
			MaxTokens: h.baseConf.MaxTokens,
			System:    helper.SystemPromptBackend,
			Messages: []models.ChatMessage{
				{
					Role:    "user",
					Content: content,
				},
			},
			MCPServers: []models.MCPServer{
				{
					Type: "url",
					URL:  h.baseConf.MCPServerURL,
					Name: "ucode",
				},
			},
			Tools: []models.MCPTool{
				{
					Type:          "mcp_toolset",
					MCPServerName: "ucode",
				},
			},
		},
		420*time.Second,
	)
}

func (h *HandlerV1) generateFrontendProject(userPrompt string) (*models.GeneratedProject, error) {
	var body = models.AnthropicRequest{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: h.baseConf.MaxTokens,
		System:    helper.SystemPromptGenerateFrontend,
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MCPServers: []models.MCPServer{
			{
				Type: "url",
				URL:  h.baseConf.MCPServerURL,
				Name: "ucode",
			},
		},
		Tools: []models.MCPTool{
			{
				Type:          "mcp_toolset",
				MCPServerName: "ucode",
			},
		},
	}

	respText, err := h.callAnthropicAPI(body, 420*time.Second)
	if err != nil {
		return nil, err
	}

	var (
		apiResponse struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}

		project models.GeneratedProject
	)

	if err = json.Unmarshal([]byte(respText), &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResponse.Content) == 0 {
		return nil, fmt.Errorf("empty content in response")
	}

	cleanedText := helper.CleanJSONResponse(apiResponse.Content[0].Text)

	if err = json.Unmarshal([]byte(cleanedText), &project); err != nil {
		return nil, fmt.Errorf("failed to parse project JSON: %w", err)
	}

	return &project, nil
}

func (h *HandlerV1) analyzeProject(userPrompt string) (*models.AnalysisResult, error) {
	var body = models.AnthropicRequest{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: 5000,
		System:    helper.SystemPromptAnalyzeFrontend,
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	respText, err := h.callAnthropicAPI(body, 120*time.Second)
	if err != nil {
		return nil, err
	}

	var (
		apiResponse struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}

		analysis models.AnalysisResult
	)

	if err = json.Unmarshal([]byte(respText), &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResponse.Content) == 0 {
		return nil, fmt.Errorf("empty content")
	}

	cleanedText := helper.CleanJSONResponse(apiResponse.Content[0].Text)

	if err = json.Unmarshal([]byte(cleanedText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	return &analysis, nil
}

func (h *HandlerV1) updateProject(userPrompt string) (*models.UpdateResult, error) {
	body := models.AnthropicRequest{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: h.baseConf.MaxTokens,
		System:    helper.SystemPromptUpdateFrontend,
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MCPServers: []models.MCPServer{
			{
				Type: "url",
				URL:  h.baseConf.MCPServerURL,
				Name: "ucode",
			},
		},
		Tools: []models.MCPTool{
			{
				Type:          "mcp_toolset",
				MCPServerName: "ucode",
			},
		},
	}

	respText, err := h.callAnthropicAPI(body, 420*time.Second)
	if err != nil {
		return nil, err
	}

	var (
		apiResponse struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}

		update models.UpdateResult
	)

	if err = json.Unmarshal([]byte(respText), &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResponse.Content) == 0 {
		return nil, fmt.Errorf("empty content")
	}

	cleanedText := helper.CleanJSONResponse(apiResponse.Content[0].Text)

	if err = json.Unmarshal([]byte(cleanedText), &update); err != nil {
		return nil, fmt.Errorf("failed to parse update: %w", err)
	}

	return &update, nil
}

// ==================== SHARED ANTHROPIC CALLER ====================

func (h *HandlerV1) callAnthropicAPI(body models.AnthropicRequest, timeout time.Duration) (string, error) {

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.baseConf.AnthropicBaseAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", h.baseConf.AnthropicAPIKey)
	req.Header.Set("anthropic-version", h.baseConf.AnthropicVersion)
	req.Header.Set("anthropic-beta", h.baseConf.AnthropicBeta)

	var client = &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	log.Println("MCP RESPONSE>>>>", string(respBytes))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	return string(respBytes), nil
}
