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

	RequestBody struct {
		Model      string        `json:"model"`
		MaxTokens  int           `json:"max_tokens"`
		Messages   []UserMessage `json:"messages"`
		MCPServers []MCPServer   `json:"mcp_servers,omitempty"`
		Tools      []Tool        `json:"tools,omitempty"`
	}

	RequestBodyAnthropic struct {
		Model      string        `json:"model"`
		MaxTokens  int           `json:"max_tokens"`
		System     string        `json:"system,omitempty"`
		Messages   []UserMessage `json:"messages"`
		MCPServers []MCPServer   `json:"mcp_servers,omitempty"`
		Tools      []Tool        `json:"tools,omitempty"`
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

	mcpResp, err := h.sendAnthropicRequest(content)
	fmt.Println("************ MCP Response ************", mcpResp)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, message+"")
	return
}

func (h *HandlerV1) sendAnthropicRequest(content string) (string, error) {
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

	frontendUserPrompt := helper.FrontendGenerateUserPrompt(models.GeneratePromptMCP{
		ProjectId:     projectId.(string),
		EnvironmentId: environmentId.(string),
		APIKey:        apiKeys.GetData()[0].GetAppId(),
		UserPrompt:    req.Prompt,
		BaseURL:       "https://admin-api.ucode.run",
	})

	project, err := h.sendAnthropicRequestFront(frontendUserPrompt)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "AI Generation Failed: "+err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, project)
}

func (h *HandlerV1) sendAnthropicRequestFront(userPrompt string) (*GeneratedProject, error) {
	var body = RequestBodyAnthropic{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: h.baseConf.MaxTokens,
		System:    helper.ClaudeFrontendSystemPrompt,
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
		cleanedText  = cleanJSONResponse(responseText)
		project      GeneratedProject
	)

	if err := json.Unmarshal([]byte(cleanedText), &project); err != nil {
		return nil, fmt.Errorf("failed to parse generated project JSON: %w\nResponse: %s", err, cleanedText)
	}

	return &project, nil
}

func cleanJSONResponse(text string) string {

	text = string(bytes.TrimSpace([]byte(text)))
	if bytes.HasPrefix([]byte(text), []byte("```json")) {
		text = string(bytes.TrimPrefix([]byte(text), []byte("```json")))
		text = string(bytes.TrimPrefix([]byte(text), []byte("```")))
	}
	if bytes.HasSuffix([]byte(text), []byte("```")) {
		text = string(bytes.TrimSuffix([]byte(text), []byte("```")))
	}

	return string(bytes.TrimSpace([]byte(text)))
}

type GeneratedProject struct {
	ProjectName string `json:"project_name"`
	Files       []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"files"`
	Env map[string]string `json:"env"`
}
