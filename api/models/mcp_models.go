package models

type (
	ProjectScope struct {
		ProjectId     string
		EnvironmentId string
		APIKey        string
	}

	// ========================== MCP Requests ==========================

	MCPRequestWithPlanning struct {
		Prompt       string            `json:"prompt"`
		ImageURLs    []string          `json:"image_urls,omitempty"`
		Method       string            `json:"method,omitempty"`
		Context      *[]InspectContext `json:"context,omitempty"`
		BackendPlan  string            `json:"backend_plan,omitempty"`
		FrontendPlan string            `json:"frontend_plan,omitempty"`
	}

	MCPRequest struct {
		Method    string            `json:"method"`
		Prompt    string            `json:"prompt"`
		ImageURLs []string          `json:"image_urls,omitempty"`
		Context   *[]InspectContext `json:"context,omitempty"`
	}

	InspectContext struct {
		TargetFile      string `json:"target_file"`
		TargetElementId string `json:"target_element_id"`
		CodeFragment    string `json:"code_fragment"`
		Tag             string `json:"tag"`
		DOMPath         string `json:"dom_path"`
		Line            int    `json:"line"`
		Column          int    `json:"column"`
		ElementName     string `json:"name"`
	}

	// ========================== Prompt Building Requests ==========================

	GeneratePromptRequest struct {
		ProjectId     string `json:"project_id"`
		EnvironmentId string `json:"environment_id"`
		APIKey        string `json:"api_key"`
		UserPrompt    string `json:"user_prompt"`
		Method        string `json:"method"`
		BaseURL       string `json:"base_url"`
	}

	AnalyzeFrontendPromptRequest struct {
		UserRequest string            `json:"user_request"`
		FileGraph   map[string]any    `json:"file_graph"`
		ProjectName string            `json:"project_name"`
		Context     *[]InspectContext `json:"context,omitempty"`
	}

	UpdateFrontendPromptRequest struct {
		UserRequest    string            `json:"user_request"`
		FilesToUpdate  []ProjectFile     `json:"files_to_update"`
		AnalysisResult AnalysisResult    `json:"analysis_result"`
		ProjectName    string            `json:"project_name"`
		Context        *[]InspectContext `json:"context,omitempty"`
	}

	// ========================== Project File Models ==========================

	ProjectFile struct {
		Path          string `json:"path"`
		Content       string `json:"content"`
		ChangeSummary string `json:"change_summary,omitempty"`
		Purpose       string `json:"purpose,omitempty"`
	}

	// ========================== AI Generated Results ==========================

	GeneratedProject struct {
		ProjectName string         `json:"project_name"`
		Files       []ProjectFile  `json:"files"`
		FileGraph   map[string]any `json:"file_graph"`
		Env         map[string]any `json:"env"`
	}

	AnalysisResult struct {
		AnalysisSummary      string         `json:"analysis_summary"`
		FilesToModify        []FileToModify `json:"files_to_modify"`
		NewFilesNeeded       []FileToCreate `json:"new_files_needed"`
		FilesToDelete        []FileToDelete `json:"files_to_delete"`
		AffectedDependencies []string       `json:"affected_dependencies"`
		EstimatedComplexity  string         `json:"estimated_complexity"`
		Risks                []string       `json:"risks"`
	}

	UpdateResult struct {
		UpdatedFiles     []ProjectFile  `json:"updated_files"`
		NewFiles         []ProjectFile  `json:"new_files"`
		DeletedFiles     []string       `json:"deleted_files"`
		FileGraphUpdates map[string]any `json:"file_graph_updates"`
		IntegrationNotes []string       `json:"integration_notes"`
	}

	FileToModify struct {
		Path       string `json:"path"`
		Reason     string `json:"reason"`
		ChangeType string `json:"change_type"`
		Priority   string `json:"priority"`
	}

	FileToCreate struct {
		Path       string `json:"path"`
		Reason     string `json:"reason"`
		ChangeType string `json:"change_type"`
	}

	FileToDelete struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	}

	// ========================== Anthropic API Models ==========================

	AnthropicRequest struct {
		Model      string        `json:"model"`
		MaxTokens  int           `json:"max_tokens"`
		System     string        `json:"system,omitempty"`
		Messages   []ChatMessage `json:"messages"`
		MCPServers []MCPServer   `json:"mcp_servers,omitempty"`
		Tools      []MCPTool     `json:"tools,omitempty"`
	}

	MCPServer struct {
		Type               string `json:"type"`
		URL                string `json:"url"`
		Name               string `json:"name"`
		AuthorizationToken string `json:"authorization_token,omitempty"`
	}

	MCPTool struct {
		Type          string `json:"type"`
		MCPServerName string `json:"mcp_server_name,omitempty"`
	}

	// ========================== Claude Tool Use (Function Calling) ==========================

	// ClaudeFunctionTool defines a structured output tool for the Anthropic tool-use API.
	// When used with tool_choice={type:"tool"}, Claude MUST call this tool and return
	// a validated JSON object — no text wrapping, no markdown, no escaping needed.
	ClaudeFunctionTool struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		InputSchema map[string]interface{} `json:"input_schema"`
	}

	// ToolChoice forces Claude to call a specific tool.
	ToolChoice struct {
		Type string `json:"type"` // "tool" | "auto" | "any"
		Name string `json:"name,omitempty"`
	}

	// AnthropicToolRequest is the request body for tool-use API calls.
	// Use this instead of AnthropicRequest for all structured generation calls
	// (architect, coder, planner, diagrams, visual edit).
	AnthropicToolRequest struct {
		Model      string               `json:"model"`
		MaxTokens  int                  `json:"max_tokens"`
		System     string               `json:"system,omitempty"`
		Messages   []ChatMessage        `json:"messages"`
		Tools      []ClaudeFunctionTool `json:"tools"`
		ToolChoice *ToolChoice          `json:"tool_choice,omitempty"`
	}

	// ToolUseBlock is one content block in the Anthropic response when stop_reason="tool_use".
	ToolUseBlock struct {
		Type  string                 `json:"type"` // "tool_use"
		ID    string                 `json:"id"`
		Name  string                 `json:"name"`
		Input map[string]interface{} `json:"input"`
	}

	// ToolUseResponse is the full Anthropic API response for tool-use calls.
	ToolUseResponse struct {
		Model      string         `json:"model"`
		ID         string         `json:"id"`
		Content    []ToolUseBlock `json:"content"`
		StopReason string         `json:"stop_reason"`
		Usage      ClaudeUsage    `json:"usage"`
	}

	ContentBlock struct {
		Type   string       `json:"type"`
		Text   string       `json:"text,omitempty"`
		Source *ImageSource `json:"source,omitempty"`
	}

	ImageSource struct {
		Type      string `json:"type" default:"url"`
		URL       string `json:"url,omitempty"`
		MediaType string `json:"media_type,omitempty"`
	}

	ChatMessage struct {
		Role    string         `json:"role"`
		Content []ContentBlock `json:"content"`
	}
)
