package models

type CreateQueryRequest struct {
	Title         string                 `json:"title"`
	QueryFolderId string                 `json:"query_folder_id"`
	Attributes    map[string]interface{} `json:"attributes"`
}

type Queries struct {
	Id            string                 `json:"guid"`
	QueryFolderId string                 `json:"query_folder_id"`
	Attributes    map[string]interface{} `json:"attributes"`
}
