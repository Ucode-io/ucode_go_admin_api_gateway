package handlers

import (
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
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
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId := c.Query("project-id")
	if !util.IsValidUUID(ProjectId) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	requestInfoStrct, err := helper.ConvertMapToStruct(req.Config.RequestInfo)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	reqStrct := pb.CreateDAGStepRequest{
		Slug:        req.Config.Slug,
		ParentId:    req.Config.ParentId,
		DagId:       req.Config.DagId,
		Type:        req.Config.Type,
		ConnectInfo: req.Config.ConnectInfo,
		RequestInfo: requestInfoStrct,
		IsParallel:  true,
		Title:       req.Config.Title,
		Description: req.Config.Description,
	}

	resp, err := services.ScenarioService().DagStepService().Create(
		c.Request.Context(),
		&reqStrct,
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
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
// @Param dag-id query string true "dag-id"
// @Param body body pb.GetAllDAGStepRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=models.GetAllDAGStepResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllDagStep(c *gin.Context) {
	req := pb.GetAllDAGStepRequest{
		Filters: &pb.Filters{},
	}
	
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error when get service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId := c.Query("project-id")
	if !util.IsValidUUID(ProjectId) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	DagId := c.Query("dag-id")
	if !util.IsValidUUID(DagId) {
		h.handleResponse(c, status_http.BadRequest, "dag-id not found")
		return
	}

	resp, err := services.ScenarioService().DagStepService().GetAll(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	respStrct := []*models.DAGStep{}

	for _, v := range resp.DagSteps {

		reqInfo, err := helper.ConvertStructToResponse(v.RequestInfo)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		dagStepConfig := models.DAGStepConfig{
			Id:          v.Id,
			Slug:        v.Slug,
			ParentId:    v.ParentId,
			DagId:       v.DagId,
			Type:        v.Type,
			ConnectInfo: v.ConnectInfo,
			RequestInfo: reqInfo,
			IsParallel:  v.IsParallel,
			Title:       v.Title,
			Description: v.Description,
		}
		uiComponent, err := helper.ConvertStructToResponse(v.UiComponent)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		respStrct = append(respStrct, &models.DAGStep{
			Config:      dagStepConfig,
			UiComponent: uiComponent,
		})
	}

	h.handleResponse(c, status_http.OK, models.GetAllDAGStepResponse{
		Steps: respStrct,
	})
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
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId := c.Query("project-id")
	if !util.IsValidUUID(ProjectId) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	resp, err := services.ScenarioService().DagStepService().Get(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
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

	if !util.IsValidUUID(dagStep.Config.Id) {
		h.handleResponse(c, status_http.BadRequest, "dagStepID not valid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId := c.Query("project-id")
	if !util.IsValidUUID(ProjectId) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	requestInfoStrct, err := helper.ConvertMapToStruct(dagStep.Config.RequestInfo)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	// conditionActionStrct, err := helper.ConvertMapToStruct(dagStep.ConditionAction)

	dagStepStrct := pb.DAGStep{
		Id:          dagStep.Config.Id,
		Slug:        dagStep.Config.Slug,
		ParentId:    dagStep.Config.ParentId,
		DagId:       dagStep.Config.DagId,
		Type:        dagStep.Config.Type,
		ConnectInfo: dagStep.Config.ConnectInfo,
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
		h.handleResponse(c, status_http.InternalServerError, err.Error())
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
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId := c.Query("project-id")
	if !util.IsValidUUID(ProjectId) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	_, err = services.ScenarioService().DagStepService().Delete(
		c.Request.Context(),
		&pb.DeleteDAGStepRequest{
			Id: dagStepID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, "success")
}
