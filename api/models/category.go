package models

type CreateCategory struct {
	Name       string                 `json:"name" bson:"name"`
	ProjectID  string                 `json:"project_id" bson:"project_id"`
	BaseUrl    string                 `json:"base_url" bson:"base_url"`
	Attributes map[string]interface{} `json:"attributes" bson:"attributes"`
}

type Category struct {
	Guid       string                 `json:"guid" bson:"guid"`
	Name       string                 `json:"name" bson:"name"`
	ProjectID  string                 `json:"project_id" bson:"project_id"`
	BaseUrl    string                 `json:"base_url" bson:"base_url"`
	Attributes map[string]interface{} `json:"attributes" bson:"attributes"`
}

type GetAllCategoriesResponse struct {
	Categories []Category `json:"categories"`
	Count      int64      `json:"count"`
}
