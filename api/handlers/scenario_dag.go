package handlers

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"

	"github.com/gin-gonic/gin"
	"github.com/saidamir98/udevs_pkg/util"
)

// ScenarioDAG godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_scenario_dag
// @Router /v1/scenario/dag [POST]
// @Summary Create scenario dag
// @Description Create scenario dag
// @Tags Section
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param body body pb.CreateDAGRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAG} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) Create(c *gin.Context) {
	var (
		req pb.CreateDAGRequest
	)
	err := c.ShouldBindJSON(&req)
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

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId, _ := c.Get("project_id")
	if !util.IsValidUUID(ProjectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project_id not found")
		return
	}

	resp, err := services.ScenarioService().DagService().Create(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// ScenarioDAG godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_all_scenario_dag
// @Router /v1/scenario/dag [GET]
// @Summary Get all scenario dag
// @Description Get all scenario dag
// @Tags Section
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param limit query int false "limit"
// @Param offset query int false "offset"
// @Param order query string false "order"
// @Param page query int false "page"
// @Success 200 {object} status_http.Response{data=pb.DAGList} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAll(c *gin.Context) {

	filter := &pb.Filters{
		Order:  "created_at",
		Offset: 0,
		Limit:  10,
		Page:   1,
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	filter.Limit = int64(limit)

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	filter.Offset = int64(offset)

	page, err := h.getPageParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	filter.Page = int64(page)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId, _ := c.Get("project_id")
	if !util.IsValidUUID(ProjectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project_id not found")
		return
	}

	resp, err := services.ScenarioService().DagService().GetAll(
		c.Request.Context(),
		&pb.GetAllDAGRequest{
			Filter: filter,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// ScenarioDAG godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_scenario_dag
// @Router /v1/scenario/dag/{id} [GET]
// @Summary Get all scenario dag
// @Description Get all scenario dag
// @Tags Section
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param project-id query string true "project-id"
// @Param table body pb.GetDAGRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAG} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) Get(c *gin.Context) {
	var (
		req pb.GetDAGRequest
	)
	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	id := c.Param("id")
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "id not valid uuid")
		return
	}
	req.Id = id

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId, _ := c.Get("project_id")
	if !util.IsValidUUID(ProjectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project_id not found")
		return
	}

	resp, err := services.ScenarioService().DagService().Get(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// ScenarioDAG godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_scenario_dag
// @Router /v1/scenario/dag [PUT]
// @Summary Update scenario dag
// @Description Update scenario dag
// @Tags Section
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table body pb.DAG  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAG} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) Update(c *gin.Context) {
	var (
		dag pb.DAG
	)
	err := c.ShouldBindJSON(&dag)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if !util.IsValidUUID(dag.GetId()) {
		h.handleResponse(c, status_http.BadRequest, "dag id not valid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId, _ := c.Get("project_id")
	if !util.IsValidUUID(ProjectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project_id not found")
		return
	}

	resp, err := services.ScenarioService().DagService().Update(
		c.Request.Context(),
		&pb.UpdateDAGRequest{
			Dag: &dag,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// ScenarioDAG godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_scenario_dag
// @Router /v1/scenario/dag/{id} [DELETE]
// @Summary Delete scenario dag
// @Description Delete scenario dag
// @Tags Section
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param project-id query string true "project-id"
// @Success 200 {object} status_http.Response{data=string} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) Delete(c *gin.Context) {

	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "id not valid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId, _ := c.Get("project_id")
	if !util.IsValidUUID(ProjectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project_id not found")
		return
	}

	_, err = services.ScenarioService().DagService().Delete(
		c.Request.Context(),
		&pb.DeleteDAGRequest{
			Id: id,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, "success")
}
