package handlers

import (
	"context"
	"errors"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// GetListLayouts godoc
// @Security ApiKeyAuth
// @ID get_list_layouts
// @Router /v1/layout/{table_id} [GET]
// @Summary Get list layouts
// @Description Get list layouts
// @Tags Layout
// @Accept json
// @Produce json
// @Param table_id path string true "table_id"
// @Success 200 {object} status_http.Response{data=object_builder_service.GetListLayoutResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListLayouts(c *gin.Context) {
	TableId := c.Param("table_id")

	if !util.IsValidUUID(TableId) {
		h.handleResponse(c, status_http.InvalidArgument, "table id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	var resourceEnvironmentId string
	if !resourceIdOk {
		resource, err := services.CompanyService().ServiceResource().GetSingle(
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
	}

	resp, err := services.BuilderService().Layout().GetAll(
		context.Background(),
		&object_builder_service.GetListLayoutRequest{
			TableId:   TableId,
			ProjectId: resourceEnvironmentId,
		},
	)

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateLayout godoc
// @Security ApiKeyAuth
// @ID update_layout
// @Router /v1/layout [PUT]
// @Summary Update layouts
// @Description Update layouts
// @Tags Layout
// @Accept json
// @Produce json
// @Param table body object_builder_service.UpdateLayoutRequest true "UpdateLayoutRequest"
// @Success 200 {object} status_http.Response{data=string} "Layout data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateLayout(c *gin.Context) {

	var input object_builder_service.UpdateLayoutRequest

	err := c.ShouldBindJSON(&input)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	if input.TableId == "" {
		h.handleResponse(c, status_http.BadRequest, errors.New("table id is required"))
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	var resourceEnvironmentId string
	if !resourceIdOk {
		resource, err := services.CompanyService().ServiceResource().GetSingle(
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
	}

	input.ProjectId = resourceEnvironmentId
	resp, err := services.BuilderService().Layout().Update(
		context.Background(),
		&input,
	)

	h.handleResponse(c, status_http.OK, resp)
}
