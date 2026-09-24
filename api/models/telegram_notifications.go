package models

type TelegramStatusNotification struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	PipelineID string `json:"pipeline_id"`
	StageID    string `json:"stage_id"`
	Template   string `json:"template"`
}

type TelegramAutomationCondition struct {
	ID       string `json:"id"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type TelegramAutomationTrigger struct {
	ID            string                           `json:"id"`
	Kind          string                           `json:"kind"`
	Table         string                           `json:"table"`
	WatchField    string                           `json:"watch_field"`
	ReportTime    string                           `json:"report_time"`
	ConditionMode string                           `json:"condition_mode"`
	Conditions    []TelegramAutomationCondition    `json:"conditions"`
	MessageFields []string                         `json:"message_fields"`
	Template      string                           `json:"template"`
	FieldOptions  map[string][]TelegramFieldOption `json:"field_options,omitempty"`
	StatusField   string                           `json:"status_field,omitempty"`
	StatusButtons []TelegramStatusButton           `json:"status_buttons,omitempty"`
}

type TelegramFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Slug  string `json:"slug,omitempty"`
}

type TelegramStatusButton struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type TelegramAutomation struct {
	ID             string                        `json:"id"`
	Name           string                        `json:"name"`
	Enabled        bool                          `json:"enabled"`
	Trigger        string                        `json:"trigger"`
	Triggers       []string                      `json:"triggers,omitempty"`
	DealEvent      string                        `json:"deal_event"`
	ReportTime     string                        `json:"report_time"`
	ConditionMode  string                        `json:"condition_mode"`
	Conditions     []TelegramAutomationCondition `json:"conditions"`
	Action         string                        `json:"action"`
	Template       string                        `json:"template"`
	TriggerConfigs []TelegramAutomationTrigger   `json:"trigger_configs,omitempty"`
	ChatID         string                        `json:"chat_id,omitempty"`
}

type TelegramNotificationGroup struct {
	ChatID    string `json:"chat_id"`
	ChatTitle string `json:"chat_title"`
}

type TelegramNotificationTemplates struct {
	NewLead     string `json:"new_lead"`
	DailyReport string `json:"daily_report"`
}

type TelegramNotificationSettings struct {
	CompanyID           string                        `json:"company_id"`
	MetaAdsAccountID    string                        `json:"meta_ads_account_id,omitempty"`
	ChatID              string                        `json:"chat_id"`
	ChatTitle           string                        `json:"chat_title"`
	BotUsername         string                        `json:"bot_username"`
	NewLeadEnabled      bool                          `json:"new_lead_enabled"`
	DailyReportEnabled  bool                          `json:"daily_report_enabled"`
	ReportTime          string                        `json:"report_time"`
	Timezone            string                        `json:"timezone"`
	Templates           TelegramNotificationTemplates `json:"templates"`
	StatusNotifications []TelegramStatusNotification  `json:"status_notifications"`
	Automations         []TelegramAutomation          `json:"automations"`
	Groups              []TelegramNotificationGroup   `json:"groups,omitempty"`
}

type TelegramNotificationTestRequest struct {
	CompanyID string `json:"company_id"`
	Type      string `json:"type"`
	RuleID    string `json:"rule_id"`
	TriggerID string `json:"trigger_id"`
}
