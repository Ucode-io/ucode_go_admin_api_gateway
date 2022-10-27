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

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool
}

// Load ...
func Load() Config {
	if err := godotenv.Load("/app/.env"); err != nil {
		fmt.Println("No .env file found")
	}

	config := Config{}

	config.ServiceName = cast.ToString(getOrReturnDefaultValue("SERVICE_NAME", "ucode_go_api_gateway"))
	config.Environment = cast.ToString(getOrReturnDefaultValue("ENVIRONMENT", DebugMode))
	config.Version = cast.ToString(getOrReturnDefaultValue("VERSION", "1.0"))

	config.ServiceHost = cast.ToString(getOrReturnDefaultValue("SERVICE_HOST", "localhost"))
	config.HTTPPort = cast.ToString(getOrReturnDefaultValue("HTTP_PORT", ":8001"))
	config.HTTPScheme = cast.ToString(getOrReturnDefaultValue("HTTP_SCHEME", "http"))

	config.MinioAccessKeyID = cast.ToString(getOrReturnDefaultValue("MINIO_ACCESS_KEY", "JqEZQP7w5XJSy2K6ZQh5VJbLWZWbcESZcVkNbakGw977FCwa"))
	config.MinioSecretAccessKey = cast.ToString(getOrReturnDefaultValue("MINIO_SECRET_KEY", "bYTX8fnBKGLhvpvQfsp63MXkBHCuEp8gScf4wUfnGANUwHxZ"))
	config.MinioEndpoint = cast.ToString(getOrReturnDefaultValue("MINIO_ENDPOINT", "test.cdn.medion.uz"))
	config.MinioProtocol = cast.ToBool(getOrReturnDefaultValue("MINIO_PROTOCOL", true))

	config.DefaultOffset = cast.ToString(getOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = cast.ToString(getOrReturnDefaultValue("DEFAULT_LIMIT", "10000000"))

	config.CorporateServiceHost = cast.ToString(getOrReturnDefaultValue("CORPORATE_SERVICE_HOST", "localhost"))
	config.CorporateGRPCPort = cast.ToString(getOrReturnDefaultValue("CORPORATE_GRPC_PORT", ":9101"))

	config.ObjectBuilderServiceHost = cast.ToString(getOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_HOST", "localhost"))
	config.ObjectBuilderGRPCPort = cast.ToString(getOrReturnDefaultValue("OBJECT_BUILDER_GRPC_PORT", ":9102"))

	config.AuthServiceHost = cast.ToString(getOrReturnDefaultValue("AUTH_SERVICE_HOST", "0.0.0.0"))
	config.AuthGRPCPort = cast.ToString(getOrReturnDefaultValue("AUTH_GRPC_PORT", ":9103"))

	config.PosServiceHost = cast.ToString(getOrReturnDefaultValue("POS_SERVICE_HOST", "localhost"))
	config.PosGRPCPort = cast.ToString(getOrReturnDefaultValue("POS_SERVICE_GRPC_PORT", ":8000"))

	config.AnalyticsServiceHost = cast.ToString(getOrReturnDefaultValue("ANALYTICS_SERVICE_HOST", "localhost"))
	config.AnalyticsGRPCPort = cast.ToString(getOrReturnDefaultValue("ANALYTICS_SERVICE_GRPC_PORT", ":9175"))

	return config
}

func getOrReturnDefaultValue(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return defaultValue
}
