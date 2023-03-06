package models

type CreateFolderReqModel struct {
	ProjectId     string `json:"project_id,omitempty"`
	Title         string `json:"title,omitempty"`
	EnvironmentId string `json:"environment_id,omitempty"`
}
