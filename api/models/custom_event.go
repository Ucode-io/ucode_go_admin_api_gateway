package models

type CreateCustomEventRequest struct {
	TableSlug  string                 `json:"table_slug"`
	EventPath  string                 `json:"event_path"`
	Label      string                 `json:"label"`
	Icon       string                 `json:"icon"`
	Url        string                 `json:"url"`
	Disable    bool                   `json:"disable"`
	ActionType string                 `json:"action_type"`
	Method     string                 `json:"method"`
	Attributes  map[string]interface{} `json:"attributes"`
}

type CustomEvent struct {
	Id         string `json:"id"`
	TableSlug  string `json:"table_slug"`
	EventPath  string `json:"event_path"`
	Label      string `json:"label"`
	Icon       string `json:"icon"`
	Url        string `json:"url"`
	Disable    bool   `json:"disable"`
	ActionType string `json:"action_type"`
	Method     string `json:"method"`
	Attributes  map[string]interface{} `json:"attributes"`
}
