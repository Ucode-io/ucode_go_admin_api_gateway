package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

const (
	// DebugMode indicates service mode is debug.
	DebugMode = "debug"
	// TestMode indicates service mode is test.
	TestMode = "test"
	// ReleaseMode indicates service mode is release.
	ReleaseMode = "release"
)

type Config struct {
	ServiceName string
	Environment string // debug, test, release
	Version     string

	ServiceHost string
	HTTPPort    string
	HTTPScheme  string

	DefaultOffset string
	DefaultLimit  string

	Postgres struct {
		Host     string
		Port     int
		Username string
		Password string
		Database string
	}

	CorporateServiceHost string
	CorporateGRPCPort    string

	ObjectBuilderServiceHost string
	ObjectBuilderGRPCPort    string

	AuthServiceHost string
	AuthGRPCPort    string

	PosServiceHost string
	PosGRPCPort    string

	AnalyticsServiceHost string
	AnalyticsGRPCPort    string

	SmsServiceHost string
	SmsGRPCPort    string

	CompanyServiceHost string
	CompanyServicePort string

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool

	UcodeNamespace string

	SecretKey string
}

// Load ...
func Load() Config {
	if err := godotenv.Load("/app/.env"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			fmt.Println("No .env file found")
		}
		fmt.Println("No /app/.env file found")
	}

	config := Config{}

	config.ServiceName = cast.ToString(GetOrReturnDefaultValue("SERVICE_NAME", "ucode/ucode_go_api_gateway"))
	config.Environment = cast.ToString(GetOrReturnDefaultValue("ENVIRONMENT", DebugMode))
	config.Version = cast.ToString(GetOrReturnDefaultValue("VERSION", "1.0"))

	config.ServiceHost = cast.ToString(GetOrReturnDefaultValue("SERVICE_HOST", "localhost"))
	config.HTTPPort = cast.ToString(GetOrReturnDefaultValue("HTTP_PORT", ":8001"))
	config.HTTPScheme = cast.ToString(GetOrReturnDefaultValue("HTTP_SCHEME", "http"))

	config.MinioAccessKeyID = cast.ToString(GetOrReturnDefaultValue("MINIO_ACCESS_KEY", "ongei0upha4DiaThioja6aip8dolai1o"))
	config.MinioSecretAccessKey = cast.ToString(GetOrReturnDefaultValue("MINIO_SECRET_KEY", "aew8aeheungohf7vaiphoh7Tusie2vei"))
	config.MinioEndpoint = cast.ToString(GetOrReturnDefaultValue("MINIO_ENDPOINT", "test.cdn.u-code.io"))
	config.MinioProtocol = cast.ToBool(GetOrReturnDefaultValue("MINIO_PROTOCOL", true))

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = cast.ToString(GetOrReturnDefaultValue("DEFAULT_LIMIT", "10000000"))

	config.Postgres.Host = cast.ToString(GetOrReturnDefaultValue("POSTGRES_HOST", "161.35.26.178"))
	config.Postgres.Port = cast.ToInt(GetOrReturnDefaultValue("POSTGRES_PORT", 30032))
	config.Postgres.Username = cast.ToString(GetOrReturnDefaultValue("POSTGRES_USERNAME", "admin_api_gateway"))
	config.Postgres.Password = cast.ToString(GetOrReturnDefaultValue("POSTGRES_PASSWORD", "aoM0zohB"))
	config.Postgres.Database = cast.ToString(GetOrReturnDefaultValue("POSTGRES_DATABASE", "admin_api_gateway"))

	config.CorporateServiceHost = cast.ToString(GetOrReturnDefaultValue("CORPORATE_SERVICE_HOST", "localhost"))
	config.CorporateGRPCPort = cast.ToString(GetOrReturnDefaultValue("CORPORATE_GRPC_PORT", ":9101"))

	config.ObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_HOST", "localhost"))
	config.ObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_GRPC_PORT", ":9102"))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", "0.0.0.0"))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ":9103"))

	config.CompanyServiceHost = cast.ToString(GetOrReturnDefaultValue("COMPANY_SERVICE_HOST", "localhost"))
	config.CompanyServicePort = cast.ToString(GetOrReturnDefaultValue("COMPANY_GRPC_PORT", ":9106"))

	config.PosServiceHost = cast.ToString(GetOrReturnDefaultValue("POS_SERVICE_HOST", "localhost"))
	config.PosGRPCPort = cast.ToString(GetOrReturnDefaultValue("POS_SERVICE_GRPC_PORT", ":8000"))

	config.AnalyticsServiceHost = cast.ToString(GetOrReturnDefaultValue("ANALYTICS_SERVICE_HOST", "localhost"))
	config.AnalyticsGRPCPort = cast.ToString(GetOrReturnDefaultValue("ANALYTICS_GRPC_PORT", ":9175"))

	config.SmsServiceHost = cast.ToString(GetOrReturnDefaultValue("SMS_SERVICE_HOST", "go-sms-service"))
	config.SmsGRPCPort = cast.ToString(GetOrReturnDefaultValue("SMS_GRPC_PORT", ":80"))

	config.UcodeNamespace = "cp-region-type-id"
	config.SecretKey = "Here$houldBe$ome$ecretKey"

	return config
}

func GetOrReturnDefaultValue(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return defaultValue
}
