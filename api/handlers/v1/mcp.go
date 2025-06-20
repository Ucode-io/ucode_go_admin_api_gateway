package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

type MCPRequest struct {
	ProjectType      string `json:"project_type"`
	ManagementSystem string `json:"management_system"`
	Industry         string `json:"industry"`
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
	var req MCPRequest

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

	// err := sendAnthropicRequest(req.ProjectType, req.ManagementSystem, req.Industry, projectId.(string), environmentId.(string))
	// if err != nil {
	// 	h.handleResponse(c, status_http.InternalServerError, err.Error())
	// 	return
	// }

	h.handleResponse(c, status_http.OK, nil)
}

func sendAnthropicRequest(projectType, managementSystem, industry, projectId, envId string) error {
	url := config.ANTHROPIC_BASE_API_URL

	// Construct the request body
	body := RequestBody{
		Model:     config.CLAUDE_MODEL,
		MaxTokens: config.MAX_TOKENS,
		Messages: []Message{
			{
				Role: "user",
				Content: fmt.Sprintf(`Task: Generate a DBML schema for an %s %s tailored for the %s industry, using PostgreSQL.

📌 Requirements:
 • Include the industry specific functional areas:
 • Don't add Users & Roles tables
 • Use proper ref: keys for relations
 • For fields like status or type, use realistic Enum definitions in proper DBML syntax. Example: Enum "tax_type" { "Fixed" "Percentage" }. Do not use comments or inline values. Use separate Enum blocks with clearly defined, realistic values.
 • Optional: use camelCase or snake_case consistently if preferred
 • Don't add any indexes 
 • Show references only in the format: Ref fk_name:table1.column1 < table2.column2. Do not include quotes or any additional options like [delete: cascade].
 • Don't incluede any quotes

🛠️ Style:
 • Use descriptive field names and comments where needed
 • Follow the design principles of relational databases
 • Ensure consistency with other systems like ProjectManagement, Payroll, and CRM

Get the current DBML schema for the project with project-id = %s and environment-id = %s.
Then, prepare a new DBML schema that excludes all existing tables from the current schema.
Finally, execute the new DBML schema using the dbml_to_ucode tool.`, projectType, managementSystem, industry, projectId, envId),
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
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", config.ANTHROPIC_API_KEY)
	req.Header.Set("anthropic-version", config.ANTHROPIC_VERSION)
	req.Header.Set("anthropic-beta", config.ANTHROPIC_BETA)

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println("Status Code:", resp.StatusCode)
	return nil
}
