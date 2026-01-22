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
	structpb "github.com/golang/protobuf/ptypes/struct"
)

// ==================== MODELS ====================
type ()

func (h *HandlerV1) MCPCall(c *gin.Context) {
	var (
		req     models.MCPRequest
		content string
		message string
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return
	}

	environmentId, ok := c.Get("environment_id")
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

	apiKey := apiKeys.GetData()[0].GetAppId()

	if req.Method == "" {
		req.Method = "project"
	}

	var generatePromptRequest = models.GenerateMcpPromptReq{
		ProjectId:     projectId.(string),
		EnvironmentId: environmentId.(string),
		Method:        req.Method,
		APIKey:        apiKey,
		UserPrompt:    req.Prompt,
	}

	content, message, err = helper.GenerateBackendUserPrompt(generatePromptRequest)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	mcpResp, err := h.sendAnthropicRequestBackend(content)
	fmt.Println("************ MCP Response ************", mcpResp)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, message+"")
	return
}

func (h *HandlerV1) sendAnthropicRequestBackend(content string) (string, error) {
	var (
		userMessage = models.McpUserMessage{
			Role:    "user",
			Content: content,
		}

		body = models.RequestBodyAnthropic{
			Model:     h.baseConf.ClaudeModel,
			MaxTokens: h.baseConf.MaxTokens,
			System:    helper.McpBackendSystemPrompt,
			Messages:  []models.McpUserMessage{userMessage},
			MCPServers: []models.MCPServer{
				{
					Type: "url",
					URL:  h.baseConf.MCPServerURL,
					Name: "ucode",
				},
			},
			McpTools: []models.McpTool{
				{
					Type:          "mcp_toolset",
					MCPServerName: "ucode",
				},
			},
		}
	)

	jsonBody, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.baseConf.AnthropicBaseAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", h.baseConf.AnthropicAPIKey)
	req.Header.Set("anthropic-version", h.baseConf.AnthropicVersion)
	req.Header.Set("anthropic-beta", h.baseConf.AnthropicBeta)

	client := &http.Client{Timeout: 420 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return string(respByte), fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return string(respByte), nil
}

func (h *HandlerV1) MCPGenerateFrontend(c *gin.Context) {
	var (
		req           models.MCPRequest
		projectId     any
		environmentId any
		ok            bool
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

	frontendUserPrompt := helper.GenerateFrontendUserPrompt(
		models.GenerateMcpPromptReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			APIKey:        apiKeys.GetData()[0].GetAppId(),
			UserPrompt:    req.Prompt,
			BaseURL:       "https://admin-api.ucode.run",
		},
	)

	project, err := h.sendAnthropicRequestFront(frontendUserPrompt)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "AI Generation Failed: "+err.Error())
		return
	}

	var (
		saveProject = pbo.CreateMcpProjectReqeust{
			ResourceEnvId: resource.ResourceEnvironmentId,
			Title:         project.ProjectName,
			Description:   "this project generated by ucode whit claude-sonnet-4-5",
		}

		projectFiles    []*pbo.McpProjectFiles
		fileGraph       = make(map[string]any)
		fileGraphStruct *structpb.Struct
	)

	for _, file := range project.Files {

		if fileGraph, ok = project.FileGraph[file.Path].(map[string]any); ok {
			log.Println("error converting file graph")
		}

		fileGraphStruct, err = helperFunc.ConvertMapToStruct(fileGraph)
		if err != nil {
			log.Println("error converting file graph")
		}

		reqFiles := &pbo.McpProjectFiles{
			FilePath:    file.Path,
			FileContent: file.Content,
			FileGraph:   fileGraphStruct,
		}

		projectFiles = append(projectFiles, reqFiles)
	}

	projectEnv, err := helperFunc.ConvertMapToStruct(project.Env)
	if err != nil {
		log.Println("error converting file graph")
	}

	saveProject.ProjectEnv = projectEnv
	saveProject.ProjectFiles = projectFiles

	response, err := services.GoObjectBuilderService().McpProject().CreateMcpProject(context.Background(), &saveProject)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		log.Printf("error creating mcp project: %v", err)
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) MCPUpdateFrontend(c *gin.Context) {
	var (
		request       models.MCPRequest
		mcpProjectId  = c.Param("mcp_project_id")
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
		filesToUpdate = make([]models.FileContent, 0)
		filesMap      = make(map[string]string)
	)

	for _, file := range projectFiles.ProjectFiles {
		filesGraphMap[file.FilePath] = file.FileGraph.AsMap()
		filesMap[file.FilePath] = file.FileContent
	}

	// ========== ЭТАП 1: АНАЛИЗ ==========
	fmt.Println("========== STEP 1: ANALYSIS ==========")
	fmt.Printf("Analyzing project '%s' for request: %s\n", projectFiles.Title, request.Prompt)

	analysisPrompt, err := helper.GenerateAnalyseFrontendUserPrompt(
		models.GenerateAnalysisPromptReq{
			UserRequest: request.Prompt,
			FileGraph:   filesGraphMap,
			ProjectName: projectFiles.Title,
		},
	)
	if err != nil {
		log.Println("AnalysisUserPrompt failed:", err)
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	analysis, err := SendAnalysisRequest(h.baseConf, analysisPrompt)
	if err != nil {
		log.Println("SendAnalysisRequest failed:", err)
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	fmt.Printf("Analysis completed. Files to modify: %d, Files to create: %d, Files to delete: %d\n",
		len(analysis.FilesToModify),
		len(analysis.NewFilesNeeded),
		len(analysis.FilesToDelete),
	)

	if len(analysis.FilesToModify) == 0 && len(analysis.NewFilesNeeded) == 0 && len(analysis.FilesToDelete) == 0 {
		log.Println("Nothing to update")
		h.HandleResponse(c, status_http.OK, "OK")
		return
	}

	fmt.Println("========== STEP 2: PREPARING FILES ==========")

	for _, fileToMod := range analysis.FilesToModify {
		if content, exists := filesMap[fileToMod.Path]; exists {
			filesToUpdate = append(filesToUpdate, models.FileContent{
				Path:    fileToMod.Path,
				Content: content,
			})
			fmt.Printf("  - Prepared file: %s (Priority: %s)\n", fileToMod.Path, fileToMod.Priority)
		} else {
			fmt.Printf("  - WARNING: File not found in project: %s\n", fileToMod.Path)
		}
	}

	if len(filesToUpdate) == 0 && len(analysis.NewFilesNeeded) == 0 {
		log.Println("Nothing to update")
		h.HandleResponse(c, status_http.OK, "OK")
		return
	}

	fmt.Println("========== STEP 3: UPDATE REQUEST ==========")
	fmt.Printf("Sending %d files for update...\n", len(filesToUpdate))

	var updateReq = models.GenerateUpdatePromptReq{
		UserRequest:    request.Prompt,
		FilesToUpdate:  filesToUpdate,
		AnalysisResult: *analysis,
		ProjectName:    projectFiles.Title,
	}

	updatePrompt, err := helper.UpdateFrontendUserPrompt(updateReq)
	if err != nil {
		log.Println("UpdateUserPrompt failed:", err)
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	update, err := SendUpdateRequest(h.baseConf, updatePrompt)
	if err != nil {
		log.Println("SendUpdateRequest failed:", err)
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	fmt.Printf("Update completed. Updated files: %d, New files: %d, Deleted files: %d\n",
		len(update.UpdatedFiles),
		len(update.NewFiles),
		len(update.DeletedFiles),
	)

	for _, file := range update.UpdatedFiles {
		filesGraphMap[file.Path] = update.FileGraphUpdates[file.Path]

		filesMap[file.Path] = file.Content
		filesGraphMap[file.Path] = file
	}

	for _, file := range update.NewFiles {
		filesGraphMap[file.Path] = update.FileGraphUpdates[file.Path]

		filesMap[file.Path] = file.Content
		filesGraphMap[file.Path] = file

	}

	var (
		saveProject = pbo.McpProject{
			ResourceEnvId: resource.ResourceEnvironmentId,
			Id:            mcpProjectId,
		}

		mcpProjectFiles  []*pbo.McpProjectFiles
		filesGraphStruck *structpb.Struct
		fileGraph        = make(map[string]any)
	)

	for filesMapKey, filesMapValue := range filesMap {

		if fileGraph, ok = filesGraphMap[filesMapKey].(map[string]any); ok {
			filesGraphStruck, err = helperFunc.ConvertMapToStruct(fileGraph)
			if err != nil {
				log.Println("ConvertMapToStruct failed:", err)
			}
		}

		var file = pbo.McpProjectFiles{
			ProjectId:   mcpProjectId,
			FilePath:    filesMapKey,
			FileContent: filesMapValue,
			FileGraph:   filesGraphStruck,
		}

		mcpProjectFiles = append(mcpProjectFiles, &file)
	}

	saveProject.ProjectFiles = mcpProjectFiles

	_, err = services.GoObjectBuilderService().McpProject().UpdateMcpProject(context.Background(), &saveProject)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		log.Printf("error updating mcp project: %v", err)
	}

	h.HandleResponse(c, status_http.OK, update)
}

func (h *HandlerV1) sendAnthropicRequestFront(userPrompt string) (*models.FrontGeneratedProject, error) {
	var body = models.RequestBodyAnthropic{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: h.baseConf.MaxTokens,
		System:    helper.ClaudeSystemPromptGenerateFrontend,
		Messages: []models.McpUserMessage{
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
		McpTools: []models.McpTool{
			{
				Type:          "mcp_toolset",
				MCPServerName: "ucode",
			},
		},
	}

	jsonBody, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, h.baseConf.AnthropicBaseAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", h.baseConf.AnthropicAPIKey)
	request.Header.Set("anthropic-version", h.baseConf.AnthropicVersion)
	request.Header.Set("anthropic-beta", h.baseConf.AnthropicBeta)

	client := &http.Client{Timeout: 420 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Println("************ Anthropic response ************:", string(respBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}

	if err = json.Unmarshal(respBytes, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResponse.Content) == 0 {
		return nil, fmt.Errorf("empty content in response")
	}

	var (
		responseText = apiResponse.Content[0].Text
		cleanedText  = helper.CleanJSONResponse(responseText)
		project      models.FrontGeneratedProject
	)

	if err = json.Unmarshal([]byte(cleanedText), &project); err != nil {
		return nil, fmt.Errorf("failed to parse generated project JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &project, nil
}

// ==================== ANTHROPIC API FUNCTIONS ====================

func SendAnalysisRequest(conf config.BaseConfig, userPrompt string) (*models.AnalysedProjectResponse, error) {
	var body = models.RequestBodyAnthropic{
		Model:     conf.ClaudeModel,
		MaxTokens: 5000,
		System:    helper.ClaudeSystemPromptAnalysisUpdateFrontend,
		Messages: []models.McpUserMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	log.Println("MODEL:", conf.ClaudeModel)

	jsonBody, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		conf.AnthropicBaseAPIURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", conf.AnthropicAPIKey)
	request.Header.Set("anthropic-version", conf.AnthropicVersion)
	request.Header.Set("anthropic-beta", conf.AnthropicBeta)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Println("CLAUDE RESPONSE:", string(respBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err = json.Unmarshal(respBytes, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResponse.Content) == 0 {
		return nil, fmt.Errorf("empty content in response")
	}

	cleanedText := helper.CleanJSONResponse(apiResponse.Content[0].Text)

	var analysis models.AnalysedProjectResponse
	if err := json.Unmarshal([]byte(cleanedText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &analysis, nil
}

func SendUpdateRequest(conf config.BaseConfig, userPrompt string) (*models.McpUpdatedProject, error) {
	var body = models.RequestBodyAnthropic{
		Model:     conf.ClaudeModel,
		MaxTokens: conf.MaxTokens,
		System:    helper.ClaudeSystemPromptUpdateFrontend,
		Messages: []models.McpUserMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MCPServers: []models.MCPServer{
			{
				Type: "url",
				URL:  conf.MCPServerURL,
				Name: "ucode",
			},
		},
		McpTools: []models.McpTool{
			{
				Type:          "mcp_toolset",
				MCPServerName: "ucode",
			},
		},
	}

	jsonBody, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		conf.AnthropicBaseAPIURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", conf.AnthropicAPIKey)
	request.Header.Set("anthropic-version", conf.AnthropicVersion)
	request.Header.Set("anthropic-beta", conf.AnthropicBeta)

	client := &http.Client{Timeout: 420 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err = json.Unmarshal(respBytes, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResponse.Content) == 0 {
		return nil, fmt.Errorf("empty content in response")
	}

	var (
		cleanedText = helper.CleanJSONResponse(apiResponse.Content[0].Text)
		update      models.McpUpdatedProject
	)

	if err = json.Unmarshal([]byte(cleanedText), &update); err != nil {
		return nil, fmt.Errorf("failed to parse update JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &update, nil
}
