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
// @ID run_scenario
// @Router /v1/scenario/run [POST]
// @Summary Run scenario
// @Description Run scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param body body models.RunScenarioRequest true "Request body"
// @Success 200 {object} status_http.Response{data=pb.RunScenarioResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RunScenario(c *gin.Context) {

	var (
		req models.RunScenarioRequest
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	if req.Body == nil {
		req.Body = make(map[string]interface{})
	}

	bodyStrct, err := helper.ConvertMapToStruct(req.Body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	runScenarioStrct := &pb.RunScenarioRequest{
		DagId:     req.DagId,
		Header:    req.Header,
		Body:      bodyStrct,
		Url:       req.Url,
		DagStepId: req.DagStepId,
		Method:    req.Method,
	}

	resp, err := services.ScenarioService().RunService().RunScenario(c.Request.Context(), runScenarioStrct)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
