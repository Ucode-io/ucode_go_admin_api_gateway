package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_variable
// @Router /v1/analytics/variable [POST]
// @Summary Create variable
// @Description Create variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable body obs.CreateVariableRequest true "CreateVariableRequestBody"
// @Success 201 {object} status_http.Response{data=obs.Variable} "Variable data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateVariable(c *gin.Context) {
	var variable obs.CreateVariableRequest

	err := c.ShouldBindJSON(&variable)
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

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	variable.ProjectId = resourceEnvironment.GetId()

	// commitID, commitGuid, err := h.CreateAutoCommit(c, environmentId.(string), config.COMMIT_TYPE_VARIABLE)
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err))
	// 	return
	// }

	// variable.CommitId = commitID
	// variable.CommitGuid = commitGuid

	resp, err := services.BuilderService().Variable().Create(
		context.Background(),
		&variable,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetSingleVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_variable_by_id
// @Router /v1/analytics/variable/{variable_id} [GET]
// @Summary Get single variable
// @Description Get single variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable_id path string true "variable_id"
// @Success 200 {object} status_http.Response{data=obs.Variable} "VariableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleVariable(c *gin.Context) {
	variableID := c.Param("variable_id")

	if !util.IsValidUUID(variableID) {
		h.handleResponse(c, status_http.InvalidArgument, "variable id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.BuilderService().Variable().GetSingle(
		context.Background(),
		&obs.VariablePrimaryKey{
			Id:        variableID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_variable
// @Router /v1/analytics/variable [PUT]
// @Summary Update variable
// @Description Update variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable body obs.Variable true "UpdateVariableRequestBody"
// @Success 200 {object} status_http.Response{data=obs.Variable} "Variable data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateVariable(c *gin.Context) {
	var variable obs.Variable

	err := c.ShouldBindJSON(&variable)
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

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	variable.ProjectId = resourceEnvironment.GetId()

	resp, err := services.BuilderService().Variable().Update(
		context.Background(),
		&variable,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteVariable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_variable
// @Router /v1/analytics/variable/{variable_id} [DELETE]
// @Summary Delete variable
// @Description Delete variable
// @Tags Variable
// @Accept json
// @Produce json
// @Param variable_id path string true "variable_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteVariable(c *gin.Context) {
	variableID := c.Param("variable_id")

	if !util.IsValidUUID(variableID) {
		h.handleResponse(c, status_http.InvalidArgument, "variable id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	resp, err := services.BuilderService().Variable().Delete(
		context.Background(),
		&obs.VariablePrimaryKey{
			Id:        variableID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// GetAllVariables godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_variable_list
// @Router /v1/analytics/variable [GET]
// @Summary Get variable list
// @Description Get variable list
// @Tags Variable
// @Accept json
// @Produce json
// @Param filters query obs.GetAllVariablesRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllVariablesResponse} "VariableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllVariables(c *gin.Context) {

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	resp, err := services.BuilderService().Variable().GetList(
		context.Background(),
		&obs.GetAllVariablesRequest{
			Slug:        c.Query("slug"),
			DashboardId: c.Query("dashboard_id"),
			ProjectId:   resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
