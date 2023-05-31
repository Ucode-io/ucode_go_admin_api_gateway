package handlers

import (
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	obs "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateEnvironment godoc
// @Security ApiKeyAuth
// @ID create_environment
// @Router /v1/environment [POST]
// @Summary Create environment
// @Description Create environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment body obs.CreateEnvironmentRequest true "CreateEnvironmentRequestBody"
// @Success 201 {object} status_http.Response{data=obs.Environment} "Environment data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateEnvironment(c *gin.Context) {
	var environmentRequest obs.CreateEnvironmentRequest

	err := c.ShouldBindJSON(&environmentRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Environment().Create(
		c.Request.Context(),
		&environmentRequest,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetSingleEnvironment godoc
// @Security ApiKeyAuth
// @ID get_environment_by_id
// @Router /v1/environment/{environment_id} [GET]
// @Summary Get single environment
// @Description Get single environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Success 200 {object} status_http.Response{data=obs.Environment} "EnvironmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleEnvironment(c *gin.Context) {
	environmentID := c.Param("environment_id")

	if !util.IsValidUUID(environmentID) {
		h.handleResponse(c, status_http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.CompanyService().Environment().GetById(
		c.Request.Context(),
		&obs.EnvironmentPrimaryKey{
			Id: environmentID,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateEnvironment godoc
// @Security ApiKeyAuth
// @ID update_environment
// @Router /v1/environment [PUT]
// @Summary Update environment
// @Description Update environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment body models.Environment true "UpdateEnvironmentRequestBody"
// @Success 200 {object} status_http.Response{data=obs.Environment} "Environment data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateEnvironment(c *gin.Context) {
	var environment models.Environment

	err := c.ShouldBindJSON(&environment)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(environment.Data)

	resp, err := h.companyServices.CompanyService().Environment().Update(
		c.Request.Context(),
		&obs.Environment{
			Id:           environment.Id,
			ProjectId:    environment.ProjectId,
			Name:         environment.Name,
			DisplayColor: environment.DisplayColor,
			Description:  environment.Description,
			Data:         structData,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteEnvironment godoc
// @Security ApiKeyAuth
// @ID delete_environment
// @Router /v1/environment/{environment_id} [DELETE]
// @Summary Delete environment
// @Description Delete environment
// @Tags Environment
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteEnvironment(c *gin.Context) {
	environmentID := c.Param("environment_id")

	if !util.IsValidUUID(environmentID) {
		h.handleResponse(c, status_http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.CompanyService().Environment().Delete(
		c.Request.Context(),
		&obs.EnvironmentPrimaryKey{Id: environmentID},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// GetAllEnvironments godoc
// @Security ApiKeyAuth
// @ID get_environment_list
// @Router /v1/environment [GET]
// @Summary Get environment list
// @Description Get environment list
// @Tags Environment
// @Accept json
// @Produce json
// @Param filters query obs.GetEnvironmentListRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetEnvironmentListResponse} "EnvironmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllEnvironments(c *gin.Context) {

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

	resp, err := h.companyServices.CompanyService().Environment().GetList(
		c.Request.Context(),
		&obs.GetEnvironmentListRequest{
			Offset:    int32(offset),
			Limit:     int32(limit),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: c.DefaultQuery("project_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
