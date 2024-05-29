package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
)

type ServiceManagerI interface {
	BuilderService() BuilderServiceI
	HighBuilderService() BuilderServiceI
	AuthService() AuthServiceI
	CompanyService() CompanyServiceI
	AnalyticsService() AnalyticsServiceI
	ApiReferenceService() ApiReferenceServiceI
	SmsService() SmsServiceI
	PosService() PosServiceI
	FunctionService() FunctionServiceI
	TemplateService() TemplateServiceI
	VersioningService() VersioningServiceI
	ScenarioService() ScenarioServiceI
	QueryService() QueryServiceI
	IntegrationPayzeService() IntegrationServicePayzeI
	WebPageService() WebPageServiceI
	ChatService() ChatServiceI
	NotificationService() NotificationServiceI
	PostgresBuilderService() PostgresBuilderServiceI
	ConvertTemplateService() ConvertTemplateServiceI
	GetBuilderServiceByType(nodeType string) BuilderServiceI
	GoObjectBuilderService() GoBuilderServiceI
}

type grpcClients struct {
	builderService           BuilderServiceI
	highBuilderService       BuilderServiceI
	authService              AuthServiceI
	companyService           CompanyServiceI
	analyticsService         AnalyticsServiceI
	apiReferenceService      ApiReferenceServiceI
	smsService               SmsServiceI
	posService               PosServiceI
	functionService          FunctionServiceI
	templateService          TemplateServiceI
	versioningService        VersioningServiceI
	scenarioService          ScenarioServiceI
	queryService             QueryServiceI
	integrationPayzeServiceI IntegrationServicePayzeI
	webPageService           WebPageServiceI
	chatService              ChatServiceI
	notificationService      NotificationServiceI
	postgresBuilderService   PostgresBuilderServiceI
	convertTemplateService   ConvertTemplateServiceI
	goObjectBuilderService   GoBuilderServiceI
}

func NewGrpcClients(ctx context.Context, cfg config.Config) (ServiceManagerI, error) {
	builderServiceClient, err := NewBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	highBuilderServiceClient, err := NewHighBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	authServiceClient, err := NewAuthServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	chatServiceClient, err := NewChatServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	companyServiceClient, err := NewCompanyServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	analyticsServiceClient, err := NewAnalyticsServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	apiReferenceClient, err := NewApiReferenceServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	smsServiceClient, err := NewSmsServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	posServiceClient, err := NewPosServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	functionServiceClient, err := NewFunctionServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	templateServiceClient, err := NewTemplateServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	queryServiceClient, err := NewQueryServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	versioningServiceClient, err := NewVersioningServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	scenarioServiceClient, err := NewScenarioServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	integrationPayzeServiceClient, err := NewPayzeServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	webPageServiceClient, err := NewWebPageServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	notificationServiceClient, err := NewNotificationServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	postgresBuilderServiceClient, err := NewPostgrespostgresBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	convertTemplateServiceClient, err := NewConvertTemplateServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	goObjectBuilderServiceClient, err := NewGoBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return grpcClients{
		apiReferenceService:      apiReferenceClient,
		analyticsService:         analyticsServiceClient,
		authService:              authServiceClient,
		builderService:           builderServiceClient,
		highBuilderService:       highBuilderServiceClient,
		posService:               posServiceClient,
		smsService:               smsServiceClient,
		companyService:           companyServiceClient,
		functionService:          functionServiceClient,
		templateService:          templateServiceClient,
		versioningService:        versioningServiceClient,
		scenarioService:          scenarioServiceClient,
		queryService:             queryServiceClient,
		integrationPayzeServiceI: integrationPayzeServiceClient,
		webPageService:           webPageServiceClient,
		chatService:              chatServiceClient,
		notificationService:      notificationServiceClient,
		postgresBuilderService:   postgresBuilderServiceClient,
		convertTemplateService:   convertTemplateServiceClient,
		goObjectBuilderService:   goObjectBuilderServiceClient,
	}, nil
}

func (g grpcClients) GetBuilderServiceByType(nodeType string) BuilderServiceI {
	switch nodeType {
	case config.LOW_NODE_TYPE:
		return g.builderService
	case config.HIGH_NODE_TYPE:
		return g.highBuilderService
	}

	return g.builderService
}

func (g grpcClients) BuilderService() BuilderServiceI {
	return g.builderService
}

func (g grpcClients) GoObjectBuilderService() GoBuilderServiceI {
	return g.goObjectBuilderService
}

func (g grpcClients) HighBuilderService() BuilderServiceI {
	return g.highBuilderService
}

func (g grpcClients) ChatService() ChatServiceI {
	return g.chatService
}

func (g grpcClients) AuthService() AuthServiceI {
	return g.authService
}

func (g grpcClients) CompanyService() CompanyServiceI {
	return g.companyService
}

func (g grpcClients) AnalyticsService() AnalyticsServiceI {
	return g.analyticsService
}

func (g grpcClients) ApiReferenceService() ApiReferenceServiceI {
	return g.apiReferenceService
}

func (g grpcClients) SmsService() SmsServiceI {
	return g.smsService
}

func (g grpcClients) PosService() PosServiceI {
	return g.posService
}

func (g grpcClients) FunctionService() FunctionServiceI {
	return g.functionService
}
func (g grpcClients) TemplateService() TemplateServiceI {
	return g.templateService
}

func (g grpcClients) VersioningService() VersioningServiceI {
	return g.versioningService
}

func (g grpcClients) ScenarioService() ScenarioServiceI {
	return g.scenarioService
}

func (g grpcClients) QueryService() QueryServiceI {
	return g.queryService
}

func (g grpcClients) IntegrationPayzeService() IntegrationServicePayzeI {
	return g.integrationPayzeServiceI
}

func (g grpcClients) WebPageService() WebPageServiceI {
	return g.webPageService
}

func (g grpcClients) NotificationService() NotificationServiceI {
	return g.notificationService
}

func (g grpcClients) PostgresBuilderService() PostgresBuilderServiceI {
	return g.postgresBuilderService
}

func (g grpcClients) ConvertTemplateService() ConvertTemplateServiceI {
	return g.convertTemplateService
}
