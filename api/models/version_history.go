package models

type CreateVersionHistoryRequest struct {
	ActionSource     string
	ActionType       string
	Previous         string
	Current          string
	UsedEnvironments map[string]bool
	Date             string
	UserInfo         string
	Request          string
}
