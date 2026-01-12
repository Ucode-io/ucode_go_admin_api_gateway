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
- **Quantity Rule**: If the user did NOT specify a number of tables/menus, strictly create approximately **10 tables** suitable for the project logic.
- **Icon Rule**: You **MUST** provide an 'icon' parameter for EVERY table. Use a valid full URL from Iconify (e.g., "https://api.iconify.design/mdi:account.svg"). Pick an icon relevant to the table's function.
- **Menu ID Rule**: You **MUST** provide 'menu_id' for every table. Use the root parent_id = "%s".
- Call create_table with: label, slug (snake_case), icon, menu_id, x-api-key.
- IMPORTANT: Save the response from each create_table call - you need table_id and slug (collection) for next steps
- Example: create_table({label: "Customers", slug: "customers", icon: "https://api.iconify.design/mdi:account.svg", menu_id: "%s", x_api_key: "%s"})
- Do NOT create folders, all tables are created at root level using the provided menu_id.
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

	type mcpResult struct {
		resp string
		err  error
	}

	var (
		mcpRespChan = make(chan mcpResult)
		ctx, cancel = context.WithTimeout(context.Background(), 500*time.Second)
	)

	defer cancel()

	go func() {
		mcpResp, mcpError := h.sendAnthropicRequest(content)
		fmt.Println("************ MCP Response ************", mcpResp)
		mcpRespChan <- mcpResult{resp: mcpResp, err: mcpError}
	}()

	select {
	case <-ctx.Done():
		h.HandleResponse(c, status_http.OK, message+"  . TIMEOUT")
		return
	case result := <-mcpRespChan:
		if result.err != nil {
			h.HandleResponse(c, status_http.InternalServerError, result.err.Error())
			return
		}

		h.HandleResponse(c, status_http.OK, result.resp)
		return
	}
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

	if len(content) > 2000 {
		maxTokens = 8192
	}

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
	apiKey := apiKeys.GetData()[0].GetAppId()
	projectIDStr := projectId.(string)
	mainMenuID := config.MainMenuID
	// Dynamic Base URL from config (assumed to be in h.baseConf or config package)
	adminBaseURL := "https://admin-api.ucode.run" // Default fallback
	// --------------------------------------------------------------------------
	// The "Universal Premium" Prompt Injection using Helper
	// --------------------------------------------------------------------------
	systemPrompt, userPrompt := h.constructFrontendPrompt(req.Prompt, projectIDStr, mainMenuID, apiKey, adminBaseURL)
	resp, err := h.sendAnthropicRequestFront(systemPrompt, userPrompt)
	log.Println("************ Anthropic response ************:", resp)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "AI Generation Failed: "+err.Error())
		return
	}
	var parsedResponse interface{}
	if json.Valid([]byte(resp)) {
		if err := json.Unmarshal([]byte(resp), &parsedResponse); err == nil {
			h.HandleResponse(c, status_http.OK, parsedResponse)
			return
		}
	}
	h.HandleResponse(c, status_http.OK, resp)
}

