package v1

import (
	"strings"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestTelegramNotificationCode(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{text: "/connect ABC123", want: "ABC123"},
		{text: "/connect@ucrmnews_bot ABC123", want: "ABC123"},
		{text: "/start ABC123", want: ""},
		{text: "/connect", want: ""},
	}

	for _, test := range tests {
		if got := telegramNotificationCode(test.text); got != test.want {
			t.Fatalf("telegramNotificationCode(%q) = %q, want %q", test.text, got, test.want)
		}
	}
}

func TestTelegramNotificationTemplate(t *testing.T) {
	settings := defaultTelegramNotificationSettings("company-1", "crm_bot")
	settings.StatusNotifications = []models.TelegramStatusNotification{{ID: "won", Template: "<b>{{deal.amount}}</b>"}}

	value, err := telegramNotificationTemplate(settings, "status_notification", "won")
	if err != nil || value != "<b>{{deal.amount}}</b>" {
		t.Fatalf("status template = %q, %v", value, err)
	}
	if _, err = telegramNotificationTemplate(settings, "status_notification", "missing"); err == nil {
		t.Fatal("missing status template should fail")
	}
}

func TestRenderTelegramNotificationTemplate(t *testing.T) {
	rendered := renderTelegramNotificationTemplate("{{lead.name}} {{deal.amount}} {{report.cpl}}")
	for _, token := range []string{"Azizbek Karimov", "1 200 000 so‘m", "20 000 so‘m"} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("rendered template %q does not contain %q", rendered, token)
		}
	}
}
