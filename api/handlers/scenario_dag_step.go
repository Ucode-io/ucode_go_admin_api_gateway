package handlers

import (
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// ScenarioDAG godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_scenario_dag_step
// @Router /v1/scenario/dag-step [POST]
// @Summary Create scenario dag step
// @Description Create scenario dag step
// @Tags Scenario
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param body body pb.CreateDAGStepRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAGStep} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateDagStep(c *gin.Context) {
	var (
		req models.DAGStep
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

	requestInfoStrct, err := helper.ConvertMapToStruct(req.RequestInfo)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	reqStrct := pb.CreateDAGStepRequest{
		Slug:        req.Slug,
		ParentId:    req.ParentId,
		DagId:       req.DagId,
		Type:        req.Type,
		ConnectInfo: &req.ConnectInfo,
		RequestInfo: requestInfoStrct,
		IsParallel:  true,
	}

	resp, err := services.ScenarioService().DagStepService().Create(
		c.Request.Context(),
		&reqStrct,
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
// @ID get_all_scenario_dag_step
// @Router /v1/scenario/dag-step [GET]
// @Summary Get All scenario dag step
// @Description Get All scenario dag step
// @Tags Scenario
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param body body pb.GetAllDAGStepRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAGStepList} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllDagStep(c *gin.Context) {
	var (
		req pb.GetAllDAGStepRequest
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

	resp, err := services.ScenarioService().DagStepService().GetAll(
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
// @ID get_scenario_dag_step
// @Router /v1/scenario/dag-step/{id} [GET]
// @Summary Get scenario dag step
// @Description Get scenario dag step
// @Tags Scenario
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param id path string true "id"
// @Param body body pb.GetDAGStepRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAGStep} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetDagStep(c *gin.Context) {
	var (
		req pb.GetDAGStepRequest
	)

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	dagStepID := c.Param("id")
	if !util.IsValidUUID(dagStepID) {
		h.handleResponse(c, status_http.BadRequest, "id not valid uuid")
		return
	}
	req.DagId = dagStepID

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

	resp, err := services.ScenarioService().DagStepService().Get(
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
// @ID update_scenario_dag_step
// @Router /v1/scenario/dag-step [PUT]
// @Summary Update scenario dag step
// @Description Update scenario dag step
// @Tags Scenario
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param body body pb.DAGStep  true "Request body"
// @Success 200 {object} status_http.Response{data=pb.DAGStep} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateDagStep(c *gin.Context) {
	var (
		dagStep models.DAGStep
	)

	err := c.ShouldBindJSON(&dagStep)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if !util.IsValidUUID(dagStep.Id) {
		h.handleResponse(c, status_http.BadRequest, "dagStepID not valid uuid")
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

	requestInfoStrct, err := helper.ConvertMapToStruct(dagStep.RequestInfo)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	// conditionActionStrct, err := helper.ConvertMapToStruct(dagStep.ConditionAction)

	dagStepStrct := pb.DAGStep{
		Id:          dagStep.Id,
		Slug:        dagStep.Slug,
		ParentId:    dagStep.ParentId,
		DagId:       dagStep.DagId,
		Type:        dagStep.Type,
		ConnectInfo: &dagStep.ConnectInfo,
		RequestInfo: requestInfoStrct,
		// ConditionAction: conditionActionStrct,
	}

	resp, err := services.ScenarioService().DagStepService().Update(
		c.Request.Context(),
		&pb.UpdateDAGStepRequest{
			DagStep: &dagStepStrct,
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
// @ID delete_scenario_dag_step
// @Router /v1/scenario/dag-step/{id} [DELETE]
// @Summary Delete scenario dag step
// @Description Delete scenario dag step
// @Tags Scenario
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=string} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteDagStep(c *gin.Context) {

	dagStepID := c.Param("id")
	if !util.IsValidUUID(dagStepID) {
		h.handleResponse(c, status_http.BadRequest, "dagStepID not found")
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

	_, err = services.ScenarioService().DagStepService().Delete(
		c.Request.Context(),
		&pb.DeleteDAGStepRequest{
			Id: dagStepID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, "success")
}
