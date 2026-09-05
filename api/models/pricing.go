package models

type PricingUsage struct {
	Current float64 `json:"current"`
	Limit   float64 `json:"limit"`
	Unit    string  `json:"unit"`
}

type AllPricingUsage struct {
	Functions       PricingUsage `json:"functions"`
	Microfrontend   PricingUsage `json:"microfrontend"`
	AssetSize       PricingUsage `json:"asset_size"`
	DatabaseSize    PricingUsage `json:"database_size"`
	Users           PricingUsage `json:"users"`
	Items           PricingUsage `json:"items"`
	Tables          PricingUsage `json:"tables"`
	ApiKeys         PricingUsage `json:"api_keys"`
	TodayTokens     PricingUsage `json:"today_tokens"`
	MonthlyTokens   PricingUsage `json:"monthly_tokens"`
	MonthlyApiCalls PricingUsage `json:"monthly_api_calls"`
	AvgResponseTime PricingUsage `json:"avg_response_time"`
	Projects        PricingUsage `json:"projects"`
}

type PerformanceMetricsResponse struct {
	AverageResponseTime float64 `json:"average_response_time"`
	ErrorRate           float64 `json:"error_rate"`
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

type CompanyStat struct {
	Current int32 `json:"current"`
	Limit   int32 `json:"limit"`
}

type CompanyTokenStat struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	// PlanTokens is the portion of usage charged against the fare limit; it excludes
	// pack-funded tokens, so PlanTokens vs Limit is the true fare-budget position.
	PlanTokens   int64 `json:"plan_tokens"`
	Limit        int64 `json:"limit"`
	LimitReached bool  `json:"limit_reached"`
}

type CompanyTokenStats struct {
	Daily   CompanyTokenStat `json:"daily"`
	Monthly CompanyTokenStat `json:"monthly"`
	// PackRemaining is the company's remaining add-on token-pack pool. When a fare
	// limit is reached, usage is automatically funded from this pool.
	PackRemaining int64 `json:"pack_remaining"`
	// ActiveSource is the bucket the next tokens will draw from: "plan" while the
	// fare budget has room, "pack" once a fare limit is reached and the pool has
	// tokens, or "exhausted" when both are spent.
	ActiveSource string `json:"active_source"`
}

type CompanyStatsResponse struct {
	Tokens       CompanyTokenStats `json:"tokens"`
	ProjectCount CompanyStat       `json:"project_count"`
	Builders     CompanyStat       `json:"builders"`
	UserCount    CompanyStat       `json:"user_count"`
}

const (
	PaymentRequiredType = "payment_required"

	PaymentCodeDatabaseLimit   = "database_limit"
	PaymentCodeAssetLimit      = "asset_limit"
	PaymentCodeTableLimit      = "table_limit"
	PaymentCodeApiCallLimit    = "api_call_limit"
	PaymentCodeTokenDayLimit   = "token_day_limit"
	PaymentCodeTokenMonthLimit = "token_month_limit"

	PaymentUnitMB       = "mb"
	PaymentUnitTables   = "tables"
	PaymentUnitRequests = "requests"
	PaymentUnitTokens   = "tokens"
)

type PaymentRequiredData struct {
	Type string `json:"type"`           // always PaymentRequiredType
	Code string `json:"code"`           // PaymentCode* constants
	Unit string `json:"unit,omitempty"` // PaymentUnit* constants
}

// Predefined sentinels — use these instead of inline literals.
var (
	PaymentDatabaseLimit = PaymentRequiredData{Type: PaymentRequiredType, Code: PaymentCodeDatabaseLimit, Unit: PaymentUnitMB}
	PaymentAssetLimit    = PaymentRequiredData{Type: PaymentRequiredType, Code: PaymentCodeAssetLimit, Unit: PaymentUnitMB}
	PaymentTableLimit    = PaymentRequiredData{Type: PaymentRequiredType, Code: PaymentCodeTableLimit, Unit: PaymentUnitTables}
	PaymentApiCallLimit  = PaymentRequiredData{Type: PaymentRequiredType, Code: PaymentCodeApiCallLimit, Unit: PaymentUnitRequests}
)

// ApiUsageBreakdownResponse answers "which requests are coming most, and how
// much of the monthly quota is left".
//
// Limit and Remaining are measured against the fare on project.fare_id — the
// same pointer the blocking worker uses — so this response can never disagree
// with whether the project is actually being refused.
type ApiUsageBreakdownResponse struct {
	Limit     int64  `json:"limit"`
	Used      int64  `json:"used"`
	Remaining *int64 `json:"remaining"`
	Unlimited bool   `json:"unlimited"`
	Blocked   bool   `json:"blocked"`
	// UsedUpdatedAt is when the counter behind Used was last written. The
	// pipeline is asynchronous, so this is typically a few minutes behind now.
	UsedUpdatedAt string                 `json:"used_updated_at"`
	From          string                 `json:"from"`
	To            string                 `json:"to"`
	Top           []ApiUsageBreakdownRow `json:"top"`
	// Other is everything the returned rows do not account for. Without filters
	// that is measured against Used, so Top + Other == Used; with filters it is
	// measured against Matched instead.
	Other int64 `json:"other"`
	// Matched is how many requests the filters selected. Equals Used when no
	// filter is set, minus anything recorded before the breakdown existed.
	Matched int64 `json:"matched"`
	// GroupBy echoes which grouping produced Top.
	GroupBy string `json:"group_by"`
}

type ApiUsageBreakdownRow struct {
	Source string `json:"source"`
	// AuthType is how the caller authenticated: api_key, bearer, or empty for
	// rows recorded before this dimension existed.
	AuthType   string `json:"auth_type"`
	Method     string `json:"method"`
	Route      string `json:"route"`
	Collection string `json:"collection"`
	// ActorID is the api_keys record id, or the user id for bearer traffic.
	// Never a credential.
	ActorID   string `json:"actor_id"`
	ActorName string `json:"actor_name"`
	// Bucket is the 15-minute interval, filled only when group_by=time.
	Bucket  string  `json:"bucket"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}
