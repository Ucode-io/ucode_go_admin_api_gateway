package api

import (
	"ucode/ucode_go_api_gateway/api/docs"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"

	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// @description This is a api gateway
// @termsOfService https://udevs.io
func SetUpAPI(r *gin.Engine, h handlers.Handler, cfg config.Config) {
	docs.SwaggerInfo.Title = cfg.ServiceName
	docs.SwaggerInfo.Version = cfg.Version
	// docs.SwaggerInfo.Host = cfg.ServiceHost + cfg.HTTPPort
	docs.SwaggerInfo.Schemes = []string{cfg.HTTPScheme}

	r.Use(customCORSMiddleware())
	r.Use(MaxAllowed(5000))

	r.GET("/ping", h.Ping)
	r.GET("/config", h.GetConfig)

	// Project
	r.POST("/v1/project", h.CreateProject)
	r.GET("/v1/project", h.GetAllProjects)
	r.DELETE("/v1/project/:project_id", h.DeleteProject)

	v1 := r.Group("/v1")
	// @securityDefinitions.apikey ApiKeyAuth
	// @in header
	// @name Authorization
	// MUST be executed before AuthMiddleware
	v1.Use(h.ProjectsMiddleware())
	v1.Use(h.AuthMiddleware())
	{
		v1.POST("/upload", h.Upload)
		v1.POST("/upload-file/:table_slug/:object_id", h.UploadFile)

		// OBJECT_BUILDER_SERVICE

		//table
		v1.POST("/table", h.CreateTable)
		v1.GET("/table/:table_id", h.GetTableByID)
		v1.GET("/table", h.GetAllTables)
		v1.PUT("/table", h.UpdateTable)
		v1.DELETE("/table/:table_id", h.DeleteTable)
		//field
		v1.POST("/field", h.CreateField)
		v1.GET("/field", h.GetAllFields)
		v1.PUT("/field", h.UpdateField)
		v1.DELETE("/field/:field_id", h.DeleteField)

		//relation
		v1.POST("/relation", h.CreateRelation)
		v1.GET("/relation", h.GetAllRelations)
		v1.PUT("/relation", h.UpdateRelation)
		v1.DELETE("/relation/:relation_id", h.DeleteRelation)

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
		// permission
		v1.POST("/permission-upsert/:app_id", h.UpsertPermissionsByAppId)
		v1.GET("/permission-get-all/:role_id", h.GetAllPermissionByRoleId)
		v1.GET("/field-permission/:role_id/:table_slug", h.GetFieldPermissions)

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

		// HTML TO PDF CONVERTER
		v1.POST("/html-to-pdf", h.ConvertHtmlToPdf)
		v1.POST("/template-to-html", h.ConvertTemplateToHtml)

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

		// Integration with AlfaLab
		v1.POST("/alfalab/directions", h.CreateDirections)
		v1.GET("/alfalab/referral", h.GetReferral)

		// company service
		v1.POST("/company", h.CreateCompany)
		v1.GET("company/:company_id", h.GetCompanyByID)
		v1.GET("company", h.GetCompanyList)
		v1.PUT("company/:company_id", h.UpdateCompany)
		v1.DELETE("company/:company_id", h.DeleteCompany)

		// project service
		v1.POST("company-project", h.CreateCompanyProject)
	    v1.GET("company-project", h.GetCompanyProjectList)
		v1.GET("company-project/:project_id", h.GetCompanyProjectById)
		v1.PUT("company-project/:project_id", h.UpdateCompanyProject)
		v1.DELETE("company-project/:project_id", h.DeleteCompanyProject)
	}
	v2 := r.Group("/v2")
	{
		v2.POST("/client-platform", h.V2CreateClientPlatform)
		v2.GET("/client-platform", h.V2GetClientPlatformList)
		v2.GET("/client-platform/:client-platform-id", h.V2GetClientPlatformByID)
		v2.GET("/client-platform-detailed/:client-platform-id", h.V2GetClientPlatformByIDDetailed)
		v2.PUT("/client-platform", h.V2UpdateClientPlatform)
		v2.DELETE("/client-platform/:client-platform-id", h.V2DeleteClientPlatform)

		// admin, dev, hr, ceo
		v2.POST("/client-type", h.V2CreateClientType)
		v2.GET("/client-type", h.V2GetClientTypeList)
		v2.GET("/client-type/:client-type-id", h.V2GetClientTypeByID)
		v2.PUT("/client-type", h.V2UpdateClientType)
		v2.DELETE("/client-type/:client-type-id", h.V2DeleteClientType)

		v2.POST("/client", h.V2AddClient)
		v2.GET("/client/:project-id", h.V2GetClientMatrix)
		v2.PUT("/client", h.V2UpdateClient)
		v2.DELETE("/client", h.V2RemoveClient)

		v2.POST("/user-info-field", h.V2AddUserInfoField)
		v2.PUT("/user-info-field", h.V2UpdateUserInfoField)
		v2.DELETE("/user-info-field/:user-info-field-id", h.V2RemoveUserInfoField)

		// PERMISSION SERVICE
		v2.GET("/role/:role-id", h.V2GetRoleByID)
		v2.GET("/role", h.V2GetRolesList)
		v2.POST("/role", h.V2AddRole)
		v2.PUT("/role", h.V2UpdateRole)
		v2.DELETE("/role/:role-id", h.V2RemoveRole)

		v2.POST("/permission", h.V2CreatePermission)
		v2.GET("/permission", h.V2GetPermissionList)
		v2.GET("/permission/:permission-id", h.V2GetPermissionByID)
		v2.PUT("/permission", h.V2UpdatePermission)
		v2.DELETE("/permission/:permission-id", h.V2DeletePermission)

		v2.POST("/permission-scope", h.V2AddPermissionScope)
		v2.DELETE("/permission-scope", h.V2RemovePermissionScope)

		v2.POST("/role-permission", h.V2AddRolePermission)
		v2.DELETE("/role-permission", h.V2RemoveRolePermission)

		v2.POST("/user", h.V2CreateUser)
		v2.GET("/user", h.V2GetUserList)
		v2.GET("/user/:user-id", h.V2GetUserByID)
		v2.PUT("/user", h.V2UpdateUser)
		v2.DELETE("/user/:user-id", h.V2DeleteUser)
		v2.POST("/login", h.V2Login)
		v2.PUT("/refresh", h.V2RefreshToken)
	}
	r.POST("/send-code", h.SendCode)
	r.POST("/verify/:sms_id/:otp", h.Verify)
	r.POST("/register-otp/:table_slug", h.RegisterOtp)
	r.POST("/send-message", h.SendMessageToEmail)
	r.POST("/verify-email/:sms_id/:otp", h.VerifyEmail)
	r.POST("/register-email-otp/:table_slug", h.RegisterEmailOtp)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}

func customCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
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

// // @description This is a api gateway
// // @termsOfService https://udevs.io
// func SetUpProjectAPIs(r *gin.Engine, h handlers.ProjectsHandler, cfg config.Config) {
// 	docs.SwaggerInfo.Title = cfg.ServiceName
// 	docs.SwaggerInfo.Version = cfg.Version
// 	// docs.SwaggerInfo.Host = cfg.ServiceHost + cfg.HTTPPort
// 	docs.SwaggerInfo.Schemes = []string{cfg.HTTPScheme}

// 	r.Use(customCORSMiddleware())
// 	r.Use(MaxAllowed(5000))

// // Project
// r.POST("/v1/project", h.CreateProject)

// 	v1 := r.Group("/v1")
// 	// @securityDefinitions.apikey ApiKeyAuth
// 	// @in header
// 	// @name Authorization

// // MUST be executed before AuthMiddleware
// v1.Use(h.ProjectsMiddleware())
// 	v1.Use(h.AuthMiddleware())

// 	{
// 		// App
// 		v1.POST("/app", h.CreateApp)
// 		v1.GET("/app", h.GetAllApps)

// 	}

// 	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

// }
