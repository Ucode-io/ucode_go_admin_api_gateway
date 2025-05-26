package config

import (
	"time"
)

const (
	COMMIT_TYPE_APP                      string = "APP"
	COMMIT_TYPE_TABLE                    string = "TABLE"
	COMMIT_TYPE_FIELD                    string = "FIELD"
	COMMIT_TYPE_RELATION                 string = "RELATION"
	COMMIT_TYPE_SECTION                  string = "SECTION"
	COMMIT_TYPE_VIEW                     string = "VIEW"
	COMMIT_TYPE_VIEW_RELATION            string = "VIEW_RELATION"
	COMMIT_TYPE_CLIENT_PLATFORM          string = "CLIENT_PLATFORM"
	COMMIT_TYPE_CLIENT_TYPE              string = "CLIENT_TYPE"
	COMMIT_TYPE_ROLE                     string = "ROLE"
	COMMIT_TYPE_TEST_LOGIN               string = "TEST_LOGIN"
	COMMIT_TYPE_CONNECTION               string = "CONNECTION"
	COMMIT_TYPE_AUTOMATIC_FILTER         string = "AUTOMATIC_FILTER"
	COMMIT_TYPE_CUSTOM_EVENT             string = "CUSTOM_EVENT"
	COMMIT_TYPE_RECORD_PERMISSION        string = "RECORD_PERMISSION"
	COMMIT_TYPE_ACTION_PERMISSION        string = "ACTION_PERMISSION"
	COMMIT_TYPE_FIELD_PERMISSION         string = "FIELD_PERMISSION"
	COMMIT_TYPE_VIEW_PERMISSION          string = "VIEW_PERMISSION"
	COMMIT_TYPE_VIEW_RELATION_PERMISSION string = "VIEW_RELATION_PERMISSION"
	COMMIT_TYPE_DASHBOARD                string = "DASHBOARD"
	COMMIT_TYPE_VARIABLE                 string = "VARIABLE"
	COMMIT_TYPE_PANEL                    string = "PANEL"
	COMMIT_TYPE_FUNCTION                 string = "FUNCTION"
	COMMIT_TYPE_SCENARIO                 string = "SCENARIO"
	LOW_NODE_TYPE                        string = "LOW"
	HIGH_NODE_TYPE                       string = "HIGH"
	ENTER_PRICE_TYPE                     string = "ENTER_PRICE"
	CACHE_WAIT                           string = "WAIT"

	STATUS_INACTIVE string = "inactive"
)

const (
	LRU_CACHE_SIZE                       = 10000
	LIMITER_RANGE                        = 100
	RATE_LIMITER_RPS_LIMIT               = 100
	REDIS_TIMEOUT          time.Duration = 5 * time.Minute
	REDIS_KEY_TIMEOUT      time.Duration = 280 * time.Second
	REDIS_WAIT_TIMEOUT     time.Duration = 1 * time.Second
	REDIS_SLEEP            time.Duration = 100 * time.Millisecond

	TIME_LAYOUT string = "15:04"

	// FaasBaseurl
	OpenFaaSBaseUrl string = "https://ofs.u-code.io/function/"
	KnativeBaseUrl  string = "knative-fn.u-code.io"

	// Function Types
	FUNCTION string = "FUNCTION"
	KNATIVE  string = "KNATIVE"

	// CustomEventTypes
	BEFORE string = "before"
	AFTER  string = "after"

	PublicStatus = "unapproved"

	InactiveStatus   string = "inactive"
	PermissionDenied string = "Permission denied"
	SessionExpired   string = "Session has been expired"
)

var (
	RelationFieldTypes = map[string]bool{
		"LOOKUP":  true,
		"LOOKUPS": true,
	}

	ConvertDocxToPdfUrl                = "https://v2.convertapi.com/convert/docx/to/pdf?Auth="
	ConvertDocxToPdfSecret             = "secret_M2phGaOMkak05F8L"
	TestNodeDocxConvertToPdfServiceUrl = "https://doc-generator.ucode.run/generate-doc"
)