// Rewritten constructFrontendPrompt that returns both the SYSTEM prompt (fixed, authoritative)
// and the USER prompt (dynamic, injected with runtime values).
// NOTE: this function uses fmt.Sprintf so ensure you have `import "fmt"` in the file.
func (h *HandlerV1) constructFrontendPrompt(userRequest, projectID, menuID, apiKey, baseURL string) (string, string) {
	// FINAL DYNAMIC SYSTEM PROMPT (immutable, used as the single system message)
	system := `You are a Senior Frontend Architect and UI/UX Designer.

Your role is ABSOLUTE and UNCHANGING:
You ALWAYS generate a PRODUCTION-READY, PREMIUM ADMIN PANEL frontend.
No matter what the user asks, you interpret the request as requirements
for an ADMIN PANEL (CRM, ERP, CMS, Dashboard, Backoffice, etc.).

You NEVER ask questions.
You NEVER wait for another message.
You NEVER output explanations.

────────────────────────────────────────────
GLOBAL OUTPUT RULES (ZERO TOLERANCE)
────────────────────────────────────────────
- Output EXACTLY ONE JSON object.
- Do NOT output markdown.
- Do NOT output comments.
- Do NOT output explanations.
- Do NOT output text before or after JSON.
- JSON MUST be parseable by JSON.parse().
- Root structure MUST be:

{
  "project_name": "<string>",
  "files": [
    { "path": "<string>", "content": "<raw file text>" }
  ]
}

- File contents MUST contain REAL newlines.
- NEVER use escaped \n inside file contents.

────────────────────────────────────────────
TECH STACK (FIXED, NEVER CHANGE)
────────────────────────────────────────────
- React 18 ONLY (createRoot is MANDATORY)
- Vite
- Tailwind CSS v2.2.19 ONLY

Allowed Tailwind colors:
gray, red, yellow, green, blue, indigo, purple, pink

FORBIDDEN colors:
slate, zinc, neutral, stone, emerald, teal, cyan, sky, amber, lime, rose, fuchsia, violet

────────────────────────────────────────────
ADMIN PANEL INTERPRETATION LOGIC
────────────────────────────────────────────
- You ALWAYS generate an ADMIN PANEL.
- The user's text only influences:
  - domain (ERP, CRM, CMS, Finance, Logistics, etc.)
  - visual style (dark, light, enterprise, modern, minimal, luxury)
- If the user mentions ANY of:
  "erp", "admin", "dashboard", "backoffice", "system"
  → default to DARK MODE.
- If the user explicitly says "light only" → light theme.
- Otherwise choose the BEST professional theme automatically.

Even if the user prompt is vague or short,
you still generate a COMPLETE, BEAUTIFUL, MODERN admin panel.

────────────────────────────────────────────
LAYOUT RULES (MANDATORY)
────────────────────────────────────────────
- Fixed sidebar layout.
- Sidebar width: 256px.
- Main content offset: ml-64.
- Sidebar height: h-screen.
- Sidebar is vertically scrollable if menus overflow.
- Main content area is scrollable independently.

────────────────────────────────────────────
NAVIGATION & ROUTER (MANDATORY)
────────────────────────────────────────────
- The generated project MUST include client-side routing using react-router-dom.
- Include BrowserRouter at app root (src/main.jsx or src/App.jsx) and a Routes config:
  - route "/" -> DashboardHome
  - route "/tables/:collection" -> DynamicTablePage
  - route "/pages/:id" or "/menu/:id" -> DynamicPage (or dashboard fallback)
- Sidebar menu clicks MUST use react-router-dom navigation (useNavigate) to change routes.
- On menu click, derive the path from the menu object:
  - If menu.type === "TABLE" and menu.table_slug exists => navigate to '/tables/${table_slug}'
  - Otherwise navigate to a menu-specific path such as '/menu/${id}' or' /pages/${id}'
- Do NOT hardcode internal variable names in the generated code examples — implement idiomatic React hook usage.
  Example (illustrative only): obtain a navigate function from useNavigate() and call navigate(pathDerivedFromMenu).
- The top app header MUST display the currently selected menu label using router state or shared state (e.g., URL param, context, or location state).
- Routes should be lazy-loaded where appropriate and include a fallback loading state (Suspense) to keep bundle sizes reasonable.
- Ensure navigation preserves scroll restoration and does not break the sticky table header behavior.

────────────────────────────────────────────
SIDEBAR & MENUS (DYNAMIC, REQUIRED)
────────────────────────────────────────────
Menus are ALWAYS dynamic.

You MUST implement runtime fetching of menus from ONE of the following:
1) MCP tool: get_menus
2) HTTP API:
   GET %%VITE_ADMIN_BASE_URL%%/v3/menus?parent_id=<parent_id>&project-id=<project_id>

Headers:
- Authorization: API-KEY
- X-API-KEY: <X_API_KEY>

Expected menu shape (example):
{
  "data": {
    "menus": [
      {
        "id": "...",
        "label": "Users",
        "icon": "https://...",
        "type": "TABLE",
        "table_slug": "users"
      }
    ]
  }
}

Rules:
- label is ALWAYS shown.
- icon is OPTIONAL.
- If icon URL is missing or fails → fallback to a react-icon.
- Image icons on dark background MUST use:
  filter invert brightness-200
- Active menu state:
  bg-blue-600 text-white font-medium rounded-lg mx-2
- Clicking a menu:
  - updates page title with menu label
  - triggers navigation/action

────────────────────────────────────────────
TABLE MENUS & DATA (DYNAMIC)
────────────────────────────────────────────
If menu.type === "TABLE":

You MUST fetch table details dynamically from ONE of:
1) MCP tool: get_table_details
2) HTTP API:
   POST %%VITE_ADMIN_BASE_URL%%/v1/table-details/:collection

Headers:
- Authorization: API-KEY
- X-API-KEY: <X_API_KEY>
- Content-Type: application/json

Body:
{ "data": {} }

Expected fields shape:
response.data?.data?.data?.fields || []

Rules:
- Fields are NEVER hardcoded.
- Tables are ALWAYS generated from fields dynamically.
- If number of fields > visible width:
  - horizontal scroll MUST appear
  - each column uses min-width
- Table layout rules:
  - ≤ 6 columns → table-fixed
  - > 6 columns → auto layout + horizontal scroll

────────────────────────────────────────────
CRITICAL: STICKY TABLE HEADER (BUG PREVENTION)
────────────────────────────────────────────
- NEVER use position: fixed for table headers.
- thead th MUST use:
  position: sticky;
  top: 0;
  z-index: 10;
- Sticky header MUST be relative to the table scroll container.
- Table scroll container MUST be inside the main content area.
- Content wrapper MUST have padding-top equal to header height.

Violating this rule is considered a critical failure.

────────────────────────────────────────────
SAFE DATA ACCESS (MANDATORY)
────────────────────────────────────────────
ALL API access MUST use optional chaining and fallbacks.

Examples:
const menus = response.data?.data?.menus || [];
const fields = response.data?.data?.data?.fields || [];

NEVER assume data exists.

────────────────────────────────────────────
REQUIRED COMPONENTS
────────────────────────────────────────────
You MUST include these exact components:

ElementLink:
- disabled input
- OPEN button
- safe value handling

ElementText:
- read-only text rendering

You may include additional components,
but these two MUST exist.

────────────────────────────────────────────
FILES THAT MUST EXIST
────────────────────────────────────────────
At minimum, generate:

- package.json
- vite.config.js
- tailwind.config.cjs
- postcss.config.cjs
- .env.example (VITE_ADMIN_BASE_URL=...)
- src/main.jsx (React 18 createRoot with BrowserRouter)
- src/App.jsx (Routes configuration)
- src/index.css
- src/api/axios.js
- src/layouts/DashboardLayout.jsx (Sidebar uses useNavigate to route)
- src/components/Sidebar.jsx (menu click performs navigate(path))
- src/components/DynamicTable.jsx
- src/components/DynamicForm.jsx
- src/pages/DashboardHome.jsx
- src/pages/DynamicTablePage.jsx (reads :collection param)
- README_HOW_TO_RUN.txt

────────────────────────────────────────────
ERROR HANDLING & FALLBACKS
────────────────────────────────────────────
- If runtime variables (project_id, parent_id, X_API_KEY, base URL) are missing:
  - still generate FULL project
  - explain injection steps inside README_HOW_TO_RUN.txt
- If API returns unexpected shape:
  - do NOT crash UI
  - log structured console.warn messages

────────────────────────────────────────────
FINAL COMMAND
────────────────────────────────────────────
Given the USER message and injected runtime variables,
IMMEDIATELY generate the FULL admin panel project
as ONE VALID JSON OBJECT.

No explanations.
No retries.
No partial output.
No markdown.
Only JSON.
`

	// USER prompt template: dynamic; will be injected with the runtime values.
	userTpl := `User request:
- Description: "%s"
- Project ID: "%s"
- Main Menu Parent ID: "%s"
- X-API-KEY: "%s"
- Base URL: "%s"

Task:
1) Generate a complete production-ready frontend-only admin project (React 18 + Vite + TailwindCSS v2.2.19) as a single JSON object with fields:
   { "project_name": "<string>", "files": [ { "path": "<path>", "content": "<file contents>" }, ... ] }
   - File contents must be plain raw file text (use real newlines in JSON string values).
   - No markdown, no extra text outside that single JSON root.
2) Default to DARK MODE when the Description includes "dark", "erp", "admin", "dashboard", or "backoffice" (unless the user explicitly requests "light only").
3) Implement client-side routing using react-router-dom:
   - Include BrowserRouter and a Routes config with at least "/" (DashboardHome) and "/tables/:collection" (DynamicTablePage).
   - Sidebar menu item clicks must navigate using useNavigate to a path derived from the menu (e.g. '/tables/${table_slug}' for TABLE menus or '/menu/${id}').
   - Top header must display selected menu label via router state or URL params.
4) Implement runtime fetching of menus and table details using MCP tools 'get_menus' and 'get_table_details' when available.
   If MCP tools are not available, include exact axios calls that the generated code will use:
   - GET %%VITE_ADMIN_BASE_URL%%/v3/menus?parent_id=%s&project-id=%s
     Headers: { Authorization: "API-KEY", "X-API-KEY": "%s" }
   - POST %%VITE_ADMIN_BASE_URL%%/v1/table-details/:collection
     Body: { "data": {} }
     Headers: same as above
5) Ensure table components follow the documented layout rules:
   - <=6 columns => table-fixed
   - >6 columns => auto layout + min-w per column + horizontal scroll
   - thead th must be position: sticky; top: 0; z-index: 10; inside the scroll container
6) Include required components ElementLink and ElementText with the behaviors described in the system prompt.
7) If any runtime env is missing, still output full project and include README_HOW_TO_RUN.txt explaining where to inject PROJECT_ID, PARENT_ID, X_API_KEY, VITE_ADMIN_BASE_URL.
8) Return EXACTLY one JSON object and nothing else.

Now produce the project JSON immediately.`

	// Build the user prompt by injecting runtime values where needed.
	user := fmt.Sprintf(userTpl,
		userRequest, // description
		projectID,
		menuID,
		apiKey,
		baseURL,
		// for axios example parameters (reused)
		menuID,
		projectID,
		apiKey,
	)

	return system, user
}

