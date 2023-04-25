package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/saidamir98/udevs_pkg/util"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
)

// CreateIntegration godoc
// @ID create_Integration
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string false "Environment-Id"
// @Router /integration [POST]
// @Summary Create Integration
// @Description Create Integration
// @Tags Integration
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param Integration body auth_service.CreateIntegrationRequest true "CreateIntegrationRequestBody"
// @Success 201 {object} status_http.Response{data=auth_service.Integration} "Integration data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateIntegration(c *gin.Context) {
	var integration auth_service.CreateIntegrationRequest

	err := c.ShouldBindJSON(&integration)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//environmentId, ok := c.Get("environment_id")
	//if !ok || !util.IsValidUUID(environmentId.(string)) {
	//	err = errors.New("error getting environment id | not valid")
	//	h.handleResponse(c, status_http.BadRequest, err)
	//	return
	//}

	//resource, err := services.CompanyService().ServiceResource().GetSingle(
	//	c.Request.Context(),
	//	&pb.GetSingleServiceResourceReq{
	//		ProjectId:     projectId,
	//		EnvironmentId: environmentId.(string),
	//		ServiceType:   pb.ServiceType_BUILDER_SERVICE,
	//	},
	//)
	//if err != nil {
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}
	integration.ProjectId = projectId

	resp, err := services.AuthService().Integration().CreateIntegration(
		c.Request.Context(),
		&integration,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetIntegrationList godoc
// @ID get_integration_list
// @Router /integration [GET]
// @Summary Get Integration List
// @Description  Get Integration List
// @Tags Integration
// @Accept json
// @Produce json
// @Param offset query integer false "offset"
// @Param limit query integer false "limit"
// @Param search query string false "search"
// @Param client-platform-id query string false "client-platform-id"
// @Param client-type-id query string false "client-type-id"
// @Param project-id query string false "project-id"
// @Success 200 {object} status_http.Response{data=auth_service.GetIntegrationListResponse} "GetIntegrationListResponseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetIntegrationList(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//@TODO::protobuff already has project_id field
	resp, err := services.AuthService().Integration().GetIntegrationList(
		c.Request.Context(),
		&auth_service.GetIntegrationListRequest{
			Limit:            int32(limit),
			Offset:           int32(offset),
			Search:           c.Query("search"),
			ClientPlatformId: c.Query("client-platform-id"),
			ClientTypeId:     c.Query("client-type-id"),
			ProjectId:        c.Query("project-id"),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetIntegrationSessions godoc
// @ID get_integration_sessions
// @Router /integration/{integration-id}/session [GET]
// @Summary Get Integration Sessions
// @Description  Get Integration Sessions
// @Tags Integration
// @Accept json
// @Produce json
// @Param integration-id path string true "integration-id"
// @Success 200 {object} status_http.Response{data=auth_service.GetIntegrationSessionsResponse} "GetIntegrationSessionsResponseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetIntegrationSessions(c *gin.Context) {
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//@TODO:: no project_id field
	resp, err := services.AuthService().Integration().GetIntegrationSessions(
		c.Request.Context(),
		&auth_service.IntegrationPrimaryKey{
			Id: c.Param("integration-id"),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// AddSessionToIntegration godoc
// @ID add_session_to_integration
// @Router /integration/{integration-id}/session [POST]
// @Summary Add Session To Integration
// @Description Add Session To Integration
// @Tags Integration
// @Accept json
// @Produce json
// @Param integration-id path string true "integration-id"
// @Param addSessionToIntegration body auth_service.AddSessionToIntegrationRequest true "AddSessionToIntegrationRequestBody"
// @Success 201 {object} status_http.Response{data=string} "Add Session To Integration Response"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) AddSessionToIntegration(c *gin.Context) {
	var login auth_service.AddSessionToIntegrationRequest

	err := c.ShouldBindJSON(&login)
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

	integrationID := c.Param("integration-id")
	if !util.IsValidUUID(integrationID) {
		h.handleResponse(c, status_http.InvalidArgument, "integration id is an invalid uuid")
		return
	}
	login.IntegrationId = integrationID

	//@TODO:: no project_id field
	resp, err := services.AuthService().Integration().AddSessionToIntegration(
		c.Request.Context(),
		&login,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetIntegrationByID godoc
// @ID get_Integration_by_id
// @Router /integration/{integration-id} [GET]
// @Summary Get Integration By ID
// @Description Get Integration By ID
// @Tags Integration
// @Accept json
// @Produce json
// @Param integration-id path string true "integration-id"
// @Success 200 {object} status_http.Response{data=auth_service.Integration} "IntegrationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetIntegrationByID(c *gin.Context) {
	IntegrationID := c.Param("integration-id")

	if !util.IsValidUUID(IntegrationID) {
		h.handleResponse(c, status_http.InvalidArgument, "Integration id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//@TODO:: no project id field
	resp, err := services.AuthService().Integration().GetIntegrationByID(
		c.Request.Context(),
		&auth_service.IntegrationPrimaryKey{
			Id: IntegrationID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteIntegration godoc
// @ID delete_Integration
// @Router /integration/{integration-id} [DELETE]
// @Summary Delete Integration
// @Description Delete Integration
// @Tags Integration
// @Accept json
// @Produce json
// @Param integration-id path string true "Integration-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteIntegration(c *gin.Context) {
	IntegrationID := c.Param("integration-id")

	if !util.IsValidUUID(IntegrationID) {
		h.handleResponse(c, status_http.InvalidArgument, "Integration id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//@TODO:: no project id field
	resp, err := services.AuthService().Integration().DeleteIntegration(
		c.Request.Context(),
		&auth_service.IntegrationPrimaryKey{
			Id: IntegrationID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// GetIntegrationToken godoc
// @ID get_integration_token
// @Router /integration/{integration-id}/session/{session-id} [GET]
// @Summary Get Integration Token
// @Description Get Integration Token
// @Tags Integration
// @Accept json
// @Produce json
// @Param integration-id path string true "integration-id"
// @Param session-id path string true "session-id"
// @Success 200 {object} status_http.Response{data=auth_service.Token} "IntegrationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetIntegrationToken(c *gin.Context) {
	integrationID := c.Param("integration-id")
	sessionID := c.Param("session-id")

	if !util.IsValidUUID(integrationID) {
		h.handleResponse(c, status_http.InvalidArgument, "Integration id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(sessionID) {
		h.handleResponse(c, status_http.InvalidArgument, "Session id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//@TODO:: no project id field
	resp, err := services.AuthService().Integration().GetIntegrationToken(
		c.Request.Context(),
		&auth_service.GetIntegrationTokenRequest{
			IntegrationId: integrationID,
			SessionId:     sessionID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// RemoveSessionFromIntegration godoc
// @ID delete_session_from_integration
// @Router /integration/{integration-id}/session/{session-id} [DELETE]
// @Summary Delete Session From Integration
// @Description Delete Session From Integration
// @Tags Integration
// @Accept json
// @Produce json
// @Param integration-id path string true "Integration-id"
// @Param session-id path string true "session-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RemoveSessionFromIntegration(c *gin.Context) {
	integrationID := c.Param("integration-id")
	sessionID := c.Param("session-id")

	if !util.IsValidUUID(integrationID) {
		h.handleResponse(c, status_http.InvalidArgument, "Integration id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(sessionID) {
		h.handleResponse(c, status_http.InvalidArgument, "Session id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.AuthService().Integration().DeleteSessionFromIntegration(
		c.Request.Context(),
		&auth_service.GetIntegrationTokenRequest{
			IntegrationId: integrationID,
			SessionId:     sessionID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
