package models

// JSONResult ..
type JSONResult struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// JSONErrorResponse ..
type JSONErrorResponse struct {
	Error string `json:"error"`
}
