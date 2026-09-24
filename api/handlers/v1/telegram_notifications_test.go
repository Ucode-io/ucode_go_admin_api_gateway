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
	rendered := renderTelegramNotificationTemplate("{{lead.name}} {{deal.amount}} {{deal.url}} {{report.cpl}}")
	for _, token := range []string{"Azizbek Karimov", "1 200 000 so‘m", "https://crm.ucode.co/deals", "20 000 so‘m"} {
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
	localTime, ok := telegramDealCreatedAt(map[string]any{"created_at": "2026-09-24T11:51:00Z"}, telegramReportLocation())
	if !ok || localTime.Format("2006-01-02 15:04") != "2026-09-24 16:51" {
		t.Fatalf("Tashkent report time = %s, %v", localTime, ok)
	}
}

func TestTelegramDailyReportGroupsTodayLeadsIntoExpandableStatuses(t *testing.T) {
	location := time.FixedZone("Asia/Tashkent", 5*60*60)
	day := time.Date(2026, 9, 24, 21, 0, 0, 0, location)
	rows := []map[string]any{
		{"created_at": "2026-09-24T06:20:00Z", "stage": "Yuk Jonatildi", "name": "Samandar", "phone": "+998 93 345 67 89"},
		{"created_at": "2026-09-24T04:15:00Z", "stage": "Yuk Jonatildi", "name": "Asadbek & Ali", "phone": "+998 90 123 45 67"},
		{"created_at": "2026-09-23T18:00:00Z", "stage": "Yuk Jonatildi", "name": "Kecha"},
		{"created_at": "2026-09-24T08:00:00Z", "stage": "Yuklab Berildi", "name": "Nurmuhammad"},
	}
	got := telegramFormatDealStatusesForDay(rows, day, "")
	for _, expected := range []string{
		"📍 <b>Yuk Jonatildi — 2</b>",
		"<blockquote expandable>09:15  <b>Asadbek &amp; Ali</b> · +998 90 123 45 67\n11:20  <b>Samandar</b> · +998 93 345 67 89</blockquote>",
		"📍 <b>Yuklab Berildi — 1</b>",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("report %q does not contain %q", got, expected)
		}
	}
	if strings.Contains(got, "Kecha") {
		t.Fatalf("yesterday's lead appeared in today's report: %q", got)
	}
}

func TestTelegramDailyReportExcludesOtherPipelines(t *testing.T) {
	day := time.Date(2026, 9, 24, 21, 0, 0, 0, time.UTC)
	rows := []map[string]any{
		{"created_at": "2026-09-24T10:36:00Z", "pipeline": []any{"Gisht Voronkasi"}, "stage": []any{"Yangi Lid"}, "name": "Armada lead"},
		{"created_at": "2026-09-24T09:42:00Z", "pipeline": []any{"Udevs"}, "stage": []any{"Новая заявка"}, "name": "Other project lead"},
	}
	got := telegramFormatDealStatusesForDay(rows, day, "Gisht Voronkasi")
	if !strings.Contains(got, "Armada lead") || strings.Contains(got, "Other project lead") {
		t.Fatalf("report included the wrong pipeline: %q", got)
	}
}

func TestTelegramDealCreatedAtAcceptsCRMMinutePrecisionUTC(t *testing.T) {
	location := time.FixedZone("Asia/Tashkent", 5*60*60)
	for _, row := range []map[string]any{
		{"start_date": "2026-09-24T10:36"},
		{"created_at": "2026-09-24T10:36:42.596468"},
	} {
		createdAt, ok := telegramDealCreatedAt(row, location)
		if !ok || createdAt.Format("2006-01-02 15:04") != "2026-09-24 15:36" {
			t.Fatalf("CRM date %#v = %s, %v", row, createdAt, ok)
		}
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

func TestRenderTelegramTemplateOmitsMissingFieldLines(t *testing.T) {
	template := "🆕 <b>Yangi lid</b>\n\n👤 Ismi: {{lead.name}}\n📞 Telefon: {{lead.phone}}\n👨‍💼 Mas’ul: {{lead.owner_name}}\n\n🔗 {{lead.url}}"
	rendered := renderTelegramNotificationTemplateWithDeal(template, map[string]any{"name": "Dilnoza"})
	if rendered != "🆕 <b>Yangi lid</b>\n\n👤 Ismi: <b>Dilnoza</b>" {
		t.Fatalf("rendered missing fields = %q", rendered)
	}
}

func TestRenderTelegramTemplateBoldsAndEscapesDealName(t *testing.T) {
	rendered := renderTelegramNotificationTemplateWithDeal("Название: {{item.name}}", map[string]any{"name": "Ali & Vali"})
	if rendered != "Название: <b>Ali &amp; Vali</b>" {
		t.Fatalf("rendered name = %q", rendered)
	}
}

func TestRenderTelegramTemplateUsesCRMFieldAliases(t *testing.T) {
	template := "📞 {{lead.phone}}\n👨‍💼 {{lead.owner_name}}\n📦 {{deal.service}}\n💵 {{deal.amount}}"
	deal := map[string]any{
		"telefon":          "+998 90 123 45 67",
		"responsible-name": "Madina",
		"xizmat_nomi":      "IELTS kursi",
		"budget":           1200000,
	}
	rendered := renderTelegramNotificationTemplateWithDeal(template, deal)
	for _, value := range []string{"+998 90 123 45 67", "Madina", "IELTS kursi", "1200000"} {
		if !strings.Contains(rendered, value) {
			t.Fatalf("rendered aliases %q does not contain %q", rendered, value)
		}
	}
}

func TestRenderTelegramTemplateBuildsClickableCRMDealURL(t *testing.T) {
	rendered := renderTelegramNotificationTemplateWithDeal(
		"🔗 {{deal.url}}",
		map[string]any{"guid": "deal id", "pipeline": "UHRMS"},
	)
	want := `🔗 <a href="https://crm.ucode.co/deals?deal=deal+id&amp;pipeline=UHRMS">CRMda ochish</a>`
	if rendered != want {
		t.Fatalf("rendered CRM link = %q, want %q", rendered, want)
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

func TestTelegramStatusRuleMatchesPipelineSpecificFieldsWithoutPipelineValue(t *testing.T) {
	rule := models.TelegramStatusNotification{PipelineID: "UHRMS", StageID: "Выиграно"}
	deal := map[string]any{"pipeline_uhrms": "Выграно", "pipeline_udevs": "Переговоры"}
	if !telegramStatusRuleMatches(rule, deal) {
		t.Fatal("the configured pipeline-specific field should match without a shared pipeline value")
	}
}