// Updated sendSimpleOpenAIRequest: fixes the "max_tokens" error (uses max_completion_tokens)
// and optionally injects MCP server info into the OpenAI request body when available in h.baseConf.
// Replace or adapt BaseConf field access if your project uses different names.
//func (h *HandlerV1) sendSimpleOpenAIRequest(systemPrompt, userPrompt string) (string, error) {
//	// --- Request/response types ---
//	type openAIMessage struct {
//		Role    string `json:"role"`
//		Content string `json:"content"`
//	}
//	type responseFormat struct {
//		Type string `json:"type"`
//	}
//
//	// Optional MCP/tool types (will be included only when MCPServerURL is present)
//	type MCPServer struct {
//		Type string `json:"type"` // e.g. "url"
//		URL  string `json:"url"`
//		Name string `json:"name"`
//	}
//	type Tool struct {
//		Type          string `json:"type"`
//		MCPServerName string `json:"mcp_server_name,omitempty"`
//	}
//
//	type openAIRequest struct {
//		Model               string          `json:"model"`
//		MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"` // NOTE: fixed param name
//		Messages            []openAIMessage `json:"messages"`
//		Temperature         float64         `json:"temperature"`
//		ResponseFormat      *responseFormat `json:"response_format,omitempty"`
//		MCPServers          []MCPServer     `json:"mcp_servers,omitempty"`
//		Tools               []Tool          `json:"tools,omitempty"`
//	}
//
//	type openAIChoice struct {
//		Message struct {
//			Content string `json:"content"`
//		} `json:"message"`
//	}
//	type openAIResponse struct {
//		Choices []openAIChoice `json:"choices"`
//		Error   any            `json:"error,omitempty"`
//	}
//
//	// --- Resolve API key (prefer baseConf, fallback to env) ---
//	var apiKey = ""
//
//	// --- Build request body ---
//	reqBody := openAIRequest{
//		Model:               "gpt-5.2",
//		MaxCompletionTokens: 65536,
//		Temperature:         0.0,
//		ResponseFormat: &responseFormat{
//			Type: "json_object",
//		},
//		Messages: []openAIMessage{
//			{Role: "system", Content: systemPrompt},
//			{Role: "user", Content: userPrompt},
//		},
//	}
//
//	mcpName := "ucode" // default name, change if you use another
//	reqBody.MCPServers = []MCPServer{
//		{
//			Type: "url",
//			URL:  h.baseConf.MCPServerURL,
//			Name: mcpName,
//		},
//	}
//	reqBody.Tools = []Tool{
//		{
//			Type:          "mcp_toolset",
//			MCPServerName: mcpName,
//		},
//	}
//
//	bodyBytes, err := json.Marshal(reqBody)
//	if err != nil {
//		return "", fmt.Errorf("failed to marshal openai request: %w", err)
//	}
//
//	// --- Determine base URL ---
//	baseURL := "https://api.openai.com/v1/chat/completions"
//
//	// --- Create HTTP request ---
//	req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(bodyBytes))
//	if err != nil {
//		return "", fmt.Errorf("failed to create request: %w", err)
//	}
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Authorization", "Bearer "+apiKey)
//
//	// --- Execute ---
//	client := &http.Client{Timeout: 600 * time.Second}
//	resp, err := client.Do(req)
//	if err != nil {
//		return "", fmt.Errorf("request failed: %w", err)
//	}
//	defer resp.Body.Close()
//
//	respBytes, err := io.ReadAll(resp.Body)
//	if err != nil {
//		return "", fmt.Errorf("failed to read response body: %w", err)
//	}
//
//	// If non-200, surface the body for debugging
//	if resp.StatusCode != http.StatusOK {
//		return "", fmt.Errorf("openai status: %d, body: %s", resp.StatusCode, string(respBytes))
//	}
//
//	// --- Parse response ---
//	var oResp openAIResponse
//	if err := json.Unmarshal(respBytes, &oResp); err != nil {
//		return "", fmt.Errorf("failed to unmarshal openai response: %w; raw: %s", err, string(respBytes))
//	}
//	if len(oResp.Choices) == 0 {
//		return "", fmt.Errorf("no choices in openai response; raw: %s", string(respBytes))
//	}
//
//	return oResp.Choices[0].Message.Content, nil
//}

