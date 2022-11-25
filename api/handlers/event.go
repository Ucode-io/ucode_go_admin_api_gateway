package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateEvent godoc
// @Security ApiKeyAuth
// @ID create_event
// @Router /v1/event [POST]
// @Summary Create event
// @Description Create event
// @Tags Event
// @Accept json
// @Produce json
// @Param event body object_builder_service.CreateEventRequest true "CreateEventRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.Event} "Event data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateEvent(c *gin.Context) {
	var event obs.CreateEventRequest

	err := c.ShouldBindJSON(&event)
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

	resp, err := services.EventService().Create(
		context.Background(),
		&event,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetEventByID godoc
// @Security ApiKeyAuth
// @ID get_event_by_id
// @Router /v1/event/{event_id} [GET]
// @Summary Get event by id
// @Description Get event by id
// @Tags Event
// @Accept json
// @Produce json
// @Param event_id path string true "event_id"
// @Success 200 {object} http.Response{data=object_builder_service.Event} "EventBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetEventByID(c *gin.Context) {
	eventID := c.Param("event_id")

	if !util.IsValidUUID(eventID) {
		h.handleResponse(c, http.InvalidArgument, "event id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.EventService().GetSingle(
		context.Background(),
		&obs.EventPrimaryKey{
			Id: eventID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetAllEvents godoc
// @Security ApiKeyAuth
// @ID get_all_events
// @Router /v1/event [GET]
// @Summary Get all events
// @Description Get all events
// @Tags Event
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetEventsListRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetEventsListResponse} "EventBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllEvents(c *gin.Context) {
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.EventService().GetList(
		context.Background(),
		&obs.GetEventsListRequest{
			TableSlug: c.DefaultQuery("table_slug", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateEvent godoc
// @Security ApiKeyAuth
// @ID update_event
// @Router /v1/event [PUT]
// @Summary Update event
// @Description Update event
// @Tags Event
// @Accept json
// @Produce json
// @Param event body object_builder_service.Event  true "UpdateEventRequestBody"
// @Success 200 {object} http.Response{data=object_builder_service.Event} "Event data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateEvent(c *gin.Context) {
	var event obs.Event

	err := c.ShouldBindJSON(&event)
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

	resp, err := services.EventService().Update(
		context.Background(),
		&event,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteEvent godoc
// @Security ApiKeyAuth
// @ID delete_event
// @Router /v1/event/{evet_id} [DELETE]
// @Summary Delete Event
// @Description Delete Event
// @Tags Event
// @Accept json
// @Produce json
// @Param event_id path string true "event_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteEvent(c *gin.Context) {
	eventID := c.Param("event_id")

	if !util.IsValidUUID(eventID) {
		h.handleResponse(c, http.InvalidArgument, "event id is an invalid uuid")
		return
	}
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.EventService().Delete(
		context.Background(),
		&obs.EventPrimaryKey{
			Id: eventID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
