package v1

import (
	"strings"
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

func crmAnalyticsTestSchema() []models.TableSchema {
	return []models.TableSchema{{
		Slug: "deals",
		Fields: []models.FieldSchema{
			{Slug: "created_at", Type: "timestamp without time zone"},
			{Slug: "deleted_at", Type: "timestamp without time zone"},
			{Slug: "stage", Type: "MULTISELECT"},
		},
	}}
}

func TestBuildCommonCRMAnalyticsPlanForYesterdayLeadCount(t *testing.T) {
	plan, ok := buildCommonCRMAnalyticsPlan(models.CRMAssistantRequest{
		Message: "kecha neshta lid keldi",
		PageContext: models.CRMAssistantPageContext{
			Now:      "2026-09-03T05:00:00Z",
			Timezone: "Asia/Tashkent",
		},
	}, crmAnalyticsTestSchema())
	if !ok {
		t.Fatal("expected common lead count analytics plan")
	}
	if plan.kind != crmAnalyticsLeadCount {
		t.Fatalf("got kind %q, want %q", plan.kind, crmAnalyticsLeadCount)
	}
	if !strings.Contains(plan.sql, `COUNT(*)`) || !strings.Contains(plan.sql, `"created_at"`) {
		t.Fatalf("unexpected SQL: %s", plan.sql)
	}
	if got, want := plan.params[0], "2026-09-02 00:00:00"; got != want {
		t.Fatalf("got start %v, want %v", got, want)
	}
	if got, want := plan.params[1], "2026-09-03 00:00:00"; got != want {
		t.Fatalf("got end %v, want %v", got, want)
	}
}

func TestBuildCommonCRMAnalyticsPlanCarriesYesterdayIntoStatusFollowUp(t *testing.T) {
	plan, ok := buildCommonCRMAnalyticsPlan(models.CRMAssistantRequest{
		Message: "kechigi kegan lidla hozir qaysi statusda",
		History: []models.CRMAssistantMessage{
			{Role: "user", Content: "kecha neshta lid keldi"},
			{Role: "assistant", Content: "Kecha 12 ta lid keldi."},
		},
		PageContext: models.CRMAssistantPageContext{
			Now:      "2026-09-03T05:00:00Z",
			Timezone: "Asia/Tashkent",
		},
	}, crmAnalyticsTestSchema())
	if !ok {
		t.Fatal("expected common lead status analytics plan")
	}
	if plan.kind != crmAnalyticsLeadStatus {
		t.Fatalf("got kind %q, want %q", plan.kind, crmAnalyticsLeadStatus)
	}
	for _, expected := range []string{`to_jsonb("stage")`, "GROUP BY 1", `"created_at"`} {
		if !strings.Contains(plan.sql, expected) {
			t.Fatalf("SQL %q does not contain %q", plan.sql, expected)
		}
	}
}

func TestBuildCommonCRMAnalyticsPlanSupportsLanguagesAndToday(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		language string
		start    string
	}{
		{name: "uzbek typo", message: "bugun neshta yangi lid keldi", language: "uz", start: "2026-09-03 00:00:00"},
		{name: "russian", message: "сколько лидов пришло вчера", language: "ru", start: "2026-09-02 00:00:00"},
		{name: "english", message: "how many leads arrived today", language: "en", start: "2026-09-03 00:00:00"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, ok := buildCommonCRMAnalyticsPlan(models.CRMAssistantRequest{
				Message: test.message,
				PageContext: models.CRMAssistantPageContext{
					Now:      "2026-09-03T05:00:00Z",
					Timezone: "Asia/Tashkent",
				},
			}, crmAnalyticsTestSchema())
			if !ok {
				t.Fatal("expected common analytics plan")
			}
			if plan.language != test.language {
				t.Fatalf("got language %q, want %q", plan.language, test.language)
			}
			if got := plan.params[0]; got != test.start {
				t.Fatalf("got start %v, want %v", got, test.start)
			}
		})
	}
}

func TestBuildCommonCRMAnalyticsPlanFallsBackForUnmatchedQuestions(t *testing.T) {
	tests := []models.CRMAssistantRequest{
		{
			Message: "eng katta budgetli deal qaysi",
			PageContext: models.CRMAssistantPageContext{
				Now:      "2026-09-03T05:00:00Z",
				Timezone: "Asia/Tashkent",
			},
		},
		{
			Message: "lidlar qaysi statusda",
			PageContext: models.CRMAssistantPageContext{
				Now:      "2026-09-03T05:00:00Z",
				Timezone: "Asia/Tashkent",
			},
		},
	}

	for _, request := range tests {
		if plan, ok := buildCommonCRMAnalyticsPlan(request, crmAnalyticsTestSchema()); ok {
			t.Fatalf("unexpected common analytics plan: %#v", plan)
		}
	}
}

func TestBuildCommonCRMAnalyticsPlanRequiresSchemaFields(t *testing.T) {
	request := models.CRMAssistantRequest{
		Message: "kecha nechta lid keldi",
		PageContext: models.CRMAssistantPageContext{
			Now:      "2026-09-03T05:00:00Z",
			Timezone: "Asia/Tashkent",
		},
	}
	if plan, ok := buildCommonCRMAnalyticsPlan(request, []models.TableSchema{{Slug: "deals"}}); ok {
		t.Fatalf("unexpected plan without created_at: %#v", plan)
	}
}

func TestFormatCommonCRMAnalyticsReplies(t *testing.T) {
	countReply, err := formatCommonCRMAnalyticsReply(crmAnalyticsPlan{
		kind:        crmAnalyticsLeadCount,
		periodLabel: "Kecha",
		language:    "uz",
	}, map[string]any{"rows": []map[string]any{{"count": float64(12)}}})
	if err != nil || countReply != "Kecha 12 ta lid keldi." {
		t.Fatalf("unexpected count reply %q, err=%v", countReply, err)
	}

	statusReply, err := formatCommonCRMAnalyticsReply(crmAnalyticsPlan{
		kind:        crmAnalyticsLeadStatus,
		periodLabel: "Kecha",
		language:    "uz",
	}, map[string]any{"rows": []map[string]any{
		{"status": "Неквалифицированный", "count": "4"},
		{"status": "Недозвон", "count": float64(3)},
		{"status": "Согласование договора", "count": int64(3)},
		{"status": "Новая заявка", "count": float64(2)},
	}})
	if err != nil {
		t.Fatalf("format status reply: %v", err)
	}
	for _, expected := range []string{"Kecha kelgan 12 ta lid", "Неквалифицированный — 4", "Новая заявка — 2"} {
		if !strings.Contains(statusReply, expected) {
			t.Fatalf("status reply %q does not contain %q", statusReply, expected)
		}
	}
}
