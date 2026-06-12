package v1

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// ==================== Agent Endpoints ====================

func (h *HandlerV1) CreateAgent(c *gin.Context) {
	var request pbo.CreateAgentRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	request.ResourceEnvId = resourceEnvId

	response, err := service.GoObjectBuilderService().Agent().CreateAgent(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.Created, response)
}

func (h *HandlerV1) GetAllAgents(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		projectId      = c.Query("project_id")
		name           = c.Query("name")
		model          = c.Query("model")
		orderBy        = c.Query("order_by")
		orderDirection = c.Query("order_direction")
		limit          = cast.ToInt32(c.DefaultQuery("limit", "20"))
		offset         = cast.ToInt32(c.DefaultQuery("offset", "0"))
	)

	response, err := service.GoObjectBuilderService().Agent().GetAllAgents(
		c.Request.Context(),
		&pbo.GetAllAgentsRequest{
			ResourceEnvId:  resourceEnvId,
			ProjectId:      projectId,
			Name:           name,
			Model:          model,
			OrderBy:        orderBy,
			OrderDirection: orderDirection,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) GetAgent(c *gin.Context) {
	var agentId = c.Param("agent-id")
	if !util.IsValidUUID(agentId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid agent-id")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().Agent().GetAgent(
		c.Request.Context(),
		&pbo.AgentPrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            agentId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) UpdateAgent(c *gin.Context) {
	var (
		request pbo.UpdateAgentRequest
		agentId = c.Param("agent-id")
	)

	if !util.IsValidUUID(agentId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid agent-id")
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	request.ResourceEnvId = resourceEnvId
	request.Id = agentId

	response, err := service.GoObjectBuilderService().Agent().UpdateAgent(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) DeleteAgent(c *gin.Context) {
	var agentId = c.Param("agent-id")
	if !util.IsValidUUID(agentId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid agent-id")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().Agent().DeleteAgent(
		c.Request.Context(),
		&pbo.AgentPrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            agentId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

// ==================== Agent Execution ====================

type runAgentRequest struct {
	Message string `json:"message"`
}

func (h *HandlerV1) RunAgent(c *gin.Context) {
	var (
		request runAgentRequest
		agentId = c.Param("agent-id")
	)

	if !util.IsValidUUID(agentId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid agent-id")
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if request.Message == "" {
		h.HandleResponse(c, status_http.BadRequest, "message is required")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	agent, err := service.GoObjectBuilderService().Agent().GetAgent(
		c.Request.Context(),
		&pbo.AgentPrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            agentId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if !agent.GetEnabled() {
		h.HandleResponse(c, status_http.InvalidArgument, "agent is disabled")
		return
	}

	run, err := h.runAgent(c.Request.Context(), service, resourceEnvId, agent, request.Message)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, run)
}

// ==================== Agent Run Endpoints (audit trail) ====================

func (h *HandlerV1) GetAgentRuns(c *gin.Context) {
	var agentId = c.Param("agent-id")
	if !util.IsValidUUID(agentId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid agent-id")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		runStatus      = c.Query("status")
		orderBy        = c.Query("order_by")
		orderDirection = c.Query("order_direction")
		limit          = cast.ToInt32(c.DefaultQuery("limit", "20"))
		offset         = cast.ToInt32(c.DefaultQuery("offset", "0"))
	)

	response, err := service.GoObjectBuilderService().Agent().GetAllAgentRuns(
		c.Request.Context(),
		&pbo.GetAllAgentRunsRequest{
			ResourceEnvId:  resourceEnvId,
			AgentId:        agentId,
			Status:         runStatus,
			OrderBy:        orderBy,
			OrderDirection: orderDirection,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) GetAgentRun(c *gin.Context) {
	var runId = c.Param("run-id")
	if !util.IsValidUUID(runId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid run-id")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().Agent().GetAgentRun(
		c.Request.Context(),
		&pbo.AgentRunPrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            runId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}
