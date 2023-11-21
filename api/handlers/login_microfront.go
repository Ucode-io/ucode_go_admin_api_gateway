package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/new_function_service"
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

	// namespace := c.GetString("namespace")
	// services, err := h.GetService(namespace)
	// if err != nil {
	// 	h.handleResponse(c, status_http.Forbidden, err)
	// 	return
	// }

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

	res, err := h.companyServices.Project().CreateProjectLoginMicroFront(
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

	resp, err := h.companyServices.Project().UpdateProjectLoginMicroFront(
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
// @Router /v1/login-microfront [GET]
// @Summary Get Project By Id
// @Description Get Project By Id
// @Tags Project login microfront
// @Accept json
// @Produce json
// @Param subdomain query string false "subdomain"
// @Param project-id query string false "project-id"
// @Success 200 {object} status_http.Response{data=models.MicrofrontForLoginPage} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetLoginMicroFrontBySubdomain(c *gin.Context) {
	subdomain := c.DefaultQuery("subdomain", "")
	projectId := c.DefaultQuery("project-id", "")

	if subdomain == "" && projectId == "" {
		h.handleResponse(c, status_http.InvalidArgument, "subdomain or project-id is required")
		return
	}
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := h.companyServices.Project().GetProjectLoginMicroFront(
		context.Background(),
		&company_service.GetProjectLoginMicroFrontRequest{
			Subdomain: subdomain,
			ProjectId: projectId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if resp.ProjectId == "" || resp.EnvironmentId == "" {
		h.handleResponse(c, status_http.OK, models.MicrofrontForLoginPage{})
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     resp.ProjectId,
			EnvironmentId: resp.EnvironmentId,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	function, err := services.FunctionService().FunctionService().GetSingle(context.Background(), &new_function_service.FunctionPrimaryKey{
		Id:        resp.MicrofrontId,
		ProjectId: resource.ResourceEnvironmentId,
	})

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, models.MicrofrontForLoginPage{
		Function:      function,
		Id:            resp.GetId(),
		MicrofrontId:  resp.GetMicrofrontId(),
		ProjectId:     resp.GetProjectId(),
		Subdomain:     resp.GetSubdomain(),
		EnvironmentId: resp.GetEnvironmentId(),
	})
}
