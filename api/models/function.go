package models

type Function struct {
	ID          string                 `json:"id"`
	Path        string                 `json:"path"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Body        map[string]interface{} `json:"body"`
	Url         string                 `json:"url"`
}

type CreateFunctionRequest struct {
	Path        string                 `json:"path"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Body        map[string]interface{} `json:"body"`
	Url         string                 `json:"url"`
	CommitId    int64                  `json:"-"`
	CommitGuid  string                 `json:"-"`
	VersionId     string                 `json:"-"`
}

type InvokeFunctionRequest struct {
	FunctionID string   `json:"function_id"`
	ObjectIDs  []string `json:"object_ids"`
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
