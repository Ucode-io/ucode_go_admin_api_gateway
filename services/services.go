package services

import (
	"errors"
	"sync"
	"ucode/ucode_go_api_gateway/config"

	"ucode/ucode_go_api_gateway/genproto/analytics_service"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/genproto/pos_service"
	"ucode/ucode_go_api_gateway/genproto/sms_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceManagerI interface {
	TableService() object_builder_service.TableServiceClient
	FieldService() object_builder_service.FieldServiceClient
	ObjectBuilderService() object_builder_service.ObjectBuilderServiceClient
	SectionService() object_builder_service.SectionServiceClient
	RelationService() object_builder_service.RelationServiceClient
	ViewService() object_builder_service.ViewServiceClient
	ClientService() auth_service.ClientServiceClient
	SessionService() auth_service.SessionServiceClient
	AppService() object_builder_service.AppServiceClient
	DashboardService() object_builder_service.DashboardServiceClient
	PanelService() object_builder_service.PanelServiceClient
	VariableService() object_builder_service.VariableServiceClient
	OfflineAppointmentService() pos_service.OfflineAppointmentServiceClient
	BookedAppointmentService() pos_service.BookedAppointmentServiceClient
	QueryService() analytics_service.QueryServiceClient
	HtmlTemplateService() object_builder_service.HtmlTemplateServiceClient
	DocumentService() object_builder_service.DocumentServiceClient
	EventService() object_builder_service.EventServiceClient
	EventLogsService() object_builder_service.EventLogsServiceClient
	ExcelService() object_builder_service.ExcelServiceClient
	PermissionService() object_builder_service.PermissionServiceClient
	CustomEventService() object_builder_service.CustomEventServiceClient
	FunctionService() object_builder_service.FunctionServiceClient
	BarcodeService() object_builder_service.BarcodeServiceClient
	IntegrationService() auth_service.IntegrationServiceClient
	ClientServiceAuth() auth_service.ClientServiceClient
	PermissionServiceAuth() auth_service.PermissionServiceClient
	UserService() auth_service.UserServiceClient
	SessionServiceAuth() auth_service.SessionServiceClient
	ObjectBuilderServiceAuth() object_builder_service.ObjectBuilderServiceClient
	SmsService() sms_service.SmsServiceClient
	LoginService() object_builder_service.LoginServiceClient
	EmailServie() auth_service.EmailOtpServiceClient
	CompanyService() company_service.CompanyServiceClient
	ProjectService() company_service.ProjectServiceClient
	QueryFolderService() object_builder_service.QueryFolderServiceClient
	QueriesService() object_builder_service.QueryServiceClient
	WebPageService() object_builder_service.WebPageServiceClient
}

type grpcClients struct {
	tableService              object_builder_service.TableServiceClient
	fieldService              object_builder_service.FieldServiceClient
	objectBuilderService      object_builder_service.ObjectBuilderServiceClient
	sectionService            object_builder_service.SectionServiceClient
	relationService           object_builder_service.RelationServiceClient
	viewService               object_builder_service.ViewServiceClient
	dashboardService          object_builder_service.DashboardServiceClient
	panelService              object_builder_service.PanelServiceClient
	variableService           object_builder_service.VariableServiceClient
	clientService             auth_service.ClientServiceClient
	sessionService            auth_service.SessionServiceClient
	appService                object_builder_service.AppServiceClient
	offlineAppointmentService pos_service.OfflineAppointmentServiceClient
	bookedAppointmentService  pos_service.BookedAppointmentServiceClient
	queryService              analytics_service.QueryServiceClient
	htmlTemplateService       object_builder_service.HtmlTemplateServiceClient
	documentService           object_builder_service.DocumentServiceClient
	eventService              object_builder_service.EventServiceClient
	eventLogsService          object_builder_service.EventLogsServiceClient
	excelService              object_builder_service.ExcelServiceClient
	permissionService         object_builder_service.PermissionServiceClient
	customEventService        object_builder_service.CustomEventServiceClient
	functionService           object_builder_service.FunctionServiceClient
	barcodeService            object_builder_service.BarcodeServiceClient
	integrationService        auth_service.IntegrationServiceClient
	clientServiceAuth         auth_service.ClientServiceClient
	permissionServiceAuth     auth_service.PermissionServiceClient
	userService               auth_service.UserServiceClient
	sessionServiceAuth        auth_service.SessionServiceClient
	objectBuilderServiceAuth  object_builder_service.ObjectBuilderServiceClient
	smsService                sms_service.SmsServiceClient
	loginService              object_builder_service.LoginServiceClient
	emailServie               auth_service.EmailOtpServiceClient
	companyService            company_service.CompanyServiceClient
	projectService            company_service.ProjectServiceClient
	queryFolderService        object_builder_service.QueryFolderServiceClient
	queriesService            object_builder_service.QueryServiceClient
	webPageService            object_builder_service.WebPageServiceClient
}

