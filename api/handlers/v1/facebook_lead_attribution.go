package v1

import (
	"net/url"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FacebookLeadAttribution resolves a small set of CRM deals against their
// original Meta form submissions without returning the leads' personal data.
func (h *HandlerV1) FacebookLeadAttribution(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	formID := strings.TrimSpace(c.Query("form_id"))
	rawGUIDs := strings.Split(c.Query("deal_guids"), ",")
	if formID == "" || len(rawGUIDs) == 0 || len(rawGUIDs) > 10 {
		h.HandleResponse(c, status_http.BadRequest, "form_id and 1-10 deal_guids are required")
		return
	}
	wanted := make(map[string]struct{}, len(rawGUIDs))
	for _, raw := range rawGUIDs {
		guid := strings.TrimSpace(raw)
		if _, err := uuid.Parse(guid); err != nil {
			h.HandleResponse(c, status_http.BadRequest, "invalid deal_guid")
			return
		}
		wanted[guid] = struct{}{}
	}
	var form *models.FacebookForm
	for _, candidate := range h.crmStoredFacebookForms(c, state) {
		if candidate.ID == formID {
			copy := candidate
			form = &copy
			break
		}
	}
	if form == nil || form.PageID == "" {
		h.HandleResponse(c, status_http.NotFound, "form is not connected to this project")
		return
	}
	resource, err := h.findFacebookResource(c.Request.Context(), state.ProjectId, state.EnvironmentId, form.PageID)
	if err != nil || resource == nil {
		h.HandleResponse(c, status_http.NotFound, "Meta page is not connected")
		return
	}
	pageToken := resource.GetSettings().GetFacebookLeads().GetPageAccessToken()
	if pageToken == "" {
		h.HandleResponse(c, status_http.ServiceUnavailable, "Meta page token is unavailable")
		return
	}
	leads, _, err := h.facebookFetchFormLeads(c.Request.Context(), formID, pageToken, time.Now().Add(-30*24*time.Hour).Unix())
	if err != nil && h.baseConf.MetaAdsAccessToken != "" {
		leads, _, err = h.facebookFetchFormLeads(c.Request.Context(), formID, h.baseConf.MetaAdsAccessToken, time.Now().Add(-30*24*time.Hour).Unix())
	}
	if err != nil {
		if token, tokenErr := h.getFacebookUserToken(c.Request.Context(), state); tokenErr == nil {
			leads, _, err = h.facebookFetchFormLeads(c.Request.Context(), formID, token, time.Now().Add(-30*24*time.Hour).Unix())
		}
	}
	if err != nil {
		h.HandleResponse(c, status_http.BadGateway, "Meta form leads are unavailable: "+err.Error())
		return
	}
	results := make([]gin.H, 0, len(wanted))
	for _, lead := range leads {
		guid := uuid.NewSHA1(uuid.NameSpaceOID, []byte("facebook-lead:"+lead.ID)).String()
		if _, match := wanted[guid]; !match {
			continue
		}
		result := gin.H{"deal_guid": guid, "ad_id": lead.AdID, "ad_name": lead.AdName, "is_organic": lead.IsOrganic}
		if lead.AdID != "" {
			token, tokenErr := h.getFacebookUserToken(c.Request.Context(), state)
			if tokenErr == nil {
				var ad struct {
					CampaignID string `json:"campaign_id"`
				}
				if h.facebookGraphGet(c.Request.Context(), lead.AdID, url.Values{"fields": {"campaign_id"}, "access_token": {token}}, &ad) == nil && ad.CampaignID != "" {
					result["campaign_id"] = ad.CampaignID
					var campaign struct {
						Name string `json:"name"`
					}
					if h.facebookGraphGet(c.Request.Context(), ad.CampaignID, url.Values{"fields": {"name"}, "access_token": {token}}, &campaign) == nil {
						result["campaign_name"] = campaign.Name
					}
				}
			}
		}
		results = append(results, result)
	}
	h.HandleResponse(c, status_http.OK, gin.H{"attribution": results})
}
