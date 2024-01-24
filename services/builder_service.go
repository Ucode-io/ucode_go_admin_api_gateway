package services

import (
	"context"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"

	grpcpool "github.com/processout/grpc-go-pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BuilderServiceI interface {
	Table() object_builder_service.TableServiceClient
	TableFolder() object_builder_service.TableFolderServiceClient
	Field() object_builder_service.FieldServiceClient
	ObjectBuilder() object_builder_service.ObjectBuilderServiceClient
	Section() object_builder_service.SectionServiceClient
	Layout() object_builder_service.LayoutServiceClient
	Relation() object_builder_service.RelationServiceClient
	View() object_builder_service.ViewServiceClient
	App() object_builder_service.AppServiceClient
	Dashboard() object_builder_service.DashboardServiceClient
	Panel() object_builder_service.PanelServiceClient
	Variable() object_builder_service.VariableServiceClient
	HtmlTemplate() object_builder_service.HtmlTemplateServiceClient
	Document() object_builder_service.DocumentServiceClient
	Event() object_builder_service.EventServiceClient
	EventLogs() object_builder_service.EventLogsServiceClient
	Excel() object_builder_service.ExcelServiceClient
	Permission() object_builder_service.PermissionServiceClient
	CustomEvent() object_builder_service.CustomEventServiceClient
	Function() object_builder_service.FunctionServiceClient
	Barcode() object_builder_service.BarcodeServiceClient
	Login() object_builder_service.LoginServiceClient
	ObjectBuilderAuth() object_builder_service.ObjectBuilderServiceClient
	QueryFolder() object_builder_service.QueryFolderServiceClient
	Queries() object_builder_service.QueryServiceClient
	WebPage() object_builder_service.WebPageServiceClient
	Cascading() object_builder_service.CascadingServiceClient
	TableHelpers() object_builder_service.TableHelpersServiceClient
	FieldsAndRelations() object_builder_service.FieldAndRelationServiceClient
	Setting() object_builder_service.SettingServiceClient
	Menu() object_builder_service.MenuServiceClient
	CustomErrorMessage() object_builder_service.CustomErrorMessageServiceClient
	ReportSetting() object_builder_service.ReportSettingServiceClient
	ItemsService() object_builder_service.ItemsServiceClient
	File() object_builder_service.FileServiceClient
	// added to managing load
	ObjectBuilderConnPool(ctx context.Context) (object_builder_service.ObjectBuilderServiceClient, *grpcpool.ClientConn, error)
}

