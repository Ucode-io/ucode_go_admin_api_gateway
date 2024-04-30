package v1

import (
	"context"
	"errors"
	"log"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

// CreateNewCustomEvent godoc
// @Security ApiKeyAuth
// @ID create_new_custom_event
// @Router /v2/custom-event [POST]
// @Summary Create New CustomEvent
// @Description Create New CustomEvent
// @Tags NewCustomEvent
// @Accept json
// @Produce json
// @Param Customevent body fc.CreateCustomEventRequest true "CreateCustomEventRequestBody"
// @Success 201 {object} status_http.Response{data=string} "CustomEvent data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateNewCustomEvent(c *gin.Context) {
	var (
		customevent models.CreateCustomEventRequest
		resp        = &fc.CustomEvent{}
	)

	err := c.ShouldBindJSON(&customevent)
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

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(customevent.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	commitID, _ := uuid.NewRandom()

	var (
		createCustomEvent = &fc.CreateCustomEventRequest{
			TableSlug:  customevent.TableSlug,
			EventPath:  customevent.EventPath,
			Label:      customevent.Label,
			Icon:       customevent.Icon,
			Url:        customevent.Url,
			Disable:    customevent.Disable,
			ActionType: customevent.ActionType,
			Method:     customevent.Method,
			Attributes: structData,
			ProjectId:  resource.ResourceEnvironmentId, //added resource id
			CommitId:   commitID.String(),
		}
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "CREATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   createCustomEvent,
			TableSlug: customevent.TableSlug,
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	resp, err = services.FunctionService().CustomEventService().Create(
		context.Background(),
		createCustomEvent,
	)

	if err != nil {
		return
	}
}

// GetNewCustomEventByID godoc
// @Security ApiKeyAuth
// @ID get_new_custom_event_by_id
// @Router /v2/custom-event/{custom_event_id} [GET]
// @Summary Get CustomEvent by id
// @Description Get CustomEvent by id
// @Tags NewCustomEvent
// @Accept json
// @Produce json
// @Param custom_event_id path string true "custom_event_id"
// @Success 200 {object} status_http.Response{data=string} "CustomEventBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetNewCustomEventByID(c *gin.Context) {
	customeventID := c.Param("custom_event_id")

	if !util.IsValidUUID(customeventID) {
		h.handleResponse(c, status_http.InvalidArgument, "Customevent id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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

	resp, err := services.FunctionService().CustomEventService().GetSingle(
		context.Background(),
		&fc.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllNewCustomEvents godoc
// @Security ApiKeyAuth
// @ID get_new_all_custom_events
// @Router /v2/custom-event [GET]
// @Summary Get all custom events
// @Description Get all custom events
// @Tags CustomEvent
// @Accept json
// @Produce json
// @Param filters query fc.GetCustomEventsListRequest false "filters"
// @Success 200 {object} status_http.Response{data=string} "CustomEventBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllNewCustomEvents(c *gin.Context) {

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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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

	tokenInfo, _ := h.GetAuthInfo(c)

	resp, err := services.FunctionService().CustomEventService().GetList(
		context.Background(),
		&fc.GetCustomEventsListRequest{
			TableSlug: c.DefaultQuery("table_slug", ""),
			ProjectId: resource.ResourceEnvironmentId,
			RoleId:    tokenInfo.GetRoleId(),
		},
	)
	if err != nil {
		log.Println("error getting custom events list: ", err)
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateNewCustomEvent godoc
// @Security ApiKeyAuth
// @ID update_new_custom_event
// @Router /v2/custom-event [PUT]
// @Summary Update New Customevent
// @Description Update new custom event
// @Tags NewCustomEvent
// @Accept json
// @Produce json
// @Param Customevent body models.CustomEvent true "UpdateCustomEventRequestBody"
// @Success 200 {object} status_http.Response{data=string} "CustomEvent data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateNewCustomEvent(c *gin.Context) {
	var (
		customevent models.CustomEvent
		resp        = &empty.Empty{}
	)

	err := c.ShouldBindJSON(&customevent)
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

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(customevent.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	var (
		updateCustomEvent = &fc.CustomEvent{
			Id:         customevent.Id,
			TableSlug:  customevent.TableSlug,
			EventPath:  customevent.EventPath,
			Label:      customevent.Label,
			Icon:       customevent.Icon,
			Url:        customevent.Url,
			Disable:    customevent.Disable,
			ActionType: customevent.ActionType,
			Method:     customevent.Method,
			Attributes: structData,
			ProjectId:  resource.ResourceEnvironmentId,
		}
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "UPDATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   updateCustomEvent,
			TableSlug: customevent.TableSlug,
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	resp, err = services.FunctionService().CustomEventService().Update(
		context.Background(),
		updateCustomEvent,
	)
	if err != nil {
		return
	}
}

// DeleteNewCustomEvent godoc
// @Security ApiKeyAuth
// @ID delete_new_custom_event
// @Router /v2/custom-event/{custom_event_id} [DELETE]
// @Summary Delete CustomEvent
// @Description Delete CustomEvent
// @Tags NewCustomEvent
// @Accept json
// @Produce json
// @Param custom_event_id path string true "custom_event_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteNewCustomEvent(c *gin.Context) {
	customeventID := c.Param("custom_event_id")

	resp := &empty.Empty{}

	if !util.IsValidUUID(customeventID) {
		h.handleResponse(c, status_http.InvalidArgument, "Customevent id is an invalid uuid")
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

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "DELETE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo: cast.ToString(userId),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	resp, err = services.FunctionService().CustomEventService().Delete(
		context.Background(),
		&fc.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}
}
