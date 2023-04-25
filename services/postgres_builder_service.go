package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PostgresBuilderServiceI interface {
	Table() object_builder_service.TableServiceClient
	Field() object_builder_service.FieldServiceClient
	Relation() object_builder_service.RelationServiceClient
	App() object_builder_service.AppServiceClient
	Dashboard() object_builder_service.DashboardServiceClient
	Panel() object_builder_service.PanelServiceClient
	Variable() object_builder_service.VariableServiceClient
	Excel() object_builder_service.ExcelServiceClient
	Permission() object_builder_service.PermissionServiceClient
	CustomEvent() object_builder_service.CustomEventServiceClient
	Barcode() object_builder_service.BarcodeServiceClient
	Login() object_builder_service.LoginServiceClient
	Cascading() object_builder_service.CascadingServiceClient
	TableHelpers() object_builder_service.TableHelpersServiceClient
	FieldsAndRelations() object_builder_service.FieldAndRelationServiceClient
	Setting() object_builder_service.SettingServiceClient
	TableFolder() object_builder_service.TableFolderServiceClient
}

func NewPostgrespostgresBuilderServiceClient(ctx context.Context, cfg config.Config) (PostgresBuilderServiceI, error) {

	connObjectBuilderService, err := grpc.DialContext(
		ctx,
		cfg.PostgresBuilderServiceHost+cfg.PostgresBuilderServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(52428800), grpc.MaxCallSendMsgSize(52428800)),
	)
	if err != nil {
		return nil, err
	}

	return &postgresBuilderServiceClient{
		tableService:         object_builder_service.NewTableServiceClient(connObjectBuilderService),
		fieldService:         object_builder_service.NewFieldServiceClient(connObjectBuilderService),
		objectBuilderService: object_builder_service.NewObjectBuilderServiceClient(connObjectBuilderService),
		relationService:      object_builder_service.NewRelationServiceClient(connObjectBuilderService),
		dashboardService:     object_builder_service.NewDashboardServiceClient(connObjectBuilderService),
		variableService:      object_builder_service.NewVariableServiceClient(connObjectBuilderService),
		panelService:         object_builder_service.NewPanelServiceClient(connObjectBuilderService),
		appService:           object_builder_service.NewAppServiceClient(connObjectBuilderService),
		excelService:         object_builder_service.NewExcelServiceClient(connObjectBuilderService),
		permissionService:    object_builder_service.NewPermissionServiceClient(connObjectBuilderService),
		customEventService:   object_builder_service.NewCustomEventServiceClient(connObjectBuilderService),
		barcodeService:       object_builder_service.NewBarcodeServiceClient(connObjectBuilderService),
		loginService:         object_builder_service.NewLoginServiceClient(connObjectBuilderService),
		cascadingService:     object_builder_service.NewCascadingServiceClient(connObjectBuilderService),
		tableHelpersService:  object_builder_service.NewTableHelpersServiceClient(connObjectBuilderService),
		fieldsAndRelations:   object_builder_service.NewFieldAndRelationServiceClient(connObjectBuilderService),
		settingService:       object_builder_service.NewSettingServiceClient(connObjectBuilderService),
		tableFolderService:   object_builder_service.NewTableFolderServiceClient(connObjectBuilderService),
	}, nil
}

type postgresBuilderServiceClient struct {
	tableService             object_builder_service.TableServiceClient
	fieldService             object_builder_service.FieldServiceClient
	objectBuilderService     object_builder_service.ObjectBuilderServiceClient
	sectionService           object_builder_service.SectionServiceClient
	relationService          object_builder_service.RelationServiceClient
	viewService              object_builder_service.ViewServiceClient
	dashboardService         object_builder_service.DashboardServiceClient
	panelService             object_builder_service.PanelServiceClient
	variableService          object_builder_service.VariableServiceClient
	appService               object_builder_service.AppServiceClient
	excelService             object_builder_service.ExcelServiceClient
	permissionService        object_builder_service.PermissionServiceClient
	customEventService       object_builder_service.CustomEventServiceClient
	barcodeService           object_builder_service.BarcodeServiceClient
	objectBuilderServiceAuth object_builder_service.ObjectBuilderServiceClient
	loginService             object_builder_service.LoginServiceClient
	cascadingService         object_builder_service.CascadingServiceClient
	tableHelpersService      object_builder_service.TableHelpersServiceClient
	fieldsAndRelations       object_builder_service.FieldAndRelationServiceClient
	settingService           object_builder_service.SettingServiceClient
	tableFolderService       object_builder_service.TableFolderServiceClient
}

func (g *postgresBuilderServiceClient) Table() object_builder_service.TableServiceClient {
	return g.tableService
}

func (g *postgresBuilderServiceClient) Field() object_builder_service.FieldServiceClient {
	return g.fieldService
}

func (g *postgresBuilderServiceClient) Section() object_builder_service.SectionServiceClient {
	return g.sectionService
}

func (g *postgresBuilderServiceClient) Relation() object_builder_service.RelationServiceClient {
	return g.relationService
}

func (g *postgresBuilderServiceClient) View() object_builder_service.ViewServiceClient {
	return g.viewService
}

func (g *postgresBuilderServiceClient) App() object_builder_service.AppServiceClient {
	return g.appService
}

func (g *postgresBuilderServiceClient) Dashboard() object_builder_service.DashboardServiceClient {
	return g.dashboardService
}

func (g *postgresBuilderServiceClient) Variable() object_builder_service.VariableServiceClient {
	return g.variableService
}

func (g *postgresBuilderServiceClient) Panel() object_builder_service.PanelServiceClient {
	return g.panelService
}

func (g *postgresBuilderServiceClient) Excel() object_builder_service.ExcelServiceClient {
	return g.excelService
}
func (g *postgresBuilderServiceClient) Permission() object_builder_service.PermissionServiceClient {
	return g.permissionService
}

func (g *postgresBuilderServiceClient) CustomEvent() object_builder_service.CustomEventServiceClient {
	return g.customEventService
}

func (g *postgresBuilderServiceClient) Barcode() object_builder_service.BarcodeServiceClient {
	return g.barcodeService
}

func (g *postgresBuilderServiceClient) TableHelpers() object_builder_service.TableHelpersServiceClient {
	return g.tableHelpersService
}

func (g *postgresBuilderServiceClient) Login() object_builder_service.LoginServiceClient {
	return g.loginService
}

func (g *postgresBuilderServiceClient) Cascading() object_builder_service.CascadingServiceClient {
	return g.cascadingService
}

func (g *postgresBuilderServiceClient) FieldsAndRelations() object_builder_service.FieldAndRelationServiceClient {
	return g.fieldsAndRelations
}

func (g *postgresBuilderServiceClient) Setting() object_builder_service.SettingServiceClient {
	return g.settingService
}

func (g *postgresBuilderServiceClient) TableFolder() object_builder_service.TableFolderServiceClient {
	return g.tableFolderService
}
