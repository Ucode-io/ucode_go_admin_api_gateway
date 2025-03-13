package models

type CommonInput struct {
	Query string         `json:"query"`
	Data  map[string]any `json:"data"`
}
