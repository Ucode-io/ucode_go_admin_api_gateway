package models

import "ucode/ucode_go_api_gateway/services"

type CreateVersionHistoryRequest struct {
	Services  services.ServiceManagerI
	NodeType  string
	ProjectId string

	ActionSource     string
	ActionType       string
	Previous         interface{}
	Current          interface{}
	UsedEnvironments map[string]bool
	Date             string
	UserInfo         string
	Request          interface{}
}
