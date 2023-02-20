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

// Scenario godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_scenario
// @Router /v1/scenario [POST]
// @Summary Create scenario
// @Description Create scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param body body models.CreateScenarioRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=models.DAG} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateFullScenario(c *gin.Context) {
	h.log.Info("CreateFullScenario", logger.String("body", ""))
	var (
		req models.CreateScenarioRequest
	)
	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.log.Error("ShouldBindJSON", logger.Error(err))
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

	dagSteps := make([]*pb.DAGStep, 0)
	for _, step := range req.Steps {

		if step.Config.RequestInfo == nil {
			step.Config.RequestInfo = make(map[string]interface{})
		}

		requestInfo, err := helper.ConvertMapToStruct(step.Config.RequestInfo)
		if err != nil {
			h.log.Error("ConvertMapToStruct requestInfo", logger.Error(err))
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		uiComponent, err := helper.ConvertMapToStruct(step.UiComponent)
		if err != nil {
			h.log.Error("ConvertMapToStruct uiComponent", logger.Error(err))
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		// conditionAction, err := helper.ConvertMapToStruct(step.ConditionAction)
		// if err != nil {
		// 	h.handleResponse(c, status_http.BadRequest, err.Error())
		// 	return
		// }

		dagSteps = append(dagSteps, &pb.DAGStep{
			Slug:        step.Config.Slug,
			Type:        step.Config.Type,
			ConnectInfo: step.Config.ConnectInfo,
			RequestInfo: requestInfo,
			IsParallel:  step.Config.IsParallel,
			UiComponent: uiComponent,
			Title:       step.Config.Title,
			Description: step.Config.Description,
		})
	}

	dag := &pb.DAG{
		Id:         req.Dag.Id,
		Title:      req.Dag.Title,
		Slug:       req.Dag.Slug,
		Type:       req.Dag.Type,
		Status:     req.Dag.Status,
		CategoryId: req.Dag.CategoryId,
	}

	serviceReq := pb.CreateScenarioRequest{
		ProjectId:     ProjectId,
		EnvironmentId: EnvironmentId.(string),
		Dag:           dag,
		Steps:         dagSteps,
	}

	resp, err := services.ScenarioService().DagService().CreateScenario(
		c.Request.Context(),
		&serviceReq,
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
