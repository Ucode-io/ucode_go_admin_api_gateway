package gpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

func SendReqToGPT(req []models.Message) ([]models.ToolCall, error) {

	cfg := config.Load()

	requestBody := models.OpenAIRequest{
		Model:        "gpt-3.5-turbo",
		Messages:     req,
		Functions:    GetDefaultFunctions(),
		FunctionCall: "auto",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	reqBody, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	reqBody.Header.Set("Content-Type", "application/json")
	reqBody.Header.Set("Authorization", "Bearer "+cfg.OpenAIApiKey)

	client := &http.Client{}
	resp, err := client.Do(reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response models.OpenAIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response.Choices[0].Message.ToolCalls, nil
}

func GetDefaultFunctions() []models.Tool {
	return []models.Tool{
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "create_menu",
				Description: "Create menu with given name",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the menu",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "create_table",
				Description: "Create table with given name",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the table",
						},
						"table_slug": map[string]interface{}{
							"type":        "string",
							"description": "The slug of the table",
						},
					},
				},
			},
		},
	}
}
