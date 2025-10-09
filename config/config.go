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

var CountReq = 0

type Config struct {
	DefaultOffset string
	DefaultLimit  string

	CorporateServiceHost string
	CorporateGRPCPort    string

	ObjectBuilderServiceHost string
	ObjectBuilderGRPCPort    string

	HighObjectBuilderServiceHost string
	HighObjectBuilderGRPCPort    string

	TemplateServiceHost string
	TemplateGRPCPort    string

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

	ConvertTemplateServiceGrpcHost string
	ConvertTemplateServiceGrpcPort string

	FunctionServiceHost string
	FunctionServicePort string
	QueryServiceHost    string
	QueryServicePort    string
	WebPageServiceHost  string
	WebPageServicePort  string

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool

	UcodeNamespace string

	CLIENT_HOST            string
	SUPERADMIN_HOST        string
	GitlabIntegrationToken string
	GitlabIntegrationURL   string
	GitlabGroupId          int
	GitlabProjectId        int
	PathToClone            string
	SecretKey              string
	PlatformType           string

	ScenarioServiceHost    string
	ScenarioGRPCPort       string
	AdminHostForCodeServer string

	GoObjectBuilderServiceHost string
	GoObjectBuilderGRPCPort    string

	GetRequestRedisHost     string
	GetRequestRedisPort     string
	GetRequestRedisDatabase int
	GetRequestRedisPassword string

	HelmRepoAddFMicroFE    string
	HelmRepoUpdateMicroFE  string
	HelmInstallMicroFE     string
	HelmUninstallMicroFE   string
	GitlabGroupIdMicroFE   int
	GitlabProjectIdMicroFE int
	GitlabHostMicroFE      string

	ServiceLink string

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

	CompanyServiceHost string
	CompanyServicePort string

	GoObjectBuilderServiceHost string
	GoObjectBuilderGRPCPort    string

	GoFunctionServiceHost     string
	GoFunctionServiceHTTPPort string

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool

	GitlabGroupIdMicroFE   int
	GitlabProjectIdMicroFE int
	GitlabHostMicroFE      string

	DefaultOffset   string
	DefaultLimit    string
	DefaultLimitInt int

	CLIENT_HOST     string
	SUPERADMIN_HOST string

	PathToClone  string
	SecretKey    string
	PlatformType string

	UcodeNamespace string
	JaegerHostPort string

	GithubClientId         string
	GithubClientSecret     string
	ProjectUrl             string
	WebhookSecret          string
	ConvertDocxToPdfSecret string

	N8NApiKey  string
	N8NBaseURL string

	StripeApiKey        string
	StripeWebhookSecret string
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

	config.HTTPBaseURL = cast.ToString(GetOrReturnDefaultValue("HTTP_BASE_URL", "https://api.admin.u-code.io"))
	config.ServiceHost = cast.ToString(GetOrReturnDefaultValue("SERVICE_HOST", ""))
	config.HTTPPort = cast.ToString(GetOrReturnDefaultValue("HTTP_PORT", ":8000"))
	config.HTTPScheme = cast.ToString(GetOrReturnDefaultValue("HTTP_SCHEME", ""))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", ""))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ""))

	config.CompanyServiceHost = cast.ToString(GetOrReturnDefaultValue("COMPANY_SERVICE_HOST", ""))
	config.CompanyServicePort = cast.ToString(GetOrReturnDefaultValue("COMPANY_GRPC_PORT", ""))

	config.MinioAccessKeyID = cast.ToString(GetOrReturnDefaultValue("MINIO_ACCESS_KEY", ""))
	config.MinioSecretAccessKey = cast.ToString(GetOrReturnDefaultValue("MINIO_SECRET_KEY", ""))
	config.MinioEndpoint = cast.ToString(GetOrReturnDefaultValue("MINIO_ENDPOINT", ""))
	config.MinioProtocol = cast.ToBool(GetOrReturnDefaultValue("MINIO_PROTOCOL", true))

	config.GitlabGroupIdMicroFE = cast.ToInt(GetOrReturnDefaultValue("GITLAB_GROUP_ID_MICROFE", 2604))
	config.GitlabProjectIdMicroFE = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFE", 0))
	config.GitlabHostMicroFE = cast.ToString(GetOrReturnDefaultValue("GITLAB_HOST_MICROFE", "test-page.u-code.io"))

	config.GoFunctionServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_FUNCTION_SERVICE_HOST", "http://localhost"))
	config.GoFunctionServiceHTTPPort = cast.ToString(GetOrReturnDefaultValue("GO_FUNCTION_SERVICE_HTTP_PORT", ":7090"))

	config.GoObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_HOST", "localhost"))
	config.GoObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_PORT", ":7107"))

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = "60"
	config.DefaultLimitInt = 20

	config.PathToClone = cast.ToString(GetOrReturnDefaultValue("CLONE_PATH", "./app"))

	config.UcodeNamespace = "u-code"
	config.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", ""))
	config.JaegerHostPort = cast.ToString(GetOrReturnDefaultValue("JAEGER_URL", ""))

	config.GithubClientId = cast.ToString(GetOrReturnDefaultValue("GITHUB_CLIENT_ID", "Ov23liaLeqZ4ihyU3CWQ"))
	config.GithubClientSecret = cast.ToString(GetOrReturnDefaultValue("GITHUB_CLIENT_SECRET", "cd5e802aa567432f8a053660dca5698678dfbe23"))
	config.ProjectUrl = "https://admin-api.ucode.run"
	config.WebhookSecret = "X8kJnsNHD9f4nRQfjs72YLSfPqxjG+PWRjxN3KBuDhE="

	config.N8NApiKey = cast.ToString(GetOrReturnDefaultValue("N8N_API_KEY", ""))
	config.N8NBaseURL = cast.ToString(GetOrReturnDefaultValue("N8N_BASE_URL", "https://n8n.u-code.io"))

	config.StripeApiKey = cast.ToString(GetOrReturnDefaultValue("STRIPE_API_KEY", "sk_test_51QvC6qCx1p2EqOQp37eMRD73jmsECnITZ1eYTn4BbYv8uLNUfGOJUf3X0j14fyjhAvcoZYucz9oCy1aEJrg7Yyp300ScU9kgfh"))
	config.StripeWebhookSecret = cast.ToString(GetOrReturnDefaultValue("STRIPE_WEBHOOK_SECRET", "whsec_cOGBaP6EVo4kRUCfeKXuSWg0JAL2avRg"))

	return config
}

