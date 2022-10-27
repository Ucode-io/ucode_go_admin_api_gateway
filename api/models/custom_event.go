package models

type CreateCustomEventRequest struct {
	TableSlug string `json:"table_slug"`
	EventPath string `json:"event_path"`
	Label     string `json:"lanel"`
	Icon      string `json:"icon"`
	Url       string `json:"url"`
	Disable   bool   `json:"disable"`
}

type CustomEvent struct {
	Id        string `json:"id"`
	TableSlug string `json:"table_slug"`
	EventPath string `json:"event_path"`
	Label     string `json:"lanel"`
	Icon      string `json:"icon"`
	Url       string `json:"url"`
	Disable   bool   `json:"disable"`
}
