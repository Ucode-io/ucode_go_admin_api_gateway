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
