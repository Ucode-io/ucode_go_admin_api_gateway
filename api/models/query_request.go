package models

type QueryRevertRequest struct {
	CommitId  string `json:"commit_id"`
	ProjectId string `json:"project_id"`
}
