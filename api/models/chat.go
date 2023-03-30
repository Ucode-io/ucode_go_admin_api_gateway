package models

// Chat ...
type Chat struct {
	Sender_name string `json:"sender_name"`
	Message     string `json:"message"`
	Types       string `json:"types"`
}

// CreateChatModule
type CreateChatRequest struct {
	UserId string `json:"user_id"`
	Chat   Chat   `json:"chat"`
}

type ChatResponse struct {
	Chat_id string `json:"chat_id"`
}

// Chat ...
type UserMessage struct {
	Sender_name string `json:"sender_name"`
	Message     string `json:"message"`
	CreatedAt   string `json:"created_at"`
	UserId      string `json:"userid"`
	Types       string `json:"types"`
}

// CreateChatModule
type CreateMessageRequest struct {
	ChatId string      `json:"chatid"`
	Chat   UserMessage `json:"chat"`
}
