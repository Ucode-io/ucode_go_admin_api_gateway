package v1

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
)

func TestTelegramScopedCRMReports(t *testing.T) {
	day := time.Date(2026, 9, 30, 21, 0, 0, 0, telegramReportLocation())
	rows := []map[string]any{{"created_at": "2026-09-29T19:00:00Z", "name": "A<&", "pipeline_sales_project": "Связались", "pipeline_ufin": "WRONG"}, {"created_at": "2026-09-30T10:00:00Z", "name": "B", "pipeline_sales_project": "Новый"}}
	messages := telegramCRMReportMessages(rows, day, "Udevs", "pipeline_sales_project")
	if len(messages) != 1 {
		t.Fatalf("messages=%d", len(messages))
	}
	for _, value := range []string{"Создано сегодня в CRM: 2", "Связались — 1", "00:00  <b>A&lt;&amp;</b>", "15:00  <b>B</b>"} {
		if !strings.Contains(messages[0], value) {
			t.Fatalf("missing %q: %s", value, messages[0])
		}
	}
	if strings.Contains(messages[0], "WRONG") || strings.Contains(messages[0], "CPL") {
		t.Fatal("mixed funnel or marketing")
	}
	if !strings.Contains(telegramCRMReportMessages(nil, day, "Ufinance", "pipeline_ufin")[0], "За сегодня лидов нет") {
		t.Fatal("missing empty report")
	}
}

func TestTelegramScopedLongReportPreservesAllNames(t *testing.T) {
	day := time.Date(2026, 9, 30, 21, 0, 0, 0, telegramReportLocation())
	rows := []map[string]any{}
	for i := 0; i < 200; i++ {
		rows = append(rows, map[string]any{"created_at": "2026-09-30T10:00:00Z", "name": "Unique lead & <escaped>", "pipeline_sales_project": "Новый"})
	}
	messages := telegramCRMReportMessages(rows, day, "Udevs", "pipeline_sales_project")
	if len(messages) <= 1 {
		t.Fatal("long report not split")
	}
	all := strings.Join(messages, "")
	if strings.Count(all, "Unique lead &amp; &lt;escaped&gt;") != 200 {
		t.Fatal("lead truncated or duplicated")
	}
	for _, message := range messages {
		if telegramHTMLUnits(message) > telegramScopedReportMessageLimit || strings.Count(message, "<blockquote expandable>") != strings.Count(message, "</blockquote>") {
			t.Fatal("unsafe message split")
		}
	}
	long := strings.Repeat("<&😀", 2000)
	if strings.Join(telegramReportNameParts(long), "") != long {
		t.Fatal("long name lost data")
	}
}

func TestTelegramScopedKeysKeepThreeMessagesAndTenantsSeparate(t *testing.T) {
	day := time.Date(2026, 9, 30, 21, 0, 0, 0, telegramReportLocation())
	candidates := []telegramScheduledReport{}
	for _, id := range []string{"marketing", "udevs", "ufinance"} {
		candidates = append(candidates, telegramScheduledReport{target: telegramNotificationTarget{ProjectID: "p", EnvironmentID: "e"}, settings: models.TelegramNotificationSettings{ChatID: "group"}, now: day, modern: true, ruleID: "existing", triggerID: id})
	}
	reports := telegramReportsForChats(candidates, map[string]bool{"group": true})
	if len(reports) != 3 {
		t.Fatal("reports collapsed")
	}
	keys := map[string]bool{}
	for _, report := range reports {
		key := telegramScopedReportKey(report)
		if keys[key] {
			t.Fatal("delivery collision")
		}
		keys[key] = true
	}
	other := reports[0]
	other.target.EnvironmentID = "other"
	if keys[telegramScopedReportKey(other)] {
		t.Fatal("cross-tenant delivery key")
	}
	other = reports[0]
	other.target.CompanyID = "other-company"
	if keys[telegramScopedReportKey(other)] {
		t.Fatal("cross-company delivery key")
	}
}

type fakeTelegramPartDelivery struct {
	sent     map[int]bool
	current  int
	failOnce bool
	calls    []string
}