func NewBuilderServiceClient(ctx context.Context, cfg config.Config) (BuilderServiceI, error) {
	connObjectBuilderService, err := grpc.DialContext(
		ctx,
		cfg.ObjectBuilderServiceHost+cfg.ObjectBuilderGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(52428800), grpc.MaxCallSendMsgSize(52428800)),
	)
	if err != nil {
		return nil, err
	}

	factory := func() (*grpc.ClientConn, error) {
		conn, err := grpc.Dial(
			cfg.ObjectBuilderServiceHost+cfg.ObjectBuilderGRPCPort,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(52428800), grpc.MaxCallSendMsgSize(52428800)))
		if err != nil {
			return nil, err
		}
		return conn, err
	}

	objectbuilderServicePool, err := grpcpool.New(factory, 12, 18, time.Second*2)
	if err != nil {
		fmt.Printf("\n\n\n\n\n Failed to create gRPC pool: %v\n", err)
		return nil, err
	}

	return &builderServiceClient{
		tableService:              object_builder_service.NewTableServiceClient(connObjectBuilderService),
		tableFolderService:        object_builder_service.NewTableFolderServiceClient(connObjectBuilderService),
		fieldService:              object_builder_service.NewFieldServiceClient(connObjectBuilderService),
		objectBuilderService:      object_builder_service.NewObjectBuilderServiceClient(connObjectBuilderService),
		sectionService:            object_builder_service.NewSectionServiceClient(connObjectBuilderService),
		relationService:           object_builder_service.NewRelationServiceClient(connObjectBuilderService),
		viewService:               object_builder_service.NewViewServiceClient(connObjectBuilderService),
		dashboardService:          object_builder_service.NewDashboardServiceClient(connObjectBuilderService),
		variableService:           object_builder_service.NewVariableServiceClient(connObjectBuilderService),
		panelService:              object_builder_service.NewPanelServiceClient(connObjectBuilderService),
		appService:                object_builder_service.NewAppServiceClient(connObjectBuilderService),
		htmlTemplateService:       object_builder_service.NewHtmlTemplateServiceClient(connObjectBuilderService),
		documentService:           object_builder_service.NewDocumentServiceClient(connObjectBuilderService),
		eventService:              object_builder_service.NewEventServiceClient(connObjectBuilderService),
		eventLogsService:          object_builder_service.NewEventLogsServiceClient(connObjectBuilderService),
		excelService:              object_builder_service.NewExcelServiceClient(connObjectBuilderService),
		permissionService:         object_builder_service.NewPermissionServiceClient(connObjectBuilderService),
		customEventService:        object_builder_service.NewCustomEventServiceClient(connObjectBuilderService),
		functionService:           object_builder_service.NewFunctionServiceClient(connObjectBuilderService),
		barcodeService:            object_builder_service.NewBarcodeServiceClient(connObjectBuilderService),
		objectBuilderServiceAuth:  object_builder_service.NewObjectBuilderServiceClient(connObjectBuilderService),
		loginService:              object_builder_service.NewLoginServiceClient(connObjectBuilderService),
		queryFolderService:        object_builder_service.NewQueryFolderServiceClient(connObjectBuilderService),
		queriesService:            object_builder_service.NewQueryServiceClient(connObjectBuilderService),
		webPageService:            object_builder_service.NewWebPageServiceClient(connObjectBuilderService),
		cascadingService:          object_builder_service.NewCascadingServiceClient(connObjectBuilderService),
		tableHelpersService:       object_builder_service.NewTableHelpersServiceClient(connObjectBuilderService),
		fieldsAndRelations:        object_builder_service.NewFieldAndRelationServiceClient(connObjectBuilderService),
		settingService:            object_builder_service.NewSettingServiceClient(connObjectBuilderService),
		layoutService:             object_builder_service.NewLayoutServiceClient(connObjectBuilderService),
		menuService:               object_builder_service.NewMenuServiceClient(connObjectBuilderService),
		customErrorMessageService: object_builder_service.NewCustomErrorMessageServiceClient(connObjectBuilderService),
		reportSettingService:      object_builder_service.NewReportSettingServiceClient(connObjectBuilderService),
		fileService:               object_builder_service.NewFileServiceClient(connObjectBuilderService),
		objectBuilderConnPool:     objectbuilderServicePool,
		itemsService:              object_builder_service.NewItemsServiceClient(connObjectBuilderService),
	}, nil
}

type builderServiceClient struct {
	tableService              object_builder_service.TableServiceClient
	tableFolderService        object_builder_service.TableFolderServiceClient
	fieldService              object_builder_service.FieldServiceClient
	objectBuilderService      object_builder_service.ObjectBuilderServiceClient
	sectionService            object_builder_service.SectionServiceClient
	relationService           object_builder_service.RelationServiceClient
	viewService               object_builder_service.ViewServiceClient
	dashboardService          object_builder_service.DashboardServiceClient
	panelService              object_builder_service.PanelServiceClient
	variableService           object_builder_service.VariableServiceClient
	appService                object_builder_service.AppServiceClient
	htmlTemplateService       object_builder_service.HtmlTemplateServiceClient
	documentService           object_builder_service.DocumentServiceClient
	eventService              object_builder_service.EventServiceClient
	eventLogsService          object_builder_service.EventLogsServiceClient
	excelService              object_builder_service.ExcelServiceClient
	permissionService         object_builder_service.PermissionServiceClient
	customEventService        object_builder_service.CustomEventServiceClient
	functionService           object_builder_service.FunctionServiceClient
	barcodeService            object_builder_service.BarcodeServiceClient
	objectBuilderServiceAuth  object_builder_service.ObjectBuilderServiceClient
	loginService              object_builder_service.LoginServiceClient
	queryFolderService        object_builder_service.QueryFolderServiceClient
	queriesService            object_builder_service.QueryServiceClient
	webPageService            object_builder_service.WebPageServiceClient
	cascadingService          object_builder_service.CascadingServiceClient
	tableHelpersService       object_builder_service.TableHelpersServiceClient
	fieldsAndRelations        object_builder_service.FieldAndRelationServiceClient
	settingService            object_builder_service.SettingServiceClient
	layoutService             object_builder_service.LayoutServiceClient
	menuService               object_builder_service.MenuServiceClient
	customErrorMessageService object_builder_service.CustomErrorMessageServiceClient
	reportSettingService      object_builder_service.ReportSettingServiceClient
	fileService               object_builder_service.FileServiceClient
	objectBuilderConnPool     *grpcpool.Pool
	itemsService              object_builder_service.ItemsServiceClient
}

