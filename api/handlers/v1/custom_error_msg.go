package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
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
func (h *HandlerV1) CreateCustomErrorMessage(c *gin.Context) {
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
			ActionType:   "CREATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &customErrorMessages,
			TableSlug: "CUSTOM_ERROR",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistory(logReq)
	}()

	customErrorMessages.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().Create(
			context.Background(),
			&customErrorMessages,
		)
		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().Create(
			context.Background(),
			&customErrorMessages,
		)
		if err != nil {
			return
		}

	}
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
func (h *HandlerV1) GetByIdCustomErrorMessage(c *gin.Context) {
	var (
		resp *obs.CustomErrorMessage
	)
	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.InvalidArgument, "custom error message id is an invalid uuid")
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
func (h *HandlerV1) GetAllCustomErrorMessage(c *gin.Context) {

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
	if c.Query("table_id") == "" && c.Query("table_slug") == "" {
		h.handleResponse(c, status_http.BadEnvironment, "table_id or table_slug is required")
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
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().GetList(
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
func (h *HandlerV1) UpdateCustomErrorMessage(c *gin.Context) {
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
			ActionType:   "UPDATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &customErrorMessages,
			TableSlug: "CUSTOM_ERROR",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(logReq)
	}()

	customErrorMessages.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).CustomErrorMessage().Update(
			context.Background(),
			&customErrorMessages,
		)
		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().CustomErrorMessage().Update(
			context.Background(),
			&customErrorMessages,
		)
		if err != nil {
			return
		}

	}
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
func (h *HandlerV1) DeleteCustomErrorMessage(c *gin.Context) {
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
			UserInfo:  cast.ToString(userId),
			TableSlug: "CUSTOM_ERROR",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(logReq)
	}()

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
			return
		}

	}
}
