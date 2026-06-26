package v1

import (
	"context"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) FacebookSubscribe(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}

	var req models.FacebookSubscribeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.PageId = strings.TrimSpace(req.PageId)

	userToken, err := h.getFacebookUserToken(c.Request.Context(), state)
	if err != nil {
		h.HandleResponse(c, status_http.BadRequest, "facebook is not connected, run connect first")
		return
	}

	pages, err := h.facebookListPages(c.Request.Context(), userToken)
	if err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	var (
		page  models.FacebookPage
		found bool

		pageName = strings.TrimSpace(req.PageName)
	)

	for _, p := range pages {
		if p.ID == req.PageId {
			page, found = p, true
			break
		}
	}
	if !found {
		h.HandleResponse(c, status_http.BadRequest, "page is not managed by the connected account")
		return
	}

	if pageName == "" {
		pageName = page.Name
	}

	if err = h.facebookSubscribePage(c.Request.Context(), page.ID, page.AccessToken); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	existing, err := h.findFacebookResource(c.Request.Context(), state.ProjectId, state.EnvironmentId, page.ID)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	credentials := &pb.FacebookLeadsCredentials{
		PageId:          page.ID,
		PageName:        pageName,
		PageAccessToken: page.AccessToken,
		ConnectedUserId: state.UserId,
		ConnectedAt:     time.Now().UTC().Format(time.RFC3339),
		Status:          config.FacebookStatusActive,
	}
	// On re-subscribe keep the saved form mapping and original connect time.
	if prev := existing.GetSettings().GetFacebookLeads(); prev != nil {
		credentials.Forms = prev.GetForms()
		if prev.GetConnectedAt() != "" {
			credentials.ConnectedAt = prev.GetConnectedAt()
		}
	}

	if existing.GetName() != "" {
		pageName = existing.GetName()
	}

	resource, err := h.companyServices.Resource().UpsertProjectResource(
		c.Request.Context(), &pb.AddResourceToProjectRequest{
			Name:          pageName,
			ProjectId:     state.ProjectId,
			EnvironmentId: state.EnvironmentId,
			Type:          pb.ResourceType_META_LEADS,
			ExternalId:    page.ID,
			Settings:      &pb.Settings{FacebookLeads: credentials},
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{"resource_id": resource.GetId(), "page_id": page.ID})
}

// FacebookSaveMapping stores the form → table field mapping. Forms absent here
// are skipped at ingestion time.
func (h *HandlerV1) FacebookSaveMapping(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}

	var req models.FacebookMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.PageId = strings.TrimSpace(req.PageId)

	resource, err := h.findFacebookResource(c.Request.Context(), state.ProjectId, state.EnvironmentId, req.PageId)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if resource == nil {
		h.HandleResponse(c, status_http.BadRequest, "facebook page is not connected for this project")
		return
	}

	credentials := resource.GetSettings().GetFacebookLeads()
	if credentials == nil {
		credentials = &pb.FacebookLeadsCredentials{PageId: req.PageId}
	}
	credentials.Forms = facebookFormsToProto(req.Forms)

	name := resource.GetName()
	if name == "" {
		name = credentials.GetPageName()
	}

	if _, err := h.companyServices.Resource().UpdateProjectResource(c.Request.Context(), &pb.ProjectResource{
		Id:            resource.GetId(),
		ProjectId:     state.ProjectId,
		EnvironmentId: state.EnvironmentId,
		Name:          name,
		Type:          pb.ResourceType_META_LEADS.String(),
		ResourceType:  int32(pb.ResourceType_META_LEADS),
		ExternalId:    req.PageId,
		Settings:      &pb.Settings{FacebookLeads: credentials},
	}); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{"resource_id": resource.GetId()})
}

