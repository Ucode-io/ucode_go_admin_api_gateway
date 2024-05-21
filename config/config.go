package config

import (
	"log"
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
}

type BaseConfig struct {
	ServiceName string
	Environment string
	Version     string

	ServiceHost string
	HTTPPort    string
	HTTPScheme  string

	AuthServiceHost string
	AuthGRPCPort    string

	CompanyServiceHost string
	CompanyServicePort string

	MinioEndpoint        string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
	MinioProtocol        bool

	GitlabGroupIdMicroFE   int
	GitlabProjectIdMicroFE int
	GitlabHostMicroFE      string

	DefaultOffset string
	DefaultLimit  string

	GoObjectBuilderServiceHost string
	GoObjectBuilderGRPCPort    string

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

	GithubClientId     string
	GithubClientSecret string
	ProjectUrl         string
	WebhookSecret      string
	//OfsHost string
}

func BaseLoad() BaseConfig {
	if err := godotenv.Load("/app/.env"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("No .env file found")
		}
		log.Println("No /app/.env file found")
	}

	config := BaseConfig{}

	config.ServiceName = cast.ToString(GetOrReturnDefaultValue("SERVICE_NAME", "ucode/ucode_go_api_gateway"))
	config.Environment = cast.ToString(GetOrReturnDefaultValue("ENVIRONMENT", DebugMode))
	config.Version = cast.ToString(GetOrReturnDefaultValue("VERSION", "1.0"))

	config.ServiceHost = cast.ToString(GetOrReturnDefaultValue("SERVICE_HOST", ""))
	config.HTTPPort = cast.ToString(GetOrReturnDefaultValue("HTTP_PORT", ""))
	config.HTTPScheme = cast.ToString(GetOrReturnDefaultValue("HTTP_SCHEME", ""))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", ""))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ""))

	config.CompanyServiceHost = cast.ToString(GetOrReturnDefaultValue("COMPANY_SERVICE_HOST", ""))
	config.CompanyServicePort = cast.ToString(GetOrReturnDefaultValue("COMPANY_GRPC_PORT", ""))

	config.MinioAccessKeyID = cast.ToString(GetOrReturnDefaultValue("MINIO_ACCESS_KEY", ""))
	config.MinioSecretAccessKey = cast.ToString(GetOrReturnDefaultValue("MINIO_SECRET_KEY", ""))
	config.MinioEndpoint = cast.ToString(GetOrReturnDefaultValue("MINIO_ENDPOINT", ""))
	config.MinioProtocol = cast.ToBool(GetOrReturnDefaultValue("MINIO_PROTOCOL", true))

	config.GitlabGroupIdMicroFE = cast.ToInt(GetOrReturnDefaultValue("GITLAB_GROUP_ID_MICROFE", 0))
	config.GitlabProjectIdMicroFE = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFE", 0))
	config.GitlabHostMicroFE = cast.ToString(GetOrReturnDefaultValue("GITLAB_HOST_MICROFE", ""))

	config.GoObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_HOST", "localhost"))
	config.GoObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_GRPC_PORT", ":7107"))

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = "100"

	config.GitlabIntegrationToken = cast.ToString(GetOrReturnDefaultValue("GITLAB_ACCESS_TOKEN", ""))
	config.GitlabIntegrationURL = cast.ToString(GetOrReturnDefaultValue("GITLAB_URL", ""))
	config.GitlabGroupId = cast.ToInt(GetOrReturnDefaultValue("GITLAB_GROUP_ID", 0))
	config.GitlabProjectId = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID", 0))
	config.GitlabProjectIdMicroFEReact = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFEReact", 0))
	config.GitlabProjectIdMicroFEAngular = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFEAngular", 0))
	config.GitlabProjectIdMicroFEVue = cast.ToInt(GetOrReturnDefaultValue("GITLAB_PROJECT_ID_MICROFEVue", 0))
	config.PathToClone = cast.ToString(GetOrReturnDefaultValue("CLONE_PATH", "./app"))

	config.UcodeNamespace = "u-code"

	config.GithubClientId = ""
	config.GithubClientSecret = ""
	config.ProjectUrl = ""
	config.WebhookSecret = ""

	return config
}

