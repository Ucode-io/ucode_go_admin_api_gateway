package models

type PricingUsage struct {
	Current float64 `json:"current"`
	Limit   float64 `json:"limit"`
	Unit    string  `json:"unit"`
}

type AllPricingUsage struct {
	Functions     PricingUsage `json:"functions"`
	Microfrontend PricingUsage `json:"microfrontend"`
	AssetSize     PricingUsage `json:"asset_size"`
	DatabaseSize  PricingUsage `json:"database_size"`
	Users         PricingUsage `json:"users"`
	Items         PricingUsage `json:"items"`
	Tables        PricingUsage `json:"tables"`
	ApiKeys       PricingUsage `json:"api_keys"`
	TodayTokens   PricingUsage `json:"today_tokens"`
	MonthlyTokens PricingUsage `json:"monthly_tokens"`
}

type TokenUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type TokenUsageResponse struct {
	Today   TokenUsage `json:"today"`
	Monthly TokenUsage `json:"monthly"`
}

type ApiMetricsResponse struct {
	Rps          float64 `json:"rps"`
	Rpm          int64   `json:"rpm"`
	Rph          int64   `json:"rph"`
	TodayCalls   int64   `json:"today_calls"`
	MonthlyCalls int64   `json:"monthly_calls"`
	LastDayCalls int64   `json:"last_day_calls"`
}

type ApiChartResponse struct {
	Chart []DailyChartPoint `json:"chart"`
}

type DailyChartPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}
