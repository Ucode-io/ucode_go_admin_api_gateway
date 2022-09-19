package models

type CommonMessage struct {
	Data map[string]interface{} `json:"data"`
}

type HtmlBody struct {
	Data map[string]interface{} `json:"data"`
	Html string                 `json:"html"`
}

type GetListRequest struct {
	TableSlug string `json:"table_slug"`
	Search    string `json:"search"`
	Limit     int32  `json:"limit"`
	Offset    int32  `json:"offset"`
}

type UpsertCommonMessage struct {
	Data map[string]interface{} `json:"data"`
	UpdatedFields []string `json:"updated_fields"`
}