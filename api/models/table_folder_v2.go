package models

type CreateTableFolderRequest struct {
	Name string `json:"name"`
}

type TableFolder struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
