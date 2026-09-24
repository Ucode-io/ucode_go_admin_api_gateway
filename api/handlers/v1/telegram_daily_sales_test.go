package v1

import "testing"

func TestTelegramDailySaleUsesAgreedPriceStageAndBrickFields(t *testing.T) {
	deal := map[string]any{
		"guid":                      "deal-1",
		"pipeline_enterprise_sales": []any{map[string]any{"label": "Narx kelishildi"}},
		"gisht_soni":                1000,
		"gisht_narxi":               1200,
		"amount":                    1200000,
	}
	sale, ok := telegramDailySaleFromDeal(deal)
	if !ok || sale.DealID != "deal-1" || sale.Bricks != 1000 || sale.Total != 1200000 {
		t.Fatalf("unexpected agreed sale: %#v, %v", sale, ok)
	}
	deal["pipeline_enterprise_sales"] = "Yuk Jonatildi"
	if _, ok := telegramDailySaleFromDeal(deal); ok {
		t.Fatal("a different stage must not count as a new agreed sale")
	}
	deal["pipeline_enterprise_sales"] = "Narx kelishildi"
	delete(deal, "amount")
	sale, ok = telegramDailySaleFromDeal(deal)
	if !ok || sale.Total != 1200000 {
		t.Fatalf("brick quantity × price fallback = %#v, %v", sale, ok)
	}
}
