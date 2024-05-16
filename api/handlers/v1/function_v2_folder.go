package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// CreateFunctionFolder godoc
// @Security ApiKeyAuth
// @ID create_function_folder
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
func (h *HandlerV1) CreateFunctionFolder(c *gin.Context) {
	var functionFolder fc.CreateFunctionFolderRequest
	//var resourceEnvironment *obs.ResourceEnvironment
	err := c.ShouldBindJSON(&functionFolder)
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

	if functionFolder.Type == "" {
		functionFolder.Type = "FUNCTION"
	}

	resp, err := services.FunctionService().FunctionFolderService().Create(
		context.Background(),
		&fc.CreateFunctionFolderRequest{
			Title:       functionFolder.Title,
			Description: functionFolder.Description,
			ProjectId:   resource.ResourceEnvironmentId,
			Type:        functionFolder.Type,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetFunctionFolderById godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) GetFunctionFolderById(c *gin.Context) {
	functionFolderID := c.Param("function_folder_id")
	//var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionFolderID) {
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
			Id:        functionFolderID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllFunctionFolder godoc
// @Security ApiKeyAuth
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
// @Param type query string true "type"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllFunctionFolder(c *gin.Context) {

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

	typeFolder := c.DefaultQuery("type", "FUNCTION")

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
		resp, err := services.FunctionService().FunctionFolderService().GetList(
			context.Background(),
			&fc.GetAllFunctionFoldersRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				Offset:    int32(offset),
				Type:      typeFolder,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		h.handleResponse(c, status_http.OK, fc.GetAllFunctionFoldersResponse{})
	}
}

// UpdateFunctionFolder godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) UpdateFunctionFolder(c *gin.Context) {
	var functionFolder models.FunctionFolder
	//var resourceEnvironment *obs.ResourceEnvironment

	err := c.ShouldBindJSON(&functionFolder)
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

	if functionFolder.Type == "" {
		functionFolder.Type = "FUNCTION"
	}

	_, err = services.FunctionService().FunctionFolderService().Update(
		context.Background(),
		&fc.FunctionFolder{
			Id:          functionFolder.Id,
			Description: functionFolder.Description,
			Title:       functionFolder.Title,
			Type:        functionFolder.Type,
			ProjectId:   resource.ResourceEnvironmentId,
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
func (h *HandlerV1) DeleteFunctionFolder(c *gin.Context) {
	functionFolderID := c.Param("function_folder_id")
	//var resourceEnvironment *obs.ResourceEnvironment

	if !util.IsValidUUID(functionFolderID) {
		h.handleResponse(c, status_http.InvalidArgument, "function folder id is an invalid uuid")
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

	resp, err := services.FunctionService().FunctionFolderService().Delete(
		context.Background(),
		&fc.FunctionFolderPrimaryKey{
			Id:        functionFolderID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
