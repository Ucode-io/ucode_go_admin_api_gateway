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
		Model         string            `json:"model"`
		MessageID     string            `json:"message_id"`
		StopReason    string            `json:"stop_reason"`
		Usage         ClaudeUsage       `json:"usage"`
		Project       *GeneratedProject `json:"project,omitempty"`
		Description   string            `json:"description"`
		PendingAction *PendingAction    `json:"pending_action,omitempty"`
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
		ProjectType string      `json:"project_type"` // "admin_panel" | "landing" | "web" | "other"
		Tables      []TablePlan `json:"tables"`
		UIStructure string      `json:"ui_structure"`
	}

	ProjectData struct {
		McpProjectId   string `json:"project_id"`
		ApiKey         string `json:"api_key"`
		UcodeProjectId string `json:"ucode_project_id"`
		EnvironmentId  string `json:"environment_id"`
		ResourceEnvId  string `json:"resource_env_id"`
	}

	// ========================== AI Database Assistant ==========================

	// FieldSchema — simplified field info sent to Claude for schema awareness
	FieldSchema struct {
		Slug  string `json:"slug"`
		Label string `json:"label"`
		Type  string `json:"type"`
	}

	// TableSchema — simplified table info with fields sent to Claude
	TableSchema struct {
		Slug   string        `json:"slug"`
		Label  string        `json:"label"`
		Fields []FieldSchema `json:"fields"`
	}

	// DatabaseActionRequest — what Claude returns as a structured database action
	DatabaseActionRequest struct {
		Action      string         `json:"action"`       // "read" | "create" | "update" | "delete" | "count" | "aggregate"
		TableSlug   string         `json:"table_slug"`
		Filters     map[string]any `json:"filters,omitempty"`
		Data        map[string]any `json:"data,omitempty"`      // for create/update
		Aggregation string         `json:"aggregation,omitempty"` // "count" | "sum" | "avg" etc.
		GroupBy     string         `json:"group_by,omitempty"`
		OrderBy     string         `json:"order_by,omitempty"`
		Limit       int            `json:"limit,omitempty"`
		Offset      int            `json:"offset,omitempty"`
		Reply       string         `json:"reply"` // AI's human-readable answer for the user
	}

	// PendingAction — stored action waiting for user confirmation
	PendingAction struct {
		ID            string         `json:"id"`
		ChatID        string         `json:"chat_id"`
		Action        string         `json:"action"`    // "create" | "update" | "delete"
		TableSlug     string         `json:"table_slug"`
		Filters       map[string]any `json:"filters,omitempty"`
		Data          map[string]any `json:"data,omitempty"`
		AffectedCount int            `json:"affected_count"`
		Description   string         `json:"description"`
		Status        string         `json:"status"` // "pending" | "confirmed" | "rejected"
		ProjectID     string         `json:"project_id"`
		ResourceEnvID string         `json:"resource_env_id"`
	}

	// ConfirmActionRequest — frontend sends this to confirm/reject a pending action
	ConfirmActionRequest struct {
		Confirmed bool `json:"confirmed"`
	}

	// DatabaseAssistantResponse — full response from the database assistant flow
	DatabaseAssistantResponse struct {
		Reply         string         `json:"reply"`
		Data          any            `json:"data,omitempty"`
		PendingAction *PendingAction `json:"pending_action,omitempty"`
	}
)
