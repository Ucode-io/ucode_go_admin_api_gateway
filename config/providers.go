package config

import "strings"

// AIProvider identifies the LLM backend that serves an individual chat.
// The value is stored on the chat record (Chat.Model) and picked per-chat by the user.
type AIProvider string

const (
	AIProviderClaude AIProvider = "claude"
	AIProviderGemini AIProvider = "gemini"
	AIProviderOpenAI AIProvider = "openai"
)

// ParseAIProvider normalises a raw string into a supported AIProvider.
// Empty, unknown, or malformed input falls back to AIProviderClaude so the
// provider field is never ambiguous downstream.
func ParseAIProvider(s string) AIProvider {
	switch AIProvider(strings.ToLower(strings.TrimSpace(s))) {
	case AIProviderClaude:
		return AIProviderClaude
	case AIProviderGemini:
		return AIProviderGemini
	case AIProviderOpenAI:
		return AIProviderOpenAI
	default:
		return AIProviderClaude
	}
}
