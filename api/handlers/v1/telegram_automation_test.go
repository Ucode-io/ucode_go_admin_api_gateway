package v1

import (
	"strings"
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestTelegramAutomationAndOrConditions(t *testing.T) {
	rule := models.TelegramAutomation{
		Enabled: true, Trigger: "deal", DealEvent: "updated", Action: "telegram_notification", ConditionMode: "and",
		Conditions: []models.TelegramAutomationCondition{
			{Field: "stage", Operator: "eq", Value: "Выиграно"},
			{Field: "amount", Operator: "gt", Value: "1000000"},
		},
	}
	deal := map[string]any{"stage": "Выграно", "amount": 800000}
	if telegramAutomationMatches(rule, "updated", deal, time.Now()) {
		t.Fatal("AND should require both conditions")
	}
	rule.ConditionMode = "or"
	if !telegramAutomationMatches(rule, "updated", deal, time.Now()) {
		t.Fatal("OR should allow either condition")
	}
	if telegramAutomationMatches(rule, "created", deal, time.Now()) {
		t.Fatal("wrong event should not dispatch")
	}
}

func TestTelegramAutomationWeekday(t *testing.T) {
	rule := models.TelegramAutomation{
		Enabled: true, Trigger: "daily_report", Action: "telegram_notification", ConditionMode: "and",
		Conditions: []models.TelegramAutomationCondition{{Field: "weekday", Operator: "eq", Value: "Dushanba"}},
	}
	monday := time.Date(2026, 9, 21, 21, 0, 0, 0, time.UTC)
	if !telegramAutomationMatches(rule, "daily_report", nil, monday) {
		t.Fatal("Uzbek weekday should match")
	}
	if telegramAutomationMatches(rule, "daily_report", nil, monday.AddDate(0, 0, 1)) {
		t.Fatal("wrong weekday should not match")
	}
}

func TestTelegramAutomationStatusConditionNeedsStageUpdate(t *testing.T) {
	rule := models.TelegramAutomation{
		Enabled: true, Trigger: "deal", DealEvent: "updated", Action: "telegram_notification", ConditionMode: "and",
		Conditions: []models.TelegramAutomationCondition{{Field: "stage", Operator: "eq", Value: "Выиграно"}},
	}
	deal := map[string]any{"stage": "Выграно", "amount": 1200000}
	if telegramAutomationMatchesUpdated(rule, deal, time.Now(), false) {
		t.Fatal("unrelated edit should not resend a status notification")
	}
	if !telegramAutomationMatchesUpdated(rule, deal, time.Now(), true) {
		t.Fatal("stage update should send")
	}
	rule.ConditionMode = "or"
	rule.Conditions = append(rule.Conditions, models.TelegramAutomationCondition{Field: "amount", Operator: "gt", Value: "1000000"})
	if !telegramAutomationMatchesUpdated(rule, deal, time.Now(), false) {
		t.Fatal("OR should still match a non-stage condition")
	}
}

func TestTelegramLegacyAutomationsKeepExistingRules(t *testing.T) {
	settings := defaultTelegramNotificationSettings("company-1", "crm_bot")
	settings.StatusNotifications = []models.TelegramStatusNotification{{ID: "status-1", Name: "Yutildi", Enabled: true, PipelineID: "UHRMS", StageID: "Выиграно", Template: "Yutildi"}}
	rules := telegramLegacyAutomations(settings)
	if len(rules) != 3 || !rules[0].Enabled || !rules[1].Enabled || !rules[2].Enabled {
		t.Fatalf("legacy rules not preserved: %+v", rules)
	}
	deal := map[string]any{"pipeline": "UHRMS", "stage": "Выграно"}
	if !telegramAutomationMatches(rules[2], "updated", deal, time.Now()) {
		t.Fatal("legacy status rule should match after migration")
	}
}

func TestTelegramAutomationMultipleTriggersUseOneAction(t *testing.T) {
	rule := models.TelegramAutomation{
		ID: "multi-1", Name: "Lid yoki hisobot", Enabled: true,
		Trigger: "new_lead", Triggers: []string{"new_lead", "daily_report"},
		ReportTime: "21:00", ConditionMode: "and", Action: "telegram_notification",
	}
	if err := validateTelegramAutomations([]models.TelegramAutomation{rule}); err != nil {
		t.Fatalf("valid multi-trigger rule rejected: %v", err)
	}
	if !telegramAutomationMatches(rule, "created", nil, time.Now()) || !telegramAutomationMatches(rule, "daily_report", nil, time.Now()) {
		t.Fatal("each selected trigger should invoke the action")
	}
	if telegramAutomationMatches(rule, "updated", nil, time.Now()) {
		t.Fatal("unselected deal event should not invoke the action")
	}
	settings := defaultTelegramNotificationSettings("company-1", "crm_bot")
	if telegramAutomationTemplate(rule, settings, "daily_report") != settings.Templates.DailyReport {
		t.Fatal("daily report should use its own default template")
	}
}

func TestTelegramAutomationTriggerConfigsMatchTableFieldAndConditions(t *testing.T) {
	rule := models.TelegramAutomation{Enabled: true, Action: "telegram_notification", TriggerConfigs: []models.TelegramAutomationTrigger{{
		ID: "update-deal", Kind: "field", Table: "deals", WatchField: "stage", ConditionMode: "and",
		Conditions: []models.TelegramAutomationCondition{{Field: "amount", Operator: "gt", Value: "1000000"}},
	}}}
	deal := map[string]any{"stage": "Won", "amount": 1500000}
	if _, ok := telegramAutomationMatchingTrigger(rule, "contacts", "update", deal, map[string]any{"stage": "Won"}, time.Now()); ok {
		t.Fatal("a deal trigger must not run for contacts")
	}
	if _, ok := telegramAutomationMatchingTrigger(rule, "deals", "update", deal, map[string]any{"name": "New"}, time.Now()); ok {
		t.Fatal("a watched-field trigger must not run for unrelated changes")
	}
	if _, ok := telegramAutomationMatchingTrigger(rule, "deals", "update", deal, map[string]any{"stage": "Won"}, time.Now()); !ok {
		t.Fatal("matching table, watched field, and condition should run")
	}
}

func TestTelegramAutomationTriggerConfigTemplateUsesSelectedFields(t *testing.T) {
	trigger := models.TelegramAutomationTrigger{Kind: "create", Table: "deals", MessageFields: []string{"name", "amount"}}
	message := renderTelegramNotificationTemplateWithDeal(telegramAutomationTriggerTemplate(trigger), map[string]any{"name": "Azizbek", "amount": 1500000})
	if !strings.Contains(message, "Azizbek") || !strings.Contains(message, "1 500 000") {
		t.Fatalf("selected fields were not rendered: %q", message)
	}
}

func TestTelegramDealMoneyRendersWithoutExponent(t *testing.T) {
	message := renderTelegramNotificationTemplateWithDeal("Summa: {{item.summa}}", map[string]any{"summa": 1000000.0})
	if message != "Summa: 1 000 000" {
		t.Fatalf("unexpected money format: %q", message)
	}
}