// sendAnthropicRequest sends a request to Anthropic (Claude) including MCP server/tool info.
// It expects h.baseConf to contain fields:
//   - AnthropiсAPIKey (string)        -> h.baseConf.AnthropicAPIKey
//   - AnthropicBaseAPIURL (string)    -> h.baseConf.AnthropicBaseAPIURL
//   - AnthropicVersion (string)       -> h.baseConf.AnthropicVersion (optional)
//   - AnthropicBeta (string)          -> h.baseConf.AnthropicBeta (optional)
//   - MCPServerURL (string)           -> h.baseConf.MCPServerURL
//   - ClaudeModel (string)            -> h.baseConf.ClaudeModel (e.g. "claude-2.1" or a supported claude model)
//   - MaxTokens (int)                 -> h.baseConf.MaxTokens (default fallback used if zero)
//
// The function will return the raw Anthropic response body (string) on success.
func (h *HandlerV1) sendAnthropicRequestFront(systemPrompt, userPrompt string) (string, error) {

	var body = RequestBodyAnthropic{
		Model:     h.baseConf.ClaudeModel,
		MaxTokens: h.baseConf.MaxTokens,
		System:    systemPrompt,
		Messages: []Message{
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
		return "", fmt.Errorf("failed to marshal anthropic request body: %w", err)
	}

	// --- Create request ---
	request, err := http.NewRequest(http.MethodPost, h.baseConf.AnthropicBaseAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create anthropic request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", h.baseConf.AnthropicAPIKey)
	request.Header.Set("anthropic-version", h.baseConf.AnthropicVersion)
	request.Header.Set("anthropic-beta", h.baseConf.AnthropicBeta)

	client := &http.Client{Timeout: 420 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return string(respBytes), fmt.Errorf("unexpected anthropic status: %d", resp.StatusCode)
	}

	return string(respBytes), nil
}
