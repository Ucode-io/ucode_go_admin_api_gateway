package anthropic

import (
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
)

var ErrMaxTokens = errors.New("generation stopped: output exceeded the token limit")

type (
	systemBlock struct {
		Type         string     `json:"type"`
		Text         string     `json:"text"`
		CacheControl *cacheCtrl `json:"cache_control,omitempty"`
	}

	cacheCtrl struct {
		Type string `json:"type"` // "ephemeral"
	}

	wireToolRequest struct {
		Model      string                      `json:"model"`
		MaxTokens  int                         `json:"max_tokens"`
		System     []systemBlock               `json:"system,omitempty"`
		Messages   []models.ChatMessage        `json:"messages"`
		Tools      []models.ClaudeFunctionTool `json:"tools"`
		ToolChoice *models.ToolChoice          `json:"tool_choice,omitempty"`
	}
)
