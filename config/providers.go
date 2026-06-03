package config

// AIProvider identifies which LLM backend the system routes all agent calls to.
// Controlled by the AI_PROVIDER environment variable (default: "claude").
type AIProvider string

const (
	AIProviderClaude AIProvider = "claude"
	AIProviderGemini AIProvider = "gemini"
	AIProviderOpenAI AIProvider = "openai"
)
