package handlers

import (
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

// CreateProject godoc
// @Security ApiKeyAuth
// @ID create_project
// @Router /v1/project [POST]
// @Summary Create project
// @Description Create project
// @Tags Project
// @Accept json
// @Produce json
// @Param project body models.CreateProjectRequest true "CreateProjectRequestBody"
// @Success 201 {object} http.Response{data=models.CreateProjectResponse} "CreateProjectResponseBody"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *ProjectsHandler) CreateProject(c *gin.Context) {
	var project models.CreateProjectRequest

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	conf := config.Config{}

	// conf.ServiceName = cast.ToString(config.GetOrReturnDefaultValue("SERVICE_NAME", "ucode_go_api_gateway"))
	// conf.Environment = cast.ToString(config.GetOrReturnDefaultValue("ENVIRONMENT", config.DebugMode))
	// conf.Version = cast.ToString(config.GetOrReturnDefaultValue("VERSION", "1.0"))

	// conf.ServiceHost = cast.ToString(config.GetOrReturnDefaultValue("SERVICE_HOST", "0.0.0.0"))
	// conf.HTTPPort = cast.ToString(config.GetOrReturnDefaultValue("HTTP_PORT", ":8001"))
	// conf.HTTPScheme = cast.ToString(config.GetOrReturnDefaultValue("HTTP_SCHEME", "http"))

	conf.MinioAccessKeyID = project.MinioCredentials.AccessKey
	conf.MinioSecretAccessKey = project.MinioCredentials.SecretKey
	conf.MinioEndpoint = project.MinioCredentials.Endpoint
	conf.MinioProtocol = project.MinioCredentials.UseSecure

	// conf.DefaultOffset = cast.ToString(config.GetOrReturnDefaultValue("DEFAULT_OFFSET", "0"))
	// conf.DefaultLimit = cast.ToString(config.GetOrReturnDefaultValue("DEFAULT_LIMIT", "10000000"))

	// get deployed node host & port
	conf.CorporateServiceHost = project.CorporateService.Host
	conf.CorporateGRPCPort = project.CorporateService.Port

	// get deployed node host & port
	conf.ObjectBuilderServiceHost = project.ObjectBuilderService.Host
	conf.ObjectBuilderGRPCPort = project.ObjectBuilderService.Port

	// get deployed node host & port
	conf.AuthServiceHost = project.AuthService.Host
	conf.AuthGRPCPort = project.AuthService.Port

	// get deployed node host & port
	conf.PosServiceHost = project.PosService.Host
	conf.PosGRPCPort = project.PosService.Port

	// get deployed node host & port
	conf.AnalyticsServiceHost = project.AnalyticsService.Host
	conf.AnalyticsGRPCPort = project.AnalyticsService.Port

	grpcClient, err := services.NewGrpcClients(conf)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	_, err = services.NewProjectGrpcsClient(h.services, grpcClient, project.ProjectNamespace)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	// @TODO::store project-info & service host-ports on database
	// @TODO::minio creds on vault
	// @TODO::automate deployment of project nodes
	// @TODO::restore services on restart (read from database)

	h.handleResponse(c, http.BadRequest, "not implemented yet")
}
