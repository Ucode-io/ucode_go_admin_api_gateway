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

// CreateCustomErrorMessage godoc
// @Security ApiKeyAuth
// @ID create_custom_error_message
// @Router /v1/custom-error-message [POST]
// @Summary Create custom error message
// @Description Create custom error message
// @Tags CustomErrorMessage
// @Accept json
// @Produce json
// @Param table body obs.CreateCustomErrorMessage  true "CreateCustomErrorMessageBody"
// @Success 200 {object} status_http.Response{data=string} "Custom Error Message data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateCustomErrorMessage(c *gin.Context) {
	var (
		customErrorMessages obs.CreateCustomErrorMessage
		resp                *obs.CustomErrorMessage
	)

	err := c.ShouldBindJSON(&customErrorMessages)
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
	customErrorMessages.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().CustomErrorMessage().Create(
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

// GetByIdCustomErrorMessage godoc
// @Security ApiKeyAuth
// @ID Get_by_id_custom_error_message
// @Router /v1/custom-error-message/{id} [GET]
// @Summary Get by id custom error message
// @Description Get by id custom error message
// @Tags CustomErrorMessage
// @Accept json
// @Produce json
// @Param id path string  true "id"
// @Success 200 {object} status_http.Response{data=string} "Custom Error Message data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetByIdCustomErrorMessage(c *gin.Context) {
	var (
		resp *obs.CustomErrorMessage
	)
	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.InvalidArgument, "custom error message id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().CustomErrorMessage().GetById(
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

// GetAllCustomErrorMessage godoc
// @Security ApiKeyAuth
// @ID get_all_custom_error_message
// @Router /v1/custom-error-message [GET]
// @Summary Get all custom error messages
// @Description Get all custom error messages
// @Tags CustomErrorMessage
// @Accept json
// @Produce json
// @Param filters query obs.GetCustomErrorMessageListRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "CustomErrorMessageBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllCustomErrorMessage(c *gin.Context) {

	var (
		resp *obs.GetCustomErrorMessageListResponse
	)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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
	if c.Query("table_id") == "" && c.Query("table_slug") == "" {
		h.handleResponse(c, status_http.BadEnvironment, "table_id or table_slug is required")
		return
	}

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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().CustomErrorMessage().GetList(
			context.Background(),
			&obs.GetCustomErrorMessageListRequest{
				TableId:   c.Query("table_id"),
				TableSlug: c.Query("table_slug"),
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
				TableId:   c.Query("table_id"),
				TableSlug: c.Query("table_slug"),
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

// UpdateCustomErrorMessage godoc
// @Security ApiKeyAuth
// @ID update_custom_error_message
// @Router /v1/custom-error-message [PUT]
// @Summary Update custom error message
// @Description Update custom error message
// @Tags CustomErrorMessage
// @Accept json
// @Produce json
// @Param table body obs.CustomErrorMessage  true "UpdateCustomErrorMessageBody"
// @Success 200 {object} status_http.Response{data=string} "Custom Error Message data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateCustomErrorMessage(c *gin.Context) {
	var (
		customErrorMessages obs.CustomErrorMessage
		resp                *emptypb.Empty
	)

	err := c.ShouldBindJSON(&customErrorMessages)
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
	customErrorMessages.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().CustomErrorMessage().Update(
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

// DeleteCustomErrorMessage godoc
// @Security ApiKeyAuth
// @ID delete_custom_error_message
// @Router /v1/custom-error-message/{id} [DELETE]
// @Summary Delete custom error message
// @Description Delete custom error message
// @Tags CustomErrorMessage
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteCustomErrorMessage(c *gin.Context) {
	var (
		resp *emptypb.Empty
	)
	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "invalid custom error message id")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().CustomErrorMessage().Delete(
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
