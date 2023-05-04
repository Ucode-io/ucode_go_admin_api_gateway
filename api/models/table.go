package models

type RevertHistoryRequest struct {
	Id string `json:"id"`
}

type InsertVersionsToCommitRequest struct {
	Id          string   `json:"id"`
	Version_ids []string `json:"version_ids"`
}

type GetAllTablesRequest struct {
	Offset    int64  `json:"offset"`
	Limit     int64  `json:"limit"`
	Search    string `json:"search"`
	ProjectId string `json:"project-id"`
	VersionId string `json:"version_id"`
	FolderId  string `json:"folder-id"`
}
