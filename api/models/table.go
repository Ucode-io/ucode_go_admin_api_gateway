package models

type RevertHistoryRequest struct {
	Id string `json:"id"`
}

type InsertVersionsToCommitRequest struct {
	Id          string   `json:"id"`
	Version_ids []string `json:"version_ids"`
}
