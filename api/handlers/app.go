package handlers

import (
	"context"
	"ucode/ucode_go_admin_api_gateway/api/http"
	obs "ucode/ucode_go_admin_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_admin_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
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
// @Param app body object_builder_service.AppRequest true "CreateAppRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.CreateAppResponse} "App data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateApp(c *gin.Context) {
	var app obs.AppRequest

	err := c.ShouldBindJSON(&app)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.services.AppService().Create(
		context.Background(),
		&app,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
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
// @Success 200 {object} http.Response{data=object_builder_service.App} "AppBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAppByID(c *gin.Context) {
	appID := c.Param("app_id")

	if !util.IsValidUUID(appID) {
		h.handleResponse(c, http.InvalidArgument, "app id is an invalid uuid")
		return
	}

	resp, err := h.services.AppService().GetByID(
		context.Background(),
		&obs.AppPrimaryKey{
			Id: appID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Param filters query object_builder_service.GetAllAppsRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetAllAppsResponse} "AppBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllApps(c *gin.Context) {
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

	resp, err := h.services.AppService().GetAll(
		context.Background(),
		&obs.GetAllAppsRequest{
			Limit:  int32(limit),
			Offset: int32(offset),
			Search: c.DefaultQuery("search", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Param app body object_builder_service.UpdateAppRequest  true "UpdateAppRequestBody"
// @Success 200 {object} http.Response{data=object_builder_service.App} "App data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateApp(c *gin.Context) {
	var app obs.UpdateAppRequest

	err := c.ShouldBindJSON(&app)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	resp, err := h.services.AppService().Update(
		context.Background(),
		&app,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteApp(c *gin.Context) {
	appID := c.Param("app_id")

	if !util.IsValidUUID(appID) {
		h.handleResponse(c, http.InvalidArgument, "app id is an invalid uuid")
		return
	}

	resp, err := h.services.AppService().Delete(
		context.Background(),
		&obs.AppPrimaryKey{
			Id: appID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
