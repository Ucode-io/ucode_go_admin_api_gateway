package v1

import (
	"context"
	"encoding/json"
	"fmt"
	redis "github.com/go-redis/redis/v8"
	"html"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"ucode/ucode_go_api_gateway/api/models"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
)

const telegramScopedReportTTL = 36 * time.Hour
const telegramScopedReportMessageLimit = 3500

// New reports retry within five minutes; legacy schedules keep their exact-minute behavior.
func telegramReportDue(trigger models.TelegramAutomationTrigger, now time.Time) bool {
	if trigger.ReportType != "crm" && trigger.ReportType != "marketing" {
		return trigger.ReportTime == "" || now.Format("15:04") == trigger.ReportTime
	}
	scheduled, err := time.Parse("15:04", trigger.ReportTime)
	if err != nil {
		return false
	}
	minutes := now.Hour()*60 + now.Minute() - scheduled.Hour()*60 - scheduled.Minute()
	return minutes >= 0 && minutes < 5
}

func validateTelegramMarketingReport(report models.MetaAdsDashboardResponse, accountID string, day time.Time) error {
	if report.Stale {
		return fmt.Errorf("Meta report is stale; current metrics unavailable")
	}
	if strings.TrimPrefix(report.Account.ID, "act_") != strings.TrimPrefix(accountID, "act_") {
		return fmt.Errorf("Meta report account does not match selected account")
	}
	generated, err := time.Parse(time.RFC3339, report.GeneratedAt)
	if err != nil || generated.In(telegramReportLocation()).Format("2006-01-02") != day.In(telegramReportLocation()).Format("2006-01-02") {
		return fmt.Errorf("Meta report freshness could not be verified")
	}
	return nil
}

