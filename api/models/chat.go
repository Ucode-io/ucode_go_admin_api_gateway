package models

import "time"

// Chat ..
type Chat struct {
	Chatid              string     `json:"chatid" binding:"required"`
	Message_sender_name string     `json:"message_sender_name" binding:"required"`
	Messages            string     `json:"messages" binding:"required"`
	Types               string     `json:"types"`
	Project_id          string     `json:"project_id"`
	Created_at          *time.Time `json:"created_at"`
}

// CreateChatModul ..
type CreateChatModul struct {
	Chatid              string `json:"chatid" binding:"required"`
	Message_sender_name string `json:"message_sender_name" binding:"required"`
	Messages            string `json:"messages" binding:"required"`
	Types               string `json:"types"`
	Project_id          string `json:"project_id"`
}
