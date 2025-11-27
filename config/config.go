package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

const (
	DebugMode = "debug"
	// TestMode indicates service mode is test.
	TestMode = "test"
	// ReleaseMode indicates service mode is release.
	ReleaseMode = "release"
)

type Config struct {
	DefaultOffset string
	DefaultLimit  string

	ObjectBuilderServiceHost string
	ObjectBuilderGRPCPort    string

	HighObjectBuilderServiceHost string
	HighObjectBuilderGRPCPort    string

	SmsServiceHost string
	SmsGRPCPort    string

	AuthServiceHost string
	AuthGRPCPort    string

	CompanyServiceHost string
	CompanyServicePort string

	TranscoderServiceHost string
	TranscoderServicePort string

	DocGeneratorGrpcHost string
	DocGeneratorGrpcPort string

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool

	UcodeNamespace string

	SecretKey string

	GoObjectBuilderServiceHost string
	GoObjectBuilderGRPCPort    string

	GetRequestRedisHost     string
	GetRequestRedisPort     string
	GetRequestRedisDatabase int
	GetRequestRedisPassword string

	OpenAIApiKey string
}

type BaseConfig struct {
	ServiceName string
	Environment string
	Version     string

	HTTPBaseURL string
	ServiceHost string
	HTTPPort    string
	HTTPScheme  string

	AuthServiceHost string
	AuthGRPCPort    string

	GoObjectBuilderServiceHost string
	GoObjectBuilderGRPCPort    string

	GoFunctionServiceHost     string
	GoFunctionServiceHTTPPort string

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool

	DefaultOffset   string
	DefaultLimit    string
	DefaultLimitInt int

	SecretKey string

	UcodeNamespace string
	JaegerHostPort string

	N8NApiKey  string
	N8NBaseURL string

	StripeApiKey        string
	StripeWebhookSecret string

	AnthropicAPIKey     string
	AnthropicBeta       string
	AnthropicBaseAPIURL string
	ClaudeModel         string
	AutomationURL       string
	OpenFaaSBaseUrl     string
	KnativeBaseUrl      string
	MaxTokens           int
	AnthropicVersion    string
	MCPServerURL        string
}

func BaseLoad() BaseConfig {
	if err := godotenv.Load("/app/.env"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("No .env file found")
		}
		log.Println("No /app/.env file found")
	}

	config := BaseConfig{}

	config.ServiceName = cast.ToString(GetOrReturnDefaultValue("SERVICE_NAME", "ucode_go_api_gateway"))
	config.Environment = cast.ToString(GetOrReturnDefaultValue("ENVIRONMENT", DebugMode))
	config.Version = cast.ToString(GetOrReturnDefaultValue("VERSION", "1.0"))

	config.HTTPBaseURL = cast.ToString(GetOrReturnDefaultValue("HTTP_BASE_URL", ""))
	config.ServiceHost = cast.ToString(GetOrReturnDefaultValue("SERVICE_HOST", ""))
	config.HTTPPort = cast.ToString(GetOrReturnDefaultValue("HTTP_PORT", ":8000"))
	config.HTTPScheme = cast.ToString(GetOrReturnDefaultValue("HTTP_SCHEME", ""))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", ""))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ""))

	config.MinioAccessKeyID = cast.ToString(GetOrReturnDefaultValue("MINIO_ACCESS_KEY", ""))
	config.MinioSecretAccessKey = cast.ToString(GetOrReturnDefaultValue("MINIO_SECRET_KEY", ""))
	config.MinioEndpoint = cast.ToString(GetOrReturnDefaultValue("MINIO_ENDPOINT", ""))
	config.MinioProtocol = cast.ToBool(GetOrReturnDefaultValue("MINIO_PROTOCOL", true))

	config.GoFunctionServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_FUNCTION_SERVICE_HOST", "http://localhost"))
	config.GoFunctionServiceHTTPPort = cast.ToString(GetOrReturnDefaultValue("GO_FUNCTION_SERVICE_HTTP_PORT", ":7090"))

	config.GoObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_HOST", "localhost"))
	config.GoObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_PORT", ":7107"))

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = "60"
	config.DefaultLimitInt = 20

	config.UcodeNamespace = "u-code"
	config.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", ""))
	config.JaegerHostPort = cast.ToString(GetOrReturnDefaultValue("JAEGER_URL", ""))

	config.N8NApiKey = cast.ToString(GetOrReturnDefaultValue("N8N_API_KEY", ""))
	config.N8NBaseURL = cast.ToString(GetOrReturnDefaultValue("N8N_BASE_URL", ""))

	config.StripeApiKey = cast.ToString(GetOrReturnDefaultValue("STRIPE_API_KEY", ""))
	config.StripeWebhookSecret = cast.ToString(GetOrReturnDefaultValue("STRIPE_WEBHOOK_SECRET", ""))

	config.AnthropicAPIKey = cast.ToString(GetOrReturnDefaultValue("ANTHROPIC_API_KEY", ""))
	config.AnthropicBeta = cast.ToString(GetOrReturnDefaultValue("ANTHROPIC_BETA", ""))
	config.AnthropicBaseAPIURL = cast.ToString(GetOrReturnDefaultValue("ANTHROPIC_BASE_API_URL", ""))
	config.AnthropicVersion = cast.ToString(GetOrReturnDefaultValue("ANTHROPIC_VERSION", ""))
	config.ClaudeModel = cast.ToString(GetOrReturnDefaultValue("CLAUDE_MODEL", ""))
	config.AutomationURL = cast.ToString(GetOrReturnDefaultValue("AUTOMATION_URL", ""))
	config.OpenFaaSBaseUrl = cast.ToString(GetOrReturnDefaultValue("OPENFAAS_BASE_URL", ""))
	config.KnativeBaseUrl = cast.ToString(GetOrReturnDefaultValue("KNATIVE_BASE_URL", ""))
	config.MaxTokens = cast.ToInt(GetOrReturnDefaultValue("MAX_TOKENS", 12000))
	config.MCPServerURL = cast.ToString(GetOrReturnDefaultValue("MCP_SERVER_URL", ""))

	return config
}

