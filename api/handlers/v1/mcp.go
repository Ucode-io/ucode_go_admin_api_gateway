package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
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

	Message struct {
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
		Model      string      `json:"model"`
		MaxTokens  int         `json:"max_tokens"`
		Messages   []Message   `json:"messages"`
		MCPServers []MCPServer `json:"mcp_servers,omitempty"`
		Tools      []Tool      `json:"tools,omitempty"`
	}

	RequestBodyAnthropic struct {
		Model      string      `json:"model"`
		MaxTokens  int         `json:"max_tokens"`
		System     string      `json:"system,omitempty"`
		Messages   []Message   `json:"messages"`
		MCPServers []MCPServer `json:"mcp_servers,omitempty"`
		Tools      []Tool      `json:"tools,omitempty"`
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

	req.Prompt += ". containing about 10 tables and icons for each table."

	switch req.Method {
	case "project":
		content = fmt.Sprintf(`You are creating a complete u-code project from scratch. Analyze the user's request and determine:
- Project Type (CRM, ERP, E-commerce, Admin Panel, etc.)
- Management Systems needed (ProjectManagement, Payroll, Inventory, Sales, etc.)
- Industry (IT, Healthcare, Finance, Retail, etc.)

Then follow these steps EXACTLY:

STEP 1: Retrieve current DBML schema (if any exists)
- Use get_dbml tool with:
  project-id = %s
  environment-id = %s
  x-api-key = %s

STEP 2: Analyze user request and generate comprehensive DBML schema
User Request: "%s"

Based on the user's request, determine:
1. What type of project/system they need (CRM, ERP, Admin Panel, etc.)
2. What functional areas/modules are required
3. What industry/domain this is for
4. What tables, relationships, and data structures are needed

Generate a complete DBML schema that:
- Matches the user's requirements EXACTLY
- Uses PostgreSQL syntax
- **EXCLUDES all existing tables** from current schema (if retrieved in STEP 1)
- Includes industry-specific functional areas based on your analysis
- DO NOT include Users or Roles tables (they already exist in u-code)
- Use proper Ref definitions: Ref fk_name: table1.column1 < table2.column2
- For status/type fields, use realistic Enum definitions: Enum "status" { "Active" "Inactive" "Pending" }
- Use camelCase or snake_case consistently
- No indexes, no quotes, no cascade options, no default values
- Use descriptive field names, NO comments
- Follow relational design principles
- If user mentions specific number of menus/tables (e.g., "10 menus"), create exactly that many

STEP 3: Create tables directly using create_table tool
- For each table, call create_table with: label, slug (snake_case), parent_id = "%s", x-api-key
- IMPORTANT: Save the response from each create_table call - you need table_id and slug (collection) for next steps
- Example: create_table({label: "Customers", slug: "customers", parent_id: "%s", x_api_key: "%s"})
- Do NOT create folders, all tables are created at root level

STEP 4: Create fields for each table (IMPORTANT: use update_table, not create_field)
- IMPORTANT INSTRUCTION FOR FIELD CREATION:
  - Do NOT call the create_field tool under any circumstances for automated field creation.
  - Instead, always aggregate fields for a table and call update_table with a fields array and relations array.
  - If the plan would create several individual fields, combine them into a single update_table call per table (bulk create).
  - If update_table cannot perform a specific field creation (server returns error), STOP and report the error back (do not retry individual create_field).
  - Use create_field only if a human explicitly requests a single manual field creation and only after explicit confirmation.
- Then use update_table to add all fields and relations for each table.

- For each table created, add ALL necessary fields based on DBML schema
- Use update_table with: tableSlug (collection slug), xapikey, fields array, relations array
- Field types: SINGLE_LINE, TEXT, NUMBER, DATE, BOOLEAN, ENUM, etc.
- Add standard fields: id (auto), name/title (required), created_at, updated_at
- Add all fields from your DBML schema
- Example: update_table({tableSlug: "customers", xapikey: "%s", fields: [...], relations: [...]})

STEP 5: Organize tables into menus
- Since no folders are created, all tables use parent_id = "%s"
- Provide view_fields JSON: { "table_slug": "primary_field_slug" }
- Example: { "customers": "name", "orders": "order_number" }

STEP 6: Execute DBML (optional - if you prefer bulk creation)
- If you want to use dbml_to_ucode for bulk creation, provide:
  - dbml: (full DBML string)
  - view_fields: (JSON object)
  - menus: (JSON object where keys are table names, values are table slugs)
  - x-api-key: %s

CRITICAL RULES:
1. All tables are created at root level: parent_id = "%s"
2. Save table_id and collection (slug) from create_table responses
3. Use collection (slug) as the "collection" parameter in create_field (only for manual single-field requests)
4. If any tool call fails, STOP and report the error - do not retry
5. Create a COMPLETE, working project - not just partial structure
6. Analyze user request carefully - if they say "10 menus", create exactly 10 tables
7. Remember: Empty folders are not used, all tables are at root level

Context:
project-id = %s
environment-id = %s
x-api-key = %s
main-menu-parent-id = %s

User Request: "%s"

Now analyze the user's request, determine project type/systems/industry, and create the complete project structure directly as tables at root level.`,
			projectId.(string), environmentId.(string), apiKey, req.Prompt,

			config.MainMenuID, config.MainMenuID, apiKey,

			apiKey,

			config.MainMenuID,

			apiKey,

			config.MainMenuID,

			projectId.(string), environmentId.(string), apiKey, config.MainMenuID,
			req.Prompt,
		)

		message = "Your project has been successfully created."

	case "table":
		content = req.Prompt
		content += fmt.Sprintf(`

Context:
project-id = %s
environment-id = %s
x-api-key = %s

Rules:
- Use available MCP tools to actually perform the action (do not just reply with text).
- When creating a table:
  - label must match the requested name
  - slug should be a safe snake_case variant of the name
- After tool execution, return the API result (or clear error).`,
			projectId.(string), environmentId.(string), apiKey)

		message = "The table has been successfully updated."
	}

	go func() {
		resp, err := h.sendAnthropicRequest(content)
		fmt.Println("************ MCP Response ************", resp)
		if err != nil {
			h.HandleResponse(c, status_http.InternalServerError, "Your request could not be processed.")
			return
		}
	}()

	time.Sleep(1 * time.Minute)

	h.HandleResponse(c, status_http.OK, message)
}

func (h *HandlerV1) sendAnthropicRequest(content string) (string, error) {
	var (
		systemContent = `You are connected to an MCP server named "ucode" and have access to tools via the mcp_toolset.
When an external action is required (get_dbml, create_menu, create_table, update_table, dbml_to_ucode), CALL the tools using the MCP tool calling mechanism.
DO NOT call the create_field tool for automated field creation. Instead, ALWAYS use update_table to add or modify fields and relations in bulk (send a fields array and relations array).
Only call create_field when a human explicitly requests a single manual field creation and after receiving explicit confirmation.
Do not invent results — call the appropriate tool with exact parameters. Use the tool names as documented.`

		userMessage = Message{
			Role:    "user",
			Content: content,
		}

		maxTokens = h.baseConf.MaxTokens
	)

	if len(content) > 2000 && maxTokens < 4000 {
		maxTokens = 4000
	}

	log.Println("MCP URL:", h.baseConf.MCPServerURL)
	log.Println("ANTHROPIC API URL:", h.baseConf.AnthropicBaseAPIURL)
	log.Println("CONTENT:", content)

	body := RequestBodyAnthropic{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: maxTokens,
		System:    systemContent,
		Messages:  []Message{userMessage},
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
