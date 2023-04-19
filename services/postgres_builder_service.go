package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/postgres_object_builder_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PostgresBuilderServiceI interface {
	Table() postgres_object_builder_service.TableServiceClient
	Field() postgres_object_builder_service.FieldServiceClient
	Relation() postgres_object_builder_service.RelationServiceClient
	App() postgres_object_builder_service.AppServiceClient
	Dashboard() postgres_object_builder_service.DashboardServiceClient
	Panel() postgres_object_builder_service.PanelServiceClient
	Variable() postgres_object_builder_service.VariableServiceClient
	Excel() postgres_object_builder_service.ExcelServiceClient
	Permission() postgres_object_builder_service.PermissionServiceClient
	CustomEvent() postgres_object_builder_service.CustomEventServiceClient
	Barcode() postgres_object_builder_service.BarcodeServiceClient
	Login() postgres_object_builder_service.LoginServiceClient
	Cascading() postgres_object_builder_service.CascadingServiceClient
	TableHelpers() postgres_object_builder_service.TableHelpersServiceClient
	FieldsAndRelations() postgres_object_builder_service.FieldAndRelationServiceClient
	Setting() postgres_object_builder_service.SettingServiceClient
	TableFolder() postgres_object_builder_service.TableFolderServiceClient
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
		tableService:         postgres_object_builder_service.NewTableServiceClient(connObjectBuilderService),
		fieldService:         postgres_object_builder_service.NewFieldServiceClient(connObjectBuilderService),
		objectBuilderService: postgres_object_builder_service.NewObjectBuilderServiceClient(connObjectBuilderService),
		relationService:      postgres_object_builder_service.NewRelationServiceClient(connObjectBuilderService),
		dashboardService:     postgres_object_builder_service.NewDashboardServiceClient(connObjectBuilderService),
		variableService:      postgres_object_builder_service.NewVariableServiceClient(connObjectBuilderService),
		panelService:         postgres_object_builder_service.NewPanelServiceClient(connObjectBuilderService),
		appService:           postgres_object_builder_service.NewAppServiceClient(connObjectBuilderService),
		excelService:         postgres_object_builder_service.NewExcelServiceClient(connObjectBuilderService),
		permissionService:    postgres_object_builder_service.NewPermissionServiceClient(connObjectBuilderService),
		customEventService:   postgres_object_builder_service.NewCustomEventServiceClient(connObjectBuilderService),
		barcodeService:       postgres_object_builder_service.NewBarcodeServiceClient(connObjectBuilderService),
		loginService:         postgres_object_builder_service.NewLoginServiceClient(connObjectBuilderService),
		cascadingService:     postgres_object_builder_service.NewCascadingServiceClient(connObjectBuilderService),
		tableHelpersService:  postgres_object_builder_service.NewTableHelpersServiceClient(connObjectBuilderService),
		fieldsAndRelations:   postgres_object_builder_service.NewFieldAndRelationServiceClient(connObjectBuilderService),
		settingService:       postgres_object_builder_service.NewSettingServiceClient(connObjectBuilderService),
		tableFolderService:   postgres_object_builder_service.NewTableFolderServiceClient(connObjectBuilderService),
	}, nil
}

type postgresBuilderServiceClient struct {
	tableService             postgres_object_builder_service.TableServiceClient
	fieldService             postgres_object_builder_service.FieldServiceClient
	objectBuilderService     postgres_object_builder_service.ObjectBuilderServiceClient
	sectionService           postgres_object_builder_service.SectionServiceClient
	relationService          postgres_object_builder_service.RelationServiceClient
	viewService              postgres_object_builder_service.ViewServiceClient
	dashboardService         postgres_object_builder_service.DashboardServiceClient
	panelService             postgres_object_builder_service.PanelServiceClient
	variableService          postgres_object_builder_service.VariableServiceClient
	appService               postgres_object_builder_service.AppServiceClient
	excelService             postgres_object_builder_service.ExcelServiceClient
	permissionService        postgres_object_builder_service.PermissionServiceClient
	customEventService       postgres_object_builder_service.CustomEventServiceClient
	barcodeService           postgres_object_builder_service.BarcodeServiceClient
	objectBuilderServiceAuth postgres_object_builder_service.ObjectBuilderServiceClient
	loginService             postgres_object_builder_service.LoginServiceClient
	cascadingService         postgres_object_builder_service.CascadingServiceClient
	tableHelpersService      postgres_object_builder_service.TableHelpersServiceClient
	fieldsAndRelations       postgres_object_builder_service.FieldAndRelationServiceClient
	settingService           postgres_object_builder_service.SettingServiceClient
	tableFolderService       postgres_object_builder_service.TableFolderServiceClient
}

func (g *postgresBuilderServiceClient) Table() postgres_object_builder_service.TableServiceClient {
	return g.tableService
}

func (g *postgresBuilderServiceClient) Field() postgres_object_builder_service.FieldServiceClient {
	return g.fieldService
}

func (g *postgresBuilderServiceClient) Section() postgres_object_builder_service.SectionServiceClient {
	return g.sectionService
}

func (g *postgresBuilderServiceClient) Relation() postgres_object_builder_service.RelationServiceClient {
	return g.relationService
}

func (g *postgresBuilderServiceClient) View() postgres_object_builder_service.ViewServiceClient {
	return g.viewService
}

func (g *postgresBuilderServiceClient) App() postgres_object_builder_service.AppServiceClient {
	return g.appService
}

func (g *postgresBuilderServiceClient) Dashboard() postgres_object_builder_service.DashboardServiceClient {
	return g.dashboardService
}

func (g *postgresBuilderServiceClient) Variable() postgres_object_builder_service.VariableServiceClient {
	return g.variableService
}

func (g *postgresBuilderServiceClient) Panel() postgres_object_builder_service.PanelServiceClient {
	return g.panelService
}

func (g *postgresBuilderServiceClient) Excel() postgres_object_builder_service.ExcelServiceClient {
	return g.excelService
}
func (g *postgresBuilderServiceClient) Permission() postgres_object_builder_service.PermissionServiceClient {
	return g.permissionService
}

func (g *postgresBuilderServiceClient) CustomEvent() postgres_object_builder_service.CustomEventServiceClient {
	return g.customEventService
}

func (g *postgresBuilderServiceClient) Barcode() postgres_object_builder_service.BarcodeServiceClient {
	return g.barcodeService
}

func (g *postgresBuilderServiceClient) TableHelpers() postgres_object_builder_service.TableHelpersServiceClient {
	return g.tableHelpersService
}

func (g *postgresBuilderServiceClient) Login() postgres_object_builder_service.LoginServiceClient {
	return g.loginService
}

func (g *postgresBuilderServiceClient) Cascading() postgres_object_builder_service.CascadingServiceClient {
	return g.cascadingService
}

func (g *postgresBuilderServiceClient) FieldsAndRelations() postgres_object_builder_service.FieldAndRelationServiceClient {
	return g.fieldsAndRelations
}

func (g *postgresBuilderServiceClient) Setting() postgres_object_builder_service.SettingServiceClient {
	return g.settingService
}

func (g *postgresBuilderServiceClient) TableFolder() postgres_object_builder_service.TableFolderServiceClient {
	return g.tableFolderService
}
