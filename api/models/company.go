package models

type Company struct {
	CompanyId   string `json:"company_id"`
	Title       string `json:"title"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
}

type CompanyCreateRequest struct {
	Title       string `json:"title"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
}

type CompanyCreateResponse struct {
	CompanyId string `json:"company_id"`
}
