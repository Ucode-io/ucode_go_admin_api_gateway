package v1

import (
	"strings"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func TestRenderTelegramAutomationMessageUsesChoiceLabel(t *testing.T) {
	trigger := models.TelegramAutomationTrigger{
		Template: "🚚 Xizmat turi: {{item.hizmat_turi}}\n🧱 Soni: {{item.gisht_soni}}",
		FieldOptions: map[string][]models.TelegramFieldOption{
			"hizmat_turi": {{Value: "choice-1", Slug: "ozi_olib_ketadi", Label: "O‘zi olib ketadi"}},
		},
	}
	message := renderTelegramAutomationMessage(trigger, map[string]any{"hizmat_turi": "ozi_olib_ketadi", "gisht_soni": 123})
	if !strings.Contains(message, "🚚 Xizmat turi: O‘zi olib ketadi") || !strings.Contains(message, "🧱 Soni: 123") {
		t.Fatalf("message should contain the display label and numeric value: %q", message)
	}
	if strings.Contains(message, "ozi_olib_ketadi") {
		t.Fatalf("message leaked a choice slug: %q", message)
	}
}

func TestRenderTelegramAutomationCompletedMessageUsesNewStatus(t *testing.T) {
	trigger := models.TelegramAutomationTrigger{
		Template:    "📍 Статус: {{item.pipeline_enterprise_sales}}",
		StatusField: "pipeline_enterprise_sales",
		FieldOptions: map[string][]models.TelegramFieldOption{
			"pipeline_enterprise_sales": {{Value: "stage-yuk", Label: "Yuk Jonatildi"}},
		},
	}
	message := renderTelegramAutomationCompletedMessage(trigger, map[string]any{"pipeline_enterprise_sales": "Narx kelishildi"}, "stage-yuk", "🚚 Yuk jo‘natildi")
	if !strings.Contains(message, "📍 Статус: Yuk Jonatildi") || !strings.Contains(message, "✅ <b>Текущий статус:</b> 🚚 Yuk jo‘natildi") {
		t.Fatalf("edited message should show the updated deal status: %q", message)
	}
	if strings.Contains(message, "Narx kelishildi") {
		t.Fatalf("edited message retained the previous status: %q", message)
	}
}
