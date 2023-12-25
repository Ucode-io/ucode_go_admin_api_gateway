package api

import (
	"encoding/json"
	"errors"
	"strings"
	"ucode/ucode_go_api_gateway/api/docs"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// SetUpAPI @description This is an api gateway
// @termsOfService https://udevs.io
func SetUpAPI(r *gin.Engine, h handlers.Handler, cfg config.BaseConfig) {
	docs.SwaggerInfo.Title = cfg.ServiceName
	docs.SwaggerInfo.Version = cfg.Version
	// docs.SwaggerInfo.Host = cfg.ServiceHost + cfg.HTTPPort
	docs.SwaggerInfo.Schemes = []string{cfg.HTTPScheme}

	r.Use(customCORSMiddleware())
	// r.Use(MaxAllowed(5000))
	// r.Use(h.NodeMiddleware())

	r.GET("/ping", h.V1.Ping)
	// r.GET("/config", h.GetConfig)

	r.Any("/x-api/*any", h.V1.RedirectAuthMiddleware(cfg), proxyMiddleware(r, &h), h.V1.Proxy)

	r.POST("/send-code", h.V1.SendCode)
	r.POST("/verify/:sms_id/:otp", h.V1.Verify)
	r.POST("/register-otp/:table_slug", h.V1.RegisterOtp)
	r.POST("/send-message", h.V1.SendMessageToEmail)
	r.POST("/verify-email/:sms_id/:otp", h.V1.VerifyEmail)
	r.POST("/register-email-otp/:table_slug", h.V1.RegisterEmailOtp)
	r.GET("/v1/login-microfront", h.V1.GetLoginMicroFrontBySubdomain)
	r.GET("/v3/note/:note-id", h.V1.GetSingleNoteWithoutToken)

	r.GET("/menu/wiki_folder", h.V1.GetWikiFolder)

	global := r.Group("/v1/global")
	global.Use(h.V1.GlobalAuthMiddleware(cfg))
	{
		global.GET("/projects", h.V1.GetGlobalCompanyProjectList)
		global.GET("/environment", h.V1.GetGlobalProjectEnvironments)
		global.GET("/template", h.V1.GetGlobalProjectTemplate)
	}

	v1 := r.Group("/v1")
	// @securityDefinitions.apikey ApiKeyAuth
	// @in header
	// @name Authorization
	v1.Use(h.V1.AuthMiddleware(cfg))
	{
		v1.POST("/menu-settings", h.V1.CreateMenuSettings)
		v1.PUT("/menu-settings", h.V1.UpdateMenuSettings)
		v1.GET("/menu-settings", h.V1.GetAllMenuSettings)
		v1.GET("/menu-settings/:id", h.V1.GetMenuSettingByID)
		v1.DELETE("/menu-settings/:id", h.V1.DeleteMenuSettings)

		v1.POST("/menu-template", h.V1.CreateMenuTemplate)
		v1.PUT("/menu-template", h.V1.UpdateMenuTemplate)
		v1.GET("/menu-template", h.V1.GetAllMenuTemplates)
		v1.GET("/menu-template/:id", h.V1.GetMenuTemplateByID)
		v1.DELETE("/menu-template/:id", h.V1.DeleteMenuTemplate)

		v1.POST("/upload", h.V1.Upload)
		v1.POST("/upload-template/:template_name", h.V1.UploadTemplate)
		v1.POST("/upload-file/:table_slug/:object_id", h.V1.UploadFile)

		// OBJECT_BUILDER_SERVICE

		//table
		v1.POST("/table", h.V1.CreateTable)
		v1.GET("/table", h.V1.GetAllTables)
		v1.GET("/table/:table_id", h.V1.GetTableByID)
		v1.POST("/table-details/:table_slug", h.V1.GetTableDetails)

		v1.PUT("/table", h.V1.UpdateTable)
		v1.DELETE("/table/:table_id", h.V1.DeleteTable)
		//field
		v1.POST("/field", h.V1.CreateField)
		v1.GET("/field", h.V1.GetAllFields)
		v1.PUT("/field", h.V1.UpdateField)
		v1.DELETE("/field/:field_id", h.V1.DeleteField)
		v1.POST("/fields-relations", h.V1.CreateFieldsAndRelations)

		//relation
		v1.POST("/relation", h.V1.CreateRelation)
		v1.GET("/relation", h.V1.GetAllRelations)
		v1.PUT("/relation", h.V1.UpdateRelation)
		v1.DELETE("/relation/:relation_id", h.V1.DeleteRelation)
		v1.GET("/get-relation-cascading/:table_slug", h.V1.GetRelationCascaders)

		//section
		v1.GET("/section", h.V1.GetAllSections)
		v1.PUT("/section", h.V1.UpdateSection)

		//view_relation
		v1.GET("/view_relation", h.V1.GetViewRelation)
		v1.PUT("/view_relation", h.V1.UpsertViewRelations)

		//object-builder
		v1.POST("/object/:table_slug", h.V1.CreateObject)
		v1.GET("/object/:table_slug/:object_id", h.V1.GetSingle)
		v1.POST("/object/get-list/:table_slug", h.V1.GetList)
		v1.GET("/object-slim/:table_slug/:object_id", h.V1.GetSingleSlim)
		v1.GET("/object-slim/get-list/:table_slug", h.V1.GetListSlim)
		v1.PUT("/object/:table_slug", h.V1.UpdateObject)
		v1.DELETE("/object/:table_slug/:object_id", h.V1.DeleteObject)
		v1.DELETE("/object/:table_slug", h.V1.DeleteManyObject)
		v1.POST("/object/excel/:table_slug", h.V1.GetListInExcel)
		v1.POST("/object-upsert/:table_slug", h.V1.UpsertObject)
		v1.PUT("/object/multiple-update/:table_slug", h.V1.MultipleUpdateObject)
		v1.POST("/object/get-financial-analytics/:table_slug", h.V1.GetFinancialAnalytics)
		v1.POST("/object/get-list-group-by/:table_slug/:column_table_slug", h.V1.GetListGroupBy)
		v1.POST("/object/get-group-by-field/:table_slug", h.V1.GetGroupByField)
		v1.POST("/object/get-list-aggregate/:table_slug", h.V1.GetListAggregate)
		v1.POST("/object/get-list-without-relation/:table_slug", h.V1.GetListWithOutRelation)

		// permission
		v1.POST("/permission-upsert/:app_id", h.V1.UpsertPermissionsByAppId)
		v1.GET("/permission-get-all/:role_id", h.V1.GetAllPermissionByRoleId)
		v1.GET("/field-permission/:role_id/:table_slug", h.V1.GetFieldPermissions)
		v1.GET("/action-permission/:role_id/:table_slug", h.V1.GetActionPermissions)
		v1.GET("/view-relation-permission/:role_id/:table_slug", h.V1.GetViewRelationPermissions)

		//many-to-many
		v1.PUT("/many-to-many", h.V1.AppendManyToMany)
		v1.DELETE("/many-to-many", h.V1.DeleteManyToMany)

		//view
		v1.POST("/view", h.V1.CreateView)
		v1.GET("/view/:view_id", h.V1.GetSingleView)
		v1.GET("/view", h.V1.GetViewList)
		v1.PUT("/view", h.V1.UpdateView)
		v1.DELETE("/view/:view_id", h.V1.DeleteView)
		v1.PUT("/update-view-order", h.V1.UpdateViewOrder)

		//anaytics dashboard
		v1.POST("/analytics/dashboard", h.V1.CreateDashboard)
		v1.GET("/analytics/dashboard/:dashboard_id", h.V1.GetSingleDashboard)
		v1.GET("/analytics/dashboard", h.V1.GetAllDashboards)
		v1.PUT("/analytics/dashboard", h.V1.UpdateDashboard)
		v1.DELETE("/analytics/dashboard/:dashboard_id", h.V1.DeleteDashboard)

		//anaytics variable
		v1.POST("/analytics/variable", h.V1.CreateVariable)
		v1.GET("/analytics/variable/:variable_id", h.V1.GetSingleVariable)
		v1.GET("/analytics/variable", h.V1.GetAllVariables)
		v1.PUT("/analytics/variable", h.V1.UpdateVariable)
		v1.DELETE("/analytics/variable/:variable_id", h.V1.DeleteVariable)

		//anaytics panel
		v1.POST("/analytics/panel/updateCoordinates", h.V1.UpdateCoordinates)
		v1.POST("/analytics/panel", h.V1.CreatePanel)
		v1.GET("/analytics/panel/:panel_id", h.V1.GetSinglePanel)
		v1.GET("/analytics/panel", h.V1.GetAllPanels)
		v1.PUT("/analytics/panel", h.V1.UpdatePanel)
		v1.DELETE("/analytics/panel/:panel_id", h.V1.DeletePanel)

		//app
		v1.POST("/app", h.V1.CreateApp)
		v1.GET("/app/:app_id", h.V1.GetAppByID)
		v1.GET("/app", h.V1.GetAllApps)
		v1.PUT("/app", h.V1.UpdateApp)
		v1.DELETE("/app/:app_id", h.V1.DeleteApp)

		// POS_SERVICE

		//appointments
		v1.GET("/offline_appointment", h.V1.GetAllOfflineAppointments)
		v1.GET("/booked_appointment", h.V1.GetAllBookedAppointments)

		v1.GET("/offline_appointment/:offline_appointment_id", h.V1.GetSingleOfflineAppointment)
		v1.GET("/booked_appointment/:booked_appointment_id", h.V1.GetSingleBookedAppointment)

		v1.PUT("/payment_status/:appointment_id", h.V1.UpdateAppointmentPaymentStatus)

		// cashbox
		v1.GET("/close-cashbox", h.V1.GetCloseCashboxInfo)
		v1.GET("/open-cashbox", h.V1.GetOpenCashboxInfo)

		// ANALYTICS_SERVICE
		// CASHBOX TRANSACTION
		v1.POST("/cashbox_transaction", h.V1.CashboxTransaction)
		// query
		v1.POST("/query", h.V1.GetQueryRows)

		// html-template
		//view
		v1.POST("/html-template", h.V1.CreateHtmlTemplate)
		v1.GET("/html-template/:html_template_id", h.V1.GetSingleHtmlTemplate)
		v1.GET("/html-template", h.V1.GetHtmlTemplateList)
		v1.PUT("/html-template", h.V1.UpdateHtmlTemplate)
		v1.DELETE("/html-template/:html_template_id", h.V1.DeleteHtmlTemplate)

		// document
		v1.POST("/document", h.V1.CreateDocument)
		v1.GET("/document/:document_id", h.V1.GetSingleDocument)
		v1.GET("/document", h.V1.GetDocumentList)
		v1.PUT("/document", h.V1.UpdateDocument)
		v1.DELETE("/document/:document_id", h.V1.DeleteDocument)

		// event
		v1.POST("/event", h.V1.CreateEvent)
		v1.GET("/event/:event_id", h.V1.GetEventByID)
		v1.GET("/event", h.V1.GetAllEvents)
		v1.PUT("/event", h.V1.UpdateEvent)
		v1.DELETE("/event/:event_id", h.V1.DeleteEvent)
		v1.GET("/event-log", h.V1.GetEventLogs)
		v1.GET("/event-log/:event_log_id", h.V1.GetEventLogById)

		// custom event
		v1.POST("/custom-event", h.V1.CreateCustomEvent)
		v1.GET("/custom-event/:custom_event_id", h.V1.GetCustomEventByID)
		v1.GET("/custom-event", h.V1.GetAllCustomEvents)
		v1.PUT("/custom-event", h.V1.UpdateCustomEvent)
		v1.DELETE("/custom-event/:custom_event_id", h.V1.DeleteCustomEvent)

		// function
		v1.POST("/function", h.V1.CreateFunction)
		v1.GET("/function/:function_id", h.V1.GetFunctionByID)
		v1.GET("/function", h.V1.GetAllNewFunctionsForApp)
		v1.PUT("/function", h.V1.UpdateFunction)
		v1.DELETE("/function/:function_id", h.V1.DeleteFunction)

		// INVOKE FUNCTION

		v1.POST("/invoke_function", h.V1.InvokeFunction)
		v1.POST("/invoke_function/:function-path", h.V1.InvokeFunctionByPath)

		// Excel Reader
		v1.GET("/excel/:excel_id", h.V1.ExcelReader)
		v1.POST("/excel/excel_to_db/:excel_id", h.V1.ExcelToDb)

		v1.GET("/barcode-generator/:table_slug", h.V1.GetNewGeneratedBarCode)
		v1.GET("/code-generator/:table_slug/:field_id", h.V1.GetNewGeneratedCode)

		// Integration with AlfaLab
		// v1.POST("/alfalab/directions", h.V1.CreateDirections)
		// v1.GET("/alfalab/referral", h.V1.GetReferral)

		v1.POST("/export-to-json", h.V1.ExportToJSON)
		v1.POST("import-from-json", h.V1.ImportFromJSON)

		// template
		v1.POST("/template-folder", h.V1.CreateTemplateFolder)
		v1.GET("/template-folder/:template-folder-id", h.V1.GetSingleTemplateFolder)
		v1.PUT("/template-folder", h.V1.UpdateTemplateFolder)
		v1.DELETE("/template-folder/:template-folder-id", h.V1.DeleteTemplateFolder)
		v1.GET("/template-folder", h.V1.GetListTemplateFolder)
		v1.GET("/template-folder/commits/:template-folder-id", h.V1.GetTemplateFolderCommits)
		v1.POST("/template", h.V1.CreateTemplate)
		v1.GET("/template/:template-id", h.V1.GetSingleTemplate)
		v1.PUT("/template", h.V1.UpdateTemplate)
		v1.DELETE("/template/:template-id", h.V1.DeleteTemplate)
		v1.GET("/template", h.V1.GetListTemplate)
		v1.GET("/template/commits/:template-id", h.V1.GetTemplateCommits)

		// HTML TO PDF CONVERTER
		v1.POST("/html-to-pdfV2", h.V1.ConvertHtmlToPdfV2)
		v1.POST("/template-to-htmlV2", h.V1.ConvertTemplateToHtmlV2)

		// HTML TO PDF CONVERTER
		v1.POST("/html-to-pdf", h.V1.ConvertHtmlToPdf)
		v1.POST("/template-to-html", h.V1.ConvertTemplateToHtml)

		// note
		v1.POST("/note-folder", h.V1.CreateNoteFolder)
		v1.GET("/note-folder/:note-folder-id", h.V1.GetSingleNoteFolder)
		v1.PUT("/note-folder", h.V1.UpdateNoteFolder)
		v1.DELETE("/note-folder/:note-folder-id", h.V1.DeleteNoteFolder)
		v1.GET("/note-folder", h.V1.GetListNoteFolder)
		v1.GET("/note-folder/commits/:note-folder-id", h.V1.GetNoteFolderCommits)
		v1.POST("/note", h.V1.CreateNote)
		v1.GET("/note/:note-id", h.V1.GetSingleNote)
		v1.PUT("/note", h.V1.UpdateNote)
		v1.DELETE("/note/:note-id", h.V1.DeleteNote)
		v1.GET("/note", h.V1.GetListNote)
		v1.GET("/note/commits/:note-id", h.V1.GetNoteCommits)
		v1.POST("/template-note/users", h.V1.CreateUserTemplate)
		v1.GET("/template-note/users", h.V1.GetListUserTemplate)
		v1.PUT("/template-note/users", h.V1.UpdateUserTemplate)
		v1.DELETE("/template-note/users/:user-permission-id", h.V1.DeleteUserTemplate)
		v1.POST("/template-note/share", h.V1.CreateSharingToken)
		v1.PUT("/template-note/share", h.V1.UpdateSharingToken)

		// api-reference
		v1.GET("/api-reference", h.V1.GetAllApiReferences)
		v1.GET("/api-reference/:api_reference_id", h.V1.GetApiReferenceByID)
		v1.GET("/category/:category_id", h.V1.GetApiCategoryByID)
		v1.GET("/category", h.V1.GetAllCategories)

		v1.GET("/layout", h.V1.GetListLayouts)
		v1.PUT("/layout", h.V1.UpdateLayout)
		v1.GET("/layout/:table_id/:menu_id", h.V1.GetSingleLayout)

		//menu
		v1.POST("/menu", h.V1.CreateMenu)
		v1.GET("/menu/:menu_id", h.V1.GetMenuByID)
		v1.GET("/menu", h.V1.GetAllMenus)
		v1.PUT("/menu", h.V1.UpdateMenu)
		v1.DELETE("/menu/:menu_id", h.V1.DeleteMenu)
		v1.PUT("menu/menu-order", h.V1.UpdateMenuOrder)

		//custom-error-message
		v1.GET("/custom-error-message", h.V1.GetAllCustomErrorMessage)
		v1.GET("/custom-error-message/:id", h.V1.GetByIdCustomErrorMessage)
		v1.PUT("/custom-error-message", h.V1.UpdateCustomErrorMessage)
		v1.POST("/custom-error-message", h.V1.CreateCustomErrorMessage)
		v1.DELETE("/custom-error-message/:id", h.V1.DeleteCustomErrorMessage)
		// table-permission
		v1.GET("/table-permission", h.V1.GetTablePermission)
		v1.PUT("/table-permission", h.V1.UpdateTablePermission)

		//report setting
		v1.GET("/get-report-setting/:id", h.V1.GetByIdReportSetting)
		v1.GET("/get-report-setting", h.V1.GetListReportSetting)
		v1.PUT("/upsert-report-setting", h.V1.UpsertReportSetting)
		v1.DELETE("/delete-report-setting/:id", h.V1.DeleteReportSetting)

		//dynamic-report
		v1.POST("/dynamic-report", h.V1.DynamicReport)
		// v1.GET("/export/dynamic-report/excel/:id", h.V1.ExportDynamicReportExcel) //TODO: should copy from parfume

		//dynamic-report template
		v1.POST("/save-pivot-template", h.V1.SavePivotTemplate)
		v1.GET("/get-pivot-template-setting/:id", h.V1.GetByIdPivotTemplate)
		v1.GET("/get-pivot-template-setting", h.V1.GetListPivotTemplate)
		v1.PUT("/upsert-pivot-template", h.V1.UpsertPivotTemplate)
		v1.DELETE("/remove-pivot-template/:id", h.V1.RemovePivotTemplate)

		v1.GET("/dynamic-report-formula", h.V1.DynamicReportFormula)

		v1.POST("/files/folder_upload", h.V1.UploadToFolder)
		v1.GET("/files/:id", h.V1.GetSingleFile)
		v1.PUT("/files", h.V1.UpdateFile)
		v1.DELETE("/files", h.V1.DeleteFiles)
		v1.DELETE("/files/:id", h.V1.DeleteFile)
		v1.GET("/files", h.V1.GetAllFiles)
	}
	v2 := r.Group("/v2")
	v2.Use(h.V1.AuthMiddleware(cfg))
	{
		// custom event
		v2.POST("/custom-event", h.V1.CreateNewCustomEvent)
		v2.GET("/custom-event/:custom_event_id", h.V1.GetNewCustomEventByID)
		v2.GET("/custom-event", h.V1.GetAllNewCustomEvents)
		v2.PUT("/custom-event", h.V1.UpdateNewCustomEvent)
		v2.DELETE("/custom-event/:custom_event_id", h.V1.DeleteNewCustomEvent)

		v2.GET("/language-json", h.V1.GetLanguageJson)

		v2.POST("/object/get-list/:table_slug", h.V1.GetListV2)
		v2.GET("/object-slim/get-list/:table_slug", h.V1.GetListSlimV2)

	}
	r.POST("/template-note/share-get", h.V1.GetObjectToken)

	v1Admin := r.Group("/v1")
	v1Admin.Use(h.V1.AdminAuthMiddleware())
	{

		// login microfront
		v1Admin.POST("/login-microfront", h.V1.BindLoginMicroFrontToProject)
		v1Admin.PUT("/login-microfront", h.V1.UpdateLoginMicroFrontProject)

		// company service
		v1.POST("/company", h.V1.CreateCompany)
		v1Admin.GET("/company/:company_id", h.V1.GetCompanyByID)
		v1Admin.GET("/company", h.V1.GetCompanyList)
		v1Admin.PUT("company/:company_id", h.V1.UpdateCompany)
		v1Admin.DELETE("/company/:company_id", h.V1.DeleteCompany)
		// project service
		v1Admin.POST("/company-project", h.V1.CreateCompanyProject)
		v1Admin.GET("/company-project", h.V1.GetCompanyProjectList)
		v1Admin.GET("/company-project/:project_id", h.V1.GetCompanyProjectById)
		v1Admin.PUT("/company-project/:project_id", h.V1.UpdateCompanyProject)
		v1Admin.DELETE("/company-project/:project_id", h.V1.DeleteCompanyProject)
		// project settings
		v1Admin.GET("/project/setting", h.V1.GetAllSettings)
		// project resource
		v1Admin.POST("/company/project/resource", h.V1.AddProjectResource)
		v1Admin.POST("/company/project/create-resource", h.V1.CreateProjectResource)
		v1Admin.DELETE("/company/project/resource", h.V1.RemoveProjectResource)
		v1Admin.GET("/company/project/resource/:resource_id", h.V1.GetResource)
		v1Admin.GET("/company/project/resource", h.V1.GetResourceList)
		v1Admin.GET("/company/project/service-resource", h.V1.GetListServiceResource)
		v1Admin.PUT("/company/project/service-resource", h.V1.UpdateServiceResource)
		v1Admin.POST("/company/project/resource/reconnect", h.V1.ReconnectProjectResource)
		v1Admin.PUT("/company/project/resource/:resource_id", h.V1.UpdateResource)
		v1Admin.POST("/company/project/configure-resource", h.V1.ConfigureProjectResource)
		v1Admin.GET("/company/project/resource-environment/:resource_id", h.V1.GetResourceEnvironment)
		v1Admin.GET("/company/project/resource-default", h.V1.GetServiceResources)
		v1Admin.PUT("/company/project/resource-default", h.V1.SetDefaultResource)

		// variable
		v1Admin.POST("/company/project/resource-variable", h.V1.AddDataToVariableResource)
		v1Admin.PUT("/company/project/resource-variable", h.V1.UpdateVariableResource)
		v1Admin.GET("/company/project/resource-variable/:project-resource-id", h.V1.GetListVariableResource)
		v1Admin.GET("/company/project/resource-variable/single", h.V1.GetSingleVariableResource)
		v1Admin.DELETE("/company/project/resource-variable/:id", h.V1.DeleteVariableResource)

		// environment service
		v1Admin.POST("/environment", h.V1.CreateEnvironment)
		v1Admin.GET("/environment/:environment_id", h.V1.GetSingleEnvironment)
		v1Admin.GET("/environment", h.V1.GetAllEnvironments)
		v1Admin.PUT("/environment", h.V1.UpdateEnvironment)
		v1Admin.DELETE("/environment/:environment_id", h.V1.DeleteEnvironment)

		// release service
		v1Admin.POST("/release", h.V1.CreateRelease)
		v1Admin.GET("/release/:project_id/:version_id", h.V1.GetReleaseByID)
		v1Admin.GET("/release/:project_id", h.V1.GetAllReleases)
		v1Admin.PUT("/release/:version_id", h.V1.UpdateRelease)
		v1Admin.DELETE("/release/:project_id/:version_id", h.V1.DeleteRelease)
		v1Admin.POST("/release/current", h.V1.SetCurrentRelease)
		v1Admin.GET("/release/current/:project_id", h.V1.GetCurrentRelease)

		// commit service
		v1Admin.POST("/commit", h.V1.CreateCommit)
		v1Admin.GET("/commit/:id", h.V1.GetCommitByID)
		v1Admin.GET("/commit", h.V1.GetAllCommits)

		// integration service
		v1Admin.POST("/generate-payze-link", h.V1.GeneratePayzeLink)
		v1Admin.POST("/payze-save-card", h.V1.PayzeSaveCard)

		//api-reference service
		v1Admin.POST("/api-reference", h.V1.CreateApiReference)
		v1Admin.PUT("/api-reference", h.V1.UpdateApiReference)
		// v1Admin.GET("/api-reference/:api_reference_id", h.V1.GetApiReferenceByID)
		// v1Admin.GET("/api-reference", h.V1.GetAllApiReferences)
		v1Admin.DELETE("/api-reference/:project_id/:api_reference_id", h.V1.DeleteApiReference)
		v1Admin.GET("/api-reference/history/:project_id/:api_reference_id", h.V1.GetApiReferenceChanges)
		v1Admin.POST("/api-reference/revert/:api_reference_id", h.V1.RevertApiReference)
		v1Admin.POST("/api-reference/select-versions/:api_reference_id", h.V1.InsertManyVersionForApiReference)

		v1Admin.POST("/category", h.V1.CreateCategory)
		v1Admin.PUT("/category", h.V1.UpdateCategory)
		// v1Admin.GET("/category/:category_id", h.V1.GetApiCategoryByID)
		// v1Admin.GET("/category", h.V1.GetAllCategories)
		v1Admin.DELETE("/category/:category_id", h.V1.DeleteCategory)

		// function folder
		v1Admin.POST("/function-folder", h.V1.CreateFunctionFolder)
		v1Admin.GET("/function-folder/:function_ifolder_d", h.V1.GetFunctionFolderById)
		v1Admin.GET("/function-folder", h.V1.GetAllFunctionFolder)
		v1Admin.PUT("/function-folder", h.V1.UpdateFunctionFolder)
		v1Admin.DELETE("/function-folder/:function_folder_id", h.V1.DeleteFunctionFolder)

		// scenario service
		v1Admin.POST("/scenario/dag", h.V1.CreateDAG)
		v1Admin.GET("/scenario/dag/:id", h.V1.GetDAG)
		v1Admin.GET("/scenario/dag", h.V1.GetAllDAG)
		v1Admin.PUT("/scenario/dag", h.V1.UpdateDAG)
		v1Admin.DELETE("/scenario/dag/:id", h.V1.DeleteDAG)

		v1Admin.POST("/scenario/dag-step", h.V1.CreateDagStep)
		v1Admin.GET("/scenario/dag-step/:id", h.V1.GetDagStep)
		v1Admin.GET("/scenario/dag-step", h.V1.GetAllDagStep)
		v1Admin.PUT("/scenario/dag-step", h.V1.UpdateDagStep)
		v1Admin.DELETE("/scenario/dag-step/:id", h.V1.DeleteDAG)

		v1Admin.POST("/scenario/category", h.V1.CreateCategoryScenario)
		v1Admin.GET("/scenario/category/:id", h.V1.GetCategoryScenario)
		v1Admin.GET("/scenario/category", h.V1.GetListCategoryScenario)

		v1Admin.POST("/scenario/run", h.V1.RunScenario)
		v1Admin.POST("/scenario", h.V1.CreateFullScenario)
		v1Admin.PUT("/scenario", h.V1.UpdateFullScenario) //--- update means also create but with new commit
		v1Admin.GET("/scenario/:id/history", h.V1.GetScenarioHistory)
		v1Admin.PUT("/scenario/:id/select-versions", h.V1.SelectVersionsScenario)
		v1Admin.POST("/scenario/revert", h.V1.RevertScenario)

		// query service
		v1Admin.POST("/query-folder", h.V1.CreateQueryRequestFolder)
		v1Admin.PUT("/query-folder", h.V1.UpdateQueryRequestFolder)
		v1Admin.GET("/query-folder", h.V1.GetListQueryRequestFolder)
		v1Admin.GET("/query-folder/:query-folder-id", h.V1.GetSingleQueryRequestFolder)
		v1Admin.DELETE("/query-folder/:query-folder-id", h.V1.DeleteQueryRequestFolder)

		v1Admin.POST("/query-request", h.V1.CreateQueryRequest)
		v1Admin.PUT("/query-request", h.V1.UpdateQueryRequest)
		v1Admin.GET("/query-request", h.V1.GetListQueryRequest)
		v1Admin.GET("/query-request/:query-id", h.V1.GetSingleQueryRequest)
		v1Admin.DELETE("/query-request/:query-id", h.V1.DeleteQueryRequest)
		v1Admin.POST("/query-request/select-versions/:query-id", h.V1.InsertManyVersionForQueryService)
		v1Admin.POST("/query-request/run", h.V1.QueryRun)
		v1Admin.GET("/query-request/:query-id/history", h.V1.GetQueryHistory)
		v1Admin.POST("/query-request/:query-id/revert", h.V1.RevertQuery)
		v1Admin.GET("/query-request/:query-id/log", h.V1.GetListQueryLog)
		v1Admin.GET("/query-request/:query-id/log/:log-id", h.V1.GetSingleQueryLog)

		// web page service
		v1Admin.POST("/webpage-folder", h.V1.CreateWebPageFolder)
		v1Admin.PUT("/webpage-folder", h.V1.UpdateWebPageFolder)
		v1Admin.GET("/webpage-folder", h.V1.GetListWebPageFolder)
		v1Admin.GET("/webpage-folder/:webpage-folder-id", h.V1.GetSingleWebPageFolder)
		v1Admin.DELETE("/webpage-folder/:webpage-folder-id", h.V1.DeleteWebPageFolder)

		v1Admin.POST("/webpage-app", h.V1.CreateWebPageApp)
		v1Admin.PUT("/webpage-app", h.V1.UpdateWebPageApp)
		v1Admin.GET("/webpage-app", h.V1.GetListWebPageApp)
		v1Admin.GET("/webpage-app/:webpage-app-id", h.V1.GetSingleWebPageApp)
		v1Admin.DELETE("/webpage-app/:webpage-app-id", h.V1.DeleteWebPageApp)

		v1Admin.POST("/webpageV2", h.V1.CreateWebPageV2)
		v1Admin.PUT("/webpageV2", h.V1.UpdateWebPageV2)
		v1Admin.GET("/webpageV2", h.V1.GetListWebPageV2)
		v1Admin.GET("/webpageV2/:webpage-id", h.V1.GetSingleWebPageV2)
		v1Admin.DELETE("/webpageV2/:webpage-id", h.V1.DeleteWebPageV2)
		v1Admin.POST("/webpageV2/select-versions/:webpage-id", h.V1.InsertManyVersionForWebPageService)
		v1Admin.GET("/webpageV2/:webpage-id/history", h.V1.GetWebPageHistory)
		v1Admin.POST("/webpageV2/:webpage-id/revert", h.V1.RevertWebPage)

		// notification service
		v1Admin.POST("/notification/user-fcmtoken", h.V1.CreateUserFCMToken)
		v1Admin.POST("/notification", h.V1.CreateNotificationUsers)
		v1Admin.GET("/notification", h.V1.GetAllNotifications)
		v1Admin.GET("/notification/:id", h.V1.GetNotificationById)
		v1Admin.GET("/notification/category", h.V1.GetListCategoryNotification)
		v1Admin.POST("/notification/category", h.V1.CreateCategoryNotification)
		v1Admin.GET("/notification/category/:id", h.V1.GetCategoryNotification)
		v1Admin.PUT("/notification/category", h.V1.UpdateCategoryNotification)
		v1Admin.DELETE("/notification/category/:id", h.V1.DeleteCategoryNotification)

		v1Admin.GET("/table-history/list/:table_id", h.V1.GetListTableHistory)
		v1Admin.GET("/table-history/:id", h.V1.GetTableHistoryById)
		v1Admin.PUT("/table-history/revert", h.V1.RevertTableHistory)
		v1Admin.PUT("/table-history", h.V1.InsetrVersionsIdsToTableHistory)

		v1Admin.POST("/redirect-url", h.V1.CreateRedirectUrl)
		v1Admin.PUT("/redirect-url", h.V1.UpdateRedirectUrl)
		v1Admin.GET("/redirect-url", h.V1.GetListRedirectUrl)
		v1Admin.GET("/redirect-url/:redirect-url-id", h.V1.GetSingleRedirectUrl)
		v1Admin.DELETE("/redirect-url/:redirect-url-id", h.V1.DeleteRedirectUrl)
		v1Admin.PUT("/redirect-url/re-order", h.V1.UpdateRedirectUrlOrder)
	}
	v2Admin := r.Group("/v2")
	v2Admin.Use(h.V1.AdminAuthMiddleware())
	v2Admin.POST("/table-folder", h.V1.CreateTableFolder)
	v2Admin.PUT("/table-folder", h.V1.UpdateTableFolder)
	v2Admin.GET("/table-folder", h.V1.GetAllTableFolders)
	v2Admin.GET("/table-folder/:id", h.V1.GetTableFolderByID)
	v2Admin.DELETE("/table-folder/:id", h.V1.DeleteTableFolder)

	function := v2Admin.Group("/functions")
	{
		function.Any("/:function-id/run", h.V1.FunctionRun)
		r.Any("v1/functions/:function-id/run", h.V1.FunctionRun)
	}

	{
		// function
		v2Admin.POST("/function", h.V1.CreateNewFunction)
		v2Admin.GET("/function/:function_id", h.V1.GetNewFunctionByID)
		v2Admin.GET("/function", h.V1.GetAllNewFunctions)
		v2Admin.PUT("/function", h.V1.UpdateNewFunction)
		v2Admin.DELETE("/function/:function_id", h.V1.DeleteNewFunction)

		// project resource /rest
		v2Admin.POST("/company/project/resource", h.V1.AddResourceToProject)
		v2Admin.PUT("/company/project/resource", h.V1.UpdateProjectResource)
		v2Admin.GET("/company/project/resource", h.V1.GetListProjectResourceList)
		v2Admin.GET("/company/project/resource/:id", h.V1.GetSingleProjectResource)
		v2Admin.DELETE("/company/project/resource/:id", h.V1.DeleteProjectResource)

		v2Admin.POST("/copy-project", h.V1.CopyProjectTemplate)

		functions := v2Admin.Group("functions")
		{
			functions.POST("/micro-frontend", h.V1.CreateMicroFrontEnd)
			functions.GET("/micro-frontend/:micro-frontend-id", h.V1.GetMicroFrontEndByID)
			functions.GET("/micro-frontend", h.V1.GetAllMicroFrontEnd)
			functions.PUT("/micro-frontend", h.V1.UpdateMicroFrontEnd)
			functions.DELETE("/micro-frontend/:micro-frontend-id", h.V1.DeleteMicroFrontEnd)
		}
	}

	// v3 for ucode version 2
	v3 := r.Group("/v3")
	v3.Use(h.V1.AdminAuthMiddleware())
	{
		// query folder
		v3.POST("/query_folder", h.V1.CreateQueryFolder)
		v3.GET("/query_folder/:guid", h.V1.GetQueryFolderByID)
		v3.GET("/query_folder", h.V1.GetQueryFolderList)
		v3.PUT("/query_folder/:guid", h.V1.UpdateQueryFolder)
		v3.DELETE("/query_folder/:guid", h.V1.DeleteQueryFolder)

		// // query
		v3.POST("/query", h.V1.CreateQuery)
		v3.GET("/query/:guid", h.V1.GetQueryByID)
		v3.GET("/query", h.V1.GetQueryList)
		v3.PUT("/query/:guid", h.V1.UpdateQuery)
		v3.DELETE("/query/:guid", h.V1.DeleteQuery)
		// // web pages
		v3.POST("/web_pages", h.V1.CreateWebPage)
		v3.GET("/web_pages/:guid", h.V1.GetWebPagesById)
		v3.GET("/web_pages", h.V1.GetWebPagesList)
		v3.PUT("/web_pages/:guid", h.V1.UpdateWebPage)
		v3.DELETE("/web_pages/:guid", h.V1.DeleteWebPage)

		v3.POST("/chat", h.V1.CreatChat)
		v3.GET("/chat", h.V1.GetChatList)
		v3.GET("/chat/:id", h.V1.GetChatByChatID)

		v3.POST("/bot", h.V1.CreateBot)
		v3.GET("/bot/:id", h.V1.GetBotTokenByBotID)
		v3.GET("/bot", h.V1.GetBotTokenList)
		v3.PUT("/bot", h.V1.UpdateBotToken)
		v3.DELETE("/bot/:id", h.V1.DeleteBotToken)

	}

	v2Version := r.Group("/v2")
	v2Version.Use(h.V2.AuthMiddleware())
	{
		// collections group
		v2Collection := v2Version.Group("/collections")
		{

			// error messages
			v2Collection.GET("/:collection/error_messages", h.V2.GetAllErrorMessage)
			v2Collection.POST("/:collection/error_messages", h.V2.CreateErrorMessage)
			v2Collection.PUT("/:collection/error_messages", h.V2.UpdateErrorMessage)
			v2Collection.GET("/:collection/error_messages/:id", h.V2.GetByIdErrorMessage)
			v2Collection.DELETE("/:collection/error_messages/:id", h.V2.DeleteErrorMessage)

			// automation
			v2Collection.GET("/:collection/automation", h.V2.GetAllAutomation)
			v2Collection.POST("/:collection/automation", h.V2.CreateAutomation)
			v2Collection.PUT("/:collection/automation", h.V2.UpdateAutomation)
			v2Collection.GET("/:collection/automation/:id", h.V2.GetByIdAutomation)
			v2Collection.DELETE("/:collection/automation/:id", h.V2.DeleteAutomation)

			// import data
			v2Collection.POST("/:collection/import/:id", h.V2.ImportData)
			v2Collection.POST("/:collection/import/fields/:id", h.V2.ExcelReader)

			// layout
			v2Collection.GET("/:collection/layout", h.V2.GetListLayouts)
			v2Collection.PUT("/:collection/layout", h.V2.UpdateLayout)
			v2Collection.GET("/:collection/layout/:menu_id", h.V2.GetSingleLayout)
			v2Collection.DELETE("/:collection/layout/:id", h.V2.DeleteLayout)
			// collection
			v2Collection.POST("", h.V2.CreateCollection)
			v2Collection.PUT("", h.V2.UpdateCollection)
			v2Collection.GET("", h.V2.GetAllCollections)
			v2Collection.GET("/:collection", h.V2.GetSingleCollection)
			v2Collection.DELETE("/:collection", h.V2.DeleteCollection)

		}

		// items group
		v2Items := v2Version.Group("/items")
		{
			v2Items.GET("/:collection", h.V2.GetAllItems)
			v2Items.GET("/:collection/:id", h.V2.GetSingleItem)
			v2Items.POST("/:collection", h.V2.CreateItem)
			v2Items.POST("/:collection/multiple-insert", h.V2.CreateItems)
			v2Items.PUT("/:collection", h.V2.UpdateItem)
			v2Items.PATCH("/:collection", h.V2.MultipleUpdateItems)
			v2Items.DELETE("/:collection", h.V2.DeleteItems)
			v2Items.DELETE("/:collection/:id", h.V2.DeleteItem)

			v2Items.PUT("/many-to-many", h.V2.AppendManyToMany)
			v2Items.DELETE("/many-to-many", h.V2.DeleteManyToMany)
		}

		// menu group
		v2Menus := v2Version.Group("/menus")
		{
			v2Menus.GET("", h.V2.GetAllMenus)
			v2Menus.GET("/:id", h.V2.GetMenuByID)
			v2Menus.PUT("", h.V2.UpdateMenu)
			v2Menus.POST("", h.V2.CreateMenu)
			v2Menus.DELETE("/:id", h.V2.DeleteMenu)
			v2Menus.PUT("/menu-order", h.V2.UpdateMenuOrder)

			v2Menus.POST("/menu-settings", h.V2.CreateMenuSettings)
			v2Menus.PUT("/menu-settings", h.V2.UpdateMenuSettings)

			v2Menus.GET("/menu-template", h.V2.GetAllMenuTemplates)
			v2Menus.GET("/menu-template/:id", h.V2.GetMenuTemplateByID)
			v2Menus.PUT("/menu-template", h.V2.UpdateMenuTemplate)
			v2Menus.POST("/menu-template", h.V2.CreateMenuTemplate)
			v2Menus.DELETE("/menu-template", h.V2.DeleteMenuTemplate)
		}

		// user group
		v2User := v2Version.Group("/user")
		{
			v2User.GET("/:id/menu-settings", h.V2.GetMenuSettingByUserID)
		}
		// view group
		v2View := v2Version.Group("/views")
		{
			v2View.GET("/:collection", h.V2.GetAllViews)
			v2View.POST("/:collection", h.V2.CreateView)
			v2View.PUT("/:collection", h.V2.UpdateView)
			v2View.DELETE("/:collection/:id", h.V2.DeleteView)
			v2View.PUT("/:collection/update-order", h.V2.UpdateViewOrder)
		}

		// fields
		v2Version.GET("/fields/:collection", h.V2.GetAllFields)
		v2Version.POST("/fields/:collection", h.V2.CreateField)
		v2Version.PUT("/fields/:collection", h.V2.UpdateField)
		v2Version.DELETE("/fields/:collection/:id", h.V2.DeleteField)

		// relations
		v2Version.GET("/relations/:collection/:id", h.V2.GetByIdRelation)
		v2Version.GET("/relations/:collection", h.V2.GetAllRelations)
		v2Version.POST("/relations/:collection", h.V2.CreateRelation)
		v2Version.PUT("/relations/:collection", h.V2.UpdateRelation)
		v2Version.DELETE("/relations/:collection/:id", h.V2.DeleteRelation)
		v2Version.GET("/relations/:collection/cascading", h.V2.GetRelationCascading)

		// utils
		v2Utils := v2Version.Group("/utils")
		{
			v2Utils.GET("/barcode/:collection/:type", h.V2.GetGeneratedBarcode)
			v2Utils.POST("/export/:collection/html-to-pdf", h.V2.ConvertHtmlToPdf)
			v2Utils.POST("/export/:collection/template-to-html", h.V2.ConvertTemplateToHtml)
			v2Utils.POST("/export/:collection", h.V2.ExportData)
		}
	}

	r.Any("/api/*any", h.V1.AuthMiddleware(cfg), proxyMiddleware(r, &h), h.V1.Proxy)

	// r.Any("/x-api/*any", h.V1.RedirectAuthMiddleware(cfg), proxyMiddleware(r, &h), h.V1.Proxy)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}

func customCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, Origin, Cache-Control, X-Requested-With, Resource-Id, Environment-Id, Platform-Type, X-API-KEY, Project-Id")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func MaxAllowed(n int) gin.HandlerFunc {
	var countReq int64
	sem := make(chan struct{}, n)
	acquire := func() {
		sem <- struct{}{}
		countReq++
	}

	release := func() {
		select {
		case <-sem:
		default:
		}
		countReq--
	}

	return func(c *gin.Context) {
		go func() {
			acquire()       // before request
			defer release() // after request

			c.Set("sem", sem)
			c.Set("count_request", countReq)
		}()

		c.Next()
	}
}

func proxyMiddleware(r *gin.Engine, h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			err error
		)
		c, err = RedirectUrl(c, h)
		if err == nil {
			r.HandleContext(c)
		}
		c.Next()
	}
}

