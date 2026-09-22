package v1

import (
	"strings"
	"testing"
	"time"

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

func TestRenderTelegramDailyReportValues(t *testing.T) {
	template := "<b>{{report.date}}</b> {{report.ad_spend}} {{report.leads_total}} {{report.statuses}}"
	rendered := renderTelegramTemplateValues(template, map[string]string{
		"{{report.date}}":        "22.09.2026",
		"{{report.ad_spend}}":    "UZS 125000.00",
		"{{report.leads_total}}": "7",
		"{{report.statuses}}":    "• Yangi — 5\n• Bog'lanildi — 2",
	})
	for _, value := range []string{"22.09.2026", "UZS 125000.00", "7", "Yangi — 5", "Bog&#39;lanildi — 2"} {
		if !strings.Contains(rendered, value) {
			t.Fatalf("rendered daily report %q does not contain %q", rendered, value)
		}
	}
}

func TestTelegramDailyReportHelpers(t *testing.T) {
	target := telegramNotificationTarget{ProjectID: "project", EnvironmentID: "environment", CompanyID: "company"}
	decoded, ok := parseTelegramNotificationTarget(encodeTelegramNotificationTarget(target))
	if !ok || decoded != target {
		t.Fatalf("target round trip = %#v, %v", decoded, ok)
	}
	createdAt, ok := telegramDealCreatedAt(map[string]any{"created_at": "2026-09-22T11:30:00Z"}, time.UTC)
	if !ok || createdAt.Format(time.RFC3339) != "2026-09-22T11:30:00Z" {
		t.Fatalf("created at = %s, %v", createdAt, ok)
	}
}

func TestTelegramStatusRuleMatchesArrayBackedDealFields(t *testing.T) {
	// The deals table stores these choice fields as arrays, even though the
	// settings UI presents each as a single select.
	rule := models.TelegramStatusNotification{
		PipelineID: "UHRMS",
		StageID:    "Выиграно",
	}
	after := map[string]any{
		"pipeline": []any{map[string]any{"value": "UHRMS"}},
		"stage":    []any{map[string]any{"label": "Выиграно"}},
	}
	if !telegramStatusRuleMatches(rule, after) {
		t.Fatal("array-backed deal fields should match the selected status rule")
	}
	if telegramDealStage(map[string]any{"stage": []any{"Переговоры"}}) == telegramDealStage(after) {
		t.Fatal("different deal stages must be distinguishable")
	}
}

func TestRenderTelegramTemplateReplacesMissingFields(t *testing.T) {
	rendered := renderTelegramNotificationTemplateWithDeal("{{lead.name}} {{lead.phone}} {{deal.custom_field}}", map[string]any{"name": "Dilnoza"})
	if rendered != "Dilnoza — —" {
		t.Fatalf("rendered missing fields = %q", rendered)
	}
}

func TestTelegramStatusRuleMatchesLegacyWinningStageSpelling(t *testing.T) {
	rule := models.TelegramStatusNotification{PipelineID: "UHRMS", StageID: "Выиграно"}
	deal := map[string]any{"pipeline": "UHRMS", "stage": "Выграно"}
	if !telegramStatusRuleMatches(rule, deal) {
		t.Fatal("legacy winning-stage spelling should match its configured rule")
	}
}

func TestTelegramStatusRuleMatchesPipelineSpecificStageField(t *testing.T) {
	rule := models.TelegramStatusNotification{PipelineID: "UHRMS", StageID: "Выиграно"}
	deal := map[string]any{"pipeline": "UHRMS", "pipeline_uhrms": "Выграно"}
	if !telegramStatusRuleMatches(rule, deal) {
		t.Fatal("pipeline-specific stage field should match the selected status rule")
	}
}
