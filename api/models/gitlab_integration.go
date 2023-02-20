package models

type CreateProject struct {
	NamespaceID          int    `json:"namespace_id"`
	Name                 string `json:"name"`
	InitializeWithReadme bool   `json:"initialize_with_readme"`
	DefaultBranch        string `json:"default_branch"`
	Visibility           string `json:"visibility"`
	Path                 string `json:"path"`
}

type GitlabIntegrationResponse struct {
	Code    int                    `json:"code"`
	Message map[string]interface{} `json:""`
}

type ResponseCreateFunction struct {
	Password string `json:"string"`
	URL      string `json:"url"`
}
