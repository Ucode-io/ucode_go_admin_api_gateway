package v2

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

// CreateErrorMessage godoc
// @Security ApiKeyAuth
// @ID create_error_message
// @Router /v2/collections/{collection}/error_messages [POST]
// @Summary Create error message
// @Description Create error message
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param table body obs.CreateCustomErrorMessage  true "CreateCustomErrorMessageBody"
// @Success 200 {object} status_http.Response{data=string} "Custom Error Message data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateErrorMessage(c *gin.Context) {
	var (
		customErrorMessages obs.CreateCustomErrorMessage
		resp                *obs.CustomErrorMessage
	)

	err := c.ShouldBindJSON(&customErrorMessages)
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

	customErrorMessages.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().Create(
			context.Background(),
			&customErrorMessages,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().Create(
			context.Background(),
			&customErrorMessages,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}
	h.handleResponse(c, status_http.OK, resp)
}

// GetByIdErrorMessage godoc
// @Security ApiKeyAuth
// @ID Get_by_id_error_message
// @Router /v2/collections/{collection}/error_messages/{id} [GET]
// @Summary Error message by id
// @Description Error message by id
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string  true "id"
// @Success 200 {object} status_http.Response{data=string} "Error Message data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetByIdErrorMessage(c *gin.Context) {
	var (
		resp *obs.CustomErrorMessage
	)
	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.InvalidArgument, "error message id is an invalid uuid")
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().GetById(
			context.Background(),
			&obs.CustomErrorMessagePK{
				Id:        c.Param("id"),
				ProjectId: resource.GetResourceEnvironmentId(),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().GetById(
			context.Background(),
			&obs.CustomErrorMessagePK{
				Id:        c.Param("id"),
				ProjectId: resource.GetResourceEnvironmentId(),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}
	h.handleResponse(c, status_http.OK, resp)
}

// GetAllErrorMessage godoc
// @Security ApiKeyAuth
// @ID get_all_error_message
// @Router /v2/collections/{collection}/error_messages [GET]
// @Summary Get all error messages
// @Description Get all error messages
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param filters query obs.GetCustomErrorMessageListRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "ErrorMessageBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllErrorMessage(c *gin.Context) {

	var (
		resp *obs.GetCustomErrorMessageListResponse
	)

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
	if c.Param("collection") == "" {
		h.handleResponse(c, status_http.BadRequest, "collection is required")
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().GetList(
			context.Background(),
			&obs.GetCustomErrorMessageListRequest{
				TableSlug: c.Param("collection"),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().GetList(
			context.Background(),
			&obs.GetCustomErrorMessageListRequest{
				TableSlug: c.Param("collection"),
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

// UpdateErrorMessage godoc
// @Security ApiKeyAuth
// @ID update_error_message
// @Router /v2/collections/{collection}/error_messages [PUT]
// @Summary Update error message
// @Description Update error message
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param table body obs.CustomErrorMessage  true "UpdateErrorMessageBody"
// @Success 200 {object} status_http.Response{data=string} "Error Message data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateErrorMessage(c *gin.Context) {
	var (
		customErrorMessages obs.CustomErrorMessage
		resp                *emptypb.Empty
	)

	err := c.ShouldBindJSON(&customErrorMessages)
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

	customErrorMessages.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().Update(
			context.Background(),
			&customErrorMessages,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().Update(
			context.Background(),
			&customErrorMessages,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}
	h.handleResponse(c, status_http.OK, resp)
}

// DeleteErrorMessage godoc
// @Security ApiKeyAuth
// @ID delete_error_message
// @Router /v2/collections/{collection}/error_messages/{id} [DELETE]
// @Summary Delete error message
// @Description Delete error message
// @Tags Collections
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteErrorMessage(c *gin.Context) {
	var (
		resp *emptypb.Empty
	)
	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "invalid custom error message id")
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
	
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().Delete(
			context.Background(),
			&obs.CustomErrorMessagePK{
				Id:        c.Param("id"),
				ProjectId: resource.GetResourceEnvironmentId(),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().Delete(
			context.Background(),
			&obs.CustomErrorMessagePK{
				Id:        c.Param("id"),
				ProjectId: resource.GetResourceEnvironmentId(),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}
	h.handleResponse(c, status_http.OK, resp)
}
