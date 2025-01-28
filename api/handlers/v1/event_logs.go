package v1

import (
	"errors"
	"strconv"
	_ "ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// GetEventLogs godoc
// @Security ApiKeyAuth
// @ID get_event_logs
// @Router /v1/event-log [GET]
// @Summary Get event logs
// @Description Get event logs
// @Tags Event
// @Accept json
// @Produce json
// @Param filters query obs.GetEventLogsListRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetEventLogListsResponse} "EventLogsBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetEventLogs(c *gin.Context) {
	page := c.Query("page")
	pageInt, _ := strconv.Atoi(page)

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	offset := (pageInt - 1) * limit

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

	res, err := services.GetBuilderServiceByType(resource.NodeType).EventLogs().GetList(
		c.Request.Context(), &obs.GetEventLogsListRequest{
			TableSlug: c.Query("table_slug"),
			Offset:    int32(offset),
			Limit:     int32(limit),
			ProjectId: resource.ResourceEnvironmentId,
		})

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.handleResponse(c, status_http.OK, res)
}

// GetEventLogById godoc
// @Security ApiKeyAuth
// @ID get_event_log
// @Router /v1/event-log/{event_log_id} [GET]
// @Summary Get event log
// @Description Get event log
// @Tags Event
// @Accept json
// @Produce json
// @Param event_log_id path string true "event_log_id"
// @Success 200 {object} status_http.Response{data=obs.EventLog} "EventLogBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetEventLogById(c *gin.Context) {
	eventLogID := c.Param("event_log_id")

	if !util.IsValidUUID(eventLogID) {
		h.handleResponse(c, status_http.InvalidArgument, "event_log_id is an invalid uuid")
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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).EventLogs().GetSingle(
		c.Request.Context(), &obs.GetEventLogById{
			Id:        eventLogID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
