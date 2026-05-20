package models

type TokenLimitData struct {
	Type       string `json:"type"`   // "payment_required"
	Code       string `json:"code"`   // "token_day_limit" | "token_month_limit"
	Period     string `json:"period"` // "day" | "month"
	Used       int64  `json:"used"`
	Limit      int64  `json:"limit"`
	Unit       string `json:"unit"` // "tokens"
	DayUsed    int64  `json:"day_used"`
	DayLimit   int64  `json:"day_limit"`
	MonthUsed  int64  `json:"month_used"`
	MonthLimit int64  `json:"month_limit"`
}

type TokenBudgetSnapshot struct {
	DayLimit   int64
	DayUsed    int64
	MonthLimit int64
	MonthUsed  int64
}