func RedirectUrl(c *gin.Context, h *handlers.Handler) (*gin.Context, error) {
	path := c.Request.URL.Path
	projectId, ok := c.Get("project_id")
	if !ok {
		return c, errors.New("something went wrong")
	}

	envId, ok := c.Get("environment_id")
	if !ok {
		return c, errors.New("something went wrong")
	}

	c.Request.Header.Add("prev_path", path)
	data := helper.MatchingData{
		ProjectId: projectId.(string),
		EnvId:     envId.(string),
		Path:      path,
	}

	// companyRedirectGetListTime := time.Now()
	res, err := h.V1.CompanyRedirectGetList(data, h.GetCompanyService(c))
	if err != nil {
		return c, errors.New("cant change")
	}
	// fmt.Println(">>>>>>>>>>>>>>>CompanyRedirectGetList:", time.Since(companyRedirectGetListTime))

	pathM, err := helper.FindUrlTo(res, data)
	if err != nil {
		return c, errors.New("cant change")
	}
	if path == pathM {
		return c, errors.New("identical path")
	}

	c.Request.URL.Path = pathM
	if strings.Contains(pathM, "/v1/functions/") {
		c.Request.Header.Add("/v1/functions/", cast.ToString(true))
	}

	c.Request.Header.Add("resource_id", cast.ToString(c.Value("resource_id")))
	c.Request.Header.Add("environment_id", cast.ToString(c.Value("environment_id")))
	c.Request.Header.Add("project_id", cast.ToString(c.Value("project_id")))
	c.Request.Header.Add("redirect", cast.ToString(true))

	auth, err := json.Marshal(c.Value("auth"))
	if err != nil {
		return c, errors.New("something went wrong")
	}
	c.Request.Header.Add("auth", string(auth))
	return c, nil
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("environment_id", "063dad1e-4596-483a-86a38-14f9b99922c3")
		u := c.Request.URL.Query()
		u.Set("project-id", "d6042238-0f60-4f30-8c1a-af78883f1d52")
		c.Request.URL.RawQuery = u.Encode()
		c.Next()
	}
}
