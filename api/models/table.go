package models

type RevertHistoryRequest struct {
	Id string `json:"id"`
}

type InsertVersionsToCommitRequest struct {
	Id          string   `json:"id"`
	Version_ids []string `json:"version_ids"`
}

type GetAllTablesRequest struct {
	Offset       int64  `json:"offset"`
	Limit        int64  `json:"limit"`
	Search       string `json:"search"`
	ProjectId    string `json:"project-id"`
	VersionId    string `json:"version_id"`
	FolderId     string `json:"folder_id"`
	IsLoginTable bool   `json:"is_login_table"`
}

type UpdateTableRequest struct {
	Id                string                 `json:"id"`
	Label             string                 `json:"label"`
	Description       string                 `json:"description"`
	Slug              string                 `json:"slug"`
	ShowInMenu        bool                   `json:"show_in_menu"`
	Icon              string                 `json:"icon"`
	SubtitleFieldSlug string                 `json:"subtitle_field_slug"`
	IsVisible         bool                   `json:"is_visible"`
	IsOwnTable        bool                   `json:"is_own_table"`
	IncrementId       IncrementId            `json:"increment_id"`
	ProjectId         string                 `json:"project_id"`
	FolderId          string                 `json:"folder_id"`
	AuthorId          string                 `json:"author_id"`
	CommitType        string                 `json:"commit_type"`
	Name              string                 `json:"name"`
	IsCached          bool                   `json:"is_cached"`
	IsLoginTable      bool                   `json:"is_login_table"`
	Attributes        map[string]interface{} `json:"attributes"`
	OrderBy           bool                   `json:"order_by"`
	SoftDelete        bool                   `json:"soft_delete"`
}
