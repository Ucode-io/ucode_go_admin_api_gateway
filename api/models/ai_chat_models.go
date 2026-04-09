package models

type (
	// ========================== Basic Requests ==========================

	// EnrichedMessage is the HTTP representation of a chat message.
	// It mirrors pbo.Message but adds a Plan field parsed from embedded content.
	EnrichedMessage struct {
		ID         string     `json:"id"`
		ChatID     string     `json:"chat_id"`
		Role       string     `json:"role"`
		Content    string     `json:"content"`
		Images     []string   `json:"images"`
		HasFiles   bool       `json:"has_files"`
		TokensUsed int32      `json:"tokens_used"`
		CreatedAt  string     `json:"created_at"`
		Plan       *HaikuPlan `json:"plan,omitempty"`
	}

	NewMessageReq struct {
		Content       string         `json:"content"`
		Images        []string       `json:"images"`
		PendingAction *PendingAction `json:"pending_action,omitempty"`
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

	QuestionOption struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}

	AiQuestion struct {
		ID      string           `json:"id"`
		Title   string           `json:"title"`
		Type    string           `json:"type"` // "single" | "multi"
		Options []QuestionOption `json:"options"`
	}

	ParsedClaudeResponse struct {
		Model         string            `json:"model"`
		MessageID     string            `json:"message_id"`
		StopReason    string            `json:"stop_reason"`
		Usage         ClaudeUsage       `json:"usage"`
		Project       *GeneratedProject `json:"project,omitempty"`
		Description   string            `json:"description"`
		PendingAction *PendingAction    `json:"pending_action,omitempty"`
		Questions     []AiQuestion      `json:"questions,omitempty"`
		Plan          *HaikuPlan        `json:"plan,omitempty"`
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

	// PlanInfraEdge — a single directed edge in the infrastructure diagram.
	PlanInfraEdge struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Label string `json:"label"`
	}

	// HaikuPlan — diagram plan returned after the user answers questionnaire questions.
	// Contains only the visual diagrams needed before code generation.
	HaikuPlan struct {
		BpmnXML      string          `json:"bpmn_xml"`
		InfraDiagram []PlanInfraEdge `json:"infra_diagram,omitempty"`
	}

	HaikuRoutingResult struct {
		NextStep       bool         `json:"next_step"`
		Intent         string       `json:"intent"`
		Reply          string       `json:"reply"`
		Clarified      string       `json:"clarified"`
		ClarifyOptions []string     `json:"clarify_options"`
		FilesNeeded    []string     `json:"files_needed"`
		HasImages      bool         `json:"has_images"`
		ProjectName    string       `json:"project_name"`
		Questions      []AiQuestion `json:"questions,omitempty"`
		Plan           *HaikuPlan   `json:"plan,omitempty"`
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
		Slug          string                   `json:"slug"`
		Label         string                   `json:"label"`
		IsLoginTable  bool                     `json:"is_login_table"`
		LoginStrategy []string                 `json:"login_strategy"`
		Fields        []TableFieldPlan         `json:"fields"`
		MockData      []map[string]interface{} `json:"mock_data"`
	}

	DesignSpec struct {
		PrimaryColor         string `json:"primary_color"`
		PrimaryHSL           string `json:"primary_hsl"`
		BackgroundColor      string `json:"background_color"`
		BackgroundHSL        string `json:"background_hsl"`
		SurfaceColor         string `json:"surface_color"`
		SurfaceHSL           string `json:"surface_hsl"`
		SidebarBackground    string `json:"sidebar_background"`
		SidebarBackgroundHSL string `json:"sidebar_background_hsl"`
		SidebarForeground    string `json:"sidebar_foreground"`
		SidebarStyle         string `json:"sidebar_style"` // "dark" | "light" | "colored"
		TextColor            string `json:"text_color"`
		TextMutedColor       string `json:"text_muted_color"`
		BorderColor          string `json:"border_color"`
		AccentColor          string `json:"accent_color"`
		AccentHSL            string `json:"accent_hsl"`
		FontFamily           string `json:"font_family"`
		BorderRadius         string `json:"border_radius"`
		DesignInspiration     string `json:"design_inspiration"`
	}

	ArchitectPlan struct {
		ProjectName string      `json:"project_name"`
		ProjectType string      `json:"project_type"` // "admin_panel" | "landing" | "web" | "other"
		Tables      []TablePlan `json:"tables"`
		UIStructure string      `json:"ui_structure"`
		Design      DesignSpec  `json:"design"`
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

	DatabaseActionRequest struct {
		Action           string         `json:"action"`
		TableSlug        string         `json:"table_slug,omitempty"`
		Filters          map[string]any `json:"filters,omitempty"`
		Data             map[string]any `json:"data,omitempty"`
		AggregationField string         `json:"aggregation_field,omitempty"`
		Aggregation      string         `json:"aggregation,omitempty"`
		GroupBy          string         `json:"group_by,omitempty"`
		OrderBy          string         `json:"order_by,omitempty"`
		Limit            int            `json:"limit,omitempty"`
		Offset           int            `json:"offset,omitempty"`
		NeedsMoreData    bool           `json:"needs_more_data"`
		QueryPlan        string         `json:"query_plan,omitempty"`
		Reply            string         `json:"reply,omitempty"`
		SuccessMessage   string         `json:"success_message,omitempty"`
		CancelMessage    string         `json:"cancel_message,omitempty"`
		ResourceEnvID    string         `json:"-"` // set by backend, not from AI

		SQL string `json:"sql,omitempty"`

		SQLParams []any `json:"sql_params,omitempty"`
	}

	PendingAction struct {
		Action             string         `json:"action"`
		TableSlug          string         `json:"table_slug,omitempty"`
		Filters            map[string]any `json:"filters,omitempty"`
		Data               map[string]any `json:"data,omitempty"`
		AffectedCount      int            `json:"affected_count,omitempty"`
		Description        string         `json:"description,omitempty"`
		ProjectID          string         `json:"project_id,omitempty"`
		ResourceEnvID      string         `json:"resource_env_id,omitempty"`
		SuccessMessage     string         `json:"success_message,omitempty"`
		CancelMessage      string         `json:"cancel_message,omitempty"`
		ConfirmationPrompt string         `json:"confirmation_prompt,omitempty"`
		Approved           bool           `json:"approved,omitempty"`

		// ── NEW: SQL mode fields ──────────────────────────────────────────────────

		SQL       string `json:"sql,omitempty"`
		SQLParams []any  `json:"sql_params,omitempty"`
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
