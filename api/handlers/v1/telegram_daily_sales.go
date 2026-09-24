package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"ucode/ucode_go_api_gateway/pkg/logger"
)

const telegramDailySalesPrefix = "telegram:daily-sales:"

type telegramDailySale struct {
	DealID string  `json:"deal_id"`
	Bricks float64 `json:"bricks"`
	Total  float64 `json:"total"`
}

type telegramDailySalesTotals struct {
	Deals  int
	Bricks float64
	Total  float64
}

func telegramDailySalesKey(projectID, environmentID string, day time.Time) string {
	return telegramDailySalesPrefix + projectID + ":" + environmentID + ":" + day.Format("2006-01-02")
}

func telegramNumber(value string) float64 {
	value = strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '.' || r == ',' || r == '-' {
			return r
		}
		return -1
	}, value)
	if strings.Contains(value, ".") {
		value = strings.ReplaceAll(value, ",", "")
	} else {
		value = strings.ReplaceAll(value, ",", ".")
	}
	number, _ := strconv.ParseFloat(value, 64)
	return number
}

func telegramDailySaleFromDeal(deal map[string]any) (telegramDailySale, bool) {
	// Armada's brick pipeline stores its stage in this dedicated select field.
	if !telegramStatusValueMatches("Narx kelishildi", telegramDealValue(deal, "pipeline_enterprise_sales")) {
		return telegramDailySale{}, false
	}
	dealID := telegramDealValue(deal, "guid", "id")
	if dealID == "" {
		return telegramDailySale{}, false
	}
	bricks, total := telegramDailySaleValues(deal)
	return telegramDailySale{DealID: dealID, Bricks: bricks, Total: total}, true
}

func telegramDailySaleValues(deal map[string]any) (float64, float64) {
	bricks := telegramNumber(telegramDealValue(deal, "gisht_soni"))
	return bricks, bricks * telegramNumber(telegramDealValue(deal, "gisht_narxi"))
}

func (h *HandlerV1) recordTelegramDailySale(ctx context.Context, projectID, environmentID string, before, after map[string]any) {
	if h.centralRedis == nil {
		return
	}
	sale, ok := telegramDailySaleFromDeal(after)
	if !ok || telegramStatusValueMatches("Narx kelishildi", telegramDealValue(before, "pipeline_enterprise_sales")) {
		return
	}
	location, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		location = time.FixedZone("Tashkent", 5*60*60)
	}
	key := telegramDailySalesKey(projectID, environmentID, time.Now().In(location))
	encoded, _ := json.Marshal(sale)
	if err := h.centralRedis.HSet(ctx, key, sale.DealID, encoded).Err(); err != nil {
		h.log.Warn("telegram daily sales: record failed", logger.Error(err))
		return
	}
	h.centralRedis.Expire(ctx, key, 45*24*time.Hour)
}

func (h *HandlerV1) telegramDailySalesForDay(ctx context.Context, target telegramNotificationTarget, day time.Time) (telegramDailySalesTotals, error) {
	var totals telegramDailySalesTotals
	if h.centralRedis == nil {
		return totals, nil
	}
	stored, err := h.centralRedis.HGetAll(ctx, telegramDailySalesKey(target.ProjectID, target.EnvironmentID, day)).Result()
	if err != nil {
		return totals, err
	}
	if len(stored) == 0 {
		return totals, nil
	}
	service, environmentID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return totals, err
	}
	for _, encoded := range stored {
		var sale telegramDailySale
		if json.Unmarshal([]byte(encoded), &sale) != nil {
			continue
		}
		// The cache identifies deals that reached the agreed-price stage today.
		// Read their current brick fields so reports also correct older cached
		// entries that were previously calculated from the unrelated amount field.
		deal, found, err := h.lookupItem(ctx, service, environmentID, "deals", sale.DealID)
		if err != nil {
			return totals, err
		}
		if !found {
			continue
		}
		sale.Bricks, sale.Total = telegramDailySaleValues(deal)
		totals.Deals++
		totals.Bricks += sale.Bricks
		totals.Total += sale.Total
	}
	return totals, nil
}

func telegramFormatBricks(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func telegramFormatUZS(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	parts := strings.SplitN(text, ".", 2)
	whole := parts[0]
	start := 0
	if strings.HasPrefix(whole, "-") {
		start = 1
	}
	for i := len(whole) - 3; i > start; i -= 3 {
		whole = whole[:i] + " " + whole[i:]
	}
	if parts[1] != "00" {
		whole += "." + parts[1]
	}
	return whole + " UZS"
}
