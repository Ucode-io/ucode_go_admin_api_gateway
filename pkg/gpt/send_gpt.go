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

	if response.Error.Message != "" {
		return nil, fmt.Errorf("not full information given")
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
							"description": "A slug generated from the name by translating it to English and converting it to lowercase. Spaces and special characters should be replaced with underscores.",
						},
						"menu": map[string]interface{}{
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
				Name:        "update_table",
				Description: "Update table with given name",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_name": map[string]interface{}{
							"type":        "string",
							"description": "The current name of the table that needs to be updated",
						},
						"new_name": map[string]interface{}{
							"type":        "string",
							"description": "The new name to be assigned to the table",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "delete_menu",
				Description: "Delete menu with given name",
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
				Name:        "update_menu",
				Description: "Update menu with given name",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_name": map[string]interface{}{
							"type":        "string",
							"description": "The old name of the menu",
						},
						"new_name": map[string]interface{}{
							"type":        "string",
							"description": "The new name of the menu",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "delete_table",
				Description: "Delete table with given name",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the table",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "create_field",
				Description: "Create a field with the given name or create many fields",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"label": map[string]interface{}{
							"type":        "string",
							"description": "The label of the field",
						},
						"slug": map[string]interface{}{
							"type":        "string",
							"description": "The slug of the field, derived from the label",
							"transform":   "translate to English, lowercase, replace spaces with underscores",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "The type of the field",
							"enum": []string{"EMAIL", "PHOTO", "TIME", "MULTISELECT", "RANDOM_NUMBERS", "FILE",
								"INCREMENT_NUMBER", "PHONE", "DATE_TIME", "FLOAT_NOLIMIT",
								"MULTI_IMAGE", "DATE_TIME_WITHOUT_TIME_ZONE", "MULTI_LINE", "CHECKBOX",
								"SWITCH", "FORMULA_FRONTEND", "NUMBER", "FLOAT", "FORMULA",
								"SINGLE_LINE", "PASSWORD", "CODABAR", "INTERNATIONAL_PHONE",
								"UUID", "INCREMENT_ID", "DATE", "MAP",
							},
							"inference": "based on label and description default return SINGLE_LINE",
						},
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The name of the table",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "update_field",
				Description: "Update field with given parameters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_label": map[string]interface{}{
							"type":        "string",
							"description": "The old label of the field",
						},
						"new_label": map[string]interface{}{
							"type":        "string",
							"description": "The new label of the field",
						},
						"new_type": map[string]interface{}{
							"type":        "string",
							"description": "The new type of the field",
							"enum": []string{"EMAIL", "PHOTO", "TIME", "MULTISELECT", "RANDOM_NUMBERS", "FILE",
								"INCREMENT_NUMBER", "PHONE", "DATE_TIME", "FLOAT_NOLIMIT",
								"MULTI_IMAGE", "DATE_TIME_WITHOUT_TIME_ZONE", "MULTI_LINE", "CHECKBOX",
								"SWITCH", "FORMULA_FRONTEND", "NUMBER", "FLOAT", "FORMULA",
								"SINGLE_LINE", "PASSWORD", "CODABAR", "INTERNATIONAL_PHONE",
								"UUID", "INCREMENT_ID", "DATE", "MAP",
							},
						},
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The table the field belongs to",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "delete_field",
				Description: "Delete field with given parameters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"label": map[string]interface{}{
							"type":        "string",
							"description": "The label of the field",
						},
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The table the field belongs to",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "create_relation",
				Description: "Create relation with given parameters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"table_from": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table from",
						},
						"table_to": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table to",
						},
						"relation_type": map[string]interface{}{
							"type":        "string",
							"description": "The relation type default Many2One. If table_to and table_from is equal return Recursive",
							"enum":        []string{"Many2One", "Many2Many", "Recursive"},
						},
						"view_field": map[string]interface{}{
							"type":        "array",
							"description": "The label of view field",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"view_type": map[string]interface{}{
							"type":        "string",
							"description": "When relation_type is Many2Many return INPUT or TABLE looking for description. Default INPUT",
							"enum":        []string{"INPUT", "TABLE"},
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "delete_relation",
				Description: "Delete relation with given parameters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"table_from": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table from",
						},
						"table_to": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table to",
						},
						"relation_type": map[string]interface{}{
							"type":        "string",
							"description": "The type of relation",
							"enum":        []string{"Many2One", "Many2Many", "Recursive"},
							"default":     "",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "create_row",
				Description: "Create row or item with given arguments",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table",
						},
						"arguments": map[string]interface{}{
							"type":        "array",
							"description": "The argument to create row or item, An array of arguments used to create the row or item. Date strings will be dynamically converted to RFC3339 format.",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "generate_row",
				Description: "Create row or item with given arguments",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table",
						},
						"count": map[string]interface{}{
							"type":        "string",
							"description": "The count of rows which should generate",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "generate_values",
				Description: "Generate values for given columns",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"arguments": map[string]interface{}{
							"type":        "array",
							"description": "Array of object which generated",
							"items": map[string]interface{}{
								"type":        "object",
								"description": "Object which generated",
								"properties": map[string]interface{}{
									"key": map[string]interface{}{
										"type":        "string",
										"description": "this column given in promt",
									},
									"value": map[string]interface{}{
										"type":        "string",
										"description": "generate by yourself value based on key or column",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "update_item",
				Description: "Update item for given table",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_data": map[string]interface{}{
							"type":        "string",
							"description": "Old data of object",
						},
						"new_data": map[string]interface{}{
							"type":        "string",
							"description": "New data of object",
						},
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "delete_item",
				Description: "Delete item for given table",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_data": map[string]interface{}{
							"type":        "string",
							"description": "Old data of object",
						},
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: models.FunctionDescription{
				Name:        "login_table",
				Description: "Change to Login table with given Table label",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"table": map[string]interface{}{
							"type":        "string",
							"description": "The label of the table",
						},
						"login": map[string]interface{}{
							"type":        "string",
							"description": "The login label of login table",
						},
						"password": map[string]interface{}{
							"type":        "string",
							"description": "The password label of login table",
						},
					},
				},
			},
		},
	}
}