// Load ...
func Load() Config {

	config := Config{}

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = "100"

	config.ObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_LOW_HOST", ""))
	config.ObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_LOW_GRPC_PORT", ""))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", ""))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ""))

	config.CompanyServiceHost = cast.ToString(GetOrReturnDefaultValue("COMPANY_SERVICE_HOST", ""))
	config.CompanyServicePort = cast.ToString(GetOrReturnDefaultValue("COMPANY_GRPC_PORT", ""))

	config.TranscoderServiceHost = cast.ToString(GetOrReturnDefaultValue("TRANSCODER_SERVICE_HOST", "localhost"))
	config.TranscoderServicePort = cast.ToString(GetOrReturnDefaultValue("TRANSCODER_GRPC_PORT", ":9110"))

	config.HighObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_HIGHT_HOST", ""))
	config.HighObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_HIGH_GRPC_PORT", ""))

	config.SmsServiceHost = cast.ToString(GetOrReturnDefaultValue("SMS_SERVICE_HOST", ""))
	config.SmsGRPCPort = cast.ToString(GetOrReturnDefaultValue("SMS_GRPC_PORT", ":2008"))

	config.GoObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_HOST", "localhost"))
	config.GoObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_PORT", ":7107"))

	config.UcodeNamespace = "cp-region-type-id"
	config.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", ""))

	config.DocGeneratorGrpcHost = cast.ToString(GetOrReturnDefaultValue("NODE_DOC_GENERATOR_SERVICE_HOST", "localhost"))
	config.DocGeneratorGrpcPort = cast.ToString(GetOrReturnDefaultValue("NODE_DOC_GENERATOR_GRPC_PORT", ":50051"))

	config.GetRequestRedisHost = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_HOST", ""))
	config.GetRequestRedisPort = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PORT", ""))
	config.GetRequestRedisDatabase = cast.ToInt(GetOrReturnDefaultValue("GET_REQUEST_REDIS_DATABASE", 0))
	config.GetRequestRedisPassword = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PASSWORD", ""))

	config.OpenAIApiKey = cast.ToString(GetOrReturnDefaultValue("OPENAI_API_KEY", "sk-proj-a2ma7TfGU0msgfY9GDsST3BlbkFJljmuOgGattnpsfQCnJ2C"))

	return config
}

func GetOrReturnDefaultValue(key string, defaultValue any) any {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return defaultValue
}
