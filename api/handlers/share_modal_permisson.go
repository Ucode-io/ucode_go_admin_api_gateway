package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetTablePermission godoc
// @Security ApiKeyAuth
// @ID get_table_permission
// @Router /v1/table-permission [GET]
// @Summary Get all table permission
// @Description Get all table permission
// @Tags TablePermission
// @Accept json
// @Produce json
// @Param filters query obs.GetPermissionsByTableSlugRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetPermissionsByTableSlugResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTablePermission(c *gin.Context) {
	var (
		//resourceEnvironment *company_service.ResourceEnvironment
		resp                  *obs.GetPermissionsByTableSlugResponse
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceEnvironmentId = resource.ResourceEnvironmentId
		nodeType = resource.NodeType
	} else {
		resourceEnvironment, err := h.companyServices.Resource().GetResourceEnvironment(
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
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
		nodeType = resourceEnvironment.GetNodeType()
	}
	authInfo, _ := h.GetAuthInfo(c)

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(nodeType).Permission().GetPermissionsByTableSlug(
			context.Background(),
			&obs.GetPermissionsByTableSlugRequest{
				ProjectId:     resourceEnvironmentId,
				CurrentRoleId: authInfo.GetRoleId(),
				RoleId:        c.DefaultQuery("role_id", ""),
				TableSlug:     c.DefaultQuery("table_slug", ""),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Permission().GetPermissionsByTableSlug(
			context.Background(),
			&obs.GetPermissionsByTableSlugRequest{
				ProjectId:     resourceEnvironmentId,
				CurrentRoleId: authInfo.GetRoleId(),
				RoleId:        c.DefaultQuery("role_id", ""),
				TableSlug:     c.DefaultQuery("table_slug", ""),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateTablePermission godoc
// @Security ApiKeyAuth
// @ID update_table_permission
// @Router /v1/table-permission [PUT]
// @Summary Update table permisison
// @Description Update table permission
// @Tags TablePermission
// @Accept json
// @Produce json
// @Param table body obs.UpdatePermissionsRequest  true "UpdateTablePermisisonRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateTablePermission(c *gin.Context) {
	var (
		tablePermission obs.UpdatePermissionsRequest
		//resourceEnvironment *company_service.ResourceEnvironment
		resp                  *emptypb.Empty
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

	err := c.ShouldBindJSON(&tablePermission)
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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
		nodeType = resource.NodeType
	} else {
		resourceEnvironment, err := h.companyServices.Resource().GetResourceEnvironment(
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
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
		nodeType = resourceEnvironment.GetNodeType()
	}

	tablePermission.ProjectId = resourceEnvironmentId
	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(nodeType).Permission().UpdatePermissionsByTableSlug(
			context.Background(),
			&tablePermission,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Permission().UpdatePermissionsByTableSlug(
			context.Background(),
			&tablePermission,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}
