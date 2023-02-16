package models

type TemplateShareRes struct {
	Data map[string]interface{} `json:"data"`
	Role string                 `json:"role"`
}
