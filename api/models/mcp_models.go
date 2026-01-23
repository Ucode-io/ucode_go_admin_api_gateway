package models

type (
	// ========================== API Requests ==========================

	MCPRequest struct {
		Method  string            `json:"method"`
		Prompt  string            `json:"prompt"`
		Context *[]InspectContext `json:"context,omitempty"`
	}

	InspectContext struct {
		TargetFile      string `json:"target_file"`
		TargetElementId string `json:"target_element_id"`
		CodeFragment    string `json:"code_fragment"`
		Tag             string `json:"tag"`
		DOMPath         string `json:"dom_path"`
		Line            int    `json:"line"`
		Column          int    `json:"column"`
	}

	// ========================== Prompt Building Requests ==========================

	BackendPromptRequest struct {
		ProjectId     string `json:"project_id"`
		EnvironmentId string `json:"environment_id"`
		APIKey        string `json:"api_key"`
		UserPrompt    string `json:"user_prompt"`
		Method        string `json:"method"`
		BaseURL       string `json:"base_url"`
	}

	FrontendPromptRequest struct {
		ProjectId     string `json:"project_id"`
		EnvironmentId string `json:"environment_id"`
		APIKey        string `json:"api_key"`
		UserPrompt    string `json:"user_prompt"`
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

	// ========================== Unified File Model ==========================

	ProjectFile struct {
		Path          string `json:"path"`
		Content       string `json:"content"`
		ChangeSummary string `json:"change_summary,omitempty"` // for updated files
		Purpose       string `json:"purpose,omitempty"`        // for new files
	}

	// ========================== AI Responses ==========================

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

	ChatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
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
)
