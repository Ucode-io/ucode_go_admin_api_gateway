package v1

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/handlers/metaads"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
)

type workspaceAdAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func (h *HandlerV1) workspaceAdAccounts(c *gin.Context) ([]workspaceAdAccount, string, bool) {
	state, ok := h.authContext(c)
	if !ok {
		return nil, "", false
	}
	token, err := h.getFacebookUserToken(c.Request.Context(), models.FacebookOAuthState{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId})
	if err != nil {
		h.HandleResponse(c, status_http.ServiceUnavailable, "Meta account is not connected")
		return nil, "", false
	}
	var result struct {
		Data []workspaceAdAccount `json:"data"`
	}
	if err := h.facebookGraphGet(c.Request.Context(), "me/adaccounts", url.Values{"fields": {"id,name,currency"}, "limit": {"100"}, "access_token": {token}}, &result); err != nil {
		h.HandleResponse(c, status_http.BadGateway, "Meta ad accounts are unavailable")
		return nil, "", false
	}
	return result.Data, token, true
}

func (h *HandlerV1) WorkspaceMetaAdsAccounts(c *gin.Context) {
	accounts, _, ok := h.workspaceAdAccounts(c)
	if !ok {
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"accounts": accounts})
}

func (h *HandlerV1) WorkspaceMetaAdsDashboard(c *gin.Context) {
	accountID := strings.TrimSpace(c.Query("account_id"))
	if accountID == "" {
		h.HandleResponse(c, status_http.BadRequest, "account_id is required")
		return
	}
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
	conf := h.baseConf
	conf.MetaAdsAdAccountID = strings.TrimPrefix(accountID, "act_")
	conf.MetaAdsAccessToken = token
	metaads.NewHandler(conf, h.centralRedis, h.log).Dashboard(c)
}
