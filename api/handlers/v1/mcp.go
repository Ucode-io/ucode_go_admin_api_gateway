package v1

import (
	"bytes"
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
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// ==================== MODELS ====================
type (
	MCPRequest struct {
		ProjectType      string   `json:"project_type"`
		ManagementSystem []string `json:"management_system"`
		Industry         string   `json:"industry"`
		Method           string   `json:"method"`
		Prompt           string   `json:"prompt"`
	}

	UserMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	MCPServer struct {
		Type               string `json:"type"`
		URL                string `json:"url"`
		Name               string `json:"name"`
		AuthorizationToken string `json:"authorization_token,omitempty"`
	}

	Tool struct {
		Type          string `json:"type"`
		MCPServerName string `json:"mcp_server_name,omitempty"`
	}

	RequestBodyAnthropic struct {
		Model      string        `json:"model"`
		MaxTokens  int           `json:"max_tokens"`
		System     string        `json:"system,omitempty"`
		Messages   []UserMessage `json:"messages"`
		MCPServers []MCPServer   `json:"mcp_servers,omitempty"`
		Tools      []Tool        `json:"tools,omitempty"`
	}

	GeneratedProject struct {
		ProjectName string `json:"project_name"`
		Files       []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
		FileGraph map[string]any    `json:"file_graph"`
		Env       map[string]string `json:"env"`
	}

	UpdateProjectReq struct {
		ProjectName string `json:"project_name"`
		Files       []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
		FileGraph   map[string]any    `json:"file_graph"`
		Env         map[string]string `json:"env"`
		UserPrompt  string            `json:"user_prompt"`
		UserContext []string          `json:"user_context"`
	}

	FileToModify struct {
		Path       string `json:"path"`
		Reason     string `json:"reason"`
		ChangeType string `json:"change_type"`
		Priority   string `json:"priority"`
	}

	UpdateResponse struct {
		UpdatedFiles     []UpdatedFile  `json:"updated_files"`
		NewFiles         []NewFile      `json:"new_files"`
		DeletedFiles     []string       `json:"deleted_files"`
		FileGraphUpdates map[string]any `json:"file_graph_updates"`
		IntegrationNotes []string       `json:"integration_notes"`
	}

	UpdatedFile struct {
		Path          string `json:"path"`
		Content       string `json:"content"`
		ChangeSummary string `json:"change_summary"`
	}

	NewFile struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Purpose string `json:"purpose"`
	}
)

func (h *HandlerV1) MCPCall(c *gin.Context) {
	var (
		req     MCPRequest
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

	var generatePromptRequest = models.GeneratePromptMCP{
		ProjectId:     projectId.(string),
		EnvironmentId: environmentId.(string),
		Method:        req.Method,
		APIKey:        apiKey,
		UserPrompt:    req.Prompt,
	}

	content, message, err = helper.GenerateBackendMCPPrompt(generatePromptRequest)
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
		userMessage = UserMessage{
			Role:    "user",
			Content: content,
		}

		body = RequestBodyAnthropic{
			Model:     h.baseConf.ClaudeModel,
			MaxTokens: h.baseConf.MaxTokens,
			System:    helper.McpBackendSystemPrompt,
			Messages:  []UserMessage{userMessage},
			MCPServers: []MCPServer{
				{
					Type: "url",
					URL:  h.baseConf.MCPServerURL,
					Name: "ucode",
				},
			},
			Tools: []Tool{
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
		req           MCPRequest
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

	frontendUserPrompt := helper.UserPromptFrontendGenerate(
		models.GeneratePromptMCP{
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

	h.HandleResponse(c, status_http.OK, project)
}

func (h *HandlerV1) MCPUpdateFrontend(c *gin.Context) {
	var (
		request       UpdateProjectReq
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

	// ========== ЭТАП 1: АНАЛИЗ ==========
	fmt.Println("========== STEP 1: ANALYSIS ==========")
	fmt.Printf("Analyzing project '%s' for request: %s\n", request.ProjectName, request.UserPrompt)

	analysisPrompt, err := helper.UserPromptAnalyseUpdateFrontend(
		models.AnalysisRequest{
			UserRequest: request.UserPrompt,
			FileGraph:   request.FileGraph,
			ProjectName: request.ProjectName,
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

	var (
		fileMap       = make(map[string]string)
		filesToUpdate = make([]models.FileContent, 0)
	)

	for _, file := range request.Files {
		fileMap[file.Path] = file.Content
	}

	for _, fileToMod := range analysis.FilesToModify {
		if content, exists := fileMap[fileToMod.Path]; exists {
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

	var updateReq = models.UpdateRequest{
		UserRequest:    request.UserPrompt,
		FilesToUpdate:  filesToUpdate,
		AnalysisResult: *analysis,
		ProjectName:    request.ProjectName,
	}

	updatePrompt, err := helper.UserPromptUpdateFrontend(updateReq)
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

	h.HandleResponse(c, status_http.OK, update)
}

func (h *HandlerV1) sendAnthropicRequestFront(userPrompt string) (*GeneratedProject, error) {
	var body = RequestBodyAnthropic{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: h.baseConf.MaxTokens,
		System:    helper.ClaudeSystemPromptGenerateFrontend,
		Messages: []UserMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MCPServers: []MCPServer{
			{
				Type: "url",
				URL:  h.baseConf.MCPServerURL,
				Name: "ucode",
			},
		},
		Tools: []Tool{
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
		project      GeneratedProject
	)

	if err = json.Unmarshal([]byte(cleanedText), &project); err != nil {
		return nil, fmt.Errorf("failed to parse generated project JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &project, nil
}

// ==================== ANTHROPIC API FUNCTIONS ====================

func SendAnalysisRequest(conf config.BaseConfig, userPrompt string) (*models.AnalysisResponse, error) {
	var body = RequestBodyAnthropic{
		Model:     conf.ClaudeModel,
		MaxTokens: 5000,
		System:    helper.ClaudeSystemPromptAnalysisUpdateFrontend,
		Messages: []UserMessage{
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

	var analysis models.AnalysisResponse
	if err := json.Unmarshal([]byte(cleanedText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &analysis, nil
}

func SendUpdateRequest(conf config.BaseConfig, userPrompt string) (*UpdateResponse, error) {
	var body = RequestBodyAnthropic{
		Model:     conf.ClaudeModel,
		MaxTokens: conf.MaxTokens,
		System:    helper.ClaudeSystemPromptUpdateFrontend,
		Messages: []UserMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MCPServers: []MCPServer{
			{
				Type: "url",
				URL:  conf.MCPServerURL,
				Name: "ucode",
			},
		},
		Tools: []Tool{
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

	cleanedText := helper.CleanJSONResponse(apiResponse.Content[0].Text)

	var update UpdateResponse
	if err := json.Unmarshal([]byte(cleanedText), &update); err != nil {
		return nil, fmt.Errorf("failed to parse update JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &update, nil
}
