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
	"github.com/spf13/cast"
	"google.golang.org/protobuf/types/known/emptypb"
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
// @Param Customevent body obs.CreateCustomEventRequest true "CreateCustomEventRequestBody"
// @Success 201 {object} status_http.Response{data=string} "CustomEvent data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateCustomEvent(c *gin.Context) {
	var (
		customevent models.CreateCustomEventRequest
		resp        *obs.CustomEvent
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

	structData, err := helper.ConvertMapToStruct(customevent.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var (
		createCustomEvent = &obs.CreateCustomEventRequest{
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
			UserInfo: cast.ToString(userId),
			Request:  createCustomEvent,
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Create(
			context.Background(),
			createCustomEvent,
		)
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Create(
			context.Background(),
			createCustomEvent,
		)
	}
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
// @Success 200 {object} status_http.Response{data=string} "CustomEventBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetCustomEventByID(c *gin.Context) {
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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).CustomEvent().GetSingle(
		context.Background(),
		&obs.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: resource.EnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Param filters query obs.GetCustomEventsListRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "CustomEventBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllCustomEvents(c *gin.Context) {
	var resp *obs.GetCustomEventsListResponse

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().GetList(
			context.Background(),
			&obs.GetCustomEventsListRequest{
				TableSlug: c.DefaultQuery("table_slug", ""),
				RoleId:    authInfo.GetRoleId(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomEvent().GetList(
			context.Background(),
			&obs.GetCustomEventsListRequest{
				TableSlug: c.DefaultQuery("table_slug", ""),
				RoleId:    authInfo.GetRoleId(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Success 200 {object} status_http.Response{data=string} "CustomEvent data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateCustomEvent(c *gin.Context) {
	var (
		customevent models.CustomEvent
		resp        *emptypb.Empty
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

	structData, err := helper.ConvertMapToStruct(customevent.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var (
		updateCustomEvent = &obs.CustomEvent{
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
			UserInfo: cast.ToString(userId),
			Request:  updateCustomEvent,
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

	resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Update(
		context.Background(),
		updateCustomEvent,
	)
	if err != nil {
		return
	}
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
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteCustomEvent(c *gin.Context) {
	var (
		resp *emptypb.Empty
	)
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

	userId, _ := c.Get("user_id")

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

	resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Delete(
		context.Background(),
		&obs.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}
}
