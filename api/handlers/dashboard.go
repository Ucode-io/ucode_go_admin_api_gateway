package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateDashboard godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_dashboard
// @Router /v1/analytics/dashboard [POST]
// @Summary Create dashboard
// @Description Create dashboard
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param dashboard body models.CreateDashboardRequest true "CreateDashboardRequestBody"
// @Success 201 {object} http.Response{data=models.Dashboard} "Dashboard data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateDashboard(c *gin.Context) {
	var dashboardRequest models.CreateDashboardRequest

	err := c.ShouldBindJSON(&dashboardRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	var dashboard = obs.CreateDashboardRequest{
		// Id:         dashboardRequest.ID,
		Name:      dashboardRequest.Name,
		Icon:      dashboardRequest.Icon,
		ProjectId: resourceEnvironment.GetId(),
	}

	resp, err := services.DashboardService().Create(
		context.Background(),
		&dashboard,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetSingleDashboard godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_dashboard_by_id
// @Router /v1/analytics/dashboard/{dashboard_id} [GET]
// @Summary Get single dashboard
// @Description Get single dashboard
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param dashboard_id path string true "dashboard_id"
// @Success 200 {object} http.Response{data=models.Dashboard} "DashboardBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleDashboard(c *gin.Context) {
	dashboardID := c.Param("dashboard_id")

	if !util.IsValidUUID(dashboardID) {
		h.handleResponse(c, http.InvalidArgument, "dashboard id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	resp, err := services.DashboardService().GetSingle(
		context.Background(),
		&obs.DashboardPrimaryKey{
			Id:        dashboardID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateDashboard godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_dashboard
// @Router /v1/analytics/dashboard [PUT]
// @Summary Update dashboard
// @Description Update dashboard
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param dashboard body models.Dashboard true "UpdateDashboardRequestBody"
// @Success 200 {object} http.Response{data=models.Dashboard} "Dashboard data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateDashboard(c *gin.Context) {
	var dashboard obs.Dashboard

	err := c.ShouldBindJSON(&dashboard)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}
	dashboard.ProjectId = resourceEnvironment.GetId()

	resp, err := services.DashboardService().Update(
		context.Background(),
		&dashboard,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteDashboard godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_dashboard
// @Router /v1/analytics/dashboard/{dashboard_id} [DELETE]
// @Summary Delete dashboard
// @Description Delete dashboard
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param dashboard_id path string true "dashboard_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteDashboard(c *gin.Context) {
	dashboardID := c.Param("dashboard_id")

	if !util.IsValidUUID(dashboardID) {
		h.handleResponse(c, http.InvalidArgument, "dashboard id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	resp, err := services.DashboardService().Delete(
		context.Background(),
		&obs.DashboardPrimaryKey{
			Id:        dashboardID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetAllDashboards godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_dashboard_list
// @Router /v1/analytics/dashboard [GET]
// @Summary Get dashboard list
// @Description Get dashboard list
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param filters query models.GetAllDashboardsRequest true "filters"
// @Success 200 {object} http.Response{data=models.GetAllDashboardsResponse} "DashboardBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllDashboards(c *gin.Context) {
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	resp, err := services.DashboardService().GetList(
		context.Background(),
		&obs.GetAllDashboardsRequest{
			Name:      c.Query("name"),
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
