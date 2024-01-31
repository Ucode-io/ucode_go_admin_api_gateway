package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// fmt.Println("test before clone")
	// var sshURL = resp.Message["ssh_url_to_repo"].(string)
	// err = gitlab.CloneForkToPath(sshURL, h.baseConf)
	// fmt.Println("clone err::", err)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	uuid, _ := uuid.NewRandom()
	// fmt.Println("test after clone")
	// fmt.Println("uuid::", uuid.String())
	// password, err := code_server.CreateCodeServer(projectName+"-"+function.Path, h.baseConf, uuid.String())
	// if err != nil {
	// 	h.handleResponse(c, status_http.InvalidArgument, err.Error())
	// 	return
	// }
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
			//Password:         password,
			//SshUrl: sshURL,
			Type: FUNCTION,
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
		var resourceList *company_service.GetResourceByEnvIDResponse
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
		faasSettings = map[string]interface{}{}
		faasPaths    []string
	)

	for key, valueObject := range easy_to_travel.AgentApiPath {
		if strings.Contains(prevPath, key) {
			faasSettings = cast.ToStringMap(valueObject)
			faasPaths = cast.ToStringSlice(faasSettings["paths"])
			break
		}
	}

	if helper.ContainsLike(faasPaths, c.Param("function-id")) {
		faasSettings["app_id"] = authInfo.Data["app_id"]
		faasSettings["node_type"] = resource.NodeType
		faasSettings["project_id"] = projectId.(string)
		faasSettings["resource_environment_id"] = resource.ResourceEnvironmentId
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

func (h *HandlerV1) EasyToTravelFunctionRun(c *gin.Context, requestData models.HttpRequest, faasSettings map[string]interface{}) (map[string]interface{}, error) {

	var (
		isCache   bool
		faasPaths = cast.ToStringSlice(faasSettings["paths"])
		params    = c.Request.URL.Query()

		invokeFunction        models.InvokeFunctionRequest
		appId                 = cast.ToString(faasSettings["app_id"])
		nodeType              = cast.ToString(faasSettings["node_type"])
		projectId             = cast.ToString(faasSettings["project_id"])
		resourceEnvironmentId = cast.ToString(faasSettings["resource_environment_id"])
	)
	if faasSettings != nil {
		isCache = cast.ToBool(faasSettings["is_cache"])
	}

	if helper.Contains(faasPaths, c.Param("function-id")) {
		for _, param := range cast.ToStringSlice(faasSettings["delete_params"]) {
			params.Del(param)
		}
	}

	var key = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("ett-%s-%s-%s", c.Request.Header.Get("Prev_path"), params.Encode(), resourceEnvironmentId)))
	_, exists := h.cache.Get(config.CACHE_WAIT)
	if isCache {
		if exists {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				functionBody, ok := h.cache.Get(key)
				if ok {
					resp := make(map[string]interface{})
					err := json.Unmarshal(functionBody, &resp)
					if err != nil {
						h.log.Error("Error while json unmarshal", logger.Any("err", err))
						return nil, err
					}

					if _, ok := resp["code"]; ok {
						return resp, nil
					}

					if helper.Contains(faasPaths, c.Param("function-id")) && faasSettings["function_name"] != nil {
						var filters = map[string]interface{}{}
						for key, val := range c.Request.URL.Query() {
							if len(val) > 0 {
								filters[key] = val[0]
							}
						}
						resp["filters"] = filters

						data, err := easy_to_travel.AgentApiGetProduct(resp)
						if err != nil {
							h.log.Error("Error while EasyToTravelAgentApiGetProduct function:", logger.Any("err", err))
							result, _ := helper.InterfaceToMap(data)
							return result, nil
						}

						resp, err = helper.InterfaceToMap(data)
						if err != nil {
							h.log.Error("Error while InterfaceToMap:", logger.Any("err", err))
							return nil, err
						}
					}

					return resp, nil
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
			projectId,
			nodeType,
		)
		if err != nil {
			h.log.Error("Error while GetProjectSrvc", logger.Any("err", err))
			return nil, err
		}
		function, err = services.FunctionService().FunctionService().GetSingle(
			context.Background(),
			&fc.FunctionPrimaryKey{
				Id:        c.Param("function-id"),
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.log.Error("Error while function service GetSingle:", logger.Any("err", err))
			return nil, err
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
			"app_id":     appId,
		},
	})

	if err != nil {
		h.log.Error("Error while function DoRequest:", logger.Any("err", err))
		return nil, err
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}

		h.log.Error("Error while function DoRequest errStr:", logger.Any("err", err))
		return nil, errors.New(errStr)
	}

	if isOwnData, ok := resp.Attributes["is_own_data"].(bool); ok {
		if isOwnData {
			if err == nil && isCache {
				jsonData, _ := json.Marshal(resp.Data)
				h.cache.Add(key, []byte(jsonData), 20*time.Second)
			}

			if _, ok := resp.Data["code"]; ok {
				return resp.Data, nil
			}

			if helper.ContainsLike(faasPaths, c.Param("function-id")) && faasSettings["function_name"] != nil {
				var filters = map[string]interface{}{}
				for key, val := range c.Request.URL.Query() {
					if len(val) > 0 {
						filters[key] = val[0]
					}
				}
				resp.Data["filters"] = filters

				data, err := easy_to_travel.AgentApiGetProduct(resp.Data)
				if err != nil {
					fmt.Println("Error while EasyToTravelAgentApiGetProduct function:", err.Error())
					result, _ := helper.InterfaceToMap(data)
					return result, nil
				}

				resp.Data, err = helper.InterfaceToMap(data)
				if err != nil {
					h.log.Error("Error while InterfaceToMap function:", logger.Any("err", err))
					return nil, err
				}
			}

			return resp.Data, nil
		}
	}

	return resp.Data, nil
}
