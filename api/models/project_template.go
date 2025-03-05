package models

import obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

type CopyProjectTemplateRequest struct {
	Menus         []*obs.MenuRequest `json:"menus"`
	FromEnvId     string             `json:"from_env_id"`
	ToEnvId       string             `json:"to_env_id"`
	FromProjectId string             `json:"from_project_id"`
	ToProjectId   string             `json:"to_project_id"`
}
