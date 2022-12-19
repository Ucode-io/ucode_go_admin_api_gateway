package models

type Company struct {
	CompanyId   string `json:"company_id"`
	Name        string `json:"name"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
}

type CompanyCreateRequest struct {
	Name        string `json:"name"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
}

type CompanyCreateResponse struct {
	CompanyId string `json:"company_id"`
}
