package v1

import (
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestTelegramCallMetricsForDayDeduplicatesAndCountsTalkTime(t *testing.T) {
	day := time.Date(2026, 9, 25, 0, 0, 0, 0, telegramReportLocation())
	rows := []map[string]any{
		{"call_uuid": "one", "client_phone": "+998 90 123 45 67", "started_at": "2026-09-25T10:00:00+05:00", "duration": 0, "status": "initiated"},
		{"call_uuid": "one", "client_phone": "+998901234567", "started_at": "2026-09-25T10:00:00+05:00", "duration": 120, "status": "answered"},
		{"call_uuid": "two", "client_phone": "+998 90 123 45 67", "started_at": "2026-09-25T11:00:00+05:00", "duration": 60, "status": "answered"},
		{"call_uuid": "three", "client_phone": "+998 90 000 00 00", "started_at": "2026-09-25T12:00:00+05:00", "duration": 0, "status": "no_answer"},
		{"call_uuid": "yesterday", "client_phone": "+998 90 123 45 67", "started_at": "2026-09-24T10:00:00+05:00", "duration": 10, "status": "answered"},
	}
	got := telegramCallMetricsForDay(rows, day, map[string]bool{"901234567": true})
	if got.Total != 3 || got.Leads != 1 || got.Seconds != 180 {
		t.Fatalf("metrics = %#v", got)
	}
}

func TestTelegramAutomationButtonsStayAvailable(t *testing.T) {
	trigger := models.TelegramAutomationTrigger{StatusButtons: []models.TelegramStatusButton{
		{ID: "load", Label: "📦 Yuklab berildi", Value: "loaded"},
		{ID: "send", Label: "🚚 Yuk jo‘natildi", Value: "sent"},
	}}
	markup := telegramAutomationButtons("token", trigger, "sent")
	buttons := markup["inline_keyboard"].([][]map[string]string)
	if len(buttons) != 2 || buttons[0][0]["callback_data"] != "crm:token:load" || buttons[1][0]["text"] != "✅ 🚚 Yuk jo‘natildi" {
		t.Fatalf("buttons = %#v", buttons)
	}
}
