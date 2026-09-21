package v1

import (
	"strconv"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

// FacebookCRMBackfill replays connected lead forms so older CRM deals receive
// the Meta ad attribution that was not stored when they were first created.
func (h *HandlerV1) FacebookCRMBackfill(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	since := time.Now().Add(-30 * 24 * time.Hour).Unix()
	if raw := strings.TrimSpace(c.Query("since")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			h.HandleResponse(c, status_http.BadRequest, "since must be a unix timestamp")
			return
		}
		since = parsed
	}
	resources, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), &pb.GetProjectResourceListRequest{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Type: pb.ResourceType_META_LEADS})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	fetched, updated, errors := 0, 0, 0
	for _, resource := range resources.GetResources() {
		credentials := resource.GetSettings().GetFacebookLeads()
		if credentials == nil || credentials.GetPageAccessToken() == "" || resource.GetExternalId() == "" {
			continue
		}
		forms, listErr := h.facebookListForms(c.Request.Context(), resource.GetExternalId(), credentials.GetPageAccessToken())
		if listErr != nil {
			errors++
			continue
		}
		for _, form := range forms {
			leads, _, fetchErr := h.facebookFetchFormLeads(c.Request.Context(), form.ID, credentials.GetPageAccessToken(), since)
			if fetchErr != nil {
				errors++
				continue
			}
			for _, lead := range leads {
				fetched++
				if writeErr := h.writeProfessionalCRMLead(c.Request.Context(), resource, lead, models.FacebookLeadChangeValue{LeadgenID: lead.ID, PageID: resource.GetExternalId(), FormID: form.ID}); writeErr != nil {
					errors++
				} else {
					updated++
				}
			}
		}
	}
	h.HandleResponse(c, status_http.OK, gin.H{"since": time.Unix(since, 0).UTC().Format(time.RFC3339), "fetched": fetched, "updated": updated, "errors": errors})
}
