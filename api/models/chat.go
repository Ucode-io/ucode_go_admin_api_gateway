package models

// Chat ...
type Chat struct {
	Chat_id        string `json:"chat_id"`
	Sender_name    string `json:"sender_name"`
	Message        string `json:"message"`
	Types          string `json:"types"`
	Environment_id string `json:"environment_id"`
	Created_at     string `json:"created_at"`
}

// CreateChatModule
type CreateChatRequest struct {
	UserId string `json:"userid"`
	Chat   Chat   `json:"chat"`
}

type ChatResponse struct {
	Chat_id string `json:"chat_id"`
}
