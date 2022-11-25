package models

type CreateProjectRequest struct {
	ProjectName        string `json:"project_name"`
	ProjectNamespace   string `json:"project_namespace"`
	MongoDBCredentials struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Database string `json:"database"`
	}
	ClickhouseCredentials struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Database string `json:"database"`
	}
	PostgresqlCredentials struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Database string `json:"database"`
	}
	MinioCredentials struct {
		Endpoint  string `json:"endpoint"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
		UseSecure bool   `json:"use_secure"`
	}
	CorporateService struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	ObjectBuilderService struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	AuthService struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	PosService struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	AnalyticsService struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
}

type CreateProjectResponse struct {
	ProjectName      string `json:"project_name"`
	ProjectNamespace string `json:"project_namespace"`
}
