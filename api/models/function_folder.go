package models

type CreateFunctionFolderRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type FunctionFolder struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Description string `json:"description"`
}
