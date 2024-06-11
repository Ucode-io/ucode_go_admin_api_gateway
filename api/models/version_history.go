package models

import "ucode/ucode_go_api_gateway/services"

type CreateVersionHistoryRequest struct {
	Services  services.ServiceManagerI
	NodeType  string
	ProjectId string

	Id               string
	ActionSource     string          `json:"action_source"`
	ActionType       string          `json:"action_type"`
	Previous         interface{}     `json:"previous"`
	Current          interface{}     `json:"current"`
	UsedEnvironments map[string]bool `json:"used_environments"`
	Date             string          `json:"date"`
	UserInfo         string          `json:"user_info"`
	Request          interface{}     `json:"request"`
	Response         interface{}     `json:"response"`
	ApiKey           string          `json:"api_key"`
	Type             string          `json:"type"`
	TableSlug        string          `json:"table_slug"`
	VersionId        string          `json:"version_id"`
	ResourceType     int
}

type MigrateUp struct {
	Id               string          `json:"id"`
	ActionSource     string          `json:"action_source"`
	ActionType       string          `json:"action_type"`
	Previous         interface{}     `json:"previus"`
	Current          interface{}     `json:"current"`
	UsedEnvironments map[string]bool `json:"used_envrironments"`
	Date             string          `json:"date"`
	UserInfo         string          `json:"user_info"`
	Request          interface{}     `json:"request"`
	Response         interface{}     `json:"response"`
	ApiKey           string          `json:"api_key"`
	Type             string          `json:"type"`
	TableSlug        string          `json:"table_slug"`
	VersionId        string          `json:"version_id"`
}

type MigrateUpRequest struct {
	Data []*MigrateUp `json:"data"`
}

type MigrateUpResponse struct {
	Ids []string `json:"ids"`
}

type PublishVersionRequest struct {
	PublishedEnvironmentID string `json:"to_environment_id"`
	PublishedVersionID     string `json:"published_version_id"`
}
