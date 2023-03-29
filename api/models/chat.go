package models

// Chat ...
type Chat struct {
	Sender_name string `json:"sender_name"`
	Message     string `json:"message"`
	Types       string `json:"types"`
}

// CreateChatModule
type CreateChatRequest struct {
	UserId string `json:"userid"`
	Chat   Chat   `json:"chat"`
}

type ChatResponse struct {
	Chat_id string `json:"chat_id"`
}
