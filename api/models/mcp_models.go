package models

type (

	// ========================== MCP Method Requests ==============================

	MCPRequest struct {
		Method  string          `json:"method"`
		Prompt  string          `json:"prompt"`
		Context *InspectContext `json:"context,omitempty"`
	}

	InspectContext struct {
		TargetFile      string `json:"target_file"`
		TargetElementId string `json:"target_element_id"`
		CodeFragment    string `json:"code_fragment"`
	}

	// =========================== Generate Prompt Requests ==========================

	GenerateMcpPromptReq struct {
		ProjectId     string `json:"project_id"`
		EnvironmentId string `json:"environment_id"`
		APIKey        string `json:"api_key"`
		UserPrompt    string `json:"user_prompt"`
		Method        string `json:"method"`
		BaseURL       string `json:"base_url"`
	}

	GenerateAnalysisPromptReq struct {
		UserRequest string          `json:"user_request"`
		FileGraph   map[string]any  `json:"file_graph"`
		ProjectName string          `json:"project_name"`
		Context     *InspectContext `json:"context,omitempty"`
	}

	GenerateUpdatePromptReq struct {
		UserRequest    string                  `json:"user_request"`
		FilesToUpdate  []FileContent           `json:"files_to_update"`
		AnalysisResult AnalysedProjectResponse `json:"analysis_result"`
		ProjectName    string                  `json:"project_name"`
	}

	FileContent struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	// ================================ AI Response ================================

	FrontGeneratedProject struct {
		ProjectName string         `json:"project_name"`
		Files       []FileContent  `json:"files"`
		FileGraph   map[string]any `json:"file_graph"`
		Env         map[string]any `json:"env"`
	}

	AnalysedProjectResponse struct {
		AnalysisSummary      string         `json:"analysis_summary"`
		FilesToModify        []FileToModify `json:"files_to_modify"`
		NewFilesNeeded       []FileToCreate `json:"new_files_needed"`
		FilesToDelete        []FileToDelete `json:"files_to_delete"`
		AffectedDependencies []string       `json:"affected_dependencies"`
		EstimatedComplexity  string         `json:"estimated_complexity"`
		Risks                []string       `json:"risks"`
	}

	McpUpdatedProject struct {
		UpdatedFiles     []McpUpdatedFile `json:"updated_files"`
		NewFiles         []McpNewFile     `json:"new_files"`
		DeletedFiles     []string         `json:"deleted_files"`
		FileGraphUpdates map[string]any   `json:"file_graph_updates"`
		IntegrationNotes []string         `json:"integration_notes"`
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

	McpUpdatedFile struct {
		Path          string `json:"path"`
		Content       string `json:"content"`
		ChangeSummary string `json:"change_summary"`
	}

	McpNewFile struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Purpose string `json:"purpose"`
	}

	// ======================= Anthropic Request Body =============================

	McpUserMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	MCPServer struct {
		Type               string `json:"type"`
		URL                string `json:"url"`
		Name               string `json:"name"`
		AuthorizationToken string `json:"authorization_token,omitempty"`
	}

	McpTool struct {
		Type          string `json:"type"`
		MCPServerName string `json:"mcp_server_name,omitempty"`
	}

	RequestBodyAnthropic struct {
		Model      string           `json:"model"`
		MaxTokens  int              `json:"max_tokens"`
		System     string           `json:"system,omitempty"`
		Messages   []McpUserMessage `json:"messages"`
		MCPServers []MCPServer      `json:"mcp_servers,omitempty"`
		McpTools   []McpTool        `json:"tools,omitempty"`
	}
)
