package v1

import (
	"context"
	"strings"
	"time"

	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
)

type telegramDailyCallMetrics struct {
	Total   int
	Leads   int
	Seconds int
}

// PBX writes an initial row and a completion row for the same call UUID.
// Count the most complete row once, just as the CRM dashboard does.
func telegramPhoneKey(phone string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if len(digits) >= 9 {
		return digits[len(digits)-9:]
	}
	return digits
}

func telegramCallMetricsForDay(rows []map[string]any, day time.Time, leadPhones map[string]bool) telegramDailyCallMetrics {
	byID := map[string]map[string]any{}
	for _, row := range rows {
		started := telegramDealValue(row, "started_at", "created_at")
		at, err := time.Parse(time.RFC3339Nano, started)
		if err != nil || at.In(day.Location()).Format("2006-01-02") != day.Format("2006-01-02") {
			continue
		}
		id := telegramDealValue(row, "call_uuid", "idempotency_key", "guid")
		if id == "" {
			continue
		}
		previous, ok := byID[id]
		if !ok || telegramNumber(telegramDealValue(row, "duration")) > telegramNumber(telegramDealValue(previous, "duration")) || telegramDealValue(previous, "status") == "initiated" && telegramDealValue(row, "status") != "initiated" {
			byID[id] = row
		}
	}
	metrics := telegramDailyCallMetrics{Total: len(byID)}
	leads := map[string]bool{}
	for _, row := range byID {
		if telegramDealValue(row, "status") != "answered" {
			continue
		}
		metrics.Seconds += int(telegramNumber(telegramDealValue(row, "duration")))
		phone := telegramPhoneKey(telegramDealValue(row, "client_phone"))
		if leadPhones[phone] {
			leads[phone] = true
		}
	}
	metrics.Leads = len(leads)
	return metrics
}

func (h *HandlerV1) telegramDailyCallsForDay(ctx context.Context, target telegramNotificationTarget, day time.Time) (telegramDailyCallMetrics, error) {
	service, environmentID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return telegramDailyCallMetrics{}, err
	}
	const pageSize = 500
	rows := make([]map[string]any, 0, pageSize)
	for offset := 0; offset < 10000; offset += pageSize {
		response, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
			TableSlug: "pbx_calls", Data: mustStruct(map[string]any{"limit": pageSize, "offset": offset}),
			ProjectId: environmentID, CompanyProjectId: target.ProjectID,
		})
		if err != nil {
			return telegramDailyCallMetrics{}, err
		}
		page := telegramResponseRows(response.GetData())
		rows = append(rows, page...)
		if len(page) < pageSize {
			break
		}
	}
	leadPhones := map[string]bool{}
	for offset := 0; offset < 10000; offset += pageSize {
		response, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
			TableSlug: "deals", Data: mustStruct(map[string]any{"limit": pageSize, "offset": offset}),
			ProjectId: environmentID, CompanyProjectId: target.ProjectID,
		})
		if err != nil {
			return telegramDailyCallMetrics{}, err
		}
		page := telegramResponseRows(response.GetData())
		for _, deal := range page {
			phone := telegramPhoneKey(telegramDealValue(deal, "phone", "phone_number", "contact_phone", "telephone", "telefon", "mobile", "mobile_phone"))
			if phone != "" {
				leadPhones[phone] = true
			}
		}
		if len(page) < pageSize {
			break
		}
	}
	return telegramCallMetricsForDay(rows, day, leadPhones), nil
}
