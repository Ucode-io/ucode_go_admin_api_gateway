package v1

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"ucode/ucode_go_api_gateway/api/status_http"
)

type workspaceCampaign struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effective_status"`
}

func (h *HandlerV1) WorkspaceMetaAdsCampaigns(c *gin.Context) {
	accountID := strings.TrimSpace(c.Query("account_id"))
	accounts, token, ok := h.workspaceAdAccounts(c)
	if !ok {
		return
	}
	allowed := false
	for _, account := range accounts {
		if account.ID == accountID {
			allowed = true
			break
		}
	}
	if !allowed {
		h.HandleResponse(c, status_http.Forbidden, "ad account is not connected to this project")
		return
	}
	result := make([]workspaceCampaign, 0)
	query := url.Values{"fields": {"id,name,status,effective_status"}, "limit": {"100"}, "access_token": {token}}
	for page := 0; page < 20; page++ {
		var response struct {
			Data   []workspaceCampaign `json:"data"`
			Paging struct {
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := h.facebookGraphGet(c.Request.Context(), accountID+"/campaigns", query, &response); err != nil {
			h.HandleResponse(c, status_http.BadGateway, "Meta campaigns are unavailable")
			return
		}
		result = append(result, response.Data...)
		if response.Paging.Next == "" || response.Paging.Cursors.After == "" {
			break
		}
		query.Set("after", response.Paging.Cursors.After)
	}
	h.HandleResponse(c, status_http.OK, gin.H{"campaigns": result})
}

func metaAdsPipelineCampaignsKey(projectID, environmentID, pipeline string) string {
	return "crm:meta-ads:pipeline-campaigns:" + projectID + ":" + environmentID + ":" + url.QueryEscape(pipeline)
}

func (h *HandlerV1) WorkspaceMetaAdsPipelineCampaigns(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	pipeline := strings.TrimSpace(c.Query("pipeline_value"))
	if pipeline == "" {
		h.HandleResponse(c, status_http.BadRequest, "pipeline_value is required")
		return
	}
	if h.centralRedis == nil {
		h.HandleResponse(c, status_http.ServiceUnavailable, "campaign settings are unavailable")
		return
	}
	body, err := h.centralRedis.Get(c.Request.Context(), metaAdsPipelineCampaignsKey(state.ProjectId, state.EnvironmentId, pipeline)).Bytes()
	if err == redis.Nil {
		h.HandleResponse(c, status_http.OK, gin.H{"campaign_ids": []string{}})
		return
	}
	if err != nil {
		h.HandleResponse(c, status_http.ServiceUnavailable, "campaign settings are unavailable")
		return
	}
	var ids []string
	if err := json.Unmarshal(body, &ids); err != nil {
		h.HandleResponse(c, status_http.ServiceUnavailable, "campaign settings are invalid")
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"campaign_ids": ids})
}

func (h *HandlerV1) SaveWorkspaceMetaAdsPipelineCampaigns(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	var req struct {
		PipelineValue string   `json:"pipeline_value" binding:"required"`
		CampaignIDs   []string `json:"campaign_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.PipelineValue = strings.TrimSpace(req.PipelineValue)
	if req.PipelineValue == "" {
		h.HandleResponse(c, status_http.BadRequest, "pipeline_value is required")
		return
	}
	if h.centralRedis == nil {
		h.HandleResponse(c, status_http.ServiceUnavailable, "campaign settings are unavailable")
		return
	}
	ids, seen := make([]string, 0, len(req.CampaignIDs)), map[string]bool{}
	for _, id := range req.CampaignIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	body, _ := json.Marshal(ids)
	if err := h.centralRedis.Set(c.Request.Context(), metaAdsPipelineCampaignsKey(state.ProjectId, state.EnvironmentId, req.PipelineValue), body, 0).Err(); err != nil {
		h.HandleResponse(c, status_http.ServiceUnavailable, "campaign settings could not be saved")
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"campaign_ids": ids})
}
