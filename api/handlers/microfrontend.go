package handlers

import (
	"context"
	"errors"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/code_server"
	"ucode/ucode_go_api_gateway/pkg/gitlab"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

const (
	MICROFE = "MICRO_FRONTEND"
)

// CreateMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID create_micro_frontend
// @Router /v2/functions/micro-frontend [POST]
// @Summary Create Micro Frontend
// @Description Create Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param MicroFrontend body models.CreateFunctionRequest true "MicroFrontend"
// @Success 201 {object} status_http.Response{data=fc.Function} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateMicroFrontEnd(c *gin.Context) {
	var (
		function models.CreateFunctionRequest
		//resourceEnvironment *obs.ResourceEnvironment
	)
	err := c.ShouldBindJSON(&function)
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

	if !util.IsValidFunctionName(function.Path) {
		h.handleResponse(c, status_http.InvalidArgument, "function path must be contains [a-z] and hyphen and numbers")
		return
	}
	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get en"))
	//	return
	//}
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	environment, err := services.CompanyService().Environment().GetById(context.Background(), &company_service.EnvironmentPrimaryKey{
		Id: environmentId.(string),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	project, err := services.CompanyService().Project().GetById(context.Background(), &company_service.GetProjectByIdRequest{
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

	respCreateFork, err := gitlab.CreateProjectFork(functionPath, gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.cfg.GitlabIntegrationURL,
		GitlabIntegrationToken: h.cfg.GitlabIntegrationToken,
		GitlabProjectId:        h.cfg.GitlabProjectIdMicroFE,
		GitlabGroupId:          h.cfg.GitlabGroupIdMicroFE,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	_, err = gitlab.UpdateProject(gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.cfg.GitlabIntegrationURL,
		GitlabIntegrationToken: h.cfg.GitlabIntegrationToken,
		GitlabProjectId:        respCreateFork.Message["id"].(int),
		GitlabGroupId:          h.cfg.GitlabGroupIdMicroFE,
	}, map[string]interface{}{
		"ci_config_path": ".gitlab-ci.yml",
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	_, err = gitlab.CreatePipeline(gitlab.IntegrationData{
		GitlabIntegrationUrl:   h.cfg.GitlabIntegrationURL,
		GitlabIntegrationToken: h.cfg.GitlabIntegrationToken,
		GitlabProjectId:        respCreateFork.Message["id"].(int),
		GitlabGroupId:          h.cfg.GitlabGroupIdMicroFE,
	}, map[string]interface{}{})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	response, err := services.FunctionService().FunctionService().Create(
		context.Background(),
		&fc.CreateFunctionRequest{
			Path:             functionPath,
			Name:             function.Name,
			Description:      function.Description,
			ProjectId:        resource.ResourceEnvironmentId,
			EnvironmentId:    environmentId.(string),
			FunctionFolderId: function.FunctionFolderId,
			Type:             MICROFE,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, response)
}

// GetMicroFrontEndByID godoc
// @Security ApiKeyAuth
// @ID get_micro_frontend_by_id
// @Router /v2/functions/micro-frontend/{micro-frontend-id} [GET]
// @Summary Get Micro Frontend By Id
// @Description Get Micro Frontend By Id
// @Tags Functions
// @Accept json
// @Produce json
// @Param micro-frontend-id path string true "micro-frontend-id"
// @Success 200 {object} status_http.Response{data=fc.Function} "Data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetMicroFrontEndByID(c *gin.Context) {
	functionID := c.Param("micro-frontend-id")
	//var resourceEnvironment *obs.ResourceEnvironment

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	h.handleResponse(c, status_http.OK, function)
}

// GetAllMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID get_all_micro_frontend
// @Router /v2/functions/micro-frontend [GET]
// @Summary Get All Micro Frontend
// @Description Get All Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "Data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllMicroFrontEnd(c *gin.Context) {

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	resp, err := services.FunctionService().FunctionService().GetList(
		context.Background(),
		&fc.GetAllFunctionsRequest{
			Search:    c.DefaultQuery("search", ""),
			Limit:     int32(limit),
			Offset:    int32(offset),
			ProjectId: resource.ResourceEnvironmentId,
			Type:      MICROFE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID update_micro_frontend
// @Router /v2/functions/micro-frontend [PUT]
// @Summary Update Micro Frontend
// @Description Update Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param Data body models.Function  true "Data"
// @Success 200 {object} status_http.Response{data=models.Function} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateMicroFrontEnd(c *gin.Context) {
	var function models.Function

	//var resourceEnvironment *obs.ResourceEnvironment
	err := c.ShouldBindJSON(&function)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	resp, err := services.FunctionService().FunctionService().Update(
		context.Background(),
		&fc.Function{
			Id:               function.ID,
			Description:      function.Description,
			Name:             function.Name,
			Path:             function.Path,
			ProjectId:        resource.ResourceEnvironmentId,
			FunctionFolderId: function.FuncitonFolderId,
			Type:             MICROFE,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID delete_micro_frontend
// @Router /v2/functions/micro-frontend/{micro-frontend-id} [DELETE]
// @Summary Delete Micro Frontend
// @Description Delete Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param micro-frontend-id path string true "micro-frontend-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteMicroFrontEnd(c *gin.Context) {
	functionID := c.Param("micro-frontend-id")
	//var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "micro frontend id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	err = code_server.DeleteCodeServerByPath(resp.Path, h.cfg)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	// delete cloned repo
	err = gitlab.DeletedClonedRepoByPath(resp.Path, h.cfg)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	// delete repo by path from gitlab
	_, err = gitlab.DeleteForkedProject(resp.Path, h.cfg)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

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
