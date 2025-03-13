package models

type CreateQueryRequest struct {
	Title         string         `json:"title"`
	QueryFolderId string         `json:"query_folder_id"`
	Attributes    map[string]any `json:"attributes"`
}

type Queries struct {
	Id            string         `json:"guid"`
	QueryFolderId string         `json:"query_folder_id"`
	Attributes    map[string]any `json:"attributes"`
}

type QueryLog struct {
	Id            string         `json:"id"`
	QueryId       string         `json:"query_id"`
	UserId        string         `json:"user_id"`
	ProjectId     string         `json:"project_id"`
	EnvironmentId string         `json:"environment_id"`
	Request       map[string]any `json:"request"`
	Response      string         `json:"response"`
	Duration      float32        `json:"duration"`
	UserData      map[string]any `json:"user_data"`
}

type QueryLogList struct {
	Logs  []QueryLog `json:"logs"`
	Count int        `json:"count"`
}