type ProjectServices struct {
	Services map[string]ServiceManagerI
	Mu       sync.Mutex
}

func NewProjectGrpcsClient(p *ProjectServices, s ServiceManagerI, namespace string) (*ProjectServices, error) {
	if p == nil {
		return nil, errors.New("p cannot be nil (nil argument of *ProjectServices)")
	}
	if s == nil {
		return nil, errors.New("s cannot be nil (nil argument of ServiceManagerI)")
	}

	p.Mu.Lock()
	defer p.Mu.Unlock()

	_, ok := p.Services[namespace]
	if ok {
		return nil, errors.New("namespace already exists with this name")
	}
	p.Services[namespace] = s
	return p, nil
}

func NewGrpcClients(cfg config.Config) (ServiceManagerI, error) {

	connObjectBuilderService, err := grpc.Dial(
		cfg.ObjectBuilderServiceHost+cfg.ObjectBuilderGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	connAuthService, err := grpc.Dial(
		cfg.AuthServiceHost+cfg.AuthGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	connPosService, err := grpc.Dial(
		cfg.PosServiceHost+cfg.PosGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	connAnalyticsService, err := grpc.Dial(
		cfg.AnalyticsServiceHost+cfg.AnalyticsGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	connSmsService, err := grpc.Dial(
		cfg.SmsServiceHost+cfg.SmsGRPCPort,
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	connCompanyService, err := grpc.Dial(
		cfg.CompanyServiceHost+cfg.CompanyServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	return &grpcClients{
		tableService:              object_builder_service.NewTableServiceClient(connObjectBuilderService),
		fieldService:              object_builder_service.NewFieldServiceClient(connObjectBuilderService),
		objectBuilderService:      object_builder_service.NewObjectBuilderServiceClient(connObjectBuilderService),
		sectionService:            object_builder_service.NewSectionServiceClient(connObjectBuilderService),
		relationService:           object_builder_service.NewRelationServiceClient(connObjectBuilderService),
		viewService:               object_builder_service.NewViewServiceClient(connObjectBuilderService),
		dashboardService:          object_builder_service.NewDashboardServiceClient(connObjectBuilderService),
		variableService:           object_builder_service.NewVariableServiceClient(connObjectBuilderService),
		panelService:              object_builder_service.NewPanelServiceClient(connObjectBuilderService),
		clientService:             auth_service.NewClientServiceClient(connAuthService),
		sessionService:            auth_service.NewSessionServiceClient(connAuthService),
		appService:                object_builder_service.NewAppServiceClient(connObjectBuilderService),
		offlineAppointmentService: pos_service.NewOfflineAppointmentServiceClient(connPosService),
		bookedAppointmentService:  pos_service.NewBookedAppointmentServiceClient(connPosService),
		queryService:              analytics_service.NewQueryServiceClient(connAnalyticsService),
		htmlTemplateService:       object_builder_service.NewHtmlTemplateServiceClient(connObjectBuilderService),
		documentService:           object_builder_service.NewDocumentServiceClient(connObjectBuilderService),
		eventService:              object_builder_service.NewEventServiceClient(connObjectBuilderService),
		eventLogsService:          object_builder_service.NewEventLogsServiceClient(connObjectBuilderService),
		excelService:              object_builder_service.NewExcelServiceClient(connObjectBuilderService),
		permissionService:         object_builder_service.NewPermissionServiceClient(connObjectBuilderService),
		customEventService:        object_builder_service.NewCustomEventServiceClient(connObjectBuilderService),
		functionService:           object_builder_service.NewFunctionServiceClient((connObjectBuilderService)),
		barcodeService:            object_builder_service.NewBarcodeServiceClient((connObjectBuilderService)),
		clientServiceAuth:         auth_service.NewClientServiceClient(connAuthService),
		permissionServiceAuth:     auth_service.NewPermissionServiceClient(connAuthService),
		userService:               auth_service.NewUserServiceClient(connAuthService),
		sessionServiceAuth:        auth_service.NewSessionServiceClient(connAuthService),
		integrationService:        auth_service.NewIntegrationServiceClient(connAuthService),
		objectBuilderServiceAuth:  object_builder_service.NewObjectBuilderServiceClient(connObjectBuilderService),
		smsService:                sms_service.NewSmsServiceClient(connSmsService),
		loginService:              object_builder_service.NewLoginServiceClient(connObjectBuilderService),
		emailServie:               auth_service.NewEmailOtpServiceClient(connAuthService),
		companyService:            company_service.NewCompanyServiceClient(connCompanyService),
		projectService:            company_service.NewProjectServiceClient(connCompanyService),
		queryFolderService:        object_builder_service.NewQueryFolderServiceClient(connObjectBuilderService),
		queriesService:            object_builder_service.NewQueryServiceClient(connObjectBuilderService),
		webPageService:            object_builder_service.NewWebPageServiceClient(connObjectBuilderService),
	}, nil
}

func (g *grpcClients) TableService() object_builder_service.TableServiceClient {
	return g.tableService
}

func (g *grpcClients) FieldService() object_builder_service.FieldServiceClient {
	return g.fieldService
}

func (g *grpcClients) ObjectBuilderService() object_builder_service.ObjectBuilderServiceClient {
	return g.objectBuilderService
}

func (g *grpcClients) SectionService() object_builder_service.SectionServiceClient {
	return g.sectionService
}

func (g *grpcClients) RelationService() object_builder_service.RelationServiceClient {
	return g.relationService
}

func (g *grpcClients) ViewService() object_builder_service.ViewServiceClient {
	return g.viewService
}

func (g *grpcClients) ClientService() auth_service.ClientServiceClient {
	return g.clientService
}

func (g *grpcClients) SessionService() auth_service.SessionServiceClient {
	return g.sessionService
}

func (g *grpcClients) AppService() object_builder_service.AppServiceClient {
	return g.appService
}

func (g *grpcClients) DashboardService() object_builder_service.DashboardServiceClient {
	return g.dashboardService
}

func (g *grpcClients) VariableService() object_builder_service.VariableServiceClient {
	return g.variableService
}

func (g *grpcClients) PanelService() object_builder_service.PanelServiceClient {
	return g.panelService
}

func (g *grpcClients) OfflineAppointmentService() pos_service.OfflineAppointmentServiceClient {
	return g.offlineAppointmentService
}

func (g *grpcClients) BookedAppointmentService() pos_service.BookedAppointmentServiceClient {
	return g.bookedAppointmentService
}

func (g *grpcClients) QueryService() analytics_service.QueryServiceClient {
	return g.queryService
}

func (g *grpcClients) HtmlTemplateService() object_builder_service.HtmlTemplateServiceClient {
	return g.htmlTemplateService
}

func (g *grpcClients) DocumentService() object_builder_service.DocumentServiceClient {
	return g.documentService
}

func (g *grpcClients) EventService() object_builder_service.EventServiceClient {
	return g.eventService
}

func (g *grpcClients) EventLogsService() object_builder_service.EventLogsServiceClient {
	return g.eventLogsService
}

func (g *grpcClients) ExcelService() object_builder_service.ExcelServiceClient {
	return g.excelService
}
func (g *grpcClients) PermissionService() object_builder_service.PermissionServiceClient {
	return g.permissionService
}

func (g *grpcClients) CustomEventService() object_builder_service.CustomEventServiceClient {
	return g.customEventService
}

func (g *grpcClients) FunctionService() object_builder_service.FunctionServiceClient {
	return g.functionService
}

func (g *grpcClients) BarcodeService() object_builder_service.BarcodeServiceClient {
	return g.barcodeService
}

// auth functions

func (g *grpcClients) ClientServiceAuth() auth_service.ClientServiceClient {
	return g.clientServiceAuth
}

func (g *grpcClients) PermissionServiceAuth() auth_service.PermissionServiceClient {
	return g.permissionServiceAuth
}

func (g *grpcClients) UserService() auth_service.UserServiceClient {
	return g.userService
}

func (g *grpcClients) SessionServiceAuth() auth_service.SessionServiceClient {
	return g.sessionServiceAuth
}

func (g *grpcClients) IntegrationService() auth_service.IntegrationServiceClient {
	return g.integrationService
}

func (g *grpcClients) ObjectBuilderServiceAuth() object_builder_service.ObjectBuilderServiceClient {
	return g.objectBuilderServiceAuth
}

func (g *grpcClients) SmsService() sms_service.SmsServiceClient {
	return g.smsService
}

func (g *grpcClients) LoginService() object_builder_service.LoginServiceClient {
	return g.loginService
}

func (g *grpcClients) EmailServie() auth_service.EmailOtpServiceClient {
	return g.emailServie
}

// this functions for multi company logic

func (g *grpcClients) CompanyService() company_service.CompanyServiceClient {
	return g.companyService
}

func (g *grpcClients) ProjectService() company_service.ProjectServiceClient {
	return g.projectService
}

// for ucode version 2

func (g *grpcClients) QueryFolderService() object_builder_service.QueryFolderServiceClient {
	return g.queryFolderService
}

func (g *grpcClients) QueriesService() object_builder_service.QueryServiceClient {
	return g.queriesService
}

func (g *grpcClients) WebPageService() object_builder_service.WebPageServiceClient {
	return g.webPageService
}
