package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

type crmAnalyticsKind string

const (
	crmAnalyticsLeadCount  crmAnalyticsKind = "lead_count"
	crmAnalyticsLeadStatus crmAnalyticsKind = "lead_status"
)

type crmAnalyticsPlan struct {
	kind        crmAnalyticsKind
	sql         string
	params      []any
	periodLabel string
	language    string
}

func (p *ChatProcessor) runCommonCRMAnalytics(
	ctx context.Context,
	req models.CRMAssistantRequest,
	schema []models.TableSchema,
) (*crmAssistantResult, bool, error) {
	plan, ok := buildCommonCRMAnalyticsPlan(req, schema)
	if !ok {
		return nil, false, nil
	}

	data, err := p.executeSQLQuery(ctx, plan.sql, plan.params, p.resourceEnvId)
	if err != nil {
		return nil, true, fmt.Errorf("execute common CRM analytics: %w", err)
	}
	reply, err := formatCommonCRMAnalyticsReply(plan, data)
	if err != nil {
		return nil, true, fmt.Errorf("format common CRM analytics: %w", err)
	}
	return &crmAssistantResult{reply: reply}, true, nil
}

func buildCommonCRMAnalyticsPlan(
	req models.CRMAssistantRequest,
	schema []models.TableSchema,
) (crmAnalyticsPlan, bool) {
	current := strings.ToLower(strings.TrimSpace(req.Message))
	if !containsAnyFold(current, "lid", "lead", "лид") {
		return crmAnalyticsPlan{}, false
	}

	deals, ok := findCRMSchemaTable(schema, "deals")
	if !ok || !hasCRMSchemaField(deals, "created_at") {
		return crmAnalyticsPlan{}, false
	}

	start, end, periodLabel, language, ok := resolveCRMRelativeDay(req)
	if !ok {
		return crmAnalyticsPlan{}, false
	}

	kind := crmAnalyticsKind("")
	switch {
	case containsAnyFold(current, "status", "stage", "bosqich", "статус", "этап") && hasCRMSchemaField(deals, "stage"):
		kind = crmAnalyticsLeadStatus
	case containsAnyFold(current, "nechta", "neshta", "qancha", "how many", "count", "сколько", "soni") &&
		containsAnyFold(current, "kel", "kegan", "yangi", "came", "arriv", "new", "приш", "нов"):
		kind = crmAnalyticsLeadCount
	default:
		return crmAnalyticsPlan{}, false
	}

	where := `"created_at" >= $1::timestamp AND "created_at" < $2::timestamp`
	if hasCRMSchemaField(deals, "deleted_at") {
		where += ` AND "deleted_at" IS NULL`
	}
	params := []any{
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	}

	if kind == crmAnalyticsLeadCount {
		return crmAnalyticsPlan{
			kind:        kind,
			sql:         `SELECT COUNT(*) AS count FROM "deals" WHERE ` + where,
			params:      params,
			periodLabel: periodLabel,
			language:    language,
		}, true
	}

	// stage is commonly a Postgres text[] but can be scalar in older projects.
	// Converting to jsonb lets one read expression support both physical shapes.
	stageExpression := `COALESCE(NULLIF(CASE WHEN jsonb_typeof(to_jsonb("stage")) = 'array' THEN to_jsonb("stage")->>0 ELSE TRIM(BOTH '"' FROM to_jsonb("stage")::text) END, ''), 'Statussiz')`
	return crmAnalyticsPlan{
		kind: crmAnalyticsLeadStatus,
		sql: `SELECT ` + stageExpression + ` AS status, COUNT(*) AS count
FROM "deals"
WHERE ` + where + `
GROUP BY 1
ORDER BY 2 DESC`,
		params:      params,
		periodLabel: periodLabel,
		language:    language,
	}, true
}

