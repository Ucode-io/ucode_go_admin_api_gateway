package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"ucode/ucode_go_api_gateway/config"
)

func main() {
	str, _ := GetOfsCode(Message{
		Role: "user",
		// Content: "write function get data from req and give it to create api table slug is: customers",
		Content: "Create function after CREATE in table Employee. Which should do: 'get data from req in data get birth_date and guid for update from it get age and give it to update api table slug is customers'",
	})
	fmt.Println(str)
}

func GetOfsCode(req Message) (string, error) {

	cfg := config.Load()

	defMessages := GetDefaultMsssages()

	defMessages = append(defMessages, req)

	requestBody := OpenAIRequest{
		Model:    "gpt-3.5-turbo-0125",
		Messages: defMessages,

		// Functions:    GetDefaultFunctions(),
		// FunctionCall: "auto",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	reqBody, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	reqBody.Header.Set("Content-Type", "application/json")
	reqBody.Header.Set("Authorization", "Bearer "+cfg.OpenAIApiKey)

	client := &http.Client{}
	resp, err := client.Do(reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response OpenAIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if response.Error.Message != "" {
		return "", fmt.Errorf("not full information given")
	}

	return response.Choices[0].Message.Content, nil
}

func GetDefaultMsssages() []Message {
	return []Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant that write open fass codes in golang",
		},
		{
			Role:    "system",
			Content: fmt.Sprintf("There is template for you for codeing ```%s```", GetTemplateCode()),
		},
		{
			Role:    "system",
			Content: "To convert variables use cast.ToInt, cast.ToString and etc...",
		},
		{
			Role:    "system",
			Content: "Return only function Handle() without any comments out of Handle() only function Handle()",
		},
		// {
		// 	Role:    "system",
		// 	Content: "let other constants and other bodies after your user or developer should or can use it",
		// },
	}
}

func GetTemplateCode() string {

	data, err := os.ReadFile("./pkg/gpt/ofs.txt")
	if err != nil {
		log.Fatal(err)
	}

	return string(data)
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type OpenAIResponse struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Created           int         `json:"created"`
	Model             string      `json:"model"`
	Choices           []Choice    `json:"choices"`
	Usage             Usage       `json:"usage"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
	Error             ErrorAI     `json:"error"`
}

type ErrorAI struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
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
