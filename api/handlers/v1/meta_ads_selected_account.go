package v1

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

type metaAdsetsPage struct {
	Data []struct {
		PromotedObject struct {
			PageID string `json:"page_id"`
		} `json:"promoted_object"`
	} `json:"data"`
	Paging struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

// WorkspaceMetaAdsSelectedAccount resolves the ad account from the Facebook
// Page assigned to a CRM pipeline. The account name is not a reliable link.
func (h *HandlerV1) WorkspaceMetaAdsSelectedAccount(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	list, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), &pb.GetProjectResourceListRequest{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Type: pb.ResourceType_META_LEADS})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	pageIDs := map[string]bool{}
	for _, resource := range list.GetResources() {
		if facebookAssignedPipeline(resource.GetSettings().GetFacebookLeads().GetCrmMapping()) != "" {
			pageIDs[resource.GetExternalId()] = true
		}
	}
	if len(pageIDs) == 0 {
		h.HandleResponse(c, status_http.NotFound, "No Facebook Page is assigned to a pipeline")
		return
	}
	accounts, token, ok := h.workspaceAdAccounts(c)
	if !ok {
		return
	}
	bestScore, bestAccount := 0, workspaceAdAccount{}
	for _, account := range accounts {
		var result metaAdsetsPage
		query := url.Values{"fields": {"promoted_object"}, "limit": {"100"}, "access_token": {token}}
		score := 0
		for page := 0; page < 10; page++ {
			result = metaAdsetsPage{}
			if err := h.facebookGraphGet(c.Request.Context(), account.ID+"/adsets", query, &result); err != nil {
				break
			}
			for _, adset := range result.Data {
				if pageIDs[adset.PromotedObject.PageID] {
					score++
				}
			}
			if result.Paging.Next == "" || result.Paging.Cursors.After == "" {
				break
			}
			query.Set("after", result.Paging.Cursors.After)
		}
		if score > bestScore {
			bestScore, bestAccount = score, account
		}
	}
	if bestScore == 0 {
		h.HandleResponse(c, status_http.NotFound, "No connected ad account has campaigns for the selected Facebook Page")
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"account": bestAccount})
}
