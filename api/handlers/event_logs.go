package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// GetEventLogs godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_event_logs
// @Router /v1/event-log [GET]
// @Summary Get event logs
// @Description Get event logs
// @Tags Event
// @Accept json
// @Produce json
// @Param filters query models.GetEventLogsListRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetEventLogListsResponse} "EventLogsBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetEventLogs(c *gin.Context) {
	page := c.Query("page")
	pageInt, _ := strconv.Atoi(page)

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}
	offset := (pageInt - 1) * limit

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	res, err := services.EventLogsService().GetList(
		context.Background(),
		&obs.GetEventLogsListRequest{
			TableSlug: c.Query("table_slug"),
			Offset:    int32(offset),
			Limit:     int32(limit),
			ProjectId: resourceId.(string),
		})

	if err != nil {
		fmt.Println("Error while get event logs, err: ", err)
		return
	}
	fmt.Println("step 3 finish")
	h.handleResponse(c, http.OK, res)
}

// GetEventLogById godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_event_log
// @Router /v1/event-log/{event_log_id} [GET]
// @Summary Get event log
// @Description Get event log
// @Tags Event
// @Accept json
// @Produce json
// @Param event_log_id path string true "event_log_id"
// @Success 200 {object} http.Response{data=object_builder_service.EventLog} "EventLogBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetEventLogById(c *gin.Context) {
	eventLogID := c.Param("event_log_id")

	if !util.IsValidUUID(eventLogID) {
		h.handleResponse(c, http.InvalidArgument, "event_log_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	resp, err := services.EventLogsService().GetSingle(
		context.Background(),
		&obs.GetEventLogById{
			Id:        eventLogID,
			ProjectId: resourceId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
