package models

type CompanyProject struct {
	Title        string `json:"title"`
	ProjectId    string `json:"project_id"`
	CompanyId    string `json:"company_id"`
	K8SNamespace string `json:"k8s_namespace"`
}

type CompanyProjectCreateRequest struct {
	Title        string `json:"title"`
	CompanyId    string `json:"company_id"`
	K8SNamespace string `json:"k8s_namespace"`
}

type CompanyProjectCreateResponse struct {
	CompanyId string `json:"company_id"`
}
