package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/code_server"
	"ucode/ucode_go_api_gateway/pkg/easy_to_travel"
	"ucode/ucode_go_api_gateway/pkg/gitlab"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

const (
	FUNCTION = "FUNCTION"
)

var (
	waitFunctionResourceMap = caching.NewConcurrentMap()
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

	environment, err := h.companyServices.Environment().GetById(context.Background(), &company_service.EnvironmentPrimaryKey{
		Id: environmentId.(string),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	project, err := h.companyServices.Project().GetById(context.Background(), &company_service.GetProjectByIdRequest{
		ProjectId: environment.GetProjectId(),
	})
	if project.GetTitle() == "" {
		err = errors.New("error project name is required")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	projectName := strings.ReplaceAll(strings.TrimSpace(project.Title), " ", "-")
	projectName = strings.ToLower(projectName)
	var functionPath = projectName + "-" + function.Path

	resp, err := gitlab.CreateProjectFork(functionPath, gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.baseConf.GitlabIntegrationURL,
		GitlabIntegrationToken: h.baseConf.GitlabIntegrationToken,
		GitlabGroupId:          h.baseConf.GitlabGroupId,
		GitlabProjectId:        h.baseConf.GitlabProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	// fmt.Println("test before clone")
	var sshURL = resp.Message["ssh_url_to_repo"].(string)
	err = gitlab.CloneForkToPath(sshURL, h.baseConf)
	// fmt.Println("clone err::", err)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	uuid, _ := uuid.NewRandom()
	// fmt.Println("test after clone")
	// fmt.Println("uuid::", uuid.String())
	password, err := code_server.CreateCodeServer(projectName+"-"+function.Path, h.baseConf, uuid.String())
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	var url = "https://" + uuid.String() + ".u-code.io"

	response, err := services.FunctionService().FunctionService().Create(
		context.Background(),
		&fc.CreateFunctionRequest{
			Path:             functionPath,
			Name:             function.Name,
			Description:      function.Description,
			ProjectId:        resource.ResourceEnvironmentId,
			EnvironmentId:    environmentId.(string),
			FunctionFolderId: function.FunctionFolderId,
			Url:              url,
			Password:         password,
			SshUrl:           sshURL,
			Type:             FUNCTION,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, response)
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

	fmt.Println("\n URL function by id path >>", functionID, "\n")
	function, err := services.FunctionService().FunctionService().GetSingle(
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
		fmt.Println("clone err::", err)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
		uuid, _ := uuid.NewRandom()
		// fmt.Println("uuid::", uuid.String())
		password, err := code_server.CreateCodeServer(function.Path, h.baseConf, uuid.String())
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
		function.Url = "https://" + uuid.String() + ".u-code.io"
		function.Password = password
	}
	// var status int
	// for {
	// 	status, err = util.DoRequestCheckCodeServer(function.Url+"/?folder=/functions/"+function.Path, "GET", nil)
	// 	if status == 200 {
	// 		break
	// 	}
	// }

	function.ProjectId = resource.ResourceEnvironmentId
	_, err = services.FunctionService().FunctionService().Update(context.Background(), function)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

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
		&company_service.EnvironmentPrimaryKey{
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
	var function models.Function

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
	environment, err := h.companyServices.Environment().GetById(context.Background(), &company_service.EnvironmentPrimaryKey{
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

	resp, err := services.FunctionService().FunctionService().Update(
		context.Background(),
		&fc.Function{
			Id:               function.ID,
			Description:      function.Description,
			Name:             function.Name,
			Path:             function.Path,
			EnvironmentId:    environment.GetId(),
			ProjectId:        resource.ResourceEnvironmentId,
			FunctionFolderId: function.FuncitonFolderId,
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
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = h.companyServices.Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = h.companyServices.Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     environment.GetId(),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	_, err = services.FunctionService().FunctionService().Delete(
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

	h.handleResponse(c, status_http.NoContent, resp)
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
		// fmt.Println("error in do request", err)
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		// fmt.Println("error in response status", err)
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

	fmt.Println("\n\n\n TEst #1")

	bodyReq, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.log.Error("cant parse body or an empty body received", logger.Any("req", c.Request))
	}

	_ = json.Unmarshal(bodyReq, &invokeFunction)
	if err != nil {
		h.log.Error("cant parse body or an empty body received", logger.Any("req", c.Request))
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
	fmt.Println("\n\n\n TEst #2", projectId, environmentId)

	var (
		resourceKey = fmt.Sprintf("%s-%s", projectId.(string), environmentId.(string))
		resource    = &pb.ServiceResourceModel{}
	)

	// resourceTime := time.Now()
	var singleResourceWaitKey = config.CACHE_WAIT + "-single-resource"
	_, singleResourceOk := h.cache.Get(singleResourceWaitKey)
	if singleResourceOk {
		ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
		defer cancel()

		for {
			singleResourceBody, ok := h.cache.Get(resourceKey)
			if ok {
				err = json.Unmarshal(singleResourceBody, &resource)
				if err != nil {
					h.log.Error("Error while unmarshal resource redis", logger.Error(err))
					return
				}
			}

			if resource.ResourceEnvironmentId != "" {
				break
			}

			if ctx.Err() == context.DeadlineExceeded {
				break
			}

			time.Sleep(config.REDIS_SLEEP)
		}
	} else {
		h.cache.Add(singleResourceWaitKey, []byte(singleResourceWaitKey), config.REDIS_KEY_TIMEOUT)
	}
	fmt.Println("\n\n\n TEst #3")

	if resource.ResourceEnvironmentId == "" {
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

		body, err := json.Marshal(resource)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.cache.Add(resourceKey, body, config.REDIS_TIMEOUT)
	}
	// fmt.Println(">>>>>>>>>>>>>>>resourceTime:", time.Since(resourceTime))
	fmt.Println("\n\n\n TEst #4")
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
	keyParams := c.Request.URL.Query()
	var ettProductPath = []string{"easy-to-travel-get-products-agent-swagger", "b693cc12-8551-475f-91d5-4913c1739df4"}

	if helper.Contains(ettProductPath, c.Param("function-id")) {
		if len(keyParams.Get("startTime")) > 0 {
			keyParams.Del("startTime")
		}

		if len(keyParams.Get("endTime")) > 0 {
			keyParams.Del("endTime")
		}
	}

	var key = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("ett-%s-%s-%s", c.Request.Header.Get("Prev_path"), keyParams.Encode(), resource.ResourceEnvironmentId)))
	_, ok = h.cache.Get(config.CACHE_WAIT)
	fmt.Println("\n\n\n TEst #5")
	if c.Request.Method == "GET" && resource.ProjectId == "1acd7a8f-a038-4e07-91cb-b689c368d855" {
		if ok {

			redisDataTime := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				functionBody, ok := h.cache.Get(key)
				if ok {
					m := make(map[string]interface{})
					err = json.Unmarshal(functionBody, &m)
					if err != nil {
						h.handleResponse(c, status_http.GRPCError, err.Error())
						return
					}

					if _, ok := m["code"]; ok {
						c.JSON(cast.ToInt(m["code"]), m)
						return
					}

					if helper.Contains(ettProductPath, c.Param("function-id")) {
						data, err := easy_to_travel.EasyToTravelAgentApiGetProduct(requestData.Params, m)
						if err != nil {
							fmt.Println("Error while EasyToTravelAgentApiGetProduct function:", err.Error())
							result, _ := helper.InterfaceToMap(data)
							c.JSON(cast.ToInt(result["code"]), result)
							return
						}

						m, err = helper.InterfaceToMap(data)
						if err != nil {
							c.JSON(http.StatusInternalServerError, m)
							return
						}
					}

					c.JSON(cast.ToInt(m["code"]), m)
					fmt.Print("\n\n ~~>> ett redis return response ", time.Since(redisDataTime), "\n\n")
					return
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		} else {
			h.cache.Add(config.CACHE_WAIT, []byte(config.CACHE_WAIT), 20*time.Second)
		}
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
	} else {
		function.Path = c.Param("function-id")
	}
	// getSingleFunctionTime := time.Now()

	// fmt.Println(">>>>>>>>>>>>>>>getSingleFunctionTime:", time.Since(getSingleFunctionTime))

	// doRequestTime := time.Now()
	fmt.Println("\n\n\n TEst #6")
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
	// fmt.Println(">>>>>>>>>>>>>>>doRequestTime:", time.Since(doRequestTime))
	fmt.Println("\n\n\n TEst #7 1")
	if isOwnData, ok := resp.Attributes["is_own_data"].(bool); ok {
		if isOwnData {
			if err == nil && c.Request.Method == "GET" && resource.ProjectId == "1acd7a8f-a038-4e07-91cb-b689c368d855" {
				jsonData, _ := json.Marshal(resp.Data)
				h.cache.Add(key, []byte(jsonData), 20*time.Second)
			}

			if _, ok := resp.Data["code"]; ok {
				c.JSON(cast.ToInt(resp.Data["code"]), resp.Data)
				return
			}

			if helper.Contains(ettProductPath, c.Param("function-id")) {
				data, err := easy_to_travel.EasyToTravelAgentApiGetProduct(requestData.Params, resp.Data)
				time.Sleep(time.Millisecond * 50)
				if err != nil {
					fmt.Println("Error while EasyToTravelAgentApiGetProduct function:", err.Error())
					result, _ := helper.InterfaceToMap(data)
					c.JSON(cast.ToInt(result["code"]), result)
					return
				}

				resp.Data, err = helper.InterfaceToMap(data)
				if err != nil {
					fmt.Println("Error while InterfaceToMap function:", err.Error())
					c.JSON(http.StatusInternalServerError, resp.Data)
					return
				}
			}

			// DoRequestCount++
			// fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::::::::::::", DoRequestCount)

			c.JSON(200, resp.Data)
			return
		}
	}
	fmt.Println("\n\n\n TEst #8")
	h.handleResponse(c, status_http.OK, resp)
}

// var DoRequestCount int
