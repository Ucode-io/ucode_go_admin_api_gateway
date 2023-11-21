package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// GetGlobalCompanyProjectList godoc
// @Security ApiKeyAuth
// @ID get_global_project_list
// @Router /v1/global/projects [GET]
// @Summary Get all global projects
// @Description Get all global projects
// @Tags Global Project
// @Accept json
// @Produce json
// @Param filters query company_service.GetProjectListRequest true "filters"
// @Success 200 {object} status_http.Response{data=company_service.GetProjectListResponse} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetGlobalCompanyProjectList(c *gin.Context) {

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

	resp, err := h.companyServices.Project().GetList(
		context.Background(),
		&company_service.GetProjectListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			CompanyId: c.DefaultQuery("company_id", ""),
		},
	)

	h.handleResponse(c, status_http.OK, resp)
}

// GetGlobalProjectEnvironments godoc
// @Security ApiKeyAuth
// @ID get_global_project_environment_list
// @Router /v1/global/environment [GET]
// @Summary Get global project environment list
// @Description Get global project environment list
// @Tags Global Project
// @Accept json
// @Produce json
// @Param filters query pb.GetEnvironmentListRequest true "filters"
// @Success 200 {object} status_http.Response{data=pb.GetEnvironmentListResponse} "EnvironmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetGlobalProjectEnvironments(c *gin.Context) {

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.Environment().GetList(
		c.Request.Context(),
		&pb.GetEnvironmentListRequest{
			Offset:    int32(offset),
			Limit:     int32(limit),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: c.DefaultQuery("project_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetGlobalProjectTemplate godoc
// @Security ApiKeyAuth
// @ID get_global_project_template
// @Router /v1/global/template [GET]
// @Summary Get global project template
// @Description Get global project template
// @Tags Global Project
// @Accept json
// @Produce json
// @Param environment-id query string true "environment-id"
// @Param project-id query string true "project-id"
// @Param filters query obs.GetAllMenusRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllMenusResponse} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetGlobalProjectTemplate(c *gin.Context) {

	var (
		resp *obs.GetAllMenusResponse
	)

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

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	authInfo, _ := h.GetAuthInfo(c)
	limit := 1000
	offset := 0

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Menu().GetAll(
			context.Background(),
			&obs.GetAllMenusRequest{
				Limit:       int32(limit),
				Offset:      int32(offset),
				Search:      c.DefaultQuery("search", ""),
				ProjectId:   resource.ResourceEnvironmentId,
				ParentId:    c.DefaultQuery("parent_id", ""),
				RoleId:      authInfo.GetRoleId(),
				ForTemplate: true,
			},
		)
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetAll(
			context.Background(),
			&obs.GetAllMenusRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				Search:    c.DefaultQuery("search", ""),
				ProjectId: resource.ResourceEnvironmentId,
				ParentId:  c.DefaultQuery("parent_id", ""),
				RoleId:    authInfo.GetRoleId(),
			},
		)
	}
	h.handleResponse(c, status_http.OK, resp)
}
