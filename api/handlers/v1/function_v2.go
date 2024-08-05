package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/code_server"
	"ucode/ucode_go_api_gateway/pkg/easy_to_travel"
	"ucode/ucode_go_api_gateway/pkg/gitlab"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

const (
	FUNCTION = "FUNCTION"
)

// CreateNewFunction godoc
// @Security ApiKeyAuth
// @ID create_new_function
// @Router /v2/function [POST]
// @Summary Create New Function
// @Description Create New Function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.CreateFunctionRequest true "CreateFunctionRequestBody"
// @Success 201 {object} status_http.Response{data=fc.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateNewFunction(c *gin.Context) {
	var (
		function models.CreateFunctionRequest
		response = &fc.Function{}
		//resourceEnvironment *obs.ResourceEnvironment
	)
	err := c.ShouldBindJSON(&function)
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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

	environment, err := h.companyServices.Environment().GetById(context.Background(), &pb.EnvironmentPrimaryKey{
		Id: environmentId.(string),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	project, _ := h.companyServices.Project().GetById(context.Background(), &pb.GetProjectByIdRequest{
		ProjectId: environment.GetProjectId(),
	})
	if project.GetTitle() == "" {
		err = errors.New("error project name is required")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	fmt.Println("UNTIL HERE OK")
	projectName := strings.ReplaceAll(strings.TrimSpace(project.Title), " ", "-")
	projectName = strings.ToLower(projectName)
	var functionPath = projectName + "-" + function.Path

	_, err = gitlab.CreateProjectFork(functionPath, gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.baseConf.GitlabIntegrationURL,
		GitlabIntegrationToken: h.baseConf.GitlabIntegrationToken,
		GitlabGroupId:          h.baseConf.GitlabGroupId,
		GitlabProjectId:        h.baseConf.GitlabProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	// var sshURL = resp.Message["ssh_url_to_repo"].(string)
	// err = gitlab.CloneForkToPath(sshURL, h.baseConf)
	uuid, _ := uuid.NewRandom()
	// password, err := code_server.CreateCodeServer(projectName+"-"+function.Path, h.baseConf, uuid.String())
	// if err != nil {
	// 	h.handleResponse(c, status_http.InvalidArgument, err.Error())
	// 	return
	// }
	var url = "https://" + uuid.String() + ".u-code.io"

	var (
		createFunction = &fc.CreateFunctionRequest{
			Path:             functionPath,
			Name:             function.Name,
			Description:      function.Description,
			ProjectId:        resource.ResourceEnvironmentId,
			EnvironmentId:    environmentId.(string),
			FunctionFolderId: function.FunctionFolderId,
			Url:              url,
			//Password:         password,
			//SshUrl: sshURL,
			Type: FUNCTION,
		}
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
			Request:   createFunction,
			TableSlug: "FUNCTION",
		}
	)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		response, err = services.FunctionService().FunctionService().Create(
			context.Background(),
			createFunction,
		)

		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = response
			h.handleResponse(c, status_http.Created, response)
		}
		go h.versionHistory(c, logReq)
	case pb.ResourceType_POSTGRESQL:

		newCreateFunction := &nb.CreateFunctionRequest{}

		err = helper.MarshalToStruct(createFunction, &newCreateFunction)
		if err != nil {
			return
		}

		response, err := services.GoObjectBuilderService().Function().Create(
			context.Background(),
			newCreateFunction,
		)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = response
			h.handleResponse(c, status_http.Created, response)
		}
		go h.versionHistory(c, logReq)
	}
}

