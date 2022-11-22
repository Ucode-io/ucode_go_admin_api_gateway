package handlers

import (
	"context"
	"time"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/project_service"
	"ucode/ucode_go_api_gateway/pkg/util"
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
func (h *Handler) CreateProject(c *gin.Context) {
	var project models.CreateProjectRequest

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	conf := config.Config{}

	conf.MinioAccessKeyID = project.MinioCredentials.AccessKey
	conf.MinioSecretAccessKey = project.MinioCredentials.SecretKey
	conf.MinioEndpoint = project.MinioCredentials.Endpoint
	conf.MinioProtocol = project.MinioCredentials.UseSecure

	conf.CorporateServiceHost = project.CorporateService.Host
	conf.CorporateGRPCPort = project.CorporateService.Port

	conf.ObjectBuilderServiceHost = project.ObjectBuilderService.Host
	conf.ObjectBuilderGRPCPort = project.ObjectBuilderService.Port

	conf.AuthServiceHost = project.AuthService.Host
	conf.AuthGRPCPort = project.AuthService.Port

	conf.PosServiceHost = project.PosService.Host
	conf.PosGRPCPort = project.PosService.Port

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	projectResponse, err := h.storage.Project().Create(ctx, &project_service.CreateProjectRequest{
		Name:                     project.ProjectName,
		Namespace:                project.ProjectNamespace,
		ObjectBuilderServiceHost: project.ObjectBuilderService.Host,
		ObjectBuilderServicePort: project.ObjectBuilderService.Port,
		AuthServiceHost:          project.AuthService.Host,
		AuthServicePort:          project.AuthService.Port,
		AnalyticsServiceHost:     project.AnalyticsService.Host,
		AnalyticsServicePort:     project.AnalyticsService.Port,
	})
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	// @DONE::store project-info & service host-ports on database
	// @TODO::minio creds on vault
	// @TODO::automate deployment of project nodes
	// @DONE::restore services on restart (read from database)

	h.handleResponse(c, http.OK, projectResponse)
}

// GetAllProjects godoc
// @Security ApiKeyAuth
// @ID get_all_projects
// @Router /v1/project [GET]
// @Summary Get all projects
// @Description Get all projects
// @Tags Project
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllProjectsRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetAllProjectsResponse} "ProjectBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllProjects(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	resp, err := h.storage.Project().GetList(ctx,
		&project_service.GetAllProjectsRequest{
			Offset: int32(offset),
			Limit:  int32(limit),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteProject godoc
// @Security ApiKeyAuth
// @ID delete_project
// @Router /v1/project/{project_id} [DELETE]
// @Summary Delete Project
// @Description Delete Project
// @Tags Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteProject(c *gin.Context) {
	projectID := c.Param("project_id")

	if !util.IsValidUUID(projectID) {
		h.handleResponse(c, http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	namespace, rowsAffected, err := h.storage.Project().Delete(ctx, &project_service.ProjectPrimaryKey{
		Id: projectID,
	})

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.RemoveService(namespace)

	h.handleResponse(c, http.NoContent, rowsAffected)
}
