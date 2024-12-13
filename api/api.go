package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ucode/ucode_go_api_gateway/api/docs"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"github.com/opentracing-contrib/go-gin/ginhttp"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cast"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// SetUpAPI @description This is an api gateway
// @termsOfService https://u-code.io/
func SetUpAPI(r *gin.Engine, h handlers.Handler, cfg config.BaseConfig, tracer opentracing.Tracer) {
	docs.SwaggerInfo.Title = cfg.ServiceName
	docs.SwaggerInfo.Version = cfg.Version
	docs.SwaggerInfo.Schemes = []string{cfg.HTTPScheme}

	r.Use(customCORSMiddleware())
	r.Use(ginhttp.Middleware(tracer))

	r.GET("/ping", h.V1.Ping)
	r.Any("/v2/upload-file/*any", gin.WrapH(http.StripPrefix("/v2/upload-file/", h.V2.MovieUpload())))

	r.Any("/x-api/*any", h.V1.RedirectAuthMiddleware(cfg), proxyMiddleware(r, &h), h.V1.Proxy)

	r.GET("/v1/login-microfront", h.V1.GetLoginMicroFrontBySubdomain)

	r.GET("/menu/wiki_folder", h.V1.GetWikiFolder)

	global := r.Group("/v1/global")
	global.Use(h.V1.GlobalAuthMiddleware(cfg))
	{
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

		// MINIO
		v1.POST("/minio/bucket-size", h.V1.BucketSize)

		v1.POST("/menu-template", h.V1.CreateMenuTemplate)
		v1.PUT("/menu-template", h.V1.UpdateMenuTemplate)
		v1.GET("/menu-template", h.V1.GetAllMenuTemplates)
		v1.GET("/menu-template/:id", h.V1.GetMenuTemplateByID)
		v1.DELETE("/menu-template/:id", h.V1.DeleteMenuTemplate)

		v1.POST("/upload", h.V1.Upload)
		v1.POST("/upload-file/:table_slug/:object_id", h.V1.UploadFile)

		// OBJECT_BUILDER_SERVICE

		//table
		v1.POST("/table", h.V1.CreateTable)
		v1.GET("/table", h.V1.GetAllTables)
		v1.GET("/table/:table_id", h.V1.GetTableByID)
		v1.POST("/table-details/:table_slug", h.V1.GetTableDetails)

		v1.PUT("/table", h.V1.UpdateTable)
		v1.DELETE("/table/:table_id", h.V1.DeleteTable)

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

		// function
		v1.POST("/function", h.V1.CreateFunction)
		v1.GET("/function/:function_id", h.V1.GetFunctionByID)
		v1.PUT("/function", h.V1.UpdateFunction)
		v1.DELETE("/function/:function_id", h.V1.DeleteFunction)

		// INVOKE FUNCTION

		v1.POST("/invoke_function", h.V1.InvokeFunction)
		v1.POST("/invoke_function/:function-path", h.V1.InvokeFunctionByPath)

		//cache
		v1.POST("/cache", h.V1.Cache)

		// Excel Reader
		v1.GET("/excel/:excel_id", h.V1.ExcelReader)
		v1.POST("/excel/excel_to_db/:excel_id", h.V1.ExcelToDb)
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

		v1.POST("/template-note/users", h.V1.CreateUserTemplate)
		v1.GET("/template-note/users", h.V1.GetListUserTemplate)
		v1.PUT("/template-note/users", h.V1.UpdateUserTemplate)
		v1.DELETE("/template-note/users/:user-permission-id", h.V1.DeleteUserTemplate)
		v1.POST("/template-note/share", h.V1.CreateSharingToken)
		v1.PUT("/template-note/share", h.V1.UpdateSharingToken)

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

		v1.POST("/files/folder_upload", h.V1.UploadToFolder)
		v1.GET("/files/:id", h.V1.GetSingleFile)
		v1.PUT("/files", h.V1.UpdateFile)
		v1.DELETE("/files", h.V1.DeleteFiles)
		v1.DELETE("/files/:id", h.V1.DeleteFile)
		v1.GET("/files", h.V1.GetAllFiles)
		v1.POST("/files/word-template", h.V1.WordTemplate)
	}
	v2 := r.Group("/v2")
	v2.Use(h.V1.AuthMiddleware(cfg))
	{

		v2.GET("/language-json", h.V1.GetLanguageJson)

		v2.POST("/object/get-list/:table_slug", h.V1.GetListV2)

		v2.PUT("/update-with/:collection", h.V1.UpdateWithParams)

		v2.POST("/erd", h.V2.UploadERD)

	}
	r.POST("/template-note/share-get", h.V1.GetObjectToken)

	v1Slim := r.Group("/v1")
	v1Slim.Use(h.V1.SlimAuthMiddleware(cfg))
	{
		v1Slim.GET("/object-slim/:table_slug/:object_id", h.V1.GetSingleSlim)
		v1.GET("/object-slim/get-list/:table_slug", h.V1.GetListSlim)
	}

	v2Slim := r.Group("/v2")
	v2Slim.Use(h.V1.SlimAuthMiddleware(cfg))
	{
		v2Slim.GET("/object-slim/get-list/:table_slug", h.V1.GetListSlimV2)
	}

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

		// airbyte
		v1Admin.GET("/company/airbyte/:id", h.V1.GetByIdAirbyte)
		v1Admin.POST("/company/airbyte", h.V1.GetListAirbyte)

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

		// function folder
		v1Admin.POST("/function-folder", h.V1.CreateFunctionFolder)
		v1Admin.GET("/function-folder/:function_ifolder_d", h.V1.GetFunctionFolderById)
		v1Admin.GET("/function-folder", h.V1.GetAllFunctionFolder)
		v1Admin.PUT("/function-folder", h.V1.UpdateFunctionFolder)
		v1Admin.DELETE("/function-folder/:function_folder_id", h.V1.DeleteFunctionFolder)

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
		function.Any("/:function-id/invoke", h.V1.FunctionRun)
	}

	{
		// project resource /rest
		projectResource := v2Admin.Group("/company/project/resource")
		{
			projectResource.POST("", h.V1.AddResourceToProject)
			projectResource.PUT("", h.V1.UpdateProjectResource)
			projectResource.GET("", h.V1.GetListProjectResourceList)
			projectResource.GET("/:id", h.V1.GetSingleProjectResource)
			projectResource.DELETE("/:id", h.V1.DeleteProjectResource)
		}

		functions := v2Admin.Group("functions")
		{
			functions.POST("/micro-frontend", h.V1.CreateMicroFrontEnd)
			functions.GET("/micro-frontend/:micro-frontend-id", h.V1.GetMicroFrontEndByID)
			functions.GET("/micro-frontend", h.V1.GetAllMicroFrontEnd)
			functions.PUT("/micro-frontend", h.V1.UpdateMicroFrontEnd)
			functions.DELETE("/micro-frontend/:micro-frontend-id", h.V1.DeleteMicroFrontEnd)
		}
	}

	{
		// docx-template v2
		v2Admin.POST("/docx-template", h.V2.CreateDocxTemplate)
		v2Admin.GET("/docx-template/:docx-template-id", h.V2.GetSingleDocxTemplate)
		v2Admin.PUT("/docx-template", h.V2.UpdateDocxTemplate)
		v2Admin.DELETE("/docx-template/:docx-template-id", h.V2.DeleteDocxTemplate)
		v2Admin.GET("/docx-template", h.V2.GetListDocxTemplate)
		v2Admin.GET("/docx-template/fields/list", h.V2.GetAllFieldsDocxTemplate)
	}

	v2Version := r.Group("/v2")
	v2Version.Use(h.V2.AuthMiddleware())
	{
		v2Version.POST("/csv/:table_slug/download", h.V2.GetListInCSV)
		v2Version.POST("/send-to-gpt", h.V2.SendToGpt)

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
			v2Items.POST("/:collection/upsert-many", h.V2.UpsertMany)
			v2Items.PUT("/:collection", h.V2.UpdateItem)
			v2Items.PUT("/:collection/:id", h.V2.UpdateItem)
			v2Items.PATCH("/:collection", h.V2.MultipleUpdateItems)
			v2Items.PATCH("/:collection/:id", h.V2.UpdateItem)
			v2Items.DELETE("/:collection", h.V2.DeleteItems)
			v2Items.DELETE("/:collection/:id", h.V2.DeleteItem)
			v2Items.POST("/:collection/aggregation", h.V2.GetListAggregation)
			v2Items.PUT("/many-to-many", h.V2.AppendManyToMany)
			v2Items.DELETE("/many-to-many", h.V2.DeleteManyToMany)
			v2Items.PUT("/update-row/:collection", h.V2.UpdateRowOrder)
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
		v2Fields := v2Version.Group("/fields")
		{
			v2Fields.GET("/:collection", h.V2.GetAllFields)
			v2Fields.POST("/:collection", h.V2.CreateField)
			v2Fields.PUT("/:collection", h.V2.UpdateField)
			v2Fields.PUT("/:collection/update-search", h.V2.UpdateSearch)
			v2Fields.DELETE("/:collection/:id", h.V2.DeleteField)
			v2Fields.GET("/:collection/with-relations", h.V2.FieldsWithPermissions)
		}

		// relations
		v2Relations := v2Version.Group("/relations")
		{
			v2Relations.GET("/:collection/:id", h.V2.GetByIdRelation)
			v2Relations.GET("/:collection", h.V2.GetAllRelations)
			v2Relations.POST("/:collection", h.V2.CreateRelation)
			v2Relations.PUT("/:collection", h.V2.UpdateRelation)
			v2Relations.DELETE("/:collection/:id", h.V2.DeleteRelation)
			v2Relations.GET("/:collection/cascading", h.V2.GetRelationCascading)
		}

		// utils
		v2Utils := v2Version.Group("/utils")
		{
			v2Utils.GET("/barcode/:collection/:type", h.V2.GetGeneratedBarcode)
			v2Utils.POST("/export/:collection/html-to-pdf", h.V2.ConvertHtmlToPdf)
			v2Utils.POST("/export/:collection/template-to-html", h.V2.ConvertTemplateToHtml)
			v2Utils.POST("/export/:collection", h.V2.ExportData)
		}

		// folder groups
		v2FolderGroups := v2Version.Group("folder-group")
		{
			v2FolderGroups.POST("", h.V2.CreateFolderGroup)
			v2FolderGroups.GET("/:id", h.V2.GetFolderGroupById)
			v2FolderGroups.GET("", h.V2.GetAllFolderGroups)
			v2FolderGroups.PUT("", h.V2.UpdateFolderGroup)
			v2FolderGroups.DELETE("/:id", h.V2.DeleteFolderGroup)
		}

		v2Files := v2Version.Group("/files")
		{
			v2Files.POST("", h.V2.UploadToFolder)
			v2Files.GET("/:id", h.V2.GetSingleFile)
			v2Files.PUT("/:id/upload", h.V2.UpdateFile)
			v2Files.DELETE("", h.V2.DeleteFiles)
			v2Files.DELETE("/:id/upload", h.V2.DeleteFile)
			v2Files.GET("", h.V2.GetAllFiles)
		}

		v2Version := v2Version.Group("/version")
		{
			v2Version.POST("", h.V2.CreateVersion)
			v2Version.PUT("", h.V2.UpdateVersion)
			v2Version.GET("", h.V2.GetVersionList)
			v2Version.POST("/publish", h.V2.PublishVersion)
			v2Version.GET("/history/:environment_id", h.V2.GetAllVersionHistory)
			v2Version.PUT("/history/:environment_id", h.V2.UpdateVersionHistory)
			v2Version.GET("/history/:environment_id/:id", h.V2.GetVersionHistoryByID)
			v2Version.POST("/history/migrate/up/:environment_id", h.V2.MigrateUp)
			v2Version.POST("/history/migrate/down/:environment_id", h.V2.MigrateDown)
		}
	}

	github := r.Group("/github")
	{
		github.GET("/login", h.V2.GithubLogin)
		github.GET("/user", h.V2.GithubGetUser)
		github.GET("/repos", h.V2.GithubGetRepos)
		github.GET("/branches", h.V2.GithubGetBranches)
	}

	proxyApi := r.Group("/v2")

	proxyFunction := proxyApi.Group("/function")
	{
		proxyFunction.POST("", h.V1.CreateNewFunction)
		proxyFunction.GET("/:function_id", h.V1.GetNewFunctionByID)
		proxyFunction.GET("", h.V1.GetAllNewFunctions)
		proxyFunction.PUT("", h.V1.UpdateNewFunction)
		proxyFunction.DELETE("/:function_id", h.V1.DeleteNewFunction)

	}

	{
		proxyApi.POST("/invoke_function/:function-path", h.V2.InvokeFunctionByPath)

		v2Webhook := proxyApi.Group("/webhook")
		{
			v2Webhook.POST("/create", h.V2.CreateWebhook)
			v2Webhook.POST("/handle", h.V2.HandleWebhook)

		}
	}

	r.Any("/api/*any", h.V1.AuthMiddleware(cfg), proxyMiddleware(r, &h), h.V1.Proxy)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}

func customCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "3600")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Headers", "*")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func proxyMiddleware(r *gin.Engine, h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error

		c, err = RedirectUrl(c, h)
		if err == nil {
			r.HandleContext(c)
		}
		c.Next()
	}
}

func RedirectUrl(c *gin.Context, h *handlers.Handler) (*gin.Context, error) {
	var path = c.Request.URL.Path

	projectId, ok := c.Get("project_id")
	if !ok {
		return c, errors.New("something went wrong")
	}

	envId, ok := c.Get("environment_id")
	if !ok {
		return c, errors.New("something went wrong")
	}

	c.Request.Header.Add("prev_path", path)
	var data = helper.MatchingData{
		ProjectId: projectId.(string),
		EnvId:     envId.(string),
		Path:      path,
	}

	res, err := h.V1.CompanyRedirectGetList(data, h.GetCompanyService(c))
	if err != nil {
		return c, errors.New("cant change")
	}

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
	c.Request.Header.Add("resource", cast.ToString(c.Value("resource")))
	c.Request.Header.Add("redirect", cast.ToString(true))

	auth, err := json.Marshal(c.Value("auth"))
	if err != nil {
		return c, errors.New("something went wrong")
	}

	c.Request.Header.Add("auth", string(auth))

	return c, nil
}
