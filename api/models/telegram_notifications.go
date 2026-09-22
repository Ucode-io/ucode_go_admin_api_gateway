package models

type TelegramStatusNotification struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	PipelineID string `json:"pipeline_id"`
	StageID    string `json:"stage_id"`
	Template   string `json:"template"`
}

type TelegramNotificationTemplates struct {
	NewLead     string `json:"new_lead"`
	DailyReport string `json:"daily_report"`
}

type TelegramNotificationSettings struct {
	CompanyID           string                        `json:"company_id"`
	ChatID              string                        `json:"chat_id"`
	ChatTitle           string                        `json:"chat_title"`
	BotUsername         string                        `json:"bot_username"`
	NewLeadEnabled      bool                          `json:"new_lead_enabled"`
	DailyReportEnabled  bool                          `json:"daily_report_enabled"`
	ReportTime          string                        `json:"report_time"`
	Timezone            string                        `json:"timezone"`
	Templates           TelegramNotificationTemplates `json:"templates"`
	StatusNotifications []TelegramStatusNotification  `json:"status_notifications"`
}

type TelegramNotificationTestRequest struct {
	CompanyID string `json:"company_id"`
	Type      string `json:"type"`
	RuleID    string `json:"rule_id"`
}
