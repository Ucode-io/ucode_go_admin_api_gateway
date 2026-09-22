package v1

import (
	"strconv"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"

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
	errorSamples := make([]string, 0, 5)
	recordError := func(err error) {
		errors++
		if err != nil && len(errorSamples) < 5 {
			errorSamples = append(errorSamples, err.Error())
		}
	}
	for _, resource := range resources.GetResources() {
		credentials := resource.GetSettings().GetFacebookLeads()
		if credentials == nil || credentials.GetPageAccessToken() == "" || resource.GetExternalId() == "" {
			continue
		}
		forms, listErr := h.facebookListForms(c.Request.Context(), resource.GetExternalId(), credentials.GetPageAccessToken())
		if listErr != nil {
			// Listing page forms requires pages_manage_ads. Lead retrieval itself
			// works with the existing leads_retrieval token, so fall back to the
			// form IDs already persisted by the CRM ingestion flow.
			forms = h.crmStoredFacebookForms(c, state)
			forms = facebookFormsForPage(forms, resource.GetExternalId())
			if len(forms) == 0 {
				recordError(listErr)
				continue
			}
		}
		for _, form := range forms {
			leads, _, fetchErr := h.facebookFetchFormLeads(c.Request.Context(), form.ID, credentials.GetPageAccessToken(), since)
			// The Ads dashboard can be configured with a separate, working Meta
			// access token. Try it only when the page token cannot read a legacy
			// form; this makes historical attribution independent of the CRM OAuth
			// connection while preserving the normal webhook flow.
			if fetchErr != nil && strings.TrimSpace(h.baseConf.MetaAdsAccessToken) != "" && h.baseConf.MetaAdsAccessToken != credentials.GetPageAccessToken() {
				leads, _, fetchErr = h.facebookFetchFormLeads(c.Request.Context(), form.ID, h.baseConf.MetaAdsAccessToken, since)
			}
			if fetchErr != nil {
				recordError(fetchErr)
				continue
			}
			for _, lead := range leads {
				fetched++
				if writeErr := h.writeProfessionalCRMLead(c.Request.Context(), resource, lead, models.FacebookLeadChangeValue{LeadgenID: lead.ID, PageID: resource.GetExternalId(), FormID: form.ID}); writeErr != nil {
					recordError(writeErr)
				} else {
					updated++
				}
			}
		}
	}
	h.HandleResponse(c, status_http.OK, gin.H{"since": time.Unix(since, 0).UTC().Format(time.RFC3339), "fetched": fetched, "updated": updated, "errors": errors, "error_samples": errorSamples})
}

// FacebookSaveCRMFormIDs stores only historical form IDs, preserving the
// connection status and all existing credentials/settings.
func (h *HandlerV1) FacebookSaveCRMFormIDs(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	var req struct {
		FormIDs []string `json:"form_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.FormIDs) == 0 {
		h.HandleResponse(c, status_http.BadRequest, "form_ids is required")
		return
	}
	ids := make(map[string]struct{}, len(req.FormIDs))
	for _, id := range req.FormIDs {
		if id = strings.TrimSpace(id); id != "" {
			ids[id] = struct{}{}
		}
	}
	list, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), &pb.GetProjectResourceListRequest{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Type: pb.ResourceType_META_LEADS})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	updated := 0
	for _, resource := range list.GetResources() {
		credentials := resource.GetSettings().GetFacebookLeads()
		if credentials == nil {
			continue
		}
		seen := map[string]struct{}{}
		forms := credentials.GetForms()
		for _, form := range forms {
			seen[form.GetFormId()] = struct{}{}
		}
		for id := range ids {
			if _, exists := seen[id]; !exists {
				forms = append(forms, &pb.FacebookLeadFormMapping{FormId: id})
				seen[id] = struct{}{}
			}
		}
		credentials.Forms = forms
		name := resource.GetName()
		if name == "" {
			name = credentials.GetPageName()
		}
		if _, err := h.companyServices.Resource().UpdateProjectResource(c.Request.Context(), &pb.ProjectResource{Id: resource.GetId(), ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Name: name, Type: pb.ResourceType_META_LEADS.String(), ResourceType: int32(pb.ResourceType_META_LEADS), ExternalId: resource.GetExternalId(), Settings: &pb.Settings{FacebookLeads: credentials}}); err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		updated++
	}
	h.HandleResponse(c, status_http.OK, gin.H{"updated_resources": updated, "form_ids": len(ids)})
}

func (h *HandlerV1) crmStoredFacebookForms(c *gin.Context, state models.FacebookOAuthState) []models.FacebookForm {
	svc, envID, err := h.resolveProjectBuilder(c.Request.Context(), state.ProjectId, state.EnvironmentId)
	if err != nil {
		return nil
	}
	resp, err := svc.GoObjectBuilderService().ObjectBuilder().GetList2(c.Request.Context(), &nb.CommonMessage{TableSlug: "lead_forms", Data: mustStruct(map[string]any{"limit": 1000, "offset": 0}), ProjectId: envID})
	if err != nil || resp.GetData() == nil {
		return nil
	}
	rows, _ := resp.GetData().AsMap()["response"].([]any)
	forms := make([]models.FacebookForm, 0, len(rows))
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := row["form_id"].(string)
		if id == "" {
			continue
		}
		name, _ := row["name"].(string)
		pageID, _ := row["page_id"].(string)
		forms = append(forms, models.FacebookForm{ID: id, Name: name, PageID: pageID})
	}
	return forms
}

// facebookFormsForPage prevents replaying every historical form against every
// connected page token. CRM persists page_id alongside form_id during normal
// lead ingestion, so this keeps the backfill within Meta's request budget.
func facebookFormsForPage(forms []models.FacebookForm, pageID string) []models.FacebookForm {
	filtered := make([]models.FacebookForm, 0, len(forms))
	for _, form := range forms {
		if form.PageID == pageID {
			filtered = append(filtered, form)
		}
	}
	return filtered
}