func (d *fakeTelegramPartDelivery) Claim(ctx context.Context, index int) (bool, error) {
	d.current = index
	claimed := !d.sent[index]
	d.sent[index] = true
	return claimed, nil
}
func (d *fakeTelegramPartDelivery) Send(ctx context.Context, message string) error {
	if d.current == 1 && d.failOnce {
		d.failOnce = false
		return &models.TelegramAPIRejectedError{Description: "definite delivery failure"}
	}
	d.calls = append(d.calls, message)
	return nil
}
func (d *fakeTelegramPartDelivery) Release(ctx context.Context, index int) error {
	delete(d.sent, index)
	return nil
}
func (d *fakeTelegramPartDelivery) MarkSent(ctx context.Context, index int) error {
	d.sent[index] = true
	return nil
}
func TestTelegramScopedRetrySkipsSuccessfulParts(t *testing.T) {
	d := &fakeTelegramPartDelivery{sent: map[int]bool{}, failOnce: true}
	parts := []string{"first", "second", "third"}
	if deliverTelegramReportParts(context.Background(), parts, d) == nil {
		t.Fatal("expected failure")
	}
	if err := deliverTelegramReportParts(context.Background(), parts, d); err != nil {
		t.Fatal(err)
	}
	if strings.Join(d.calls, ",") != "first,second,third" {
		t.Fatalf("duplicate sends: %v", d.calls)
	}
}

func TestTelegramScopedReportRetryWindow(t *testing.T) {
	trigger := models.TelegramAutomationTrigger{ReportType: "crm", ReportTime: "21:00"}
	for minute, want := range map[int]bool{59: false, 0: true, 1: true, 4: true, 5: false} {
		hour := 21
		if minute == 59 {
			hour = 20
		}
		now := time.Date(2026, 9, 30, hour, minute, 0, 0, telegramReportLocation())
		if telegramReportDue(trigger, now) != want {
			t.Fatalf("hour=%d minute=%d", hour, minute)
		}
	}
	trigger.ReportType = ""
	if telegramReportDue(trigger, time.Date(2026, 9, 30, 21, 1, 0, 0, telegramReportLocation())) {
		t.Fatal("legacy schedule changed")
	}
}

func TestTelegramMarketingFreshnessAndAccount(t *testing.T) {
	day := time.Date(2026, 9, 30, 21, 0, 0, 0, telegramReportLocation())
	report := models.MetaAdsDashboardResponse{GeneratedAt: "2026-09-30T15:00:00Z", Account: models.MetaAdsAccount{ID: "843587364827107"}}
	if err := validateTelegramMarketingReport(report, "act_843587364827107", day); err != nil {
		t.Fatal(err)
	}
	report.Stale = true
	if validateTelegramMarketingReport(report, "act_843587364827107", day) == nil {
		t.Fatal("stale report accepted")
	}
	report.Stale = false
	if validateTelegramMarketingReport(report, "other", day) == nil {
		t.Fatal("wrong account accepted")
	}
	report.GeneratedAt = "2026-09-29T10:00:00Z"
	if validateTelegramMarketingReport(report, "843587364827107", day) == nil {
		t.Fatal("old report accepted")
	}
}

type fakeUncertainTelegramDelivery struct {
	fakeTelegramPartDelivery
	attempts int
}

func (d *fakeUncertainTelegramDelivery) Send(ctx context.Context, message string) error {
	d.attempts++
	return errors.New("transport timeout, delivery unknown")
}
func TestTelegramScopedUncertainDeliveryNotResent(t *testing.T) {
	d := &fakeUncertainTelegramDelivery{fakeTelegramPartDelivery: fakeTelegramPartDelivery{sent: map[int]bool{}}}
	if deliverTelegramReportParts(context.Background(), []string{"one"}, d) == nil {
		t.Fatal("expected uncertain error")
	}
	if err := deliverTelegramReportParts(context.Background(), []string{"one"}, d); err != nil {
		t.Fatal(err)
	}
	if d.attempts != 1 {
		t.Fatal("uncertain message resent")
	}
}

func TestTelegramScopedCanonicalStageFallback(t *testing.T) {
	day := time.Date(2026, 9, 30, 21, 0, 0, 0, telegramReportLocation())
	rows := []map[string]any{{"created_at": "2026-09-30T11:41:48Z", "name": "Lead", "stage": []any{"Новая заявка"}, "pipeline_sales_project": nil, "pipeline_ufin": "WRONG"}}
	message := telegramCRMReportMessages(rows, day, "Udevs", "pipeline_sales_project")[0]
	if !strings.Contains(message, "Новая заявка — 1") || strings.Contains(message, "Без статуса") || strings.Contains(message, "WRONG") {
		t.Fatal("canonical current stage lost")
	}
}
