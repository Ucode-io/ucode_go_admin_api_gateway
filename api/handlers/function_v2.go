package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/code_server"
	"ucode/ucode_go_api_gateway/pkg/gitlab_integration"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateNewFunction godoc
// @Security ApiKeyAuth
// @ID create_new_function
// @Router /v1/new/function [POST]
// @Summary Create New Function
// @Description Create New Function
// @Tags Function
// @Accept json
// @Produce json
// @Param project_id query string true "project_id"
// @Param Function body models.CreateFunctionRequest true "CreateFunctionRequestBody"
// @Success 201 {object} status_http.Response{data=models.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateNewFunction(c *gin.Context) {
	var function models.CreateFunctionRequest
	err := c.ShouldBindJSON(&function)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	// structData, err := helper.ConvertMapToStruct(function.Body)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	if !util.IsValidFunctionName(function.Path) {
		h.handleResponse(c, status_http.InvalidArgument, "function path must be contains [a-z] and hyphen")
		return
	}
	project, err := services.CompanyService().Project().GetById(context.Background(), &company_service.GetProjectByIdRequest{
		ProjectId: c.DefaultQuery("project_id", ""),
	})

	if project.Title == "" {
		err = errors.New("error project name is required")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	projectName := strings.TrimSpace(project.Title)
	projectName = strings.ToLower(projectName)

	resp, err := gitlab_integration.CreateProjectFork(h.cfg, projectName+"-"+function.Path)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	fmt.Println("test before clone")
	err = gitlab_integration.CloneForkToPath(resp.Message["http_url_to_repo"].(string), h.cfg)
	fmt.Println("clone err::", err)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	uuid, _ := uuid.NewRandom()
	fmt.Println("test after clone")
	password, err := code_server.CreateCodeServer(projectName+"-"+function.Path, h.cfg, uuid.String())
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	responseMap := map[string]string{
		"password": password,
		"url":      "https://" + uuid.String() + ".u-code.io",
	}

	// response, err := services.FunctionService().FunctionService().Create(
	// 	context.Background(),
	// 	&fc.CreateFunctionRequest{
	// 		Path:        function.Path,
	// 		Name:        function.Name,
	// 		Description: function.Description,
	// 		Body:        structData,
	// 		ProjectId:   project.ProjectId,
	// 	},
	// )

	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	h.handleResponse(c, status_http.Created, responseMap)
}

// GetNewFunctionByID godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_new_function_by_id
// @Router /v1/new/function/{function_id} [GET]
// @Summary Get Function by id
// @Description Get Function by id
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 200 {object} status_http.Response{data=models.Function} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetNewFunctionByID(c *gin.Context) {
	functionID := c.Param("function_id")

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
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

	resp, err := services.FunctionService().FunctionService().GetSingle(
		context.Background(),
		&fc.FunctionPrimaryKey{
			Id:        functionID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllFunctions godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_all_new_functions
// @Router /v1/new/function [GET]
// @Summary Get all functions
// @Description Get all functions
// @Tags Function
// @Accept json
// @Produce json
// @Param filters query new_function_service.GetAllFunctionsRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllNewFunctions(c *gin.Context) {

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

	resp, err := services.FunctionService().FunctionService().GetList(
		context.Background(),
		&fc.GetAllFunctionsRequest{
			Search:    c.DefaultQuery("search", ""),
			Limit:     int32(limit),
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateNewFunction godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_new_function
// @Router /v1/function [PUT]
// @Summary Update new function
// @Description Update new function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.Function  true "UpdateFunctionRequestBody"
// @Success 200 {object} status_http.Response{data=models.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateNewFunction(c *gin.Context) {
	var function models.Function

	err := c.ShouldBindJSON(&function)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(function.Body)
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

	resp, err := services.FunctionService().FunctionService().Update(
		context.Background(),
		&fc.Function{
			Id:          function.ID,
			Description: function.Description,
			Name:        function.Name,
			Path:        function.Path,
			Body:        structData,
			ProjectId:   resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteNewFunction godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_new_function
// @Router /v1/function/{function_id} [DELETE]
// @Summary Delete Function
// @Description Delete Function
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteNewFunction(c *gin.Context) {
	functionID := c.Param("function_id")

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
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

	resp, err := services.BuilderService().Function().Delete(
		context.Background(),
		&obs.FunctionPrimaryKey{
			Id:        functionID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
