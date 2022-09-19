package models

type CommonInput struct {
	Query string                 `json:"query"`
	Data  map[string]interface{} `json:"data"`
}
