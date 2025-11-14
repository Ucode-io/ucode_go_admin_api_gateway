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

	r.GET("/v1/fare", h.V1.GetAllFares)
	r.GET("v1/chart", h.V1.GetChart)
	r.Any("v1/functions/:function-id/run", h.V1.FunctionRun)

	r.Any("/v1/transcoder/webhook", h.V1.TranscoderWebhook)

	// Real Stripe PaymentIntent endpoint
	r.POST("/stripe/webhook", h.V1.StripeWebhook)

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
		v1.POST("/upload-file/:collection/:object_id", h.V1.UploadFile)

		v1.POST("/menu/template", h.V1.CreateProjectMenuTemplate)
		v1.GET("/menu/template", h.V1.GetProjectMenuTemplates)

		// OBJECT_BUILDER_SERVICE

		//table
		v1.POST("/table", h.V1.CreateTable)
		v1.GET("/table", h.V1.GetAllTables)
		v1.GET("/table/:table_id", h.V1.GetTableByID)
		v1.POST("/table-details/:collection", h.V1.GetTableDetails)
		v1.PUT("/table", h.V1.UpdateTable)
		v1.PUT("/table/:collection/mcp", h.V1.UpdateTableByMCP)
		v1.DELETE("/table/:table_id", h.V1.DeleteTable)

		v1.POST("/connections", h.V1.CreateConnectionAndSchema)
		v1.GET("/connections/:connection_id/tables", h.V1.GetTrackedUntrackedTables)
		v1.GET("/connections", h.V1.GetTrackedConnections)
		v1.POST("/connections/:connection_id/tables/track", h.V1.TrackTablesByIds)
		v1.POST("/connections/:connection_id/tables/:table_id", h.V1.UntrackTableById)

		//view_relation
		v1.GET("/view_relation", h.V1.GetViewRelation)
		v1.PUT("/view_relation", h.V1.UpsertViewRelations)

		//object-builder
		v1.POST("/object/:collection", h.V1.CreateObject)
		v1.GET("/object/:collection/:object_id", h.V1.GetSingle)
		v1.POST("/object/get-list/:collection", h.V1.GetList)
		v1.PUT("/object/:collection", h.V1.UpdateObject)
		v1.DELETE("/object/:collection/:object_id", h.V1.DeleteObject)
		v1.DELETE("/object/:collection", h.V1.DeleteManyObject)
		v1.POST("/object/excel/:collection", h.V1.GetListInExcel)
		v1.POST("/object-upsert/:collection", h.V1.UpsertObject)
		v1.PUT("/object/multiple-update/:collection", h.V1.MultipleUpdateObject)
		v1.POST("/object/get-list-aggregate/:collection", h.V1.GetListAggregate)

		//many-to-many
		v1.PUT("/many-to-many", h.V1.AppendManyToMany)
		v1.DELETE("/many-to-many", h.V1.DeleteManyToMany)

		//view
		v1.POST("/html-template", h.V1.CreateHtmlTemplate)
		v1.GET("/html-template/:html_template_id", h.V1.GetSingleHtmlTemplate)
		v1.GET("/html-template", h.V1.GetHtmlTemplateList)
		v1.PUT("/html-template", h.V1.UpdateHtmlTemplate)
		v1.DELETE("/html-template/:html_template_id", h.V1.DeleteHtmlTemplate)

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

		// template
		v1.POST("/template", h.V1.CreateTemplate)
		v1.GET("/template/:template-id", h.V1.GetSingleTemplate)
		v1.DELETE("/template/:template-id", h.V1.DeleteTemplate)
		v1.GET("/template", h.V1.GetListTemplate)
		v1.POST("/template/execute", h.V1.ExecuteTemplate)

		// HTML TO PDF CONVERTER
		v1.POST("/html-to-pdf", h.V1.ConvertHtmlToPdf)
		v1.POST("/template-to-html", h.V1.ConvertTemplateToHtml)

		v1.POST("/files/folder_upload", h.V1.UploadToFolder)
		v1.GET("/files/:id", h.V1.GetSingleFile)
		v1.PUT("/files", h.V1.UpdateFile)
		v1.DELETE("/files", h.V1.DeleteFiles)
		v1.DELETE("/files/:id", h.V1.DeleteFile)
		v1.GET("/files", h.V1.GetAllFiles)
		v1.POST("/files/word-template", h.V1.WordTemplate)

		v1.POST("/language", h.V1.CreateLanguage)
		v1.GET("/language/:id", h.V1.GetByIdLanguage)
		v1.GET("/language", h.V1.GetListLanguage)
		v1.PUT("/language", h.V1.UpdateLanguage)
		v1.DELETE("/language/:id", h.V1.DeleteLanguage)

		fare := v1.Group("/fare")
		{
			fare.POST("", h.V1.CreateFare)
			fare.GET("/:id", h.V1.GetFare)
			fare.PUT("", h.V1.UpdateFare)
			fare.DELETE("/:id", h.V1.DeleteFare)
			fare.POST("/calculate-price", h.V1.CalculatePrice)

			fareItem := fare.Group("/item")
			{
				fareItem.POST("", h.V1.CreateFareItem)
				fareItem.GET("", h.V1.GetAllFareItem)
				fareItem.GET("/:id", h.V1.GetFareItem)
				fareItem.PUT("", h.V1.UpdateFareItem)
				fareItem.DELETE("/:id", h.V1.DeleteFareItem)
			}
		}
		transaction := v1.Group("/transaction")
		{
			transaction.POST("", h.V1.CreateTransaction)
			transaction.GET("", h.V1.GetAllTransactions)
			transaction.GET("/:id", h.V1.GetTransaction)
			transaction.PUT("", h.V1.UpdateTransaction)
		}
		payment := v1.Group("/payment")
		{
			payment.POST("/intent", h.V1.CreatePaymentIntent)
			payment.POST("/get-verify-code", h.V1.GetVerifyCode)
			payment.POST("/verify", h.V1.Verify)
			payment.GET("/card-list", h.V1.GetAllProjectCards)
			payment.POST("/receipt-pay", h.V1.ReceiptPay)
			payment.DELETE("/card/:id", h.V1.DeleteProjectCard)
		}
		discount := v1.Group("/discounts")
		{
			discount.GET("", h.V1.ListDiscounts)
		}

		metabase := v1.Group("/metabase")
		{
			metabase.POST("/dashboard", h.V1.GetMetabaseDashboards)
			metabase.POST("/public-url", h.V1.GetMetabasePublicUrl)
		}

		transcoder := v1.Group("/transcoder")
		{
			transcoder.GET("/pipeline", h.V1.GetListPipeline)
		}

		v1.PUT("/subscription", h.V1.UpdateSubscriptionEndDate)
	}

	v2 := r.Group("/v2")
	v2.Use(h.V1.AuthMiddleware(cfg))
	{
		v2.POST("/object/get-list/:collection", h.V1.GetListV2)
		v2.PUT("/update-with/:collection", h.V1.UpdateWithParams)

	}

	v1Slim := r.Group("/v1")
	v1Slim.Use(h.V1.SlimAuthMiddleware(cfg))
	{
		v1Slim.GET("/object-slim/:collection/:object_id", h.V1.GetSingleSlim)
		v1Slim.GET("/object-slim/get-list/:collection", h.V1.GetListSlim)
	}

	v2Slim := r.Group("/v2")
	v2Slim.Use(h.V1.SlimAuthMiddleware(cfg))
	{
		v2Slim.GET("/object-slim/get-list/:collection", h.V1.GetListSlimV2)
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

		company := v1Admin.Group("/companies")
		{
			company.GET("", h.V1.ListCompanies)
			company.POST("", h.V1.CreateCompany)
			company.GET("/:company_id", h.V1.GetCompanyByID)
			company.PUT("/:company_id", h.V1.UpdateCompany)
			company.DELETE("/:company_id", h.V1.DeleteCompany)
			company.GET("/:company_id/projects", h.V1.ListCompanyProjects)
			company.POST("/:company_id/projects", h.V1.CreateCompanyProject)
		}

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
		v1Admin.PATCH("/company/project/attach-fare", h.V1.AttachFareToProject)

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

		v1Admin.POST("/redirect-url", h.V1.CreateRedirectUrl)
		v1Admin.PUT("/redirect-url", h.V1.UpdateRedirectUrl)
		v1Admin.GET("/redirect-url", h.V1.GetListRedirectUrl)
		v1Admin.GET("/redirect-url/:redirect-url-id", h.V1.GetSingleRedirectUrl)
		v1Admin.DELETE("/redirect-url/:redirect-url-id", h.V1.DeleteRedirectUrl)
		v1Admin.PUT("/redirect-url/re-order", h.V1.UpdateRedirectUrlOrder)

		v1Admin.POST("dbml-to-ucode", h.V1.DbmlToUcode)
		v1Admin.POST("mcp-call", h.V1.MCPCall)
	}

	v2Admin := r.Group("/v2")
	v2Admin.Use(h.V1.AdminAuthMiddleware())

	function := v2Admin.Group("/functions")
	{
		function.Any("/:function-id/run", h.V1.FunctionRun)
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
	}

	{
		// docx-template v2
		v2Admin.POST("/docx-template", h.V2.CreateDocxTemplate)
		v2Admin.GET("/docx-template/:docx-template-id", h.V2.GetSingleDocxTemplate)
		v2Admin.PUT("/docx-template", h.V2.UpdateDocxTemplate)
		v2Admin.DELETE("/docx-template/:docx-template-id", h.V2.DeleteDocxTemplate)
		v2Admin.GET("/docx-template", h.V2.GetListDocxTemplate)
		v2Admin.GET("/docx-template/fields/list", h.V2.GetAllFieldsDocxTemplate)
		// docx-constructor
		v2Admin.POST("/docx-template/convert/pdf", h.V2.ConvertDocxToPdf)
		// HTML TO PDF CONVERTER
		v2Admin.POST("/html/convert", h.V2.ConvertHtmlToDocxOrPdf)
	}

	clientV2 := r.Group("/v2")
	clientV2.Use(h.V2.AuthMiddleware())
	// items group
	v2Items := clientV2.Group("/items")
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
		v2Items.PUT("/many-to-many", h.V2.AppendManyToMany)                  // TODO test
		v2Items.DELETE("/many-to-many", h.V2.DeleteManyToMany)               // TODO test
		v2Items.PUT("/update-row/:collection", h.V2.UpdateRowOrder)          // TODO test
		v2Items.POST("/:collection/tree", h.V2.AgTree)                       // TODO test
		v2Items.POST("/:collection/board/structure", h.V2.GetBoardStructure) // TODO test
		v2Items.POST("/:collection/board", h.V2.GetBoardData)                // TODO test
	}

	v2Version := r.Group("/v2")
	v2Version.Use(h.V1.AuthMiddleware(cfg))
	{
		v2Version.POST("/csv/:collection/download", h.V2.GetListInCSV)
		v2Version.POST("/send-to-gpt", h.V2.SendToGpt)

		// collections group
		v2Collection := v2Version.Group("/collections")
		{
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

		// menu group
		v2Menus := v2Version.Group("/menus")
		{
			v2Menus.GET("", h.V2.GetAllMenus)
			v2Menus.GET("/:id", h.V2.GetMenuByID)
			v2Menus.PUT("", h.V2.UpdateMenu)
			v2Menus.POST("", h.V2.CreateMenu)
			v2Menus.DELETE("/:id", h.V2.DeleteMenu)
			v2Menus.PUT("/menu-order", h.V2.UpdateMenuOrder)
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
			v2Fields.POST("/:collection", h.V2.CreateField)
			v2Fields.GET("/:collection", h.V2.GetAllFields)
			v2Fields.GET("/:collection/with-relations", h.V2.FieldsWithPermissions)
			v2Fields.PUT("/:collection", h.V2.UpdateField)
			v2Fields.PUT("/:collection/update-search", h.V2.UpdateSearch)
			v2Fields.DELETE("/:collection/:id", h.V2.DeleteField)
		}

		// relations
		v2Relations := v2Version.Group("/relations")
		{
			v2Relations.POST("/:collection", h.V2.CreateRelation)
			v2Relations.GET("/:collection/:id", h.V2.GetByIdRelation)
			v2Relations.GET("/:collection", h.V2.GetAllRelations)
			v2Relations.GET("/:collection/cascading", h.V2.GetRelationCascading)
			v2Relations.PUT("/:collection", h.V2.UpdateRelation)
			v2Relations.DELETE("/:collection/:id", h.V2.DeleteRelation)
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
			v2Version.GET("/history/:environment_id/excel", h.V2.VersionHistoryExcelDownload)
		}
	}

	github := r.Group("/v1/github")
	{
		github.GET("/login", h.V2.GithubLogin)
		github.GET("/user", h.V2.GithubGetUser)
		github.GET("/repos", h.V2.GithubGetRepos)
		github.GET("/branches", h.V2.GithubGetBranches)
	}

	gitlab := r.Group("/v1/gitlab")
	{
		gitlab.GET("/login", h.V2.GitlabLogin)
		gitlab.GET("/user", h.V2.GitlabGetUser)
		gitlab.GET("/repos", h.V2.GitlabGetRepos)
		gitlab.GET("/branches", h.V2.GitlabGetBranches)
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

	proxyFunctions := proxyApi.Group("functions")
	{
		proxyFunctions.POST("/micro-frontend", h.V1.CreateMicroFrontEnd)
		proxyFunctions.GET("/micro-frontend/:micro-frontend-id", h.V1.GetMicroFrontEndByID)
		proxyFunctions.GET("/micro-frontend", h.V1.GetAllMicroFrontEnd)
		proxyFunctions.PUT("/micro-frontend", h.V1.UpdateMicroFrontEnd)
		proxyFunctions.DELETE("/micro-frontend/:micro-frontend-id", h.V1.DeleteMicroFrontEnd)
	}

	proxyGrafana := proxyApi.Group("/grafana")
	{
		proxyGrafana.POST("/loki", h.V2.GetGrafanaFunctionLogs)
		proxyGrafana.GET("/function", h.V2.GetGrafanaFunctionList)
	}

	{
		proxyApi.POST("/invoke_function/:function-path", h.V2.InvokeFunctionByPath)
		proxyApi.POST("/invoke_function/:function-path/*any", h.V2.InvokeFunctionByPath)

		v2Webhook := proxyApi.Group("/webhook")
		{
			v2Webhook.POST("/create", h.V2.CreateWebhook)
			v2Webhook.POST("/handle", h.V2.HandleWebhook)

		}
	}

	v3 := r.Group("/v3")
	v3.Use(h.V1.AuthMiddleware(cfg))
	v3Menus := v3.Group("/menus")
	{
		v3Menus.GET("", h.V3.GetAllMenus)
		v3Menus.GET("/:menu_id", h.V3.GetMenuByID)
		v3Menus.PUT("", h.V3.UpdateMenu)
		v3Menus.POST("", h.V3.CreateMenu)
		v3Menus.DELETE("/:menu_id", h.V3.DeleteMenu)
		v3Menus.PUT("/order", h.V3.UpdateMenuOrder)
		v3Menus.PUT("/menu-order", h.V3.UpdateMenuOrder)

		v3views := v3Menus.Group("/:menu_id/views")
		{
			v3views.GET("", h.V3.GetAllViews)
			v3views.POST("", h.V3.CreateView)
			v3views.PUT("", h.V3.UpdateView)
			v3views.DELETE("/:view_id", h.V3.DeleteView)
			v3views.PUT("/update-order", h.V3.UpdateViewOrder)

			v3table := v3views.Group("/:view_id/tables")
			{
				v3table.POST("", h.V3.CreateTable)
				v3table.GET("", h.V3.GetAllTables)
				v3table.GET("/:collection", h.V3.GetTableByID)
				v3table.POST("/:collection", h.V3.GetTableDetails)
				v3table.PUT("", h.V3.UpdateTable)
				v3table.DELETE("/:collection", h.V3.DeleteTable)

				v3Layout := v3table.Group("/:collection/layout")
				{
					// layout
					v3Layout.GET("", h.V3.GetListLayouts)
					v3Layout.PUT("", h.V3.UpdateLayout)
					v3Layout.POST("", h.V3.GetSingleLayout)
					v3Layout.DELETE("/:id", h.V3.DeleteLayout)
				}

				v3items := v3table.Group("/:collection/items")
				{
					v3items.POST("/list", h.V3.GetListV2)
					v3items.GET("/:id", h.V3.GetSingleItem)
					v3items.POST("", h.V3.CreateItem)
					v3items.POST("/multiple-insert", h.V3.CreateItems)
					v3items.POST("/upsert-many", h.V3.UpsertMany)
					v3items.PUT("", h.V3.UpdateItem)
					v3items.PUT("/:id", h.V3.UpdateItem)
					v3items.PATCH("", h.V3.MultipleUpdateItems)
					v3items.PATCH("/:id", h.V3.UpdateItem)
					v3items.DELETE("", h.V3.DeleteItems)
					v3items.DELETE("/:id", h.V3.DeleteItem)
					v3items.POST("/aggregation", h.V3.GetListAggregation)
					v3items.PUT("/many-to-many", h.V3.AppendManyToMany)
					v3items.DELETE("/many-to-many", h.V3.DeleteManyToMany)
					v3items.PUT("/update-row", h.V3.UpdateRowOrder)
					v3items.POST("/tree", h.V3.AgTree)
				}
			}
		}
	}

	knative := r.Group("/v1/knative")
	knative.POST("/:function-path/without-auth", h.V2.InvokeInAdminWithoutAuth)
	knative.POST("/:function-path/without-data", h.V2.InvokeInAdminWithoutData)
	knative.POST("/:function-path/proxy/noauth", h.V2.InvokeInAdminProxyWithoutAuth)
	knative.Use(h.V1.AuthMiddleware(cfg))
	{
		knative.POST("/:function-path", h.V2.InvokeInAdmin)
		knative.POST("/:function-path/auth-data", h.V2.InvokeInAdminAuthData)
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