// Load ...
func Load() Config {

	config := Config{}

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = "100"

	config.CorporateServiceHost = cast.ToString(GetOrReturnDefaultValue("CORPORATE_SERVICE_HOST", ""))
	config.CorporateGRPCPort = cast.ToString(GetOrReturnDefaultValue("CORPORATE_GRPC_PORT", ":2025"))

	config.ObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_LOW_HOST", ""))
	config.ObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_LOW_GRPC_PORT", ""))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", ""))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ""))
	config.ScenarioGRPCPort = ":5001"

	config.CompanyServiceHost = cast.ToString(GetOrReturnDefaultValue("COMPANY_SERVICE_HOST", ""))
	config.CompanyServicePort = cast.ToString(GetOrReturnDefaultValue("COMPANY_GRPC_PORT", ""))

	config.TranscoderServiceHost = cast.ToString(GetOrReturnDefaultValue("TRANSCODER_SERVICE_HOST", "localhost"))
	config.TranscoderServicePort = cast.ToString(GetOrReturnDefaultValue("TRANSCODER_GRPC_PORT", ":9110"))

	config.HighObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_HIGHT_HOST", ""))
	config.HighObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_HIGH_GRPC_PORT", ""))

	config.TemplateServiceHost = cast.ToString(GetOrReturnDefaultValue("TEMPLATE_SERVICE_HOST", ""))
	config.TemplateGRPCPort = cast.ToString(GetOrReturnDefaultValue("TEMPLATE_GRPC_PORT", ":2012"))

	config.SmsServiceHost = cast.ToString(GetOrReturnDefaultValue("SMS_SERVICE_HOST", ""))
	config.SmsGRPCPort = cast.ToString(GetOrReturnDefaultValue("SMS_GRPC_PORT", ":2008"))

	config.ConvertTemplateServiceGrpcHost = cast.ToString(GetOrReturnDefaultValue("CONVERT_TEMPLATE_SERVICE_HOST", ""))
	config.ConvertTemplateServiceGrpcPort = cast.ToString(GetOrReturnDefaultValue("CONVERT_TEMPLATE_GRPC_PORT", ":2006"))

	config.FunctionServiceHost = cast.ToString(GetOrReturnDefaultValue("FUNCTION_SERVICE_HOST", ""))
	config.FunctionServicePort = cast.ToString(GetOrReturnDefaultValue("FUNCTION_GRPC_PORT", ":2005"))

	config.GoObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_HOST", "localhost"))
	config.GoObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_GRPC_PORT", ":7107"))

	config.QueryServiceHost = cast.ToString(GetOrReturnDefaultValue("QUERY_SERVICE_HOST", ""))
	config.QueryServicePort = cast.ToString(GetOrReturnDefaultValue("QUERY_GRPC_PORT", ":3001"))
	config.WebPageServiceHost = cast.ToString(GetOrReturnDefaultValue("WEB_PAGE_SERVICE_HOST", ""))
	config.WebPageServicePort = cast.ToString(GetOrReturnDefaultValue("WEB_PAGE_GRPC_PORT", ":2004"))

	config.UcodeNamespace = "cp-region-type-id"
	config.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", ""))

	config.DocGeneratorGrpcHost = cast.ToString(GetOrReturnDefaultValue("NODE_DOC_GENERATOR_SERVICE_HOST", "localhost"))
	config.DocGeneratorGrpcPort = cast.ToString(GetOrReturnDefaultValue("NODE_DOC_GENERATOR_GRPC_PORT", ":50051"))

	config.GetRequestRedisHost = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_HOST", ""))
	config.GetRequestRedisPort = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PORT", ""))
	config.GetRequestRedisDatabase = cast.ToInt(GetOrReturnDefaultValue("GET_REQUEST_REDIS_DATABASE", 0))
	config.GetRequestRedisPassword = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PASSWORD", ""))

	config.ServiceLink = ""

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
