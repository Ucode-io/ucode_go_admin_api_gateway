package config

import "time"

const (
	COMMIT_TYPE_APP                      = "APP"
	COMMIT_TYPE_TABLE                    = "TABLE"
	COMMIT_TYPE_FIELD                    = "FIELD"
	COMMIT_TYPE_RELATION                 = "RELATION"
	COMMIT_TYPE_SECTION                  = "SECTION"
	COMMIT_TYPE_VIEW                     = "VIEW"
	COMMIT_TYPE_VIEW_RELATION            = "VIEW_RELATION"
	COMMIT_TYPE_CLIENT_PLATFORM          = "CLIENT_PLATFORM"
	COMMIT_TYPE_CLIENT_TYPE              = "CLIENT_TYPE"
	COMMIT_TYPE_ROLE                     = "ROLE"
	COMMIT_TYPE_TEST_LOGIN               = "TEST_LOGIN"
	COMMIT_TYPE_CONNECTION               = "CONNECTION"
	COMMIT_TYPE_AUTOMATIC_FILTER         = "AUTOMATIC_FILTER"
	COMMIT_TYPE_CUSTOM_EVENT             = "CUSTOM_EVENT"
	COMMIT_TYPE_RECORD_PERMISSION        = "RECORD_PERMISSION"
	COMMIT_TYPE_ACTION_PERMISSION        = "ACTION_PERMISSION"
	COMMIT_TYPE_FIELD_PERMISSION         = "FIELD_PERMISSION"
	COMMIT_TYPE_VIEW_PERMISSION          = "VIEW_PERMISSION"
	COMMIT_TYPE_VIEW_RELATION_PERMISSION = "VIEW_RELATION_PERMISSION"
	COMMIT_TYPE_DASHBOARD                = "DASHBOARD"
	COMMIT_TYPE_VARIABLE                 = "VARIABLE"
	COMMIT_TYPE_PANEL                    = "PANEL"
	COMMIT_TYPE_FUNCTION                 = "FUNCTION"
	COMMIT_TYPE_SCENARIO                 = "SCENARIO"
	LOW_NODE_TYPE                        = "LOW"
	HIGH_NODE_TYPE                       = "HIGH"
	ENTER_PRICE_TYPE                     = "ENTER_PRICE"
	CACHE_WAIT                           = "WAIT"
)

const (
	LRU_CACHE_SIZE         = 10000
	REDIS_TIMEOUT          = 5 * time.Minute
	REDIS_KEY_TIMEOUT      = 280 * time.Second
	REDIS_WAIT_TIMEOUT     = 1 * time.Second
	REDIS_SLEEP            = 100 * time.Millisecond
	LIMITER_RANGE          = 100
	RATE_LIMITER_RPS_LIMIT = 100

	TIME_LAYOUT = "15:04"
)

var (
	DynamicReportFormula = []string{"SUM", "COUNT", "AVERAGE", "MAX", "MIN", "FIRST", "LAST", "END_FIRST", "END_LAST"}
	BarcodeTypes         = map[string]int{
		"barcode": 1,
		"codabar": 1,
	}
)

var NodeDocxConvertToPdfServiceUrl = "https://crowe-doc-generator.u-code.io/generate-doc"
