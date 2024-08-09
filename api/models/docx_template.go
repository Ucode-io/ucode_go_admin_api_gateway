package models

type DocxTemplateVariables struct {
	ID        string                 `json:"id"`
	TableSlug string                 `json:"table_slug"`
	Data      map[string]interface{} `json:"data"`
}