func (h *HandlerV1) FacebookIntegration(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}

	list, err := h.companyServices.Resource().GetProjectResourceList(
		c.Request.Context(),
		&pb.GetProjectResourceListRequest{
			ProjectId:     state.ProjectId,
			EnvironmentId: state.EnvironmentId,
			Type:          pb.ResourceType_META_LEADS,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	statuses := make([]models.FacebookIntegrationStatus, 0, len(list.GetResources()))

	for _, resource := range list.GetResources() {
		status := models.FacebookIntegrationStatus{
			Connected:  true,
			ResourceId: resource.GetId(),
			PageId:     resource.GetExternalId(),
		}

		if credentials := resource.GetSettings().GetFacebookLeads(); credentials != nil {
			status.PageName = credentials.GetPageName()
			status.Status = credentials.GetStatus()
			status.ConnectedAt = credentials.GetConnectedAt()
			status.Forms = facebookFormsFromProto(credentials.GetForms())
		}

		statuses = append(statuses, status)
	}

	h.HandleResponse(c, status_http.OK, gin.H{"connected": len(statuses) > 0, "integrations": statuses})
}

func (h *HandlerV1) FacebookDisconnect(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}

	resourceID := strings.TrimSpace(c.Param("id"))
	if resourceID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "resource id is required")
		return
	}

	resource, err := h.companyServices.Resource().GetSingleProjectResouece(c.Request.Context(), &pb.PrimaryKeyProjectResource{
		Id:            resourceID,
		ProjectId:     state.ProjectId,
		EnvironmentId: state.EnvironmentId,
	})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if credentials := resource.GetSettings().GetFacebookLeads(); credentials != nil {
		pageID := strings.TrimSpace(credentials.GetPageId())
		pageToken := strings.TrimSpace(credentials.GetPageAccessToken())
		if pageID != "" && pageToken != "" {
			if err := h.facebookUnsubscribePage(c.Request.Context(), pageID, pageToken); err != nil {
				// Page may already be unsubscribed or the token expired; log and
				// proceed with removal so the integration can be cleaned up.
				h.log.Warn("facebook disconnect: unsubscribe failed: " + err.Error())
			}
		}
	}

	if _, err := h.companyServices.Resource().DeleteProjectResource(c.Request.Context(), &pb.PrimaryKeyProjectResource{
		Id:            resourceID,
		ProjectId:     state.ProjectId,
		EnvironmentId: state.EnvironmentId,
	}); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{"resource_id": resourceID})
}

func (h *HandlerV1) findFacebookResource(ctx context.Context, projectID, environmentID, pageID string) (*pb.ProjectResource, error) {
	list, err := h.companyServices.Resource().GetProjectResourceList(
		ctx, &pb.GetProjectResourceListRequest{
			ProjectId:     projectID,
			EnvironmentId: environmentID,
			Type:          pb.ResourceType_META_LEADS,
		},
	)

	if err != nil {
		return nil, err
	}

	for _, resource := range list.GetResources() {
		if resource.GetExternalId() == pageID {
			return resource, nil
		}
	}
	return nil, nil
}

func facebookFormsToProto(forms []models.FacebookFormMapping) []*pb.FacebookLeadFormMapping {
	result := make([]*pb.FacebookLeadFormMapping, 0, len(forms))
	for _, form := range forms {
		fields := make([]*pb.FacebookLeadFieldMapping, 0, len(form.Fields))
		for _, field := range form.Fields {
			fields = append(fields, &pb.FacebookLeadFieldMapping{
				LeadField:  field.LeadField,
				TableField: field.TableField,
				Required:   field.Required,
			})
		}
		result = append(result, &pb.FacebookLeadFormMapping{
			FormId:    form.FormId,
			FormName:  form.FormName,
			TableSlug: form.TableSlug,
			Fields:    fields,
		})
	}
	return result
}

func facebookFormsFromProto(forms []*pb.FacebookLeadFormMapping) []models.FacebookFormMapping {
	result := make([]models.FacebookFormMapping, 0, len(forms))
	for _, form := range forms {
		fields := make([]models.FacebookFieldMapping, 0, len(form.GetFields()))
		for _, field := range form.GetFields() {
			fields = append(fields, models.FacebookFieldMapping{
				LeadField:  field.GetLeadField(),
				TableField: field.GetTableField(),
				Required:   field.GetRequired(),
			})
		}
		result = append(result, models.FacebookFormMapping{
			FormId:    form.GetFormId(),
			FormName:  form.GetFormName(),
			TableSlug: form.GetTableSlug(),
			Fields:    fields,
		})
	}
	return result
}