// GetNewFunctionByID godoc
// @Security ApiKeyAuth
// @ID get_new_function_by_id
// @Router /v2/function/{function_id} [GET]
// @Summary Get Function by id
// @Description Get Function by id
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 200 {object} status_http.Response{data=fc.Function} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetNewFunctionByID(c *gin.Context) {
	functionID := c.Param("function_id")
	//var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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

	var function = &fc.Function{}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		function, err = services.FunctionService().FunctionService().GetSingle(
			context.Background(),
			&fc.FunctionPrimaryKey{
				Id:        functionID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		if function.Url == "" {
			err = gitlab.CloneForkToPath(function.GetSshUrl(), h.baseConf)
			if err != nil {
				h.handleResponse(c, status_http.InvalidArgument, err.Error())
				return
			}
			uuid, _ := uuid.NewRandom()
			password, err := code_server.CreateCodeServer(function.Path, h.baseConf, uuid.String())
			if err != nil {
				h.handleResponse(c, status_http.InvalidArgument, err.Error())
				return
			}
			function.Url = "https://" + uuid.String() + ".u-code.io"
			function.Password = password
		}

		function.ProjectId = resource.ResourceEnvironmentId
		_, err = services.FunctionService().FunctionService().Update(context.Background(), function)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Function().GetSingle(
			context.Background(),
			&nb.FunctionPrimaryKey{
				Id:        functionID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		err = helper.MarshalToStruct(resp, &function)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	// var status int
	// for {
	// 	status, err = util.DoRequestCheckCodeServer(function.Url+"/?folder=/functions/"+function.Path, "GET", nil)
	// 	if status == 200 {
	// 		break
	// 	}
	// }

	h.handleResponse(c, status_http.OK, function)
}

// GetAllNewFunctions godoc
// @Security ApiKeyAuth
// @ID get_all_new_functions
// @Router /v2/function [GET]
// @Summary Get all functions
// @Description Get all functions
// @Tags Function
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllNewFunctions(c *gin.Context) {

	//var resourceEnvironment *obs.ResourceEnvironment
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	environment, err := h.companyServices.Environment().GetById(
		context.Background(),
		&pb.EnvironmentPrimaryKey{
			Id: environmentId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
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

		resp, err := services.FunctionService().FunctionService().GetList(
			context.Background(),
			&fc.GetAllFunctionsRequest{
				Search:        c.DefaultQuery("search", ""),
				Limit:         int32(limit),
				Offset:        int32(offset),
				ProjectId:     resource.ResourceEnvironmentId,
				EnvironmentId: environment.GetId(),
				Type:          FUNCTION,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Function().GetList(
			context.Background(),
			&nb.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
				Type:      FUNCTION,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	}

}

// UpdateNewFunction godoc
// @Security ApiKeyAuth
// @ID update_new_function
// @Router /v2/function [PUT]
// @Summary Update new function
// @Description Update new function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.Function  true "UpdateFunctionRequestBody"
// @Success 200 {object} status_http.Response{data=models.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateNewFunction(c *gin.Context) {
	var (
		function models.Function
		resp     = &empty.Empty{}
	)

	//var resourceEnvironment *obs.ResourceEnvironment
	err := c.ShouldBindJSON(&function)
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	environment, _ := h.companyServices.Environment().GetById(context.Background(), &pb.EnvironmentPrimaryKey{
		Id: environmentId.(string),
	})

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
		updateFunction = &fc.Function{
			Id:               function.ID,
			Description:      function.Description,
			Name:             function.Name,
			Path:             function.Path,
			EnvironmentId:    environment.GetId(),
			ProjectId:        resource.ResourceEnvironmentId,
			FunctionFolderId: function.FuncitonFolderId,
		}
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
			Request:   &updateFunction,
			TableSlug: "FUNCTION",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	resp, err = services.FunctionService().FunctionService().Update(
		context.Background(),
		updateFunction,
	)
	if err != nil {
		return
	}
}

// DeleteNewFunction godoc
// @Security ApiKeyAuth
// @ID delete_new_function
// @Router /v2/function/{function_id} [DELETE]
// @Summary Delete New Function
// @Description Delete New Function
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteNewFunction(c *gin.Context) {
	functionID := c.Param("function_id")
	//var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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

	resp, err := services.FunctionService().FunctionService().GetSingle(
		context.Background(),
		&fc.FunctionPrimaryKey{
			Id:        functionID,
			ProjectId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	// delete code server
	err = code_server.DeleteCodeServerByPath(resp.Path, h.baseConf)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	// delete cloned repo
	err = gitlab.DeletedClonedRepoByPath(resp.Path, h.baseConf)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	// delete repo by path from gitlab
	_, err = gitlab.DeleteForkedProject(resp.Path, h.baseConf)
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
			TableSlug: "FUNCTION",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	_, err = services.FunctionService().FunctionService().Delete(
		context.Background(),
		&fc.FunctionPrimaryKey{
			Id:        functionID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}
}

// GetAllNewFunctionsForApp godoc
// @Security ApiKeyAuth
// @ID get_all_new_functions_for_app
// @Router /v2/function-for-app [GET]
// @Summary Get all functions
// @Description Get all functions
// @Tags Function
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllNewFunctionsForApp(c *gin.Context) {

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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
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
		resp, err := services.FunctionService().FunctionService().GetList(
			context.Background(),
			&fc.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
				Type:      FUNCTION,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Function().GetList(
			context.Background(),
			&nb.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
				Type:      FUNCTION,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	}
}

// InvokeFunctionByPath godoc
// @Security ApiKeyAuth
// @Param function-path path string true "function-path"
// @ID invoke_function_by_path
// @Router /v1/invoke_function/{function-path} [POST]
// @Summary Invoke Function By Path
// @Description Invoke Function By Path
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionByPathRequest body models.CommonMessage true "InvokeFunctionByPathRequest"
// @Success 201 {object} status_http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) InvokeFunctionByPath(c *gin.Context) {
	var invokeFunction models.CommonMessage

	err := c.ShouldBindJSON(&invokeFunction)
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	apiKeys, err := h.authService.ApiKey().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     resource.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if len(apiKeys.Data) < 1 {
		h.handleResponse(c, status_http.InvalidArgument, "Api key not found")
		return
	}
	authInfo, _ := h.GetAuthInfo(c)
	invokeFunction.Data["user_id"] = authInfo.GetUserId()
	invokeFunction.Data["project_id"] = authInfo.GetProjectId()
	invokeFunction.Data["environment_id"] = authInfo.GetEnvId()
	invokeFunction.Data["app_id"] = apiKeys.GetData()[0].GetAppId()

	resp, err := util.DoRequest("https://ofs.u-code.io/function/"+c.Param("function-path"), "POST", models.NewInvokeFunctionRequest{
		Data: invokeFunction.Data,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status_http.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// FunctionRun godoc
// @Security ApiKeyAuth
// @ID function_run
// @Router /v2/functions/{function-id}/run [POST]
// @Summary Function Run
// @Description Function Run
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionRequest body models.InvokeFunctionRequest true "InvokeFunctionRequest"
// @Success 200 {object} status_http.Response{data=models.InvokeFunctionResponse} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) FunctionRun(c *gin.Context) {
	var (
		requestData    models.HttpRequest
		invokeFunction models.InvokeFunctionRequest
	)

	bodyReq, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.log.Error("cant parse body or an empty body received", logger.Any("req", c.Request))
	}

	_ = json.Unmarshal(bodyReq, &invokeFunction)
	if err != nil {
		h.log.Error("cant parse body or an empty body received", logger.Any("req", c.Request))
	}

	if cast.ToBool(c.GetHeader("/v1/functions/")) {
		var authData = models.AuthData{}
		err = json.Unmarshal([]byte(c.GetHeader("auth")), &authData)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, "cant get auth info")
			return
		}

		c.Set("auth", authData)
		c.Set("resource_id", c.GetHeader("resource_id"))
		c.Set("environment_id", c.GetHeader("environment_id"))
		c.Set("project_id", c.GetHeader("project_id"))
		c.Set("resource", c.GetHeader("resource"))
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

	var resource *pb.ServiceResourceModel
	resourceBody, ok := c.Get("resource")
	if resourceBody != "" && ok {
		var resourceList *pb.GetResourceByEnvIDResponse
		err = json.Unmarshal([]byte(resourceBody.(string)), &resourceList)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		for _, resourceObject := range resourceList.ServiceResources {
			if resourceObject.Title == pb.ServiceType_name[1] {
				resource = &pb.ServiceResourceModel{
					Id:                    resourceObject.Id,
					ServiceType:           resourceObject.ServiceType,
					ProjectId:             resourceObject.ProjectId,
					Title:                 resourceObject.Title,
					ResourceId:            resourceObject.ResourceId,
					ResourceEnvironmentId: resourceObject.ResourceEnvironmentId,
					EnvironmentId:         resourceObject.EnvironmentId,
					ResourceType:          resourceObject.ResourceType,
					NodeType:              resourceObject.NodeType,
				}
				break
			}
		}
	} else {
		resource, err = h.companyServices.ServiceResource().GetSingle(
			c.Request.Context(),
			&pb.GetSingleServiceResourceReq{
				ProjectId:     projectId.(string),
				EnvironmentId: environmentId.(string),
				ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	authInfoAny, ok := c.Get("auth")
	if !ok {
		h.handleResponse(c, status_http.InvalidArgument, "cant get auth info")
		return
	}

	authInfo := authInfoAny.(models.AuthData)
	requestData.Method = c.Request.Method
	requestData.Headers = c.Request.Header
	requestData.Path = c.Request.URL.Path
	requestData.Params = c.Request.URL.Query()
	requestData.Body = bodyReq
	var (
		prevPath     = c.Request.Header.Get("Prev_path")
		faasSettings easy_to_travel.FaasSetting
		faasPaths    []string
	)

	for key, valueObject := range easy_to_travel.AgentApiPath {
		if strings.Contains(prevPath, key) {
			var (
				mu    = sync.Mutex{}
				value = cast.ToStringMap(valueObject)
			)

			mu.Lock()
			faasSettings = easy_to_travel.FaasSetting{
				Paths:        cast.ToStringSlice(value["paths"]),
				IsCache:      cast.ToBool(value["is_cache"]),
				Continue:     cast.ToBool(value["continue"]),
				DeleteParams: cast.ToStringSlice(value["delete_params"]),
			}

			if function, ok := value["function"].(func(services.ServiceManagerI, []byte) string); ok {
				faasSettings.Function = function
			}
			faasPaths = faasSettings.Paths
			mu.Unlock()

			break
		}
	}

	if helper.ContainsLike(faasPaths, c.Param("function-id")) {
		faasSettings.AppId = cast.ToString(authInfo.Data["app_id"])
		faasSettings.NodeType = resource.NodeType
		faasSettings.ProjectId = projectId.(string)
		faasSettings.ResourceEnvironmentId = resource.ResourceEnvironmentId

		resp, err := h.EasyToTravelFunctionRun(c, requestData, faasSettings)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		c.JSON(cast.ToInt(resp["code"]), resp)
		return
	}

	var function = &fc.Function{}
	if util.IsValidUUID(c.Param("function-id")) {
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
			function, err = services.FunctionService().FunctionService().GetSingle(
				context.Background(),
				&fc.FunctionPrimaryKey{
					Id:        c.Param("function-id"),
					ProjectId: resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case pb.ResourceType_POSTGRESQL:
			resp, err := services.GoObjectBuilderService().Function().GetSingle(
				context.Background(),
				&nb.FunctionPrimaryKey{
					Id:        c.Param("function-id"),
					ProjectId: resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			err = helper.MarshalToStruct(resp, &function)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	} else {
		function.Path = c.Param("function-id")
	}

	resp, err := util.DoRequest("https://ofs.u-code.io/function/"+function.Path, "POST", models.FunctionRunV2{
		Auth:        models.AuthData{},
		RequestData: requestData,
		Data: map[string]interface{}{
			"object_ids": invokeFunction.ObjectIDs,
			"attributes": invokeFunction.Attributes,
			"app_id":     authInfo.Data["app_id"],
		},
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status_http.InvalidArgument, errStr)
		return
	}

	if isOwnData, ok := resp.Attributes["is_own_data"].(bool); ok {
		if isOwnData {
			if _, ok := resp.Data["code"]; ok {
				c.JSON(cast.ToInt(resp.Data["code"]), resp.Data)
				return
			}

			c.JSON(200, resp.Data)
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// InvokeFunctionByPath godoc
// @ID wayll-payment
// @Router /v1/wayll-payment [POST]
// @Summary Invoke Function By Path
// @Description Invoke Function By Path
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionByPathRequest body models.Wayll true "InvokeFunctionByPathRequest"
// @Success 201 {object} status_http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) WayllPayment(c *gin.Context) {
	var wayll models.Wayll
	err := c.ShouldBindJSON(&wayll)
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
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	apiKeys, err := h.authService.ApiKey().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     resource.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if len(apiKeys.Data) < 1 {
		h.handleResponse(c, status_http.InvalidArgument, "Api key not found")
		return
	}

	invokeFunction := models.CommonMessage{}

	invokeFunction.Data = make(map[string]interface{})

	invokeFunction.Data = map[string]interface{}{
		"data": wayll,
	}

	authInfo, _ := h.GetAuthInfo(c)
	invokeFunction.Data["user_id"] = authInfo.GetUserId()
	invokeFunction.Data["project_id"] = authInfo.GetProjectId()
	invokeFunction.Data["environment_id"] = authInfo.GetEnvId()
	invokeFunction.Data["app_id"] = apiKeys.GetData()[0].GetAppId()

	resp, err := util.DoRequest("https://ofs.u-code.io/function/wayll-payment", "POST", models.NewInvokeFunctionRequest{
		Data: invokeFunction.Data,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status_http.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
