package v1

import (
	"strings"
	"testing"
)

func TestTelegramLeadCommentRendering(t *testing.T) {
	template := "🆕 Новый лид\n👤 {{item.name}}\n📞 {{item.phone}}\n📍 {{item.stage}}\n📝 Комментарий: {{item.notes}}"
	item := map[string]any{"name": "Actual name", "phone": "+998901234567", "stage": "Новая заявка", "notes": "Первая строка <b>не HTML</b> &\nВторая строка"}
	result := renderTelegramNotificationTemplateWithDeal(template, item)
	if !strings.Contains(result, "&lt;b&gt;не HTML&lt;/b&gt; &amp;\nВторая строка") {
		t.Fatal(result)
	}
	for _, notes := range []any{nil, "", "   "} {
		item["notes"] = notes
		result = renderTelegramNotificationTemplateWithDeal(template, item)
		if strings.Contains(result, "Комментарий") {
			t.Fatal(result)
		}
	}
	item["notes"] = strings.Repeat("😀<&\n", 10000)
	result = renderTelegramNotificationTemplateWithDeal(template, item)
	if telegramHTMLUnits(result) > 2000 || !strings.Contains(result, "полный комментарий — в CRM") {
		t.Fatal("unbounded comment", telegramHTMLUnits(result))
	}
	for _, field := range []string{"Actual name", "+998901234567", "Новая заявка"} {
		if !strings.Contains(result, field) {
			t.Fatal("lost existing field", field)
		}
	}
	if len(item["notes"].(string)) < 10000 {
		t.Fatal("changed source notes")
	}
}
