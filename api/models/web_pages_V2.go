package models

import (
	"google.golang.org/protobuf/types/known/structpb"
)

type CreateFolderReqModel struct {
	ProjectId string `json:"project_id,omitempty"`
	Title     string `json:"title,omitempty"`
}

type UpdateFolderReqModel struct {
	Id        string `json:"id,omitempty"`
	ProjectId string `json:"project_id,omitempty"`
	Title     string `json:"title,omitempty"`
}

type CreateWebPageReqModel struct {
	Title      string           `json:"title,omitempty"`
	ProjectId  string           `json:"project_id,omitempty"`
	FolderId   string           `json:"folder_id,omitempty"`
	AppId      string           `json:"app_id,omitempty"`
	Components *structpb.Struct `json:"components,omitempty"`
	Icon       string           `json:"icon,omitempty"`
}

type UpdateWebPageReqModel struct {
	Id         string           `json:"id,omitempty"`
	Title      string           `json:"title,omitempty"`
	ProjectId  string           `json:"project_id,omitempty"`
	FolderId   string           `json:"folder_id,omitempty"`
	AppId      string           `json:"app_id,omitempty"`
	Components *structpb.Struct `json:"components,omitempty"`
	Icon       string           `json:"icon,omitempty"`
}

type RevertWebPageReqModel struct {
	VersionId   string `json:"version_id,omitempty"`
	Id          string `json:"id,omitempty"`
	OldCommitId string `json:"old_commit_id,omitempty"`
	ProjectId   string `json:"project_id,omitempty"`
}

type ManyVersionsModel struct {
	VersionIds  []string `json:"version_ids,omitempty"`
	ProjectId   string   `json:"project_id,omitempty"`
	Id          string   `json:"id,omitempty"`
	OldCommitId string   `json:"old_commit_id,omitempty"`
}
