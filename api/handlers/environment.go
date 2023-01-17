package handlers

import (
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateEnvironment godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_environment
// @Router /v1/environment [POST]
// @Summary Create environment
// @Description Create environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment body obs.CreateEnvironmentRequest true "CreateEnvironmentRequestBody"
// @Success 201 {object} http.Response{data=obs.Environment} "Environment data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateEnvironment(c *gin.Context) {
	var environmentRequest obs.CreateEnvironmentRequest

	err := c.ShouldBindJSON(&environmentRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.EnvironmentService().Create(
		c.Request.Context(),
		&environmentRequest,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetSingleEnvironment godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_environment_by_id
// @Router /v1/environment/{environment_id} [GET]
// @Summary Get single environment
// @Description Get single environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Success 200 {object} http.Response{data=obs.Environment} "EnvironmentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleEnvironment(c *gin.Context) {
	environmentID := c.Param("environment_id")

	if !util.IsValidUUID(environmentID) {
		h.handleResponse(c, http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.EnvironmentService().GetById(
		c.Request.Context(),
		&obs.EnvironmentPrimaryKey{
			Id: environmentID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateEnvironment godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_environment
// @Router /v1/environment [PUT]
// @Summary Update environment
// @Description Update environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment body obs.Environment true "UpdateEnvironmentRequestBody"
// @Success 200 {object} http.Response{data=obs.Environment} "Environment data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateEnvironment(c *gin.Context) {
	var environment obs.Environment

	err := c.ShouldBindJSON(&environment)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.EnvironmentService().Update(
		c.Request.Context(),
		&environment,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteEnvironment godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_environment
// @Router /v1/environment/{environment_id} [DELETE]
// @Summary Delete environment
// @Description Delete environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteEnvironment(c *gin.Context) {
	environmentID := c.Param("environment_id")

	if !util.IsValidUUID(environmentID) {
		h.handleResponse(c, http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.EnvironmentService().Delete(
		c.Request.Context(),
		&obs.EnvironmentPrimaryKey{Id: environmentID},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetAllEnvironments godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_environment_list
// @Router /v1/environment [GET]
// @Summary Get environment list
// @Description Get environment list
// @Tags Environment
// @Accept json
// @Produce json
// @Param filters query obs.GetEnvironmentListRequest true "filters"
// @Success 200 {object} http.Response{data=obs.GetEnvironmentListResponse} "EnvironmentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllEnvironments(c *gin.Context) {

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.EnvironmentService().GetList(
		c.Request.Context(),
		&obs.GetEnvironmentListRequest{
			Offset:    int32(offset),
			Limit:     int32(limit),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: c.DefaultQuery("project_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
