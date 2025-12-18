package v1

import (
	"context"

	hHelper "ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type GenerateAIUIRequest struct {
	Prompt           string   `json:"prompt" binding:"required"`
	ProjectType      string   `json:"project_type"`
	ManagementSystem []string `json:"management_system"`
}

func (h *HandlerV1) GenerateAIUI(c *gin.Context) {
	var req GenerateAIUIRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
	 h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
	 return
	}
	
	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
	 h.HandleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
	 return
	}

	if projectId == "" || environmentId == "" {
		h.HandleResponse(c, status_http.BadRequest, "project_id or environment_id missing")
		return
	}

	apiKeys, err := h.AuthService().ApiKey().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: cast.ToString(environmentId),
		ProjectId:     cast.ToString(projectId),
	 })
	 if err != nil {
		h.HandleResponse(c, status_http.GRPCError, "error getting api keys by environment id")
		return
	 }

	 if len(apiKeys.Data) == 0 {
		h.HandleResponse(c, status_http.GRPCError, "error no api key for this environment")
		return
	 }

	payload := map[string]any{
		"prompt":             req.Prompt,
		"project_id":         projectId,
		"environment_id":     environmentId,
		"x_api_key":          apiKeys.Data[0].AppId,
		"project_type":       req.ProjectType,
		"management_system":  req.ManagementSystem,
	}

	resp, err := hHelper.DoInvokeMCPAIUI(c, h, payload)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, resp)
}
