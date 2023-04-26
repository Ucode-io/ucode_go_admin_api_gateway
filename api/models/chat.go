package models

// Chat ...
type Chat struct {
	Sender_name  string `json:"sender_name"`
	PhoneNumber  string `json:"phone_number"`
	PlatformType string `json:"platform_type"`
}

// CreateChatModule
type CreateChatRequest struct {
	UserId string `json:"user_id"`
	Chat   Chat   `json:"chat"`
}

type UpdateBotToken struct {
	BotId    string `json:"bot_id"`
	BotToken string `json:"bot_token"`
}

type ChatResponse struct {
	Chat_id string `json:"chat_id"`
}

type CreateBotRequest struct {
	BotToken      string `json:"bot_token"`
	EnvironmentID string `json:"environment_id"`
}
