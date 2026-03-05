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

	// HaikuRoutingResult — ответ Haiku роутера
	// intent values:
	//   "chat"             — оффтоп, Haiku отвечает сам, next_step=false
	//   "project_question" — вопрос о структуре проекта, Haiku отвечает из графа, next_step=false
	//   "project_inspect"  — вопрос требующий чтения контента файлов (пиксели, цвета, логика), next_step=true
	//   "code_change"      — изменение/генерация кода, next_step=true
	HaikuRoutingResult struct {
		NextStep    bool     `json:"next_step"`
		Intent      string   `json:"intent"`
		Reply       string   `json:"reply"`        // готовый ответ если next_step=false
		Clarified   string   `json:"clarified"`    // уточнённый запрос для code_change
		FilesNeeded []string `json:"files_needed"` // нужные файлы для project_inspect
	}

	// SonnetPlanResult — план Sonnet: какие файлы создать/изменить
	SonnetPlanResult struct {
		FilesToChange []FilePlan `json:"files_to_change"`
		FilesToCreate []FilePlan `json:"files_to_create"`
		Summary       string     `json:"summary"`
	}

	FilePlan struct {
		Path        string `json:"path"`
		Description string `json:"description"`
	}
)
