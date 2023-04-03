package handlers

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// CreateFunctionFolder godoc
// @Security ApiKeyAuth
// @ID create_function_folder
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @Router /v1/function-folder [POST]
// @Summary Create Function Folder
// @Description Create Function Folder
// @Tags FunctionFolder
// @Accept json
// @Produce json
// @Param Function body models.CreateFunctionFolderRequest true "CreateFunctionFolderRequestBody"
// @Success 201 {object} status_http.Response{data=models.FunctionFolder} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateFunctionFolder(c *gin.Context) {

	fmt.Println("test 11")
	var functionFolder models.CreateFunctionFolderRequest
	var resourceEnvironment *obs.ResourceEnvironment
	err := c.ShouldBindJSON(&functionFolder)
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get en"))
		return
	}
	fmt.Println("test 12")
	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	environment, err := services.CompanyService().Environment().GetById(context.Background(), &company_service.EnvironmentPrimaryKey{
		Id: environmentId.(string),
	})
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     environment.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	resp, err := services.FunctionService().FunctionFolderService().Create(
		context.Background(),
		&fc.CreateFunctionFolderRequest{
			Title:         functionFolder.Title,
			Description:   functionFolder.Description,
			ProjectId:     resourceEnvironment.GetId(),
			EnvironmentId: environmentId.(string),
		},
	)

	fmt.Println("test 13")
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	fmt.Println("test 14")

	h.handleResponse(c, status_http.Created, resp)
}

// GetFunctionFolderById godoc
// @Security ApiKeyAuth
// @Param Environment-Id header string true "Environment-Id"
// @ID get_function_folder_by_id
// @Router /v1/function-folder/{function_folder_id} [GET]
// @Summary Get Function by id
// @Description Get Function by id
// @Tags FunctionFolder
// @Accept json
// @Produce json
// @Param function_folder_id path string true "function_folder_id"
// @Success 200 {object} status_http.Response{data=models.FunctionFolder} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetFunctionFolderById(c *gin.Context) {
	functionFolderID := c.Param("function_folder_id")
	var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionFolderID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get en"))
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	environment, err := services.CompanyService().Environment().GetById(
		context.Background(),
		&company_service.EnvironmentPrimaryKey{
			Id: environmentId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     environment.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	resp, err := services.FunctionService().FunctionService().GetSingle(
		context.Background(),
		&fc.FunctionPrimaryKey{
			Id:        functionFolderID,
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
// @Param Environment-Id header string true "Environment-Id"
// @ID get_all_function_folders
// @Router /v1/function-folder [GET]
// @Summary Get all function folders
// @Description Get all function folders
// @Tags FunctionFolder
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllFunctionFolder(c *gin.Context) {

	var resourceEnvironment *obs.ResourceEnvironment
	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	offset, err := h.getOffsetParam(c)
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get en"))
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	environment, err := services.CompanyService().Environment().GetById(
		context.Background(),
		&company_service.EnvironmentPrimaryKey{
			Id: environmentId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     environment.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	fmt.Println("test eeee")
	resp, err := services.FunctionService().FunctionFolderService().GetList(
		context.Background(),
		&fc.GetAllFunctionFoldersRequest{
			Search:        c.DefaultQuery("search", ""),
			Limit:         int32(limit),
			Offset:        int32(offset),
			ProjectId:     resourceEnvironment.GetId(),
			EnvironmentId: environment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateFunctionFolder godoc
// @Security ApiKeyAuth
// @Param Environment-Id header string true "Environment-Id"
// @ID update_function_folder
// @Router /v1/function-folder [PUT]
// @Summary Update function folder
// @Description Update function folder
// @Tags FunctionFolder
// @Accept json
// @Produce json
// @Param Function body models.FunctionFolder  true "UpdateFunctionFolderRequestBody"
// @Success 200 {object} status_http.Response{data=models.FunctionFolder} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateFunctionFolder(c *gin.Context) {
	var functionFolder models.FunctionFolder
	var resourceEnvironment *obs.ResourceEnvironment

	err := c.ShouldBindJSON(&functionFolder)
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get en"))
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	environment, err := services.CompanyService().Environment().GetById(
		context.Background(),
		&company_service.EnvironmentPrimaryKey{
			Id: environmentId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     environment.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	_, err = services.FunctionService().FunctionFolderService().Update(
		context.Background(),
		&fc.FunctionFolder{
			Id:            functionFolder.Id,
			Description:   functionFolder.Description,
			Title:         functionFolder.Title,
			EnvironmentId: environment.GetId(),
			ProjectId:     resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, functionFolder)
}

// DeleteFunctionFolder godoc
// @Security ApiKeyAuth
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_function_folder
// @Router /v1/function-folder/{function_folder_id} [DELETE]
// @Summary Delete Function Folder
// @Description Delete Function Folder
// @Tags FunctionFolder
// @Accept json
// @Produce json
// @Param function_folder_id path string true "function_folder_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteFunctionFolder(c *gin.Context) {
	functionFolderID := c.Param("function_folder_id")
	var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionFolderID) {
		h.handleResponse(c, status_http.InvalidArgument, "function folder id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get en"))
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	environment, err := services.CompanyService().Environment().GetById(
		context.Background(),
		&company_service.EnvironmentPrimaryKey{
			Id: environmentId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     environment.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.FunctionService().FunctionFolderService().Delete(
		context.Background(),
		&fc.FunctionFolderPrimaryKey{
			Id:        functionFolderID,
			ProjectId: resourceEnvironment.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
