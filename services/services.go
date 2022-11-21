package services

import (
	"errors"
	"sync"
	"ucode/ucode_go_api_gateway/config"

	"ucode/ucode_go_api_gateway/genproto/analytics_service"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/corporate_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/genproto/pos_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceManagerI interface {
	CompanyService() corporate_service.CompanyServiceClient
	BranchService() corporate_service.BranchServiceClient
	RequisiteService() corporate_service.RequisiteServiceClient
	CategoryService() corporate_service.CategoryServiceClient
	SubcategoryService() corporate_service.SubcategoryServiceClient
	ProductService() corporate_service.ProductServiceClient
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
	CashboxTransactionService() object_builder_service.CashboxTransactionClient
	HtmlTemplateService() object_builder_service.HtmlTemplateServiceClient
	DocumentService() object_builder_service.DocumentServiceClient
	EventService() object_builder_service.EventServiceClient
	EventLogsService() object_builder_service.EventLogsServiceClient
	ExcelService() object_builder_service.ExcelServiceClient
	PermissionService() object_builder_service.PermissionServiceClient
	CustomEventService() object_builder_service.CustomEventServiceClient
	FunctionService() object_builder_service.FunctionServiceClient
	BarcodeService() object_builder_service.BarcodeServiceClient
}

type grpcClients struct {
	companyService            corporate_service.CompanyServiceClient
	branchService             corporate_service.BranchServiceClient
	requisiteService          corporate_service.RequisiteServiceClient
	categoryService           corporate_service.CategoryServiceClient
	subcategoryService        corporate_service.SubcategoryServiceClient
	productService            corporate_service.ProductServiceClient
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
	cashboxTransactionService object_builder_service.CashboxTransactionClient
	htmlTemplateService       object_builder_service.HtmlTemplateServiceClient
	documentService           object_builder_service.DocumentServiceClient
	eventService              object_builder_service.EventServiceClient
	eventLogsService          object_builder_service.EventLogsServiceClient
	excelService              object_builder_service.ExcelServiceClient
	permissionService         object_builder_service.PermissionServiceClient
	customEventService        object_builder_service.CustomEventServiceClient
	functionService           object_builder_service.FunctionServiceClient
	barcodeService            object_builder_service.BarcodeServiceClient
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

	connCorporateService, err := grpc.Dial(
		cfg.CorporateServiceHost+cfg.CorporateGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

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
	if err != nil {
		return nil, err
	}

	return &grpcClients{
		companyService:            corporate_service.NewCompanyServiceClient(connCorporateService),
		branchService:             corporate_service.NewBranchServiceClient(connCorporateService),
		requisiteService:          corporate_service.NewRequisiteServiceClient(connCorporateService),
		categoryService:           corporate_service.NewCategoryServiceClient(connCorporateService),
		subcategoryService:        corporate_service.NewSubcategoryServiceClient(connCorporateService),
		productService:            corporate_service.NewProductServiceClient(connCorporateService),
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
		cashboxTransactionService: object_builder_service.NewCashboxTransactionClient(connObjectBuilderService),
		htmlTemplateService:       object_builder_service.NewHtmlTemplateServiceClient(connObjectBuilderService),
		documentService:           object_builder_service.NewDocumentServiceClient(connObjectBuilderService),
		eventService:              object_builder_service.NewEventServiceClient(connObjectBuilderService),
		eventLogsService:          object_builder_service.NewEventLogsServiceClient(connObjectBuilderService),
		excelService:              object_builder_service.NewExcelServiceClient(connObjectBuilderService),
		permissionService:         object_builder_service.NewPermissionServiceClient(connObjectBuilderService),
		customEventService:        object_builder_service.NewCustomEventServiceClient(connObjectBuilderService),
		functionService:           object_builder_service.NewFunctionServiceClient((connObjectBuilderService)),
		barcodeService:            object_builder_service.NewBarcodeServiceClient((connObjectBuilderService)),
	}, nil
}

func (g *grpcClients) CompanyService() corporate_service.CompanyServiceClient {
	return g.companyService
}

func (g *grpcClients) BranchService() corporate_service.BranchServiceClient {
	return g.branchService
}

func (g *grpcClients) RequisiteService() corporate_service.RequisiteServiceClient {
	return g.requisiteService
}

func (g *grpcClients) CategoryService() corporate_service.CategoryServiceClient {
	return g.categoryService
}

func (g *grpcClients) SubcategoryService() corporate_service.SubcategoryServiceClient {
	return g.subcategoryService
}

func (g *grpcClients) ProductService() corporate_service.ProductServiceClient {
	return g.productService
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

func (g *grpcClients) CashboxTransactionService() object_builder_service.CashboxTransactionClient {
	return g.cashboxTransactionService
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
