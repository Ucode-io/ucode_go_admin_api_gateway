package models

type CreateWebPageRequest struct {
	Title      string         `json:"title"`
	Components map[string]any `json:"components"`
}

type WebPage struct {
	Id         string         `json:"guid"`
	Title      string         `json:"title"`
	Components map[string]any `json:"components"`
}
