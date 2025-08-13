package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

type MCPRequest struct {
	ProjectType      string   `json:"project_type"`
	ManagementSystem []string `json:"management_system"`
	Industry         string   `json:"industry"`
	Method           string   `json:"method"`
	Prompt           string   `json:"prompt"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MCPServer struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

type RequestBody struct {
	Model     string      `json:"model"`
	MaxTokens int         `json:"max_tokens"`
	Messages  []Message   `json:"messages"`
	MCPServer []MCPServer `json:"mcp_servers"`
}

func (h *HandlerV1) MCPCall(c *gin.Context) {
	var (
		req     MCPRequest
		content string
		message string
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return
	}

	apiKeys, err := h.authService.ApiKey().GetList(c.Request.Context(), &as.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     projectId.(string),
		Limit:         1,
		Offset:        0,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if len(apiKeys.Data) < 1 {
		h.handleResponse(c, status_http.InvalidArgument, "Api key not found")
		return
	}

	apiKey := apiKeys.GetData()[0].GetAppId()

	if req.Method == "" {
		req.Method = "project"
	}

	switch req.Method {
	case "project":
		content = fmt.Sprintf(`
1. Retrieve the current DBML schema using: project-id = %s  environment-id = %s
2. Generate a DBML schema for an %s %s tailored for the %s industry, using PostgreSQL. **excluding all existing tables from the current schema**.
📌 Requirements:
 • Include the industry specific functional areas:
 • Do NOT include Users or Roles tables.
 • Use proper Ref definitions for relationships, in this format: Ref fk_name: table1.column1 < table2.column2
 • For fields like status or type, use realistic Enum definitions in proper DBML syntax. Example: Enum "tax_type" { "Fixed" "Percentage" }. Use separate Enum blocks with clearly defined, realistic values and wrap all enum values in double quotes to ensure compatibility.
 • Optional: use camelCase or snake_case consistently if preferred
 • Do not include indexes 
 • Do not include quotes, additional options (e.g., [delete: cascade]) and default values.
 • Use descriptive field names and don't use comments
 • Do not include comments anywhere in the schema.
 • Follow relational design principles and ensure consistency with systems like ProjectManagement, Payroll, and CRM.

3. Organize the new tables into **menus** by their functional purpose.
4. Provide a view_fields JSON that maps each table to its most important column: Example: { "customer": "name" }
5. Execute the new DBML schema using the dbml_to_ucode tool:
   Use X-API-KEY = %s  
⚠️ Attempt any operation **once only** — do not retry on failure. If the dbml_to_ucode tool returns an error, **end the operation immediately**.
`, projectId.(string), environmentId.(string), req.ProjectType, strings.Join(req.ManagementSystem, "/"), "IT", apiKey)

		message = fmt.Sprintf("Your request for %s %s has been successfully processed.", req.ProjectType, strings.Join(req.ManagementSystem, ", "))
	case "table":
		content = req.Prompt
		content += fmt.Sprintf(`
x-api-key = %s
		`, apiKey)
		message = "The table has been successfully updated."

	}

	resp, err := sendAnthropicRequest(content)
	fmt.Println("************ MCP Response ************", resp)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, "Your request could not be processed.")
		return
	}

	h.handleResponse(c, status_http.OK, message)
}

func sendAnthropicRequest(content string) (string, error) {
	url := config.ANTHROPIC_BASE_API_URL

	// Construct the request body
	body := RequestBody{
		Model:     config.CLAUDE_MODEL,
		MaxTokens: config.MAX_TOKENS,
		Messages: []Message{
			{
				Role:    "user",
				Content: content,
			},
		},
		MCPServer: []MCPServer{
			{
				Type: "url",
				URL:  config.MCP_SERVER_URL,
				Name: "ucode",
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", config.ANTHROPIC_API_KEY)
	req.Header.Set("anthropic-version", config.ANTHROPIC_VERSION)
	req.Header.Set("anthropic-beta", config.ANTHROPIC_BETA)

	client := &http.Client{Timeout: 300 * time.Second}
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
