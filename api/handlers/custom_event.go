package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateCustomEvent godoc
// @Security ApiKeyAuth
// @ID create_custom_event
// @Router /v1/custom-event [POST]
// @Summary Create CustomEvent
// @Description Create CustomEvent
// @Tags CustomEvent
// @Accept json
// @Produce json
// @Param Customevent body object_builder_service.CreateCustomEventRequest true "CreateCustomEventRequestBody"
// @Success 201 {object} http.Response{data=string} "CustomEvent data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateCustomEvent(c *gin.Context) {
	var customevent obs.CreateCustomEventRequest

	err := c.ShouldBindJSON(&customevent)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}
	customevent.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.CustomEventService().Create(
		context.Background(),
		&customevent,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetCustomEventByID godoc
// @Security ApiKeyAuth
// @ID get_custom_event_by_id
// @Router /v1/custom-event/{custom_event_id} [GET]
// @Summary Get CustomEvent by id
// @Description Get CustomEvent by id
// @Tags CustomEvent
// @Accept json
// @Produce json
// @Param custom_event_id path string true "custom_event_id"
// @Success 200 {object} http.Response{data=string} "CustomEventBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCustomEventByID(c *gin.Context) {
	customeventID := c.Param("custom_event_id")

	if !util.IsValidUUID(customeventID) {
		h.handleResponse(c, http.InvalidArgument, "Customevent id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.CustomEventService().GetSingle(
		context.Background(),
		&obs.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetAllCustomEvents godoc
// @Security ApiKeyAuth
// @ID get_all_custom_events
// @Router /v1/custom-event [GET]
// @Summary Get all custom events
// @Description Get all custom events
// @Tags CustomEvent
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetCustomEventsListRequest true "filters"
// @Success 200 {object} http.Response{data=string} "CustomEventBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllCustomEvents(c *gin.Context) {
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.CustomEventService().GetList(
		context.Background(),
		&obs.GetCustomEventsListRequest{
			TableSlug: c.DefaultQuery("table_slug", ""),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateCustomEvent godoc
// @Security ApiKeyAuth
// @ID update_Customevent
// @Router /v1/custom-event [PUT]
// @Summary Update Customevent
// @Description Update custom event
// @Tags CustomEvent
// @Accept json
// @Produce json
// @Param Customevent body models.CustomEvent true "UpdateCustomEventRequestBody"
// @Success 200 {object} http.Response{data=string} "CustomEvent data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateCustomEvent(c *gin.Context) {
	var customevent models.CustomEvent

	err := c.ShouldBindJSON(&customevent)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.CustomEventService().Update(
		context.Background(),
		&obs.CustomEvent{
			Id:        customevent.Id,
			EventPath: customevent.EventPath,
			Disable:   customevent.Disable,
			Icon:      customevent.Icon,
			TableSlug: customevent.TableSlug,
			Url:       customevent.Url,
			Label:     customevent.Label,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteCustomEvent godoc
// @Security ApiKeyAuth
// @ID delete_custom_event
// @Router /v1/custom-event/{custom_event_id} [DELETE]
// @Summary Delete CustomEvent
// @Description Delete CustomEvent
// @Tags CustomEvent
// @Accept json
// @Produce json
// @Param custom_event_id path string true "custom_event_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteCustomEvent(c *gin.Context) {
	customeventID := c.Param("custom_event_id")

	if !util.IsValidUUID(customeventID) {
		h.handleResponse(c, http.InvalidArgument, "Customevent id is an invalid uuid")
		return
	}
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.CustomEventService().Delete(
		context.Background(),
		&obs.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
