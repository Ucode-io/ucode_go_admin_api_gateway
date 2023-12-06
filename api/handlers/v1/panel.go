package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// UpdateCoordinates godoc
// @Security ApiKeyAuth
// @ID panel_coordinates
// @Router /v1/analytics/panel/updateCoordinates [POST]
// @Summary Update panel coordinates
// @Description Update panel coordinates
// @Tags Panel
// @Accept json
// @Produce json
// @Param panel_coordinates body obs.UpdatePanelCoordinatesRequest true "UpdatePanelCoordinatesRequestBody"
// @Success 201 {object} status_http.Response{data=obs.PanelCoordinates} "Coordinates data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateCoordinates(c *gin.Context) {
	var panelCoordinates obs.UpdatePanelCoordinatesRequest

	err := c.ShouldBindJSON(&panelCoordinates)
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

	panelCoordinates.ProjectId = resource.ResourceEnvironmentId

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Panel().UpdateCoordinates(
		context.Background(),
		&panelCoordinates,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetSinglePanel godoc
// @Security ApiKeyAuth
// @ID get_panel_by_id
// @Router /v1/analytics/panel/{panel_id} [GET]
// @Summary Get single panel
// @Description Get single panel
// @Tags Panel
// @Accept json
// @Produce json
// @Param panel_id path string true "panel_id"
// @Success 200 {object} status_http.Response{data=models.Panel} "PanelBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSinglePanel(c *gin.Context) {
	panelID := c.Param("panel_id")

	if !util.IsValidUUID(panelID) {
		h.handleResponse(c, status_http.InvalidArgument, "panel id is an invalid uuid")
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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Panel().GetSingle(
		context.Background(),
		&obs.PanelPrimaryKey{
			Id:        panelID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// CreatePanel godoc
// @Security ApiKeyAuth
// @ID create_panel
// @Router /v1/analytics/panel [POST]
// @Summary Create panel
// @Description Create panel
// @Tags Panel
// @Accept json
// @Produce json
// @Param table body models.CreatePanelRequest true "CreatePanelRequestBody"
// @Success 201 {object} status_http.Response{data=models.Panel} "Panel data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreatePanel(c *gin.Context) {
	var panelRequest models.CreatePanelRequest

	err := c.ShouldBindJSON(&panelRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(panelRequest.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var panel = obs.CreatePanelRequest{
		Query:         panelRequest.Query,
		Coordinates:   panelRequest.Coordinates,
		Attributes:    attributes,
		Title:         panelRequest.Title,
		DashboardId:   panelRequest.DashboardID,
		HasPagination: panelRequest.HasPagination,
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
	panel.ProjectId = resource.ResourceEnvironmentId

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Panel().Create(
		context.Background(),
		&panel,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetAllPanels godoc
// @Security ApiKeyAuth
// @ID get_all_panels
// @Router /v1/analytics/panel [GET]
// @Summary Get all panels
// @Description Get all panels
// @Tags Panel
// @Accept json
// @Produce json
// @Param filters query obs.GetAllPanelsRequest true "filters"
// @Success 200 {object} status_http.Response{data=models.GetAllPanelsResponse} "PanelBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllPanels(c *gin.Context) {

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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Panel().GetList(
		context.Background(),
		&obs.GetAllPanelsRequest{
			Title:     c.DefaultQuery("title", ""),
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {

		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdatePanel godoc
// @Security ApiKeyAuth
// @ID update_panel
// @Router /v1/analytics/panel [PUT]
// @Summary Update panel
// @Description Update panel
// @Tags Panel
// @Accept json
// @Produce json
// @Param relation body models.Panel  true "UpdatePanelRequestBody"
// @Success 200 {object} status_http.Response{data=models.Panel} "Panel data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdatePanel(c *gin.Context) {
	var panelRequest models.Panel

	err := c.ShouldBindJSON(&panelRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(panelRequest.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var panel = obs.Panel{
		Id:            panelRequest.ID,
		Query:         panelRequest.Query,
		Coordinates:   panelRequest.Coordinates,
		Attributes:    attributes,
		Title:         panelRequest.Title,
		DashboardId:   panelRequest.DashboardID,
		HasPagination: panelRequest.HasPagination,
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

	panel.ProjectId = resource.ResourceEnvironmentId

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Panel().Update(
		context.Background(),
		&panel,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeletePanel godoc
// @Security ApiKeyAuth
// @ID delete_panel
// @Router /v1/analytics/panel/{panel_id} [DELETE]
// @Summary Delete Panel
// @Description Delete Panel
// @Tags Panel
// @Accept json
// @Produce json
// @Param panel_id path string true "panel_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeletePanel(c *gin.Context) {
	panelID := c.Param("panel_id")

	if !util.IsValidUUID(panelID) {
		h.handleResponse(c, status_http.InvalidArgument, "panel id is an invalid uuid")
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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Panel().Delete(
		context.Background(),
		&obs.PanelPrimaryKey{
			Id:        panelID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
