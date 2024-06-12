package models

type CacheRequest struct {
	ProjectId string
	Key       string
	Value     map[string]interface{} `json:"value"`
	NodeType  string
	Method    string
	Keys      []string
}

type CacheResponse struct {
	Value map[string]interface{} `json:"value"`
}
