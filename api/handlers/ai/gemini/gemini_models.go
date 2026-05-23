package gemini

import "errors"

var ErrMaxTokens = errors.New("generation stopped: output exceeded the token limit")

type (
	geminiRequest struct {
		SystemInstruction *geminiContent   `json:"system_instruction,omitempty"`
		Contents          []geminiContent  `json:"contents"`
		Tools             []geminiTool     `json:"tools,omitempty"`
		ToolConfig        *geminiToolCfg   `json:"tool_config,omitempty"`
		GenerationConfig  generationConfig `json:"generation_config"`
	}

	geminiContent struct {
		Role  string       `json:"role,omitempty"` // "user" | "model"
		Parts []geminiPart `json:"parts"`
	}

	geminiPart struct {
		Text         string           `json:"text,omitempty"`
		InlineData   *geminiInline    `json:"inline_data,omitempty"`
		FunctionCall *geminiFuncCall  `json:"function_call,omitempty"`
	}

	geminiInline struct {
		MimeType string `json:"mime_type"`
		Data     string `json:"data"` // base64-encoded
	}

	geminiFuncCall struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	}

	geminiTool struct {
		FunctionDeclarations []funcDeclaration `json:"function_declarations"`
	}

	funcDeclaration struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	}

	geminiToolCfg struct {
		FunctionCallingConfig funcCallingConfig `json:"function_calling_config"`
	}

	funcCallingConfig struct {
		Mode                 string   `json:"mode"` // "AUTO" | "ANY" | "NONE"
		AllowedFunctionNames []string `json:"allowed_function_names,omitempty"`
	}

	generationConfig struct {
		MaxOutputTokens int `json:"max_output_tokens"`
	}

	geminiResponse struct {
		Candidates    []geminiCandidate `json:"candidates"`
		UsageMetadata geminiUsage       `json:"usage_metadata"`
	}

	geminiCandidate struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finish_reason"` // "STOP" | "MAX_TOKENS" | "SAFETY" | ...
	}

	geminiUsage struct {
		PromptTokenCount     int `json:"prompt_token_count"`
		CandidatesTokenCount int `json:"candidates_token_count"`
	}
)