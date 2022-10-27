package models

type ExcelToDbRequest struct {
	TableSlug string                 `json:"table_slug"`
	Data      map[string]interface{} `json:"data"`
}

type ExcelToDbResponse struct {
	Message string `json:"message"`
}
