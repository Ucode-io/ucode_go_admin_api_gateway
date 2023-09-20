package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// BindLoginMicroFrontToProject godoc
// @Security ApiKeyAuth
// @ID bind_login_micro_front_to_project
// @Router /v1/login-microfront [POST]
// @Summary Bind login microfrotn to project
// @Description Bind login microfrotn to project
// @Tags Project login microfront
// @Accept json
// @Produce json
// @Param data body pb.ProjectLoginMicroFrontend true "ProjectLoginMicroFrontend"
// @Success 201 {object} status_http.Response{data=pb.ProjectLoginMicroFrontend} "ProjectLoginMicroFrontend"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) BindLoginMicroFrontToProject(c *gin.Context) {
	var (
		data pb.ProjectLoginMicroFrontend
		//resourceEnvironment *obs.ResourceEnvironment
	)

	err := c.ShouldBindJSON(&data)
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

	data.ProjectId = projectId.(string)
	data.EnvironmentId = environmentId.(string)

	res, err := services.CompanyService().Project().CreateProjectLoginMicroFront(
		context.Background(),
		&data,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// UpdateLoginMicroFront godoc
// @Security ApiKeyAuth
// @ID update_login_microfront
// @Router /v1/login-microfront [PUT]
// @Summary Update Login MicroFront Project
// @Description Update Login MicroFront Project
// @Tags Project login microfront
// @Accept json
// @Produce json
// @Param Company body company_service.ProjectLoginMicroFrontend  true "ProjectLoginMicroFrontend"
// @Success 200 {object} status_http.Response{data=company_service.ProjectLoginMicroFrontend} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateLoginMicroFrontProject(c *gin.Context) {
	var req company_service.ProjectLoginMicroFrontend

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Project().UpdateProjectLoginMicroFront(
		context.Background(),
		&req,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetLoginMicroFrontBySubdomain godoc
// @Security ApiKeyAuth
// @ID get_login_microfront_by_subdomain
// @Router /v1/login-microfront/{subdomain} [GET]
// @Summary Get Project By Id
// @Description Get Project By Id
// @Tags Project login microfront
// @Accept json
// @Produce json
// @Param subdomain path string true "subdomain"
// @Success 200 {object} status_http.Response{data=company_service.Project} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetLoginMicroFrontBySubdomain(c *gin.Context) {
	subdomain := c.Param("subdomain")

	resp, err := h.companyServices.CompanyService().Project().GetProjectLoginMicroFront(
		context.Background(),
		&company_service.GetProjectLoginMicroFrontRequest{
			Subdomain: subdomain,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
