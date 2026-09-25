package v1

import (
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestTelegramReportsForChatsPrefersAutomation(t *testing.T) {
	now := time.Now()
	legacy := telegramScheduledReport{settings: models.TelegramNotificationSettings{ChatID: "group"}, template: "old", now: now}
	modern := telegramScheduledReport{settings: models.TelegramNotificationSettings{ChatID: "group"}, template: "new", now: now, modern: true}
	for _, candidates := range [][]telegramScheduledReport{{legacy, modern}, {modern, legacy}} {
		reports := telegramReportsForChats(candidates, map[string]bool{"group": true})
		if len(reports) != 1 || reports[0].template != "new" {
			t.Fatalf("reports = %#v, want only automation report", reports)
		}
	}
	if reports := telegramReportsForChats([]telegramScheduledReport{legacy}, map[string]bool{"group": true}); len(reports) != 0 {
		t.Fatalf("legacy report sent despite a configured automation: %#v", reports)
	}
}

func TestTelegramReportsForChatsOnePerGroup(t *testing.T) {
	candidates := []telegramScheduledReport{
		{settings: models.TelegramNotificationSettings{ChatID: "first"}, template: "one"},
		{settings: models.TelegramNotificationSettings{ChatID: "first"}, template: "two"},
		{settings: models.TelegramNotificationSettings{ChatID: "second"}, template: "three"},
	}
	reports := telegramReportsForChats(candidates, nil)
	if len(reports) != 2 || reports[0].template != "one" || reports[1].template != "three" {
		t.Fatalf("reports = %#v, want one per group", reports)
	}
}

func TestTelegramReportsForChatsKeepsDistinctRules(t *testing.T) {
	candidates := []telegramScheduledReport{
		{settings: models.TelegramNotificationSettings{ChatID: "group"}, template: "sales", modern: true, ruleID: "sales", triggerID: "one"},
		{settings: models.TelegramNotificationSettings{ChatID: "group"}, template: "calls", modern: true, ruleID: "calls", triggerID: "two"},
		{settings: models.TelegramNotificationSettings{ChatID: "group"}, template: "sales", modern: true, ruleID: "sales", triggerID: "one"},
	}
	reports := telegramReportsForChats(candidates, map[string]bool{"group": true})
	if len(reports) != 2 || reports[0].template != "sales" || reports[1].template != "calls" {
		t.Fatalf("reports = %#v, want separate sales and calls", reports)
	}
}
