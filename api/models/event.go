package models

type EventLog struct {
	Guid          string `json:"guid" bson:"guid"`
	ID            string `json:"id" bson:"id"`
	Date          string `json:"date" bson:"date"`
	TableSlug     string `json:"table_slug" bson:"table_slug"`
	EffectedTable string `json:"effected_table" bson:"effected_table"`
}

type GetEventLogsListRequest struct {
	Limit     string `json:"limit" bson:"limit"`
	Page      string `json:"page" bson:"page"`
	TableSlug string `json:"table_slug" bson:"table_slug"`
}

type GetEventLogListsResponse struct {
	EventLogs []EventLog `json:"event_logs" bson:"event_logs"`
	Count     int32      `json:"count" bson:"count"`
}