func validTelegramReportStatusField(field string) bool {
	if !strings.HasPrefix(field, "pipeline_") {
		return false
	}
	for _, r := range field {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return len(field) > len("pipeline_") && len(field) <= 100
}

func (h *HandlerV1) scopedTelegramReportMessages(ctx context.Context, report telegramScheduledReport) ([]string, error) {
	if report.reportType == "marketing" {
		const template = "📣 <b>Маркетинговый отчёт · {{report.date}}</b>\n🕘 На {{report.as_of}} · Ташкент\n\n👥 Лиды по данным Meta: {{report.leads_total}}\n💸 Расходы: {{report.ad_spend}}\n💰 CPL Meta (расходы / лиды Meta): {{report.cpl}}"
		message, err := h.telegramDailyReportMessageWithTemplate(ctx, report.target, report.settings, report.now, template, "")
		if err != nil {
			h.log.Warn("telegram marketing report data unavailable", logger.Error(err))
			message = "📣 <b>Маркетинговый отчёт · " + report.now.In(telegramReportLocation()).Format("02.01.2006") + "</b>\n🕘 На " + report.now.In(telegramReportLocation()).Format("15:04") + " · Ташкент\n\n⚠️ Данные Meta за сегодня недоступны. Лиды, расходы и CPL не подтверждены."
		}
		return []string{message}, nil
	}
	if report.reportType != "crm" || report.pipeline == "" || report.statusField != "" && !validTelegramReportStatusField(report.statusField) {
		return nil, fmt.Errorf("invalid scoped CRM report")
	}
	service, environmentID, err := h.resolveProjectBuilder(ctx, report.target.ProjectID, report.target.EnvironmentID)
	if err != nil {
		return nil, err
	}
	pipelineResponse, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{TableSlug: "project_pipeline", Data: mustStruct(map[string]any{"guid": report.pipeline, "limit": 2, "offset": 0}), ProjectId: environmentID, CompanyProjectId: report.target.ProjectID})
	if err != nil {
		return nil, err
	}
	name := ""
	for _, row := range telegramResponseRows(pipelineResponse.GetData()) {
		if telegramDealValue(row, "guid") == report.pipeline {
			name = telegramDealValue(row, "label", "name")
		}
	}
	if name == "" {
		return nil, fmt.Errorf("report pipeline not found in current project")
	}
	fields, err := service.GoObjectBuilderService().Field().GetAll(ctx, &nb.GetAllFieldsRequest{TableSlug: "deals", ProjectId: environmentID, Limit: 500})
	if err != nil {
		return nil, err
	}
	matchingFields := []string{}
	for _, field := range fields.GetFields() {
		if field.GetAttributes() != nil && telegramValueString(field.GetAttributes().AsMap()["pipeline_id"]) == report.pipeline && validTelegramReportStatusField(field.GetSlug()) && (report.statusField == "" || report.statusField == field.GetSlug()) {
			matchingFields = append(matchingFields, field.GetSlug())
		}
	}
	if len(matchingFields) != 1 {
		return nil, fmt.Errorf("report requires one verified status field for selected pipeline")
	}
	report.statusField = matchingFields[0]
	report.now = report.now.In(telegramReportLocation())
	start := time.Date(report.now.Year(), report.now.Month(), report.now.Day(), 0, 0, 0, 0, telegramReportLocation())
	end := start.AddDate(0, 0, 1)
	const pageSize = 500
	rows := []map[string]any{}
	seen := map[string]bool{}
	for offset := 0; ; offset += pageSize {
		response, queryErr := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{TableSlug: "deals", Data: mustStruct(map[string]any{"created_at": map[string]any{"$gte": start.UTC().Format(time.RFC3339), "$lt": end.UTC().Format(time.RFC3339)}, "pipeline": []any{name}, "limit": pageSize, "offset": offset}), ProjectId: environmentID, CompanyProjectId: report.target.ProjectID})
		if queryErr != nil {
			return nil, queryErr
		}
		page := telegramResponseRows(response.GetData())
		for _, row := range page {
			created, ok := telegramDealCreatedAt(row, start.Location())
			id := telegramDealValue(row, "guid", "id")
			// Fail rather than scan historical records or publish mixed/incomplete totals if filtering/pagination is ignored.
			if !ok || created.Before(start) || !created.Before(end) || !telegramDealPipelineMatches(name, row) || id == "" || seen[id] {
				return nil, fmt.Errorf("CRM report query did not honor day, funnel, or pagination scope")
			}
			seen[id] = true
			rows = append(rows, row)
		}
		if len(page) < pageSize {
			break
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	messages := telegramCRMReportMessages(rows, report.now, name, report.statusField)
	for _, message := range messages {
		if telegramHTMLUnits(message) > telegramScopedReportMessageLimit {
			return nil, fmt.Errorf("CRM report heading exceeds safe message length")
		}
	}
	return messages, nil
}

func telegramHTMLUnits(value string) int { return len(utf16.Encode([]rune(value))) }

// Split plain text before escaping so entities and HTML tags never cross message boundaries.
func telegramReportNameParts(value string) []string {
	parts := []string{}
	current := ""
	for _, r := range value {
		next := current + string(r)
		if telegramHTMLUnits(html.EscapeString(next)) > 1000 && current != "" {
			parts = append(parts, current)
			current = string(r)
		} else {
			current = next
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func telegramCRMReportMessages(rows []map[string]any, day time.Time, pipelineName, statusField string) []string {
	groups := map[string][]telegramDailyDeal{}
	for _, row := range rows {
		created, ok := telegramDealCreatedAt(row, telegramReportLocation())
		if !ok {
			continue
		}
		status := telegramDealValue(row, "stage", "stage_id", "status", statusField)
		if status == "" {
			status = "Без статуса"
		}
		name := telegramDealValue(row, "name", "full_name", "contact_name")
		if name == "" {
			name = "Без имени"
		}
		groups[status] = append(groups[status], telegramDailyDeal{Name: name, CreatedAt: created})
	}
	header := "📋 <b>" + html.EscapeString(pipelineName) + " · " + day.Format("02.01.2006") + "</b>\n🕘 На " + day.In(telegramReportLocation()).Format("15:04") + " · Ташкент\n👥 Создано сегодня в CRM: " + fmt.Sprint(len(rows)) + "\n📍 Текущие статусы на момент отчёта\n\n"
	keys := []string{}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	messages := []string{}
	current := header
	for _, status := range keys {
		deals := groups[status]
		sort.SliceStable(deals, func(i, j int) bool { return deals[i].CreatedAt.Before(deals[j].CreatedAt) })
		heading := "📍 <b>" + html.EscapeString(status) + " — " + fmt.Sprint(len(deals)) + "</b>\n"
		lines := ""
		flush := func() {
			if lines != "" {
				current += heading + "<blockquote expandable>" + lines + "</blockquote>\n\n"
				lines = ""
			}
		}
		for _, deal := range deals {
			for _, part := range telegramReportNameParts(deal.Name) {
				line := deal.CreatedAt.Format("15:04") + "  <b>" + html.EscapeString(part) + "</b>\n"
				if telegramHTMLUnits(current+heading+"<blockquote expandable>"+lines+line+"</blockquote>\n\n") > telegramScopedReportMessageLimit {
					flush()
					if current != header {
						messages = append(messages, current)
						current = header
					}
				}
				lines += line
			}
		}
		flush()
	}
	if len(keys) == 0 {
		current += "За сегодня лидов нет."
	}
	return append(messages, current)
}

func telegramScopedReportKey(report telegramScheduledReport) string {
	return telegramNotificationsDailyLockPrefix + "scoped:" + report.target.ProjectID + ":" + report.target.EnvironmentID + ":" + report.target.CompanyID + ":" + report.settings.ChatID + ":" + report.now.Format("2006-01-02") + ":" + report.ruleID + ":" + report.triggerID
}

func (h *HandlerV1) sendScopedTelegramReport(report telegramScheduledReport) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	key := telegramScopedReportKey(report)
	lock, err := h.centralRedis.SetNX(ctx, key+":building", "1", 2*time.Minute).Result()
	if err != nil || !lock {
		return
	}
	defer h.centralRedis.Del(context.Background(), key+":building")
	snapshot, err := h.centralRedis.Get(ctx, key+":snapshot").Result()
	var messages []string
	if err == nil {
		err = json.Unmarshal([]byte(snapshot), &messages)
	} else if err == redis.Nil {
		messages, err = h.scopedTelegramReportMessages(ctx, report)
		if err == nil {
			var encoded []byte
			encoded, err = json.Marshal(messages)
			if err == nil {
				err = h.centralRedis.Set(ctx, key+":snapshot", string(encoded), telegramScopedReportTTL).Err()
			}
		}
	}
	if err != nil {
		h.log.Error("telegram scoped report build failed", logger.Error(err))
		return
	}
	delivery := telegramRedisReportDelivery{client: h.centralRedis, key: key, chatID: report.settings.ChatID, bot: newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken)}
	if err := deliverTelegramReportParts(ctx, messages, &delivery); err != nil {
		h.log.Error("telegram scoped report send failed", logger.Error(err))
	}
}

type telegramRedisReportDelivery struct {
	client *redis.Client
	key    string
	chatID string
	bot    *telegramAPIClient
}

func (d *telegramRedisReportDelivery) Claim(ctx context.Context, index int) (bool, error) {
	return d.client.SetNX(ctx, fmt.Sprintf("%s:part:%d", d.key, index), "sending", telegramScopedReportTTL).Result()
}
func (d *telegramRedisReportDelivery) Send(ctx context.Context, message string) error {
	_, err := d.bot.sendHTMLMessage(ctx, d.chatID, message)
	return err
}
func (d *telegramRedisReportDelivery) Release(ctx context.Context, index int) error {
	return d.client.Del(ctx, fmt.Sprintf("%s:part:%d", d.key, index)).Err()
}
func (d *telegramRedisReportDelivery) MarkSent(ctx context.Context, index int) error {
	return d.client.Set(ctx, fmt.Sprintf("%s:part:%d", d.key, index), "sent", telegramScopedReportTTL).Err()
}
