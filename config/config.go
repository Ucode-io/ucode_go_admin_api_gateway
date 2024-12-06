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

	PosServiceHost string
	PosGRPCPort    string

	AnalyticsServiceHost string
	AnalyticsGRPCPort    string

	SmsServiceHost string
	SmsGRPCPort    string

	VersioningServiceHost string
	VersioningGRPCPort    string

	AuthServiceHost string
	AuthGRPCPort    string

	CompanyServiceHost string
	CompanyServicePort string

	IntegrationServiceHost string
	IntegrationGRPCPort    string

	ApiReferenceServiceHost string
	ApiReferenceServicePort string

	ChatServiceGrpcHost string
	ChatServiceGrpcPort string

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

	NotificationServiceHost string
	NotificationGRPCPort    string

	PostgresBuilderServiceHost string
	PostgresBuilderServicePort string

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

	DefaultOffset string
	DefaultLimit  string

	CLIENT_HOST                   string
	SUPERADMIN_HOST               string
	GitlabIntegrationToken        string
	GitlabIntegrationURL          string
	GitlabGroupId                 int
	GitlabProjectId               int
	GitlabProjectIdMicroFEReact   int
	GitlabProjectIdMicroFEAngular int
	GitlabProjectIdMicroFEVue     int
	PathToClone                   string
	SecretKey                     string
	PlatformType                  string

	UcodeNamespace string
	JaegerHostPort string

	GithubClientId         string
	GithubClientSecret     string
	ProjectUrl             string
	WebhookSecret          string
	ConvertDocxToPdfSecret string
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

	config.GitlabIntegrationToken = cast.ToString(GetOrReturnDefaultValue("GITLAB_ACCESS_TOKEN", "glpat-KxgpdFWZFXHyBG6x9A2x"))
	config.GitlabIntegrationURL = cast.ToString(GetOrReturnDefaultValue("GITLAB_URL", "https://gitlab.udevs.io"))
	config.GitlabGroupId = cast.ToInt(GetOrReturnDefaultValue("GITLAB_GROUP_ID", 5466))
	config.GitlabProjectId = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID", 4622))
	config.GitlabProjectIdMicroFEReact = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFEReact", 1993))
	config.GitlabProjectIdMicroFEAngular = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFEAngular", 0))
	config.GitlabProjectIdMicroFEVue = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFEVue", 0))
	config.PathToClone = cast.ToString(GetOrReturnDefaultValue("CLONE_PATH", "./app"))

	config.UcodeNamespace = "u-code"
	config.SecretKey = cast.ToString(GetOrReturnDefaultValue("SECRET_KEY", ""))
	config.JaegerHostPort = cast.ToString(GetOrReturnDefaultValue("JAEGER_URL", ""))

	config.GithubClientId = cast.ToString(GetOrReturnDefaultValue("GITHUB_CLIENT_ID", "Ov23liaLeqZ4ihyU3CWQ"))
	config.GithubClientSecret = cast.ToString(GetOrReturnDefaultValue("GITHUB_CLIENT_SECRET", "cd5e802aa567432f8a053660dca5698678dfbe23"))
	config.ProjectUrl = "https://admin-api.ucode.run"
	config.WebhookSecret = "X8kJnsNHD9f4nRQfjs72YLSfPqxjG+PWRjxN3KBuDhE="

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

	config.HighObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_HIGHT_HOST", ""))
	config.HighObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_HIGH_GRPC_PORT", ""))

	config.TemplateServiceHost = cast.ToString(GetOrReturnDefaultValue("TEMPLATE_SERVICE_HOST", ""))
	config.TemplateGRPCPort = cast.ToString(GetOrReturnDefaultValue("TEMPLATE_GRPC_PORT", ":2012"))

	config.IntegrationServiceHost = cast.ToString(GetOrReturnDefaultValue("INTEGRATION_SERVICE_HOST", ""))
	config.IntegrationGRPCPort = cast.ToString(GetOrReturnDefaultValue("INTEGRATION_GRPC_PORT", ":4001"))

	config.PosServiceHost = cast.ToString(GetOrReturnDefaultValue("POS_SERVICE_HOST", ""))
	config.PosGRPCPort = cast.ToString(GetOrReturnDefaultValue("POS_SERVICE_GRPC_PORT", ":2011"))

	config.AnalyticsServiceHost = cast.ToString(GetOrReturnDefaultValue("ANALYTICS_SERVICE_HOST", ""))
	config.AnalyticsGRPCPort = cast.ToString(GetOrReturnDefaultValue("ANALYTICS_GRPC_PORT", ":2010"))

	config.SmsServiceHost = cast.ToString(GetOrReturnDefaultValue("SMS_SERVICE_HOST", ""))
	config.SmsGRPCPort = cast.ToString(GetOrReturnDefaultValue("SMS_GRPC_PORT", ":2008"))

	config.VersioningServiceHost = cast.ToString(GetOrReturnDefaultValue("VERSIONING_SERVICE_HOST", ""))
	config.VersioningGRPCPort = cast.ToString(GetOrReturnDefaultValue("VERSIONING_GRPC_PORT", ":2009"))

	config.ApiReferenceServiceHost = cast.ToString(GetOrReturnDefaultValue("API_REF_SERVICE_HOST", ""))
	config.ApiReferenceServicePort = cast.ToString(GetOrReturnDefaultValue("API_REF_GRPC_PORT", ":2007"))

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

	config.ChatServiceGrpcHost = cast.ToString(GetOrReturnDefaultValue("CHAT_SERVICE_HOST", ""))
	config.ChatServiceGrpcPort = cast.ToString(GetOrReturnDefaultValue("CHAT_GRPC_PORT", ":2112"))
	config.NotificationServiceHost = cast.ToString(GetOrReturnDefaultValue("NOTIFICATION_SERVICE_HOST", ""))
	config.NotificationGRPCPort = cast.ToString(GetOrReturnDefaultValue("NOTIFICATION_GRPC_PORT", ":2001"))

	config.PostgresBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("NODE_POSTGRES_SERVICE_HOST", ""))
	config.PostgresBuilderServicePort = cast.ToString(GetOrReturnDefaultValue("NODE_POSTGRES_SERVICE_PORT", ":2002"))

	config.GetRequestRedisHost = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_HOST", ""))
	config.GetRequestRedisPort = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PORT", ""))
	config.GetRequestRedisDatabase = cast.ToInt(GetOrReturnDefaultValue("GET_REQUEST_REDIS_DATABASE", 0))
	config.GetRequestRedisPassword = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PASSWORD", ""))

	config.ServiceLink = ""

	config.OpenAIApiKey = cast.ToString(GetOrReturnDefaultValue("OPENAI_API_KEY", "sk-proj-a2ma7TfGU0msgfY9GDsST3BlbkFJljmuOgGattnpsfQCnJ2C"))

	return config
}

func GetOrReturnDefaultValue(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return defaultValue
}
