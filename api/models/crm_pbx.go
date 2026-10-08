package models

type CRMPBXRequest struct {
	Data CRMPBXData `json:"data"`
}

type CRMPBXData struct {
	Method string         `json:"method"`
	Object map[string]any `json:"object_data,omitempty"`
}

type CRMPBXIdentity struct {
	Project     string `json:"project_id"`
	Environment string `json:"environment_id"`
	User        string `json:"user_id"`
	AuthUser    string `json:"user_id_auth"`
	Role        string `json:"role_id"`
	ClientType  string `json:"client_type_id"`
	AppVerified bool   `json:"app_context_verified"`
}