// Load ...
func Load() Config {

	config := Config{}

	config.DefaultOffset = cast.ToString(GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	config.DefaultLimit = "100"

	config.CorporateServiceHost = cast.ToString(GetOrReturnDefaultValue("CORPORATE_SERVICE_HOST", ""))
	config.CorporateGRPCPort = cast.ToString(GetOrReturnDefaultValue("CORPORATE_GRPC_PORT", ""))

	config.ObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_LOW_HOST", ""))
	config.ObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_LOW_GRPC_PORT", ""))

	config.AuthServiceHost = cast.ToString(GetOrReturnDefaultValue("AUTH_SERVICE_HOST", ""))
	config.AuthGRPCPort = cast.ToString(GetOrReturnDefaultValue("AUTH_GRPC_PORT", ""))

	config.CompanyServiceHost = cast.ToString(GetOrReturnDefaultValue("COMPANY_SERVICE_HOST", ""))
	config.CompanyServicePort = cast.ToString(GetOrReturnDefaultValue("COMPANY_GRPC_PORT", ""))

	config.HighObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_SERVICE_HIGHT_HOST", ""))
	config.HighObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("OBJECT_BUILDER_HIGH_GRPC_PORT", ""))

	config.TemplateServiceHost = cast.ToString(GetOrReturnDefaultValue("TEMPLATE_SERVICE_HOST", ""))
	config.TemplateGRPCPort = cast.ToString(GetOrReturnDefaultValue("TEMPLATE_GRPC_PORT", ""))

	config.IntegrationServiceHost = cast.ToString(GetOrReturnDefaultValue("INTEGRATION_SERVICE_HOST", ""))
	config.IntegrationGRPCPort = cast.ToString(GetOrReturnDefaultValue("INTEGRATION_GRPC_PORT", ""))

	config.PosServiceHost = cast.ToString(GetOrReturnDefaultValue("POS_SERVICE_HOST", ""))
	config.PosGRPCPort = cast.ToString(GetOrReturnDefaultValue("POS_SERVICE_GRPC_PORT", ""))

	config.AnalyticsServiceHost = cast.ToString(GetOrReturnDefaultValue("ANALYTICS_SERVICE_HOST", ""))
	config.AnalyticsGRPCPort = cast.ToString(GetOrReturnDefaultValue("ANALYTICS_GRPC_PORT", ""))

	config.SmsServiceHost = cast.ToString(GetOrReturnDefaultValue("SMS_SERVICE_HOST", ""))
	config.SmsGRPCPort = cast.ToString(GetOrReturnDefaultValue("SMS_GRPC_PORT", ""))

	config.VersioningServiceHost = cast.ToString(GetOrReturnDefaultValue("VERSIONING_SERVICE_HOST", ""))
	config.VersioningGRPCPort = cast.ToString(GetOrReturnDefaultValue("VERSIONING_GRPC_PORT", ""))

	config.ApiReferenceServiceHost = cast.ToString(GetOrReturnDefaultValue("API_REF_SERVICE_HOST", ""))
	config.ApiReferenceServicePort = cast.ToString(GetOrReturnDefaultValue("API_REF_GRPC_PORT", ""))

	config.ConvertTemplateServiceGrpcHost = cast.ToString(GetOrReturnDefaultValue("CONVERT_TEMPLATE_SERVICE_HOST", ""))
	config.ConvertTemplateServiceGrpcPort = cast.ToString(GetOrReturnDefaultValue("CONVERT_TEMPLATE_GRPC_PORT", ""))

	config.FunctionServiceHost = cast.ToString(GetOrReturnDefaultValue("FUNCTION_SERVICE_HOST", ""))
	config.FunctionServicePort = cast.ToString(GetOrReturnDefaultValue("FUNCTION_GRPC_PORT", ""))

	config.GoObjectBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_SERVICE_HOST", "localhost"))
	config.GoObjectBuilderGRPCPort = cast.ToString(GetOrReturnDefaultValue("GO_OBJECT_BUILDER_GRPC_PORT", ":7107"))

	config.QueryServiceHost = cast.ToString(GetOrReturnDefaultValue("QUERY_SERVICE_HOST", ""))
	config.QueryServicePort = cast.ToString(GetOrReturnDefaultValue("QUERY_GRPC_PORT", ""))
	config.WebPageServiceHost = cast.ToString(GetOrReturnDefaultValue("WEB_PAGE_SERVICE_HOST", ""))
	config.WebPageServicePort = cast.ToString(GetOrReturnDefaultValue("WEB_PAGE_GRPC_PORT", ""))

	config.UcodeNamespace = "cp-region-type-id"
	config.SecretKey = ""

	config.ChatServiceGrpcHost = cast.ToString(GetOrReturnDefaultValue("CHAT_SERVICE_HOST", ""))
	config.ChatServiceGrpcPort = cast.ToString(GetOrReturnDefaultValue("CHAT_GRPC_PORT", ""))

	config.NotificationServiceHost = cast.ToString(GetOrReturnDefaultValue("NOTIFICATION_SERVICE_HOST", ""))
	config.NotificationGRPCPort = cast.ToString(GetOrReturnDefaultValue("NOTIFICATION_GRPC_PORT", ""))

	config.PostgresBuilderServiceHost = cast.ToString(GetOrReturnDefaultValue("NODE_POSTGRES_SERVICE_HOST", ""))
	config.PostgresBuilderServicePort = cast.ToString(GetOrReturnDefaultValue("NODE_POSTGRES_SERVICE_PORT", ""))

	config.GetRequestRedisHost = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_HOST", ""))
	config.GetRequestRedisPort = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PORT", ""))
	config.GetRequestRedisDatabase = cast.ToInt(GetOrReturnDefaultValue("GET_REQUEST_REDIS_DATABASE", 0))
	config.GetRequestRedisPassword = cast.ToString(GetOrReturnDefaultValue("GET_REQUEST_REDIS_PASSWORD", ""))

	return config
}

func GetOrReturnDefaultValue(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return defaultValue
}
