package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
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
// @Param panel_coordinates body object_builder_service.UpdatePanelCoordinatesRequest true "UpdatePanelCoordinatesRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.PanelCoordinates} "Coordinates data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateCoordinates(c *gin.Context) {
	var panel_coordinates obs.UpdatePanelCoordinatesRequest

	err := c.ShouldBindJSON(&panel_coordinates)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PanelService().UpdateCoordinates(
		context.Background(),
		&panel_coordinates,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
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
// @Success 200 {object} http.Response{data=models.Panel} "PanelBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSinglePanel(c *gin.Context) {
	panelID := c.Param("panel_id")

	if !util.IsValidUUID(panelID) {
		h.handleResponse(c, http.InvalidArgument, "panel id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PanelService().GetSingle(
		context.Background(),
		&obs.PanelPrimaryKey{
			Id: panelID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Success 201 {object} http.Response{data=models.Panel} "Panel data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreatePanel(c *gin.Context) {
	var panelRequest models.CreatePanelRequest

	err := c.ShouldBindJSON(&panelRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(panelRequest.Attributes)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PanelService().Create(
		context.Background(),
		&panel,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
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
// @Param filters query object_builder_service.GetAllPanelsRequest true "filters"
// @Success 200 {object} http.Response{data=models.GetAllPanelsResponse} "PanelBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllPanels(c *gin.Context) {

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PanelService().GetList(
		context.Background(),
		&obs.GetAllPanelsRequest{
			Title: c.DefaultQuery("title", ""),
		},
	)

	if err != nil {

		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Success 200 {object} http.Response{data=models.Panel} "Panel data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdatePanel(c *gin.Context) {
	var panelRequest models.Panel

	err := c.ShouldBindJSON(&panelRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(panelRequest.Attributes)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PanelService().Update(
		context.Background(),
		&panel,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeletePanel(c *gin.Context) {
	panelID := c.Param("panel_id")

	if !util.IsValidUUID(panelID) {
		h.handleResponse(c, http.InvalidArgument, "panel id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PanelService().Delete(
		context.Background(),
		&obs.PanelPrimaryKey{
			Id: panelID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
