package handlers

import (
	"context"
	"fmt"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// GetCompanyProjectById godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_company_project_id
// @Router /v1/company-project/{project_id} [GET]
// @Summary Get Project By Id
// @Description Get Project By Id
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Param company_id query string false "company_id"
// @Success 200 {object} status_http.Response{data=company_service.Project} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyProjectById(c *gin.Context) {
	projectId := c.Param("project_id")

	resp, err := h.companyServices.CompanyService().Project().GetById(
		context.Background(),
		&company_service.GetProjectByIdRequest{
			ProjectId: projectId,
			CompanyId: c.DefaultQuery("company_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetCompanyProjectList godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_project_list
// @Router /v1/company-project [GET]
// @Summary Get all projects
// @Description Get all projects
// @Tags Company Project
// @Accept json
// @Produce json
// @Param filters query company_service.GetProjectListRequest true "filters"
// @Success 200 {object} status_http.Response{data=company_service.GetProjectListResponse} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyProjectList(c *gin.Context) {

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

	resp, err := h.companyServices.CompanyService().Project().GetList(
		context.Background(),
		&company_service.GetProjectListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			CompanyId: c.DefaultQuery("company_id", ""),
		},
	)
	fmt.Println("projects::", resp.GetProjects())

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateCompanyProject godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_project
// @Router /v1/company-project/{project_id} [PUT]
// @Summary Update Project
// @Description Update Project
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Param Company body company_service.Project  true "CompanyProjectCreateRequest"
// @Success 200 {object} status_http.Response{data=company_service.Project} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateCompanyProject(c *gin.Context) {
	var project company_service.Project

	projectId := c.Param("project_id")
	project.ProjectId = projectId

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Project().Update(
		context.Background(),
		&project,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteCompanyProject godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_project
// @Router /v1/company-project/{project_id} [DELETE]
// @Summary Delete Project
// @Description Delete Project
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Success 204 {object} status_http.Response{data=string} "Data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteCompanyProject(c *gin.Context) {
	projectId := c.Param("project_id")

	resp, err := h.companyServices.CompanyService().Project().Delete(
		context.Background(),
		&company_service.DeleteProjectRequest{
			ProjectId: projectId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
