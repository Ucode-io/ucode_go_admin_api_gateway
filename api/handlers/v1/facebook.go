package v1

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *HandlerV1) authContext(c *gin.Context) (models.FacebookOAuthState, bool) {
	projectID, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectID.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return models.FacebookOAuthState{}, false
	}

	environmentID, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentID.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return models.FacebookOAuthState{}, false
	}

	userID := ""
	if value, ok := c.Get("user_id"); ok {
		userID, _ = value.(string)
	}

	return models.FacebookOAuthState{
		ProjectId:     projectID.(string),
		EnvironmentId: environmentID.(string),
		UserId:        userID,
	}, true
}

func (h *HandlerV1) FacebookWebhookVerify(c *gin.Context) {
	var (
		mode      = c.Query("hub.mode")
		token     = c.Query("hub.verify_token")
		challenge = c.Query("hub.challenge")
	)

	if mode == "subscribe" && token == h.baseConf.FacebookWebhookVerifyToken && token != "" {
		c.String(http.StatusOK, challenge)
		return
	}

	c.String(http.StatusForbidden, "verification failed")
}

func (h *HandlerV1) FacebookWebhookReceive(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	if !h.verifyFacebookSignature(c.GetHeader(config.FacebookSignatureHeader), body) {
		h.log.Warn("facebook webhook: signature verification failed")
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	var event models.FacebookLeadWebhookEvent

	if err = json.Unmarshal(body, &event); err != nil {
		h.log.Error("facebook webhook: invalid payload", logger.Error(err))
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	// Meta requires a fast ack (~20s); acknowledge first, ingest out of band.
	c.JSON(http.StatusOK, gin.H{"ok": true})

	go h.processFacebookLeadEvent(event)
}

// processFacebookLeadEvent runs in its own goroutine after the webhook is acked,
// so it owns a fresh context and recovers from panics to never take the gateway
// down on a malformed payload. A Page may be wired to several project
// environments (fan-out), so every match receives the lead.
func (h *HandlerV1) processFacebookLeadEvent(event models.FacebookLeadWebhookEvent) {
	defer func() {
		if r := recover(); r != nil {
			h.log.Error("facebook lead: recovered from panic", logger.Any("panic", r))
		}
	}()

	ctx := context.Background()
	for _, entry := range event.Entry {
		for _, change := range entry.Changes {
			if change.Field != config.FacebookWebhookFieldLead {
				continue
			}
			h.ingestFacebookLead(ctx, entry.ID, change.Value)
		}
	}
}

func (h *HandlerV1) ingestFacebookLead(ctx context.Context, pageID string, value models.FacebookLeadChangeValue) {
	if value.PageID != "" {
		pageID = value.PageID
	}

	list, err := h.companyServices.Resource().GetProjectResourcesByExternalId(
		ctx, &pb.GetByExternalIdRequest{
			ExternalId: pageID,
			Type:       pb.ResourceType_META_LEADS,
		},
	)
	if err != nil {
		h.log.Error("facebook lead: resolve resources failed", logger.Error(err))
		return
	}

	var (
		resources = list.GetResources()
		lead      *models.FacebookLead
	)

	if len(resources) == 0 {
		h.log.Warn("facebook lead: no project mapped for page",
			logger.String("page_id", pageID),
			logger.String("form_id", value.FormID),
		)
		return
	}

	for _, resource := range resources {
		var (
			credentials = resource.GetSettings().GetFacebookLeads()
			mapping     *pb.FacebookLeadFormMapping
		)

		if credentials == nil {
			continue
		}

		for _, form := range credentials.GetForms() {
			if form.GetFormId() == value.FormID {
				mapping = form
				break
			}
		}
		if mapping == nil {
			h.log.Info("facebook lead: form not mapped, skipping",
				logger.String("page_id", pageID),
				logger.String("form_id", value.FormID),
				logger.String("project_id", resource.GetProjectId()),
			)
			continue
		}

		if lead == nil {
			// TEST: skip Graph fetch — test webhooks send fake leadgen ids that 404.
			// fetched, err := h.facebookFetchLead(ctx, value.LeadgenID, credentials.GetPageAccessToken())
			// if err != nil {
			// 	h.log.Error("facebook lead: fetch failed", logger.Error(err),
			// 		logger.String("leadgen_id", value.LeadgenID))
			// 	return
			// }
			// lead = &fetched
			lead = &models.FacebookLead{}
		}

		if err := h.writeFacebookLead(ctx, resource, mapping, *lead, value); err != nil {
			h.log.Error("facebook lead: write failed", logger.Error(err),
				logger.String("project_id", resource.GetProjectId()),
				logger.String("table_slug", mapping.GetTableSlug()),
			)
		}
	}
}

func (h *HandlerV1) writeFacebookLead(ctx context.Context, resource *pb.ProjectResource, mapping *pb.FacebookLeadFormMapping, lead models.FacebookLead, value models.FacebookLeadChangeValue) error {
	//var (
	//	values = facebookLeadValues(lead.FieldData)
	//	data   = map[string]any{}
	//)
	//
	//for _, field := range mapping.GetFields() {
	//	raw, present := values[strings.ToLower(field.GetLeadField())]
	//	if !present || raw == "" {
	//		if field.GetRequired() {
	//			return fmt.Errorf("required lead field %q is missing", field.GetLeadField())
	//		}
	//		continue
	//	}
	//	data[field.GetTableField()] = raw
	//}
	//
	// TEST: fill every mapped field with a placeholder instead of real lead data.
	data := map[string]any{}
	for _, field := range mapping.GetFields() {
		data[field.GetTableField()] = "hello world"
	}

	if len(data) == 0 {
		h.log.Info("facebook lead: no mapped fields produced data, skipping",
			logger.String("leadgen_id", value.LeadgenID),
			logger.String("table_slug", mapping.GetTableSlug()),
		)
		return nil
	}

	data["guid"] = uuid.NewSHA1(uuid.NameSpaceOID, []byte("facebook-lead:"+value.LeadgenID)).String()

	resourceModel, err := h.companyServices.ServiceResource().GetSingle(ctx, &pb.GetSingleServiceResourceReq{
		ProjectId:     resource.GetProjectId(),
		EnvironmentId: resource.GetEnvironmentId(),
		ServiceType:   pb.ServiceType_BUILDER_SERVICE,
	})
	if err != nil {
		return err
	}

	services, err := h.GetProjectSrvc(ctx, resource.GetProjectId(), resourceModel.NodeType)
	if err != nil {
		return err
	}

	structData, err := helper.ConvertMapToStruct(data)
	if err != nil {
		return err
	}

	_, err = services.GetBuilderServiceByType(resourceModel.NodeType).ObjectBuilder().Create(ctx, &obs.CommonMessage{
		TableSlug: mapping.GetTableSlug(),
		Data:      structData,
		ProjectId: resourceModel.ResourceEnvironmentId,
	})
	return err
}

func facebookLeadValues(fieldData []models.FacebookFieldData) map[string]string {
	values := make(map[string]string, len(fieldData))

	for _, field := range fieldData {
		if len(field.Values) > 0 {
			values[strings.ToLower(field.Name)] = field.Values[0]
		}
	}
	return values
}

func (h *HandlerV1) verifyFacebookSignature(header string, body []byte) bool {
	secret := strings.TrimSpace(h.baseConf.FacebookAppSecret)
	if secret == "" {
		return false
	}

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, config.FacebookSignaturePrefix) {
		return false
	}

	provided, err := hex.DecodeString(strings.TrimPrefix(header, config.FacebookSignaturePrefix))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return subtle.ConstantTimeCompare(provided, expected) == 1
}
