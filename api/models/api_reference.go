package models

type CreateApiReference struct {
	AdditionalUrl    string                 `json:"additional_url"`
	Desc             string                 `json:"desc"`
	Method           string                 `json:"method"`
	CategoryID       string                 `json:"category_id"`
	ExternalUrl      string                 `json:"external_url"`
	Authentification bool                   `json:"authentification"`
	Title            string                 `json:"title"`
	NewWindow        bool                   `json:"new_window"`
	Attributes       map[string]interface{} `json:"attributes"`
	ProjectID        string                 `json:"project_id"`
}

type ApiReference struct {
	Guid             string                 `json:"guid"`
	AdditionalUrl    string                 `json:"additional_url"`
	Desc             string                 `json:"desc"`
	Method           string                 `json:"method"`
	CategoryID       string                 `json:"category_id"`
	ExternalUrl      string                 `json:"external_url"`
	Authentification bool                   `json:"authentification"`
	Title            string                 `json:"title"`
	NewWindow        bool                   `json:"new_window"`
	Attributes       map[string]interface{} `json:"attributes"`
	ProjectID        string                 `json:"project_id"`
}

type GetAllApiReferenceResponse struct {
	ApiReferences []ApiReference `json:"api_refences"`
	Count         int64          `json:"count"`
}