func (g *builderServiceClient) ObjectBuilderConnPool(ctx context.Context) (object_builder_service.ObjectBuilderServiceClient, *grpcpool.ClientConn, error) {
	conn, err := g.objectBuilderConnPool.Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	service := object_builder_service.NewObjectBuilderServiceClient(conn)

	return service, conn, nil
}

func (g *builderServiceClient) Table() object_builder_service.TableServiceClient {
	return g.tableService
}

func (g *builderServiceClient) TableFolder() object_builder_service.TableFolderServiceClient {
	return g.tableFolderService
}

func (g *builderServiceClient) Field() object_builder_service.FieldServiceClient {
	return g.fieldService
}

func (g *builderServiceClient) ObjectBuilder() object_builder_service.ObjectBuilderServiceClient {
	return g.objectBuilderService
}

func (g *builderServiceClient) Section() object_builder_service.SectionServiceClient {
	return g.sectionService
}

func (g *builderServiceClient) Layout() object_builder_service.LayoutServiceClient {
	return g.layoutService
}

func (g *builderServiceClient) Relation() object_builder_service.RelationServiceClient {
	return g.relationService
}

func (g *builderServiceClient) View() object_builder_service.ViewServiceClient {
	return g.viewService
}

func (g *builderServiceClient) App() object_builder_service.AppServiceClient {
	return g.appService
}

func (g *builderServiceClient) Dashboard() object_builder_service.DashboardServiceClient {
	return g.dashboardService
}

func (g *builderServiceClient) Variable() object_builder_service.VariableServiceClient {
	return g.variableService
}

func (g *builderServiceClient) Panel() object_builder_service.PanelServiceClient {
	return g.panelService
}

func (g *builderServiceClient) HtmlTemplate() object_builder_service.HtmlTemplateServiceClient {
	return g.htmlTemplateService
}

func (g *builderServiceClient) Document() object_builder_service.DocumentServiceClient {
	return g.documentService
}

func (g *builderServiceClient) Event() object_builder_service.EventServiceClient {
	return g.eventService
}

func (g *builderServiceClient) EventLogs() object_builder_service.EventLogsServiceClient {
	return g.eventLogsService
}

func (g *builderServiceClient) Excel() object_builder_service.ExcelServiceClient {
	return g.excelService
}
func (g *builderServiceClient) Permission() object_builder_service.PermissionServiceClient {
	return g.permissionService
}

func (g *builderServiceClient) CustomEvent() object_builder_service.CustomEventServiceClient {
	return g.customEventService
}

func (g *builderServiceClient) Function() object_builder_service.FunctionServiceClient {
	return g.functionService
}

func (g *builderServiceClient) Barcode() object_builder_service.BarcodeServiceClient {
	return g.barcodeService
}

func (g *builderServiceClient) TableHelpers() object_builder_service.TableHelpersServiceClient {
	return g.tableHelpersService
}

func (g *builderServiceClient) ObjectBuilderAuth() object_builder_service.ObjectBuilderServiceClient {
	return g.objectBuilderServiceAuth
}

func (g *builderServiceClient) Login() object_builder_service.LoginServiceClient {
	return g.loginService
}

func (g *builderServiceClient) QueryFolder() object_builder_service.QueryFolderServiceClient {
	return g.queryFolderService
}

func (g *builderServiceClient) Queries() object_builder_service.QueryServiceClient {
	return g.queriesService
}

func (g *builderServiceClient) WebPage() object_builder_service.WebPageServiceClient {
	return g.webPageService
}

func (g *builderServiceClient) Cascading() object_builder_service.CascadingServiceClient {
	return g.cascadingService
}

func (g *builderServiceClient) FieldsAndRelations() object_builder_service.FieldAndRelationServiceClient {
	return g.fieldsAndRelations
}

func (g *builderServiceClient) Setting() object_builder_service.SettingServiceClient {
	return g.settingService
}

func (g *builderServiceClient) Menu() object_builder_service.MenuServiceClient {
	return g.menuService
}
func (g *builderServiceClient) CustomErrorMessage() object_builder_service.CustomErrorMessageServiceClient {
	return g.customErrorMessageService
}

func (g *builderServiceClient) ReportSetting() object_builder_service.ReportSettingServiceClient {
	return g.reportSettingService
}

func (g *builderServiceClient) File() object_builder_service.FileServiceClient {
	return g.fileService
}

func (g *builderServiceClient) ItemsService() object_builder_service.ItemsServiceClient {
	return g.itemsService
}
