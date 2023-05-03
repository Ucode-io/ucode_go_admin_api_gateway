package models

type CreateTableFolderRequest struct {
	Title     string `json:"title"`
	ParentdId string `json:"parentd_id"`
}

type GetAllTableFoldersRequest struct {
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
	Search string `json:"search"`
}
