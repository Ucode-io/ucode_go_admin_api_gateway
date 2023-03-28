package api

import (
	"ucode/ucode_go_api_gateway/api/docs"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"

	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// SetUpAPI @description This is an api gateway
// @termsOfService https://udevs.io
func SetUpAPI(r *gin.Engine, h handlers.Handler, cfg config.Config) {
	docs.SwaggerInfo.Title = cfg.ServiceName
	docs.SwaggerInfo.Version = cfg.Version
	// docs.SwaggerInfo.Host = cfg.ServiceHost + cfg.HTTPPort
	docs.SwaggerInfo.Schemes = []string{cfg.HTTPScheme}

	r.Use(customCORSMiddleware())
	r.Use(MaxAllowed(5000))
	r.Use(h.NodeMiddleware())

	r.GET("/ping", h.Ping)
	r.GET("/config", h.GetConfig)

	r.POST("/send-code", h.SendCode)
	r.POST("/verify/:sms_id/:otp", h.Verify)
	r.POST("/register-otp/:table_slug", h.RegisterOtp)
	r.POST("/send-message", h.SendMessageToEmail)
	r.POST("/verify-email/:sms_id/:otp", h.VerifyEmail)
	r.POST("/register-email-otp/:table_slug", h.RegisterEmailOtp)

	v1 := r.Group("/v1")
	// @securityDefinitions.apikey ApiKeyAuth
	// @in header
	// @name Authorization
	v1.Use(h.AuthMiddleware(cfg))
	{
		v1.POST("/upload", h.Upload)
		v1.POST("/upload-file/:table_slug/:object_id", h.UploadFile)

		// OBJECT_BUILDER_SERVICE

		//table
		v1.POST("/table", h.CreateTable)
		v1.GET("/table", h.GetAllTables)
		v1.GET("/table/:table_id", h.GetTableByID)

		v1.PUT("/table", h.UpdateTable)
		v1.DELETE("/table/:table_id", h.DeleteTable)
		//field
		v1.POST("/field", h.CreateField)
		v1.GET("/field", h.GetAllFields)
		v1.PUT("/field", h.UpdateField)
		v1.DELETE("/field/:field_id", h.DeleteField)
		v1.POST("/fields-relations", h.CreateFieldsAndRelations)

		//relation
		v1.POST("/relation", h.CreateRelation)
		v1.GET("/relation", h.GetAllRelations)
		v1.PUT("/relation", h.UpdateRelation)
		v1.DELETE("/relation/:relation_id", h.DeleteRelation)
		v1.GET("/get-relation-cascading/:table_slug", h.GetRelationCascaders)

		//section
		v1.GET("/section", h.GetAllSections)
		v1.PUT("/section", h.UpdateSection)

		//view_relation
		v1.GET("/view_relation", h.GetViewRelation)
		v1.PUT("/view_relation", h.UpsertViewRelations)

		//object-builder
		v1.POST("/object/:table_slug", h.CreateObject)
		v1.GET("/object/:table_slug/:object_id", h.GetSingle)
		v1.POST("/object/get-list/:table_slug", h.GetList)
		v1.PUT("/object/:table_slug", h.UpdateObject)
		v1.DELETE("/object/:table_slug/:object_id", h.DeleteObject)
		v1.POST("/object/object-details/:table_slug", h.GetObjectDetails)
		v1.POST("/object/excel/:table_slug", h.GetListInExcel)
		v1.POST("/object-upsert/:table_slug", h.UpsertObject)
		v1.PUT("/object/multiple-update/:table_slug", h.MultipleUpdateObject)
		v1.POST("/object/get-financial-analytics/:table_slug", h.GetFinancialAnalytics)

		// permission
		v1.POST("/permission-upsert/:app_id", h.UpsertPermissionsByAppId)
		v1.GET("/permission-get-all/:role_id", h.GetAllPermissionByRoleId)
		v1.GET("/field-permission/:role_id/:table_slug", h.GetFieldPermissions)
		v1.GET("/action-permission/:role_id/:table_slug", h.GetActionPermissions)
		v1.GET("/view-relation-permission/:role_id/:table_slug", h.GetViewRelationPermissions)

		//many-to-many
		v1.PUT("/many-to-many", h.AppendManyToMany)
		v1.DELETE("/many-to-many", h.DeleteManyToMany)

		//view
		v1.POST("/view", h.CreateView)
		v1.GET("/view/:view_id", h.GetSingleView)
		v1.GET("/view", h.GetViewList)
		v1.PUT("/view", h.UpdateView)
		v1.DELETE("/view/:view_id", h.DeleteView)

		//anaytics dashboard
		v1.POST("/analytics/dashboard", h.CreateDashboard)
		v1.GET("/analytics/dashboard/:dashboard_id", h.GetSingleDashboard)
		v1.GET("/analytics/dashboard", h.GetAllDashboards)
		v1.PUT("/analytics/dashboard", h.UpdateDashboard)
		v1.DELETE("/analytics/dashboard/:dashboard_id", h.DeleteDashboard)

		//anaytics variable
		v1.POST("/analytics/variable", h.CreateVariable)
		v1.GET("/analytics/variable/:variable_id", h.GetSingleVariable)
		v1.GET("/analytics/variable", h.GetAllVariables)
		v1.PUT("/analytics/variable", h.UpdateVariable)
		v1.DELETE("/analytics/variable/:variable_id", h.DeleteVariable)

		//anaytics panel
		v1.POST("/analytics/panel/updateCoordinates", h.UpdateCoordinates)
		v1.POST("/analytics/panel", h.CreatePanel)
		v1.GET("/analytics/panel/:panel_id", h.GetSinglePanel)
		v1.GET("/analytics/panel", h.GetAllPanels)
		v1.PUT("/analytics/panel", h.UpdatePanel)
		v1.DELETE("/analytics/panel/:panel_id", h.DeletePanel)

		//app
		v1.POST("/app", h.CreateApp)
		v1.GET("/app/:app_id", h.GetAppByID)
		v1.GET("/app", h.GetAllApps)
		v1.PUT("/app", h.UpdateApp)
		v1.DELETE("/app/:app_id", h.DeleteApp)

		// POS_SERVICE

		//appointments
		v1.GET("/offline_appointment", h.GetAllOfflineAppointments)
		v1.GET("/booked_appointment", h.GetAllBookedAppointments)

		v1.GET("/offline_appointment/:offline_appointment_id", h.GetSingleOfflineAppointment)
		v1.GET("/booked_appointment/:booked_appointment_id", h.GetSingleBookedAppointment)

		v1.PUT("/payment_status/:appointment_id", h.UpdateAppointmentPaymentStatus)

		// cashbox
		v1.GET("/close-cashbox", h.GetCloseCashboxInfo)
		v1.GET("/open-cashbox", h.GetOpenCashboxInfo)

		// ANALYTICS_SERVICE
		// CASHBOX TRANSACTION
		v1.POST("/cashbox_transaction", h.CashboxTransaction)
		// query
		v1.POST("/query", h.GetQueryRows)

		// html-template
		//view
		v1.POST("/html-template", h.CreateHtmlTemplate)
		v1.GET("/html-template/:html_template_id", h.GetSingleHtmlTemplate)
		v1.GET("/html-template", h.GetHtmlTemplateList)
		v1.PUT("/html-template", h.UpdateHtmlTemplate)
		v1.DELETE("/html-template/:html_template_id", h.DeleteHtmlTemplate)

		// document
		v1.POST("/document", h.CreateDocument)
		v1.GET("/document/:document_id", h.GetSingleDocument)
		v1.GET("/document", h.GetDocumentList)
		v1.PUT("/document", h.UpdateDocument)
		v1.DELETE("/document/:document_id", h.DeleteDocument)

		// event
		v1.POST("/event", h.CreateEvent)
		v1.GET("/event/:event_id", h.GetEventByID)
		v1.GET("/event", h.GetAllEvents)
		v1.PUT("/event", h.UpdateEvent)
		v1.DELETE("/event/:event_id", h.DeleteEvent)
		v1.GET("/event-log", h.GetEventLogs)
		v1.GET("/event-log/:event_log_id", h.GetEventLogById)

		// custom event
		v1.POST("/custom-event", h.CreateCustomEvent)
		v1.GET("/custom-event/:custom_event_id", h.GetCustomEventByID)
		v1.GET("/custom-event", h.GetAllCustomEvents)
		v1.PUT("/custom-event", h.UpdateCustomEvent)
		v1.DELETE("/custom-event/:custom_event_id", h.DeleteCustomEvent)

		// function
		v1.POST("/function", h.CreateFunction)
		v1.GET("/function/:function_id", h.GetFunctionByID)
		v1.GET("/function", h.GetAllFunctions)
		v1.PUT("/function", h.UpdateFunction)
		v1.DELETE("/function/:function_id", h.DeleteFunction)

		// INVOKE FUNCTION

		v1.POST("/invoke_function", h.InvokeFunction)

		// Excel Reader
		v1.GET("/excel/:excel_id", h.ExcelReader)
		v1.POST("/excel/excel_to_db/:excel_id", h.ExcelToDb)

		v1.GET("/barcode-generator/:table_slug", h.GetNewGeneratedBarCode)
		v1.GET("/code-generator/:table_slug/:field_id", h.GetNewGeneratedCode)

		// Integration with AlfaLab
		v1.POST("/alfalab/directions", h.CreateDirections)
		v1.GET("/alfalab/referral", h.GetReferral)

		v1.POST("/export-to-json", h.ExportToJSON)
		v1.POST("import-from-json", h.ImportFromJSON)

		// template
		v1.POST("/template-folder", h.CreateTemplateFolder)
		v1.GET("/template-folder/:template-folder-id", h.GetSingleTemplateFolder)
		v1.PUT("/template-folder", h.UpdateTemplateFolder)
		v1.DELETE("/template-folder/:template-folder-id", h.DeleteTemplateFolder)
		v1.GET("/template-folder", h.GetListTemplateFolder)
		v1.GET("/template-folder/commits/:template-folder-id", h.GetTemplateFolderCommits)
		v1.POST("/template", h.CreateTemplate)
		v1.GET("/template/:template-id", h.GetSingleTemplate)
		v1.PUT("/template", h.UpdateTemplate)
		v1.DELETE("/template/:template-id", h.DeleteTemplate)
		v1.GET("/template", h.GetListTemplate)
		v1.GET("/template/commits/:template-id", h.GetTemplateCommits)

		// HTML TO PDF CONVERTER
		v1.POST("/html-to-pdfV2", h.ConvertHtmlToPdfV2)
		v1.POST("/template-to-htmlV2", h.ConvertTemplateToHtmlV2)

		// HTML TO PDF CONVERTER
		v1.POST("/html-to-pdf", h.ConvertHtmlToPdf)
		v1.POST("/template-to-html", h.ConvertTemplateToHtml)

		// note
		v1.POST("/note-folder", h.CreateNoteFolder)
		v1.GET("/note-folder/:note-folder-id", h.GetSingleNoteFolder)
		v1.PUT("/note-folder", h.UpdateNoteFolder)
		v1.DELETE("/note-folder/:note-folder-id", h.DeleteNoteFolder)
		v1.GET("/note-folder", h.GetListNoteFolder)
		v1.GET("/note-folder/commits/:note-folder-id", h.GetNoteFolderCommits)
		v1.POST("/note", h.CreateNote)
		v1.GET("/note/:note-id", h.GetSingleNote)
		v1.PUT("/note", h.UpdateNote)
		v1.DELETE("/note/:note-id", h.DeleteNote)
		v1.GET("/note", h.GetListNote)
		v1.GET("/note/commits/:note-id", h.GetNoteCommits)
		v1.POST("/template-note/users", h.CreateUserTemplate)
		v1.GET("/template-note/users", h.GetListUserTemplate)
		v1.PUT("/template-note/users", h.UpdateUserTemplate)
		v1.DELETE("/template-note/users/:user-permission-id", h.DeleteUserTemplate)
		v1.POST("/template-note/share", h.CreateSharingToken)
		v1.PUT("/template-note/share", h.UpdateSharingToken)

	}
	r.POST("/template-note/share-get", h.GetObjectToken)

	v1Admin := r.Group("/v1")
	v1Admin.Use(h.AdminAuthMiddleware())
	{
		// company service
		// v1.POST("/company", h.CreateCompany)
		v1Admin.GET("/company/:company_id", h.GetCompanyByID)
		v1Admin.GET("/company", h.GetCompanyList)
		v1Admin.PUT("company/:company_id", h.UpdateCompany)
		v1Admin.DELETE("/company/:company_id", h.DeleteCompany)

		// project service
		v1Admin.POST("/company-project", h.CreateCompanyProject)
		v1Admin.GET("/company-project", h.GetCompanyProjectList)
		v1Admin.GET("/company-project/:project_id", h.GetCompanyProjectById)
		v1Admin.PUT("/company-project/:project_id", h.UpdateCompanyProject)
		v1Admin.DELETE("/company-project/:project_id", h.DeleteCompanyProject)

		v1Admin.POST("/company/project/resource", h.AddProjectResource)
		v1Admin.POST("/company/project/create-resource", h.CreateProjectResource)
		v1Admin.DELETE("/company/project/resource", h.RemoveProjectResource)
		v1Admin.GET("/company/project/resource/:resource_id", h.GetResource)
		v1Admin.GET("/company/project/resource", h.GetResourceList)
		v1Admin.POST("/company/project/resource/reconnect", h.ReconnectProjectResource)
		v1Admin.PUT("/company/project/resource/:resource_id", h.UpdateResource)
		v1Admin.POST("/company/project/configure-resource", h.ConfigureProjectResource)
		v1Admin.GET("/company/project/resource-environment/:resource_id", h.GetResourceEnvironment)
		v1Admin.GET("/company/project/resource-default", h.GetServiceResources)
		v1Admin.PUT("/company/project/resource-default", h.SetDefaultResource)

		// environment service
		v1Admin.POST("/environment", h.CreateEnvironment)
		v1Admin.GET("/environment/:environment_id", h.GetSingleEnvironment)
		v1Admin.GET("/environment", h.GetAllEnvironments)
		v1Admin.PUT("/environment", h.UpdateEnvironment)
		v1Admin.DELETE("/environment/:environment_id", h.DeleteEnvironment)

		// release service
		v1Admin.POST("/release", h.CreateRelease)
		v1Admin.GET("/release/:project_id/:version_id", h.GetReleaseByID)
		v1Admin.GET("/release/:project_id", h.GetAllReleases)
		v1Admin.PUT("/release/:version_id", h.UpdateRelease)
		v1Admin.DELETE("/release/:project_id/:version_id", h.DeleteRelease)
		v1Admin.POST("/release/current", h.SetCurrentRelease)
		v1Admin.GET("/release/current/:project_id", h.GetCurrentRelease)

		// commit service
		v1Admin.POST("/commit", h.CreateCommit)
		v1Admin.GET("/commit/:id", h.GetCommitByID)
		v1Admin.GET("/commit", h.GetAllCommits)

		// integration service
		v1Admin.POST("/generate-payze-link", h.GeneratePayzeLink)
		v1Admin.POST("/payze-save-card", h.PayzeSaveCard)

		//api-reference service
		v1Admin.POST("/api-reference", h.CreateApiReference)
		v1Admin.PUT("/api-reference", h.UpdateApiReference)
		v1Admin.GET("/api-reference/:api_reference_id", h.GetApiReferenceByID)
		v1Admin.GET("/api-reference", h.GetAllApiReferences)
		v1Admin.DELETE("/api-reference/:project_id/:api_reference_id", h.DeleteApiReference)
		v1Admin.GET("/api-reference/history/:project_id/:api_reference_id", h.GetApiReferenceChanges)
		v1Admin.POST("/api-reference/revert/:api_reference_id", h.RevertApiReference)
		v1Admin.POST("/api-reference/select-versions/:api_reference_id", h.InsertManyVersionForApiReference)

		v1Admin.POST("/category", h.CreateCategory)
		v1Admin.PUT("/category", h.UpdateCategory)
		v1Admin.GET("/category/:category_id", h.GetApiCategoryByID)
		v1Admin.GET("/category", h.GetAllCategories)
		v1Admin.DELETE("/category/:category_id", h.DeleteCategory)

		// function folder
		v1Admin.POST("/function-folder", h.CreateFunctionFolder)
		v1Admin.GET("/function-folder/:function_ifolder_d", h.GetFunctionFolderById)
		v1Admin.GET("/function-folder", h.GetAllFunctionFolder)
		v1Admin.PUT("/function-folder", h.UpdateFunctionFolder)
		v1Admin.DELETE("/function-folder/:function_folder_id", h.DeleteFunctionFolder)

		// scenario service
		v1Admin.POST("/scenario/dag", h.CreateDAG)
		v1Admin.GET("/scenario/dag/:id", h.GetDAG)
		v1Admin.GET("/scenario/dag", h.GetAllDAG)
		v1Admin.PUT("/scenario/dag", h.UpdateDAG)
		v1Admin.DELETE("/scenario/dag/:id", h.DeleteDAG)

		v1Admin.POST("/scenario/dag-step", h.CreateDagStep)
		v1Admin.GET("/scenario/dag-step/:id", h.GetDagStep)
		v1Admin.GET("/scenario/dag-step", h.GetAllDagStep)
		v1Admin.PUT("/scenario/dag-step", h.UpdateDagStep)
		v1Admin.DELETE("/scenario/dag-step/:id", h.DeleteDAG)

		v1Admin.POST("/scenario/category", h.CreateCategoryScenario)
		v1Admin.GET("/scenario/category/:id", h.GetCategoryScenario)
		v1Admin.GET("/scenario/category", h.GetListCategoryScenario)

		v1Admin.POST("/scenario/run", h.RunScenario)
		v1Admin.POST("/scenario", h.CreateFullScenario)
		v1Admin.PUT("/scenario", h.UpdateFullScenario) //--- update means also create but with new commit
		v1Admin.GET("/scenario/:id/history", h.GetScenarioHistory)
		v1Admin.PUT("/scenario/:id/select-versions", h.SelectVersionsScenario)
		v1Admin.POST("/scenario/revert", h.RevertScenario)

		// query service
		v1Admin.POST("/query-folder", h.CreateQueryRequestFolder)
		v1Admin.PUT("/query-folder", h.UpdateQueryRequestFolder)
		v1Admin.GET("/query-folder", h.GetListQueryRequestFolder)
		v1Admin.GET("/query-folder/:query-folder-id", h.GetSingleQueryRequestFolder)
		v1Admin.DELETE("/query-folder/:query-folder-id", h.DeleteQueryRequestFolder)

		v1Admin.POST("/query-request", h.CreateQueryRequest)
		v1Admin.PUT("/query-request", h.UpdateQueryRequest)
		v1Admin.GET("/query-request", h.GetListQueryRequest)
		v1Admin.GET("/query-request/:query-id", h.GetSingleQueryRequest)
		v1Admin.DELETE("/query-request/:query-id", h.DeleteQueryRequest)
		v1Admin.POST("/query-request/select-versions/:query-id", h.InsertManyVersionForQueryService)
		v1Admin.POST("/query-request/run", h.QueryRun)
		v1Admin.GET("/query-request/:query-id/history", h.GetQueryHistory)
		v1Admin.POST("/query-request/:query-id/revert", h.RevertQuery)
		v1Admin.GET("/query-request/:query-id/log", h.GetListQueryLog)
		v1Admin.GET("/query-request/:query-id/log/:log-id", h.GetSingleQueryLog)

		// web page service
		v1Admin.POST("/webpage-folder", h.CreateWebPageFolder)
		v1Admin.PUT("/webpage-folder", h.UpdateWebPageFolder)
		v1Admin.GET("/webpage-folder", h.GetListWebPageFolder)
		v1Admin.GET("/webpage-folder/:webpage-folder-id", h.GetSingleWebPageFolder)
		v1Admin.DELETE("/webpage-folder/:webpage-folder-id", h.DeleteWebPageFolder)

		v1Admin.POST("/webpageV2", h.CreateWebPageV2)
		v1Admin.PUT("/webpageV2", h.UpdateWebPageV2)
		v1Admin.GET("/webpageV2", h.GetListWebPageV2)
		v1Admin.GET("/webpageV2/:webpage-id", h.GetSingleWebPageV2)
		v1Admin.DELETE("/webpageV2/:webpage-id", h.DeleteWebPageV2)
		v1Admin.POST("/webpageV2/select-versions/:webpage-id", h.InsertManyVersionForWebPageService)
		v1Admin.GET("/webpageV2/:webpage-id/history", h.GetWebPageHistory)
		v1Admin.POST("/webpageV2/:webpage-id/revert", h.RevertWebPage)
	}
	v2Admin := r.Group("/v2")
	v2Admin.Use(h.AdminAuthMiddleware())
	{

		// function
		v2Admin.POST("/function", h.CreateNewFunction)
		v2Admin.GET("/function/:function_id", h.GetNewFunctionByID)
		v2Admin.GET("/function", h.GetAllNewFunctions)
		v2Admin.PUT("/function", h.UpdateNewFunction)
		v2Admin.DELETE("/function/:function_id", h.DeleteNewFunction)

		// custom event
		v2Admin.POST("/custom-event", h.CreateNewCustomEvent)
		v2Admin.GET("/custom-event/:custom_event_id", h.GetNewCustomEventByID)
		v2Admin.GET("/custom-event", h.GetAllNewCustomEvents)
		v2Admin.PUT("/custom-event", h.UpdateNewCustomEvent)
		v2Admin.DELETE("/custom-event/:custom_event_id", h.DeleteNewCustomEvent)
	}

	// v3 for ucode version 2
	v3 := r.Group("/v3")
	{
		// query folder
		v3.POST("/query_folder", h.CreateQueryFolder)
		v3.GET("/query_folder/:guid", h.GetQueryFolderByID)
		v3.GET("/query_folder", h.GetQueryFolderList)
		v3.PUT("/query_folder/:guid", h.UpdateQueryFolder)
		v3.DELETE("/query_folder/:guid", h.DeleteQueryFolder)

		// // query
		v3.POST("/query", h.CreateQuery)
		v3.GET("/query/:guid", h.GetQueryByID)
		v3.GET("/query", h.GetQueryList)
		v3.PUT("/query/:guid", h.UpdateQuery)
		v3.DELETE("/query/:guid", h.DeleteQuery)
		// // web pages
		v3.POST("/web_pages", h.CreateWebPage)
		v3.GET("/web_pages/:guid", h.GetWebPagesById)
		v3.GET("/web_pages", h.GetWebPagesList)
		v3.PUT("/web_pages/:guid", h.UpdateWebPage)
		v3.DELETE("/web_pages/:guid", h.DeleteWebPage)

		v3.POST("/chat_service", h.CreatChat)
		v3.GET("/chat_service/chatid/:chatid", h.GetListByChatId)
		v3.GET("/chat_service/projectid/:projectid", h.GetListByProjectId)

	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}

func customCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, Origin, Cache-Control, X-Requested-With, Resource-Id, Environment-Id, Platform-Type, X-API-KEY")
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
		acquire()       // before request
		defer release() // after request

		c.Set("sem", sem)
		c.Set("count_request", countReq)

		c.Next()
	}
}
