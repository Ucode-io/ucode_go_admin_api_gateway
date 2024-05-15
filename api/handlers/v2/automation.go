package v2

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateAutomation godoc
// @Security ApiKeyAuth
// @ID create_automation
// @Router /v2/collections/{collection}/automation [POST]
// @Summary Create Automation
// @Description Create Automation
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param Automation body obs.CreateCustomEventRequest true "AutomationRequestBody"
// @Success 201 {object} status_http.Response{data=string} "Automation data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateAutomation(c *gin.Context) {
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
		resource.GetProjectId(),
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Create(
			context.Background(),
			&obs.CreateCustomEventRequest{
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
			},
		)
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Create(
			context.Background(),
			&obs.CreateCustomEventRequest{
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
			},
		)
	}

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetByIdAutomation godoc
// @Security ApiKeyAuth
// @ID get_automatio_by_id
// @Router /v2/collections/{collection}/automation/{id} [GET]
// @Summary Get Automation by id
// @Description Get Automation by id
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=string} "AutomationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetByIdAutomation(c *gin.Context) {
	customeventID := c.Param("id")

	if !util.IsValidUUID(customeventID) {
		h.handleResponse(c, status_http.InvalidArgument, "automation id is an invalid uuid")
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
		resource.GetProjectId(),
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

// GetAllAutomation godoc
// @Security ApiKeyAuth
// @ID get_all_automation
// @Router /v2/collections/{collection}/automation [GET]
// @Summary Get all automation
// @Description Get all automation
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param filters query obs.GetCustomEventsListRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "AutomationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllAutomation(c *gin.Context) {
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
	fmt.Println("here coming >>>>>> ", resource.ResourceType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		_, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().GetList(
			context.Background(),
			&obs.GetCustomEventsListRequest{
				TableSlug: c.DefaultQuery("table_slug", ""),
				RoleId:    authInfo.GetRoleId(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		fmt.Println(resp)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		_, err = services.PostgresBuilderService().CustomEvent().GetList(
			context.Background(),
			&obs.GetCustomEventsListRequest{
				TableSlug: c.DefaultQuery("table_slug", ""),
				RoleId:    authInfo.GetRoleId(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		// if err != nil {
		// 	h.handleResponse(c, status_http.GRPCError, err.Error())
		// 	return
		// }
	}

	h.handleResponse(c, status_http.OK, obs.GetCustomEventsListResponse{})
}

// UpdateAutomation godoc
// @Security ApiKeyAuth
// @ID update_automation
// @Router /v2/collections/{collection}/automation [PUT]
// @Summary Update Automation
// @Description Update automation
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param Customevent body models.CustomEvent true "UpdateAutomationRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Automation data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateAutomation(c *gin.Context) {
	var customevent models.CustomEvent

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
		resource.GetProjectId(),
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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Update(
		context.Background(),
		&obs.CustomEvent{
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
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteAutomation godoc
// @Security ApiKeyAuth
// @ID delete_automation
// @Router /v2/collections/{collection}/automation/{id} [DELETE]
// @Summary Delete Automation
// @Description Delete Automation
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteAutomation(c *gin.Context) {
	customeventID := c.Param("id")

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
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).CustomEvent().Delete(
		context.Background(),
		&obs.CustomEventPrimaryKey{
			Id:        customeventID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
