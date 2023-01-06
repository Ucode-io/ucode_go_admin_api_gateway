package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID create_variable
// @Router /v1/analytics/variable [POST]
// @Summary Create variable
// @Description Create variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable body object_builder_service.CreateVariableRequest true "CreateVariableRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.Variable} "Variable data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateVariable(c *gin.Context) {
	var variable obs.CreateVariableRequest

	err := c.ShouldBindJSON(&variable)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	variable.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.VariableService().Create(
		context.Background(),
		&variable,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetSingleVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_variable_by_id
// @Router /v1/analytics/variable/{variable_id} [GET]
// @Summary Get single variable
// @Description Get single variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable_id path string true "variable_id"
// @Success 200 {object} http.Response{data=object_builder_service.Variable} "VariableBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleVariable(c *gin.Context) {
	variableID := c.Param("variable_id")

	if !util.IsValidUUID(variableID) {
		h.handleResponse(c, http.InvalidArgument, "variable id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := services.VariableService().GetSingle(
		context.Background(),
		&obs.VariablePrimaryKey{
			Id:        variableID,
			ProjectId: resourceId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID update_variable
// @Router /v1/analytics/variable [PUT]
// @Summary Update variable
// @Description Update variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable body object_builder_service.Variable true "UpdateVariableRequestBody"
// @Success 200 {object} http.Response{data=object_builder_service.Variable} "Variable data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateVariable(c *gin.Context) {
	var variable obs.Variable

	err := c.ShouldBindJSON(&variable)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	variable.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.VariableService().Update(
		context.Background(),
		&variable,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID delete_variable
// @Router /v1/analytics/variable/{variable_id} [DELETE]
// @Summary Delete variable
// @Description Delete variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable_id path string true "variable_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteVariable(c *gin.Context) {
	variableID := c.Param("variable_id")

	if !util.IsValidUUID(variableID) {
		h.handleResponse(c, http.InvalidArgument, "variable id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	resp, err := services.VariableService().Delete(
		context.Background(),
		&obs.VariablePrimaryKey{
			Id:        variableID,
			ProjectId: resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetAllVariables godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_variable_list
// @Router /v1/analytics/variable [GET]
// @Summary Get variable list
// @Description Get variable list
// @Tags Variable
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllVariablesRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetAllVariablesResponse} "VariableBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllVariables(c *gin.Context) {

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	resp, err := services.VariableService().GetList(
		context.Background(),
		&obs.GetAllVariablesRequest{
			Slug:        c.Query("slug"),
			DashboardId: c.Query("dashboard_id"),
			ProjectId:   resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
