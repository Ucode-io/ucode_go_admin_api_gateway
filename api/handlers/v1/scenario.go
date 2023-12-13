package v1

import (
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Scenario godoc
// @Security ApiKeyAuth
// @ID create_scenario
// @Router /v1/scenario [POST]
// @Summary Create scenario
// @Description Create scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param body body models.CreateScenarioRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=models.DAG} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateFullScenario(c *gin.Context) {
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}
	if !util.IsValidUUID(projectId.(string)) {
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
			Slug:         step.Config.Slug,
			Type:         step.Config.Type,
			ConnectInfo:  step.Config.ConnectInfo,
			RequestInfo:  requestInfo,
			IsParallel:   step.Config.IsParallel,
			UiComponent:  uiComponent,
			Title:        step.Config.Title,
			Description:  step.Config.Description,
			CallbackType: step.Config.CallbackType,
		})
	}

	if req.Dag.Attributes == nil {
		req.Dag.Attributes = make(map[string]interface{})
	}
	attributes, err := helper.ConvertMapToStruct(req.Dag.Attributes)
	if err != nil {
		h.log.Error("ConvertMapToStruct attributes", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	dag := &pb.DAG{
		Id:         req.Dag.Id,
		Title:      req.Dag.Title,
		Slug:       req.Dag.Slug,
		Type:       req.Dag.Type,
		Status:     req.Dag.Status,
		CategoryId: req.Dag.CategoryId,
		Attributes: attributes,
	}

	serviceReq := pb.CreateScenarioRequest{
		ProjectId:     projectId.(string),
		EnvironmentId: EnvironmentId.(string),
		Dag:           dag,
		Steps:         dagSteps,
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	serviceReq.CommitInfo = &pb.CommitInfo{
		Guid:       "",
		CommitType: config.COMMIT_TYPE_SCENARIO,
		Name:       fmt.Sprintf("Auto Created Commit Create Scenario - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  projectId.(string),
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

// Scenario godoc
// @Security ApiKeyAuth
// @ID update_scenario
// @Router /v1/scenario [PUT]
// @Summary Update scenario
// @Description Update scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param body body models.CreateScenarioRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=models.DAG} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateFullScenario(c *gin.Context) {
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}
	if !util.IsValidUUID(projectId.(string)) {
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
			Slug:         step.Config.Slug,
			Type:         step.Config.Type,
			ConnectInfo:  step.Config.ConnectInfo,
			RequestInfo:  requestInfo,
			IsParallel:   step.Config.IsParallel,
			UiComponent:  uiComponent,
			Title:        step.Config.Title,
			Description:  step.Config.Description,
			CallbackType: step.Config.CallbackType,
		})
	}

	if req.Dag.Attributes == nil {
		req.Dag.Attributes = make(map[string]interface{})
	}
	attributes, err := helper.ConvertMapToStruct(req.Dag.Attributes)
	if err != nil {
		h.log.Error("ConvertMapToStruct attributes", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	dag := &pb.DAG{
		Id:         req.Dag.Id,
		Title:      req.Dag.Title,
		Slug:       req.Dag.Slug,
		Type:       req.Dag.Type,
		Status:     req.Dag.Status,
		CategoryId: req.Dag.CategoryId,
		Attributes: attributes,
	}

	serviceReq := pb.CreateScenarioRequest{
		ProjectId:     projectId.(string),
		EnvironmentId: EnvironmentId.(string),
		Dag:           dag,
		Steps:         dagSteps,
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	serviceReq.CommitInfo = &pb.CommitInfo{
		Guid:       "",
		CommitType: config.COMMIT_TYPE_SCENARIO,
		Name:       fmt.Sprintf("Auto Created Commit Update Scenario - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  projectId.(string),
	}

	resp, err := services.ScenarioService().DagService().Update(
		c.Request.Context(),
		&serviceReq,
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// Scenario godoc
// @Security ApiKeyAuth
// @ID scenario_history
// @Router /v1/scenario/{id}/history/ [GET]
// @Summary Get History scenario
// @Description Get History scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param id path string true "dag-id"
// @Success 200 {object} status_http.Response{data=pb.GetScenarioHistoryResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetScenarioHistory(c *gin.Context) {

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
	if !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "id not found")
		return
	}

	resp, err := services.ScenarioService().DagService().GetScenarioHistory(
		c.Request.Context(),
		&pb.GetScenarioHistoryRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			DagId:         c.Param("id"),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// Scenario godoc
// @Security ApiKeyAuth
// @ID select_version_scenario
// @Router /v1/scenario/{id}/select-versions [PUT]
// @Summary Select Versions scenario
// @Description Select Versions scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param id path string true "dag-id"
// @Param body body pb.CommitWithRelease  true "Request body"
// @Success 200 {object} status_http.Response{data=emptypb.Empty} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) SelectVersionsScenario(c *gin.Context) {

	req := pb.CommitWithRelease{}

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
	if !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "id not found")
		return
	}

	req.ProjectId = projectId.(string)
	req.EnvironmentId = EnvironmentId.(string)
	req.Id = c.Param("id")

	_, err = services.ScenarioService().DagService().SelectManyScenarioVersions(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, &emptypb.Empty{})
}

// Scenario godoc
// @Security ApiKeyAuth
// @ID revert_scenario
// @Router /v1/scenario/revert [POST]
// @Summary Revert scenario
// @Description Revert scenario
// @Tags Scenario
// @Accept json
// @Produce json
// @Param body body pb.RevertScenarioRequest  true "Request body"
// @Success 200 {object} status_http.Response{data=emptypb.Empty} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) RevertScenario(c *gin.Context) {

	req := pb.RevertScenarioRequest{}

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

	req.ProjectId = projectId.(string)
	req.EnvironmentId = EnvironmentId.(string)

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	req.CommitInfo = &pb.CommitInfo{
		Guid:       "",
		CommitType: config.COMMIT_TYPE_SCENARIO,
		Name:       fmt.Sprintf("Auto Created Commit revert Scenario - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  projectId.(string),
		VersionIds: req.GetVersionIds(),
	}

	_, err = services.ScenarioService().DagService().RevertScenario(
		c.Request.Context(),
		&req,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, &emptypb.Empty{})
}
