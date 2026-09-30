package v1

import (
	"strings"
	"testing"
	"time"
)

func TestTelegramContactPhoneRelations(t *testing.T) {
	for _, test := range []struct {
		name string
		item map[string]any
		want string
	}{
		{"direct wins", map[string]any{"phone": "+998901111111", "contacts_id_data": map[string]any{"phone": "+998902222222"}}, "+998901111111"},
		{"linked mobile", map[string]any{"contacts_id_data": map[string]any{"mobile": "+998903333333"}}, "+998903333333"},
		{"linked phones", map[string]any{"contact": map[string]any{"phones": []any{"", "+998904444444"}}}, "+998904444444"},
		{"missing", map[string]any{"contacts_id": "d34b4e48-4213-4569-8841-276a8d16c969"}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := telegramContactPhone(test.item); got != test.want {
				t.Fatalf("got %q want %q", got, test.want)
			}
		})
	}
	if got := telegramContactID(map[string]any{"contacts_id": []any{map[string]any{"guid": "d34b4e48-4213-4569-8841-276a8d16c969"}}}); got != "d34b4e48-4213-4569-8841-276a8d16c969" {
		t.Fatal(got)
	}
	if got := telegramContactID(map[string]any{"contacts_id": "Aziz"}); got != "" {
		t.Fatal("must not query contacts by name")
	}
	message := renderTelegramNotificationTemplateWithDeal("📞 Телефон: {{item.phone}}", map[string]any{"phone": "+998901111111"})
	if !strings.Contains(message, "+998901111111") {
		t.Fatal(message)
	}
	missing := renderTelegramNotificationTemplateWithDeal("📞 Телефон: {{item.phone}}", map[string]any{})
	if strings.Contains(missing, "+998") {
		t.Fatal("fabricated phone")
	}
}

func TestTelegramSimpleReportHeadingKeepsLeadTime(t *testing.T) {
	day := time.Date(2026, 9, 30, 21, 0, 0, 0, telegramReportLocation())
	messages := telegramCRMReportMessages([]map[string]any{{"name": "Real lead", "stage": "Новая заявка", "created_at": "2026-09-30T08:00:00Z"}}, day, "Udevs", "stage")
	if !strings.HasPrefix(messages[0], "📋 <b>Udevs · 30.09.2026</b>\n") {
		t.Fatal(messages)
	}
	for _, forbidden := range []string{"TEST", "Ташкент", "🕘", "не завершён"} {
		if strings.Contains(messages[0], forbidden) {
			t.Fatal(messages)
		}
	}
	if !strings.Contains(messages[0], "13:00") {
		t.Fatal("lead time lost", messages)
	}
}
