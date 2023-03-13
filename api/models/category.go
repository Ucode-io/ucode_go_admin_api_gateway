package models

type CreateCategory struct {
	Name       string                 `json:"name"`
	ProjectID  string                 `json:"project_id"`
	BaseUrl    string                 `json:"base_url"`
	Attributes map[string]interface{} `json:"attributes"`
}

type Category struct {
	Guid       string                 `json:"guid"`
	Name       string                 `json:"name"`
	ProjectID  string                 `json:"project_id"`
	BaseUrl    string                 `json:"base_url"`
	Attributes map[string]interface{} `json:"attributes"`
	CommitId   string                 `json:"commit_id"`
	VersionId  string                 `json:"version_id"`
}

type GetAllCategoriesResponse struct {
	Categories []Category `json:"categories"`
	Count      int64      `json:"count"`
}

type UpdateCategoryRequest struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}
