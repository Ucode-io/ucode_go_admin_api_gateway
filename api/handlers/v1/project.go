package v1

import (
	"context"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// GetCompanyProjectById godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) GetCompanyProjectById(c *gin.Context) {
	projectId, ok := c.Get("project_id")
	if !ok && !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project id is required")
		return
	}

	resp, err := h.companyServices.Project().GetById(
		context.Background(),
		&company_service.GetProjectByIdRequest{
			ProjectId: projectId.(string),
			CompanyId: c.Query("company_id"),
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
func (h *HandlerV1) GetCompanyProjectList(c *gin.Context) {
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

	authInfo, _ := h.GetAuthAdminInfo(c)
	userProjects, err := h.authService.User().GetUserProjects(context.Background(), &auth_service.UserPrimaryKey{
		Id: authInfo.GetUserId(),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var projectIds []string
	for _, userProject := range userProjects.GetCompanies() {
		if userProject.GetId() == c.DefaultQuery("company_id", "") {
			projectIds = userProject.GetProjectIds()
		}
	}

	resp, err := h.companyServices.Project().GetList(
		context.Background(),
		&company_service.GetProjectListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			CompanyId: c.DefaultQuery("company_id", ""),
		},
	)

	projectsMap := make(map[string]*company_service.Project)
	for _, project := range resp.GetProjects() {
		projectsMap[project.GetProjectId()] = project
	}

	var availableProjects = make([]*company_service.Project, 0, len(projectIds))

	for _, id := range projectIds {
		if val, ok := projectsMap[id]; ok {
			availableProjects = append(availableProjects, val)
		}
	}

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	resp.Projects = availableProjects

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateCompanyProject godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) UpdateCompanyProject(c *gin.Context) {
	var project company_service.Project

	projectId := c.Param("project_id")
	project.ProjectId = projectId

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.Project().Update(
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
func (h *HandlerV1) DeleteCompanyProject(c *gin.Context) {
	projectId := c.Param("project_id")

	resp, err := h.companyServices.Project().Delete(
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

// GetCompanyProjectList godoc
// @Security ApiKeyAuth
// @ID get_project_list
// @Router /v1/companies/{company_id}/projects [GET]
// @Summary Get all projects
// @Description Get all projects
// @Tags Company Project
// @Accept json
// @Produce json
// @Param company_id path string true "company_id"
// @Param filters query company_service.GetProjectListRequest true "filters"
// @Success 200 {object} status_http.Response{data=company_service.GetProjectListResponse} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) ListCompanyProjects(c *gin.Context) {
	companyId := c.Param("company_id")
	if companyId == "" {
		h.handleResponse(c, status_http.BadRequest, "company_id is required")
		return
	}

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

	resp, err := h.companyServices.Project().GetList(c.Request.Context(),
		&company_service.GetProjectListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.Query("search"),
			CompanyId: companyId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
