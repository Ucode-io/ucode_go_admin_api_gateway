package models

type CreateQueryFolderRequest struct {
	Title     string `json:"title"`
	ParentId string `json:"project_id"`
}

type QueryFolder struct {
	Id        string `json:"guid"`
	Title     string `json:"title"`
	ParentId string `json:"project_id"`
}
