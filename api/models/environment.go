package models

type Environment struct {
	Id           string                 `json:"id"`
	ProjectId    string                 `json:"project_id"`
	Name         string                 `json:"name"`
	DisplayColor string                 `json:"display_color"`
	Description  string                 `json:"description"`
	Data         map[string]interface{} `json:"data"`
}
