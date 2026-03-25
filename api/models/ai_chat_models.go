package models

type (
	// ========================== Basic Requests ==========================

	NewMessageReq struct {
		Content string   `json:"content"`
		Images  []string `json:"images"`
	}

	SendMessageRequest struct {
		UserPrompt    string `json:"user_prompt"`
		ChatId        string `json:"chat_id"`
		ProjectId     string `json:"project_id"`
		ResourceEnvId string `json:"resource_env_id"`
	}

	// ========================== Project File Models ==========================

	GraphNode struct {
		Path      string `json:"path"`
		FileGraph any    `json:"file_graph,omitempty"`
	}

	// ========================== Anthropic Response Models ==========================

	// ClaudeResponse — полный ответ от Anthropic API
	ClaudeResponse struct {
		Model        string         `json:"model"`
		ID           string         `json:"id"`
		Type         string         `json:"type"`
		Role         string         `json:"role"`
		Content      []ContentBlock `json:"content"`
		StopReason   string         `json:"stop_reason"`
		StopSequence *string        `json:"stop_sequence"`
		Usage        ClaudeUsage    `json:"usage"`
	}

	ClaudeUsage struct {
		InputTokens              int    `json:"input_tokens"`
		OutputTokens             int    `json:"output_tokens"`
		CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
		ServiceTier              string `json:"service_tier"`
		InferenceGeo             string `json:"inference_geo"`
	}

	ParsedClaudeResponse struct {
		Model       string            `json:"model"`
		MessageID   string            `json:"message_id"`
		StopReason  string            `json:"stop_reason"`
		Usage       ClaudeUsage       `json:"usage"`
		Project     *GeneratedProject `json:"project,omitempty"`
		Description string            `json:"description"`
	}

	// ========================== Classification ==========================

	RequestClassification struct {
		RequiresBackend  bool   `json:"requires_backend"`
		RequiresFrontend bool   `json:"requires_frontend"`
		BackendReason    string `json:"backend_reason"`
		FrontendReason   string `json:"frontend_reason"`
		Confidence       string `json:"confidence"`
	}

	// ========================== AI Agent Routing ==========================

	HaikuRoutingResult struct {
		NextStep    bool     `json:"next_step"`
		Intent      string   `json:"intent"`
		Reply       string   `json:"reply"`        // готовый ответ если next_step=false
		Clarified   string   `json:"clarified"`    // уточнённый запрос для code_change
		FilesNeeded []string `json:"files_needed"` // нужные файлы для project_inspect
		HasImages   bool     `json:"has_images"`   // есть ли изображения в запросе
		ProjectName string   `json:"project_name"` // осмысленное имя проекта (max 3 слова)
	}

	SonnetPlanResult struct {
		FilesToChange []FilePlan `json:"files_to_change"`
		FilesToCreate []FilePlan `json:"files_to_create"`
		Summary       string     `json:"summary"`
	}

	FilePlan struct {
		Path        string `json:"path"`
		Description string `json:"description"`
	}

	// ========================== Architect Plan ==========================
	TableFieldPlan struct {
		Slug  string `json:"slug"`
		Label string `json:"label"`
		Type  string `json:"type"` // SINGLE_LINE, NUMBER, EMAIL, PHONE, DATE, etc.
	}

	TablePlan struct {
		Slug     string           `json:"slug"`
		Label    string           `json:"label"`
		Fields   []TableFieldPlan `json:"fields"`
		MockData []map[string]any `json:"mock_data"` // 3-5 реалистичных записей
	}

	ArchitectPlan struct {
		ProjectName string      `json:"project_name"`
		Tables      []TablePlan `json:"tables"`
		UIStructure string      `json:"ui_structure"`
	}

	ProjectData struct {
		McpProjectId   string `json:"project_id"`
		ApiKey         string `json:"api_key"`
		UcodeProjectId string `json:"ucode_project_id"`
	}
)
