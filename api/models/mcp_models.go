package models

type (
	GeneratePromptMCP struct {
		ProjectId     string `json:"project_id"`
		EnvironmentId string `json:"environment_id"`
		APIKey        string `json:"api_key"`
		UserPrompt    string `json:"user_prompt"`
		Method        string `json:"method"`
		BaseURL       string `json:"base_url"`
	}
)
