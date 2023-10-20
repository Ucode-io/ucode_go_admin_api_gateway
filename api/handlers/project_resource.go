package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// AddResourceToProject godoc
// @Security ApiKeyAuth
// @ID add_resource_to_project
// @Router /v2/company/project/resource [POST]
// @Summary Add rosource to project
// @Description Add rosource to project
// @Tags Project resource
// @Accept json
// @Produce json
// @Param data body pb.AddResourceToProjectRequest true "AddResourceToProjectRequest"
// @Success 200 {object} status_http.Response{data=company_service.ProjectResource} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) AddResourceToProject(c *gin.Context) {

	var request = &pb.AddResourceToProjectRequest{}

	err := c.ShouldBindJSON(&request)
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
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)

	resp, err := h.companyServices.CompanyService().Resource().AddResourceToProject(
		context.Background(),
		request,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateProjectResource godoc
// @Security ApiKeyAuth
// @ID update_project_resource
// @Router /v2/company/project/resource [PUT]
// @Summary Update Project resource
// @Description update Project resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param Company body company_service.ProjectResource  true "ProjectResource"
// @Success 200 {object} status_http.Response{data=company_service.Empty} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateProjectResource(c *gin.Context) {
	var request = &pb.ProjectResource{}

	err := c.ShouldBindJSON(&request)
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
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)

	resp, err := h.companyServices.CompanyService().Resource().UpdateProjectResource(
		context.Background(),
		request,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListProjectResource godoc
// @Security ApiKeyAuth
// @ID get_list_project_resource
// @Router /v2/company/project/resource [GET]
// @Summary Get list project resource
// @Description Get list project resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param type query string false "type"
// @Success 200 {object} status_http.Response{data=company_service.ListProjectResource} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListProjectResourceList(c *gin.Context) {

	request := &pb.GetProjectResourceListRequest{}

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

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)

	if c.DefaultQuery("type", "") != "" && pb.ResourceType(pb.ResourceType_value[c.DefaultQuery("type", "")]) != 0 {
		request.Type = pb.ResourceType(pb.ResourceType_value[c.DefaultQuery("type", "")])
	}

	resp, err := h.companyServices.CompanyService().Resource().GetProjectResourceList(
		context.Background(),
		request,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetProjectResourceByID godoc
// @Security ApiKeyAuth
// @ID get_single_project_resource
// @Router /v2/company/project/resource/{id} [GET]
// @Summary Get single variable resource
// @Description Get single variable resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=pb.ProjectResource} "ProjectResource"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleProjectResource(c *gin.Context) {

	request := &pb.PrimaryKeyProjectResource{}

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

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)
	request.Id = c.Param("id")

	resp, err := h.companyServices.CompanyService().Resource().GetSingleProjectResouece(
		context.Background(),
		request,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeletevariableResource godoc
// @Security ApiKeyAuth
// @ID delete_variable_resource
// @Router /v2/company/project/resource/{id} [DELETE]
// @Summary Delete variable resource
// @Description Delete variable resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=pb.Empty} "VariableResource"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteProjectResource(c *gin.Context) {

	request := &pb.PrimaryKeyProjectResource{}

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

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)
	request.Id = c.Param("id")

	resp, err := h.companyServices.CompanyService().Resource().DeleteProjectResource(
		c.Request.Context(),
		request,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
