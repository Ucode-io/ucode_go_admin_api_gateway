package v1

import (
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestTelegramCreateDeliveriesPrefersAutomationForSharedGroup(t *testing.T) {
	deal := map[string]any{"name": "Test lead"}
	legacy := models.TelegramNotificationSettings{
		ChatID: "group", NewLeadEnabled: true,
		Templates: models.TelegramNotificationTemplates{NewLead: "Yangi lid: {{lead.name}}"},
	}
	modern := models.TelegramNotificationSettings{
		ChatID: "group", Automations: []models.TelegramAutomation{{
			Enabled: true, Action: "telegram_notification",
			TriggerConfigs: []models.TelegramAutomationTrigger{{Kind: "create", Table: "deals", Template: "Создание: {{item.name}}"}},
		}},
	}
	for _, settings := range [][]models.TelegramNotificationSettings{{legacy, modern}, {modern, legacy}} {
		got := telegramCreateDeliveries(settings, deal, time.Now())
		if len(got) != 1 || got[0].ChatID != "group" || got[0].Message != "Создание: Test lead" {
			t.Fatalf("deliveries = %#v, want one automation message", got)
		}
	}
	legacy.Automations = []models.TelegramAutomation{{
		ID: "legacy-new-lead", Enabled: true, Trigger: "new_lead", Action: "telegram_notification", Template: legacy.Templates.NewLead,
	}}
	got := telegramCreateDeliveries([]models.TelegramNotificationSettings{modern, legacy}, deal, time.Now())
	if len(got) != 1 || got[0].Message != "Создание: Test lead" {
		t.Fatalf("deliveries with persisted legacy rule = %#v, want one automation message", got)
	}
	modern.Automations[0].Enabled = false
	got = telegramCreateDeliveries([]models.TelegramNotificationSettings{legacy, modern}, deal, time.Now())
	if len(got) != 0 {
		t.Fatalf("deliveries with disabled automation = %#v, want no legacy fallback", got)
	}
}

func TestTelegramCreateDeliveriesKeepsOtherGroupsIndependent(t *testing.T) {
	settings := []models.TelegramNotificationSettings{
		{ChatID: "first", NewLeadEnabled: true, Templates: models.TelegramNotificationTemplates{NewLead: "First"}},
		{ChatID: "second", NewLeadEnabled: true, Templates: models.TelegramNotificationTemplates{NewLead: "Second"}},
	}
	got := telegramCreateDeliveries(settings, map[string]any{}, time.Now())
	if len(got) != 2 || got[0].Message != "First" || got[1].Message != "Second" {
		t.Fatalf("deliveries = %#v, want one message per group", got)
	}
}

func TestTelegramCreateDeliveriesRoutesRulesToTheirGroups(t *testing.T) {
	rule := func(chatID, message string) models.TelegramAutomation {
		return models.TelegramAutomation{Enabled: true, Action: "telegram_notification", ChatID: chatID,
			TriggerConfigs: []models.TelegramAutomationTrigger{{Kind: "create", Table: "deals", Template: message}}}
	}
	settings := models.TelegramNotificationSettings{ChatID: "latest", Automations: []models.TelegramAutomation{
		rule("sales", "sales message"), rule("owners", "owners message"),
	}}
	got := telegramCreateDeliveries([]models.TelegramNotificationSettings{settings}, map[string]any{}, time.Now())
	if len(got) != 2 || got[0].ChatID != "sales" || got[1].ChatID != "owners" {
		t.Fatalf("deliveries = %#v, want each automation routed to its group", got)
	}
}
