package models

import (
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage"
)

type SendToGptRequest struct {
	Promt string `json:"promt"`
}

type OpenAIRequest struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	Functions    []Tool    `json:"tools"`
	FunctionCall string    `json:"tool_choice"`
}

type OpenAIResponse struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Created           int         `json:"created"`
	Model             string      `json:"model"`
	Choices           []Choice    `json:"choices"`
	Usage             Usage       `json:"usage"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type FunctionTool struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionTool `json:"function"`
}

type MessageChoice struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      MessageChoice `json:"message"`
	Logprobs     interface{}   `json:"logprobs"`
	FinishReason string        `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Type     string              `json:"type"`
	Function FunctionDescription `json:"function"`
}

type FunctionDescription struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type CreateMenuAI struct {
	Label    string
	UserId   string
	Resource *pb.ServiceResourceModel
	Service  services.ServiceManagerI
	Redis    storage.RedisStorageI
	Promt    string
}

type DeleteMenuAI struct {
	Label    string
	UserId   string
	Resource *pb.ServiceResourceModel
	Service  services.ServiceManagerI
}

type UpdateMenuAI struct {
	NewLabel string
	OldLabel string
	UserId   string
	Resource *pb.ServiceResourceModel
	Service  services.ServiceManagerI
}

type CreateTableAI struct {
	Label         string
	UserId        string
	TableSlug     string
	EnvironmentId string
	Menu          string
	Resource      *pb.ServiceResourceModel
	Service       services.ServiceManagerI
}

type UpdateTableAI struct {
	NewLabel      string
	OldLabel      string
	UserId        string
	EnvironmentId string
	Resource      *pb.ServiceResourceModel
	Service       services.ServiceManagerI
}
