package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateCategoryScenario godoc
// @Security ApiKeyAuth
// @ID create_category_for_scenario
// @Router /v1/scenario/category [POST]
// @Summary Create category
// @Description Create category
// @Tags Scenario-category
// @Accept json
// @Produce json
// @Param app body models.CreateCategory true "Request body"
// @Success 201 {object} status_http.Response{data=models.Category} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateCategoryScenario(c *gin.Context) {
	var (
		req models.CreateCategory
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&company_service.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			ServiceType:   company_service.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ScenarioService().CategoryService().Create(
		context.Background(),
		&pb.CreateCategoryRequest{
			Name:          req.Name,
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetCategoryScenario godoc
// @Security ApiKeyAuth
// @ID get_single_category_for_scenario
// @Router /v1/scenario/category/{id} [GET]
// @Summary Get single category
// @Description Get single category
// @Tags Scenario-category
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=models.Category} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCategoryScenario(c *gin.Context) {

	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "id is invalid uuid")
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&company_service.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			ServiceType:   company_service.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ScenarioService().CategoryService().Get(
		context.Background(),
		&pb.GetCategoryRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			Guid:          id,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListCategoryScenario godoc
// @Security ApiKeyAuth
// @ID get_list_category_for_scenario
// @Router /v1/scenario/category [GET]
// @Summary Get list category
// @Description Get list category
// @Tags Scenario-category
// @Accept json
// @Produce json
// @Param offset query string false "offset"
// @Param limit query string false "limit"
// @Param page query string false "page"
// @Success 200 {object} status_http.Response{data=models.GetAllCategoriesResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListCategoryScenario(c *gin.Context) {

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	page, err := h.getPageParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&company_service.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			ServiceType:   company_service.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ScenarioService().CategoryService().GetList(
		c.Request.Context(),
		&pb.GetListCategoryRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			Filter: &pb.Filters{
				Offset: int64(offset),
				Limit:  int64(limit),
				Page:   int64(page),
			},
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
