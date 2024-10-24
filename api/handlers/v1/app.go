package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateApp godoc
// @Security ApiKeyAuth
// @ID create_app
// @Router /v1/app [POST]
// @Summary Create app
// @Description Create app
// @Tags App
// @Accept json
// @Produce json
// @Param app body obs.AppRequest true "CreateAppRequestBody"
// @Success 201 {object} status_http.Response{data=obs.CreateAppResponse} "App data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateApp(c *gin.Context) {
	var (
		app  obs.AppRequest
		resp *obs.CreateAppResponse
	)

	err := c.ShouldBindJSON(&app)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	app.ProjectId = resource.ResourceEnvironmentId

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).App().Create(
			context.Background(),
			&app,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// Does Not Implemented
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetAppByID godoc
// @Security ApiKeyAuth
// @ID get_app_by_id
// @Router /v1/app/{app_id} [GET]
// @Summary Get app by id
// @Description Get app by id
// @Tags App
// @Accept json
// @Produce json
// @Param app_id path string true "app_id"
// @Success 200 {object} status_http.Response{data=obs.App} "AppBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAppByID(c *gin.Context) {
	appID := c.Param("app_id")
	var (
		resp *obs.App
	)

	if !util.IsValidUUID(appID) {
		h.handleResponse(c, status_http.InvalidArgument, "app id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).App().GetByID(
			context.Background(),
			&obs.AppPrimaryKey{
				Id:        appID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// Does Not Implemented
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllApps godoc
// @Security ApiKeyAuth
// @ID get_all_apps
// @Router /v1/app [GET]
// @Summary Get all apps
// @Description Get all apps
// @Tags App
// @Accept json
// @Produce json
// @Param X-API-KEY header string true "API key for the endpoint"
// @Param filters query obs.GetAllAppsRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllAppsResponse} "AppBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllApps(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	var (
		resp *obs.GetAllAppsResponse
	)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit := 100

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	limit = 100

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	authInfo, _ := h.GetAuthInfo(c)
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).App().GetAll(
			context.Background(),
			&obs.GetAllAppsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				Search:    c.DefaultQuery("search", ""),
				ProjectId: resource.ResourceEnvironmentId,
				RoleId:    authInfo.GetRoleId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// Does Not Implemented
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateApp godoc
// @Security ApiKeyAuth
// @ID update_app
// @Router /v1/app [PUT]
// @Summary Update app
// @Description Update app
// @Tags App
// @Accept json
// @Produce json
// @Param app body obs.UpdateAppRequest  true "UpdateAppRequestBody"
// @Success 200 {object} status_http.Response{data=obs.App} "App data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateApp(c *gin.Context) {
	var (
		app  obs.UpdateAppRequest
		resp *emptypb.Empty
	)

	err := c.ShouldBindJSON(&app)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	app.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).App().Update(
			context.Background(),
			&app,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// Does Not Implemented
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteApp godoc
// @Security ApiKeyAuth
// @ID delete_app
// @Router /v1/app/{app_id} [DELETE]
// @Summary Delete App
// @Description Delete App
// @Tags App
// @Accept json
// @Produce json
// @Param app_id path string true "app_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteApp(c *gin.Context) {
	appID := c.Param("app_id")
	var (
		resp *emptypb.Empty
	)

	if !util.IsValidUUID(appID) {
		h.handleResponse(c, status_http.InvalidArgument, "app id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).App().Delete(
			context.Background(),
			&obs.AppPrimaryKey{
				Id:        appID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// Does Not Implemented
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
