package models

type Function struct {
	ID               string `json:"id"`
	Path             string `json:"path"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	FuncitonFolderId string `json:"function_folder_id"`
}

type CreateFunctionRequest struct {
	Path             string `json:"path"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	CommitId         int64  `json:"-"`
	CommitGuid       string `json:"-"`
	VersionId        string `json:"-"`
	FunctionFolderId string `json:"function_folder_id"`
}

type InvokeFunctionRequest struct {
	FunctionID string   `json:"function_id"`
	ObjectIDs  []string `json:"object_ids"`
	Attributes map[string]interface{}
}

type InvokeFunctionResponse struct {
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

type NewInvokeFunctionRequest struct {
	Data map[string]interface{} `json:"data"`
}

type InvokeFunctionRequestWithAppId struct {
	ObjectIDs []string `json:"object_ids"`
	AppID     string   `json:"app_id"`
}

type GetByIdFunctionResponse struct {
	Password         string `json:"password"`
	URL              string `json:"url"`
	ID               string `json:"id"`
	Path             string `json:"path"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	FuncitonFolderId string `json:"function_folder_id"`
}