func resolveCRMRelativeDay(req models.CRMAssistantRequest) (time.Time, time.Time, string, string, bool) {
	texts := []string{strings.ToLower(req.Message)}
	for index := len(req.History) - 1; index >= 0; index-- {
		if req.History[index].Role == "user" {
			texts = append(texts, strings.ToLower(req.History[index].Content))
		}
	}

	offset := 0
	period := ""
	language := detectCRMRequestLanguage(strings.ToLower(req.Message))
	for _, text := range texts {
		switch {
		case containsAnyFold(text, "kecha", "kechagi", "kechigi", "yesterday", "вчера"):
			offset = -1
			period = "yesterday"
		case containsAnyFold(text, "bugun", "today", "сегодня"):
			offset = 0
			period = "today"
		}
		if period != "" {
			break
		}
	}
	if period == "" {
		return time.Time{}, time.Time{}, "", "", false
	}

	now, err := time.Parse(time.RFC3339, strings.TrimSpace(req.PageContext.Now))
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false
	}
	location := now.Location()
	if requestedLocation, locationErr := time.LoadLocation(strings.TrimSpace(req.PageContext.Timezone)); locationErr == nil {
		location = requestedLocation
	}
	localNow := now.In(location)
	start := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location).AddDate(0, 0, offset)
	end := start.AddDate(0, 0, 1)

	labels := map[string]map[string]string{
		"uz": {"today": "Bugun", "yesterday": "Kecha"},
		"ru": {"today": "Сегодня", "yesterday": "Вчера"},
		"en": {"today": "Today", "yesterday": "Yesterday"},
	}
	return start, end, labels[language][period], language, true
}

func formatCommonCRMAnalyticsReply(plan crmAnalyticsPlan, data any) (string, error) {
	rows, err := crmAnalyticsRows(data)
	if err != nil {
		return "", err
	}

	if plan.kind == crmAnalyticsLeadCount {
		if len(rows) == 0 {
			return "", fmt.Errorf("count query returned no rows")
		}
		count, ok := crmAnalyticsInteger(rows[0]["count"])
		if !ok {
			return "", fmt.Errorf("count query returned an invalid count")
		}
		switch plan.language {
		case "ru":
			return fmt.Sprintf("%s пришло %d лидов.", plan.periodLabel, count), nil
		case "en":
			return fmt.Sprintf("%s, %d leads came in.", plan.periodLabel, count), nil
		default:
			return fmt.Sprintf("%s %d ta lid keldi.", plan.periodLabel, count), nil
		}
	}

	lines := make([]string, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		count, ok := crmAnalyticsInteger(row["count"])
		if !ok {
			return "", fmt.Errorf("status query returned an invalid count")
		}
		status := strings.TrimSpace(fmt.Sprintf("%v", row["status"]))
		if status == "" || status == "<nil>" {
			status = "Statussiz"
		}
		total += count
		lines = append(lines, fmt.Sprintf("- %s — %d", status, count))
	}

	header := ""
	switch plan.language {
	case "ru":
		header = fmt.Sprintf("%s пришло %d лидов. Сейчас они на этапах:", plan.periodLabel, total)
	case "en":
		header = fmt.Sprintf("%s, %d leads came in. Their current stages are:", plan.periodLabel, total)
	default:
		header = fmt.Sprintf("%s kelgan %d ta lid hozir quyidagi statuslarda:", plan.periodLabel, total)
	}
	if len(lines) == 0 {
		return header + "\n- Ma’lumot yo‘q", nil
	}
	return header + "\n" + strings.Join(lines, "\n"), nil
}

func crmAnalyticsRows(data any) ([]map[string]any, error) {
	result, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected analytics result type")
	}
	rows, ok := result["rows"].([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("analytics result has no rows")
	}
	return rows, nil
}

func crmAnalyticsInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float32:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func findCRMSchemaTable(schema []models.TableSchema, slug string) (models.TableSchema, bool) {
	for _, table := range schema {
		if table.Slug == slug {
			return table, true
		}
	}
	return models.TableSchema{}, false
}

func hasCRMSchemaField(table models.TableSchema, slug string) bool {
	for _, field := range table.Fields {
		if field.Slug == slug {
			return true
		}
	}
	return false
}

func containsAnyFold(value string, candidates ...string) bool {
	value = strings.ToLower(value)
	for _, candidate := range candidates {
		if strings.Contains(value, strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

func detectCRMRequestLanguage(value string) string {
	if containsAnyFold(value, "вчера", "сегодня", "лид", "статус", "этап", "сколько", "приш") {
		return "ru"
	}
	if containsAnyFold(value, "yesterday", "today", "lead", "status", "stage", "how many", "came", "arrived") {
		return "en"
	}
	return "uz"
}
