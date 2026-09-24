package v1

import (
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestNormalizeTelegramNotificationGroupsMigratesExistingSettings(t *testing.T) {
	settings := models.TelegramNotificationSettings{
		ChatID: "crm", ChatTitle: "CRM bot",
		Automations: []models.TelegramAutomation{{ID: "rule"}},
	}
	normalizeTelegramNotificationGroups(&settings)
	if len(settings.Groups) != 1 || settings.Groups[0].ChatID != "crm" || settings.Automations[0].ChatID != "crm" {
		t.Fatalf("settings = %#v, want legacy chat migrated to groups and rule", settings)
	}
}
