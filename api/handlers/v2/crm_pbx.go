package v2

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"ucode/ucode_go_api_gateway/api/models"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
)

func denyCRMPBX(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "error", "data": gin.H{"message": "record unavailable"}})
}

func crmPBXMethod(method string) bool {
	switch method {
	case crmPBXIdentityMethod, "crm_deals_page", "crm_missed_calls", "crm_deal_transfer", "crm_deal_capability", "crm_pbx_history_wake", "crm_pbx_history", "crm_pbx_recording_ref", "crm_pbx_recording", "crm_pbx_recording_access":
		return true
	}
	return crmPBXProtectedLegacyMethod(method)
}

func crmPBXProtectedLegacyMethod(method string) bool {
	switch method {
	case "pbx_get_recording", "pbx_save_stereo_recording", "pbx_save_transcript", "pbx_list_untranscribed", "pbx_debug":
		return true
	}
	return false
}

func crmPBXActor(actor *as.V2HasAccessUserRes) bool {
	if actor == nil || actor.GetId() == "" || actor.GetProjectId() != crmReviewProject || actor.GetEnvId() != crmReviewEnvironment {
		return false
	}
	if _, err := uuid.Parse(actor.GetUserId()); err != nil || len(actor.GetUserId()) != 36 {
		return false
	}
	if actor.GetRoleId() == crmReviewRole && actor.GetClientTypeId() == crmReviewClientType {
		switch actor.GetUserIdAuth() {
		case "5f73ff09-69a9-458d-8868-5009e6c8291c", "eeae89f7-c111-4fb1-bea0-02c38586ceb1", "5e7f763e-120b-4b9a-a451-04775cee55a7":
			// The native membership predicate independently resolves the unique
			// auth linkage; an admin CRM GUID is never guessed here.
			return true
		}
	}
	if actor.GetRoleId() != crmPBXOperatorRole || actor.GetClientTypeId() != crmPBXOperatorClient {
		return false
	}
	switch actor.GetUserIdAuth() {
	case "50d88de6-c51e-429a-bb6c-0a284273a3d1":
		return actor.GetUserId() == "76fad06c-7c1a-4967-8037-f698279c4335"
	case "78aa2409-650e-4606-9ca0-21c867e068b2":
		return actor.GetUserId() == "99543ad4-9db0-48b8-8da6-39e15c3efd4c"
	case "def73797-3d1e-4b67-b335-635cd316baba":
		return actor.GetUserId() == "f10bc331-e67e-4c45-b6bf-70222c4ad959"
	}
	return false
}

// Return false only for the original, positively outside-scope dispatch. The
// business path never asks frozen function-service to author identity claims.
func (h *HandlerV2) handleCRMPBX(c *gin.Context) bool {
	cfg := h.baseConf.CRMNative
	if (!cfg.Enabled && !cfg.InboundEnabled) || c.Param("function-path") != crmPBXPath {
		return false
	}
	// A shared function slug is not tenant identity. Leave every non-target or
	// missing tuple on the original proxy, before touching its body or auth.
	if c.Query("project-id") != crmReviewProject || c.GetHeader("Environment-Id") != crmReviewEnvironment {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, crmPBXTranscriptRequestLimit+1))
	if err != nil || len(body) > crmPBXTranscriptRequestLimit {
		denyCRMPBX(c)
		return true
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var probe struct {
		Data struct {
			Method string `json:"method"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &probe) != nil {
		denyCRMPBX(c)
		return true
	}
	// These legacy methods are protected by the target PBX consumer too. They
	// need the original Bearer on the direct bridge rather than frozen proxy.
	if !strings.HasPrefix(probe.Data.Method, "crm_") && !(cfg.Enabled && crmPBXProtectedLegacyMethod(probe.Data.Method)) {
		return false
	}
	if probe.Data.Method != "pbx_save_transcript" && len(body) > crmPBXRequestLimit {
		denyCRMPBX(c)
		return true
	}
	c.Header("Cache-Control", "no-store")
	query, parts := c.Request.URL.Query(), strings.Fields(c.GetHeader("Authorization"))
	if c.Request.URL.Path != crmPBXRoute || c.Request.Method != http.MethodPost || !crmPBXMethod(probe.Data.Method) || len(query) != 1 || len(query["project-id"]) != 1 || query.Get("project-id") != crmReviewProject || len(c.Request.Header.Values("Environment-Id")) != 1 || c.GetHeader("Environment-Id") != crmReviewEnvironment || len(c.Request.Header.Values("Authorization")) != 1 || len(parts) != 2 || parts[0] != "Bearer" || len(parts[1]) > 8192 || c.GetHeader("X-API-KEY") != "" || cfg.Project != crmReviewProject || cfg.Environment != crmReviewEnvironment || cfg.ResourceEnvironment != crmReviewResourceEnvironment {
		denyCRMPBX(c)
		return true
	}
	var request models.CRMPBXRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(&struct{}{}) != io.EOF || request.Data.Method != probe.Data.Method {
		denyCRMPBX(c)
		return true
	}
	if !cfg.Enabled && request.Data.Method != crmPBXIdentityMethod && request.Data.Method != "crm_pbx_history_wake" && request.Data.Method != "crm_missed_calls" && request.Data.Method != "crm_deals_page" {
		denyCRMPBX(c)
		return true
	}
	if request.Data.Method == crmPBXIdentityMethod && (len(request.Data.Object) != 0 || len(c.Request.Header.Values("X-CRM-PBX-App-Context")) != 1) {
		denyCRMPBX(c)
		return true
	}
	if request.Data.Method != crmPBXIdentityMethod && len(c.Request.Header.Values("X-CRM-PBX-App-Context")) != 0 {
		denyCRMPBX(c)
		return true
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	actor, err := h.crmPBXSession(ctx, parts[1])
	if err != nil || !crmPBXActor(actor) {
		denyCRMPBX(c)
		return true
	}
	resource, err := h.crmPBXResource(ctx)
	if err != nil {
		denyCRMPBX(c)
		return true
	}
	app, err := h.crmPBXApp(ctx)
	if err != nil {
		denyCRMPBX(c)
		return true
	}
	if request.Data.Method == crmPBXIdentityMethod {
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-CRM-PBX-App-Context")), []byte(app)) != 1 {
			denyCRMPBX(c)
			return true
		}
		c.JSON(http.StatusOK, gin.H{"status": "OK", "data": models.CRMPBXIdentity{Project: crmReviewProject, Environment: crmReviewEnvironment, User: actor.GetUserId(), AuthUser: actor.GetUserIdAuth(), Role: actor.GetRoleId(), ClientType: actor.GetClientTypeId(), AppVerified: true}})
		return true
	}
	if err = h.crmPBXFunction(ctx, actor.GetUserId(), resource); err != nil {
		denyCRMPBX(c)
		return true
	}
	data := map[string]any{"method": request.Data.Method, "object_data": request.Data.Object, "user_id": actor.GetUserId(), "project_id": crmReviewProject, "environment_id": crmReviewEnvironment, "app_id": app}
	response, err := forwardCRMPBX(ctx, h.baseConf.KnativeBaseUrl, data, c.GetHeader("Authorization"), app)
	if err != nil {
		denyCRMPBX(c)
		return true
	}
	c.Data(http.StatusOK, "application/json", response)
	return true
}

func (h *HandlerV2) crmPBXSession(ctx context.Context, token string) (*as.V2HasAccessUserRes, error) {
	service, conn, err := h.authService.Session(ctx)
	if err != nil {
		return nil, errors.New("record unavailable")
	}
	defer conn.Close()
	return service.V2HasAccessUser(ctx, &as.V2HasAccessUserReq{AccessToken: token, Path: crmPBXRoute, Method: http.MethodPost, ProjectId: crmReviewProject, EnvironmentId: crmReviewEnvironment})
}

// The configured record must already exist. No first-key fallback, creation,
// persistent app value or generic client grant is introduced.
func (h *HandlerV2) crmPBXApp(ctx context.Context) (string, error) {
	id := h.baseConf.CRMNative.ServerScopeID
	if _, err := uuid.Parse(id); err != nil || len(id) != 36 {
		return "", errors.New("record unavailable")
	}
	key, err := h.authService.ApiKey().Get(ctx, &as.GetReq{Id: id})
	if err != nil || key == nil || key.GetId() != id || key.GetProjectId() != crmReviewProject || key.GetEnvironmentId() != crmReviewEnvironment || key.GetDisable() || strings.ToUpper(key.GetStatus()) != "ACTIVE" || key.GetAppId() == "" || len(key.GetAppId()) > 8192 {
		return "", errors.New("record unavailable")
	}
	return key.GetAppId(), nil
}

func (h *HandlerV2) crmPBXResource(ctx context.Context) (*pb.ServiceResourceModel, error) {
	resource, err := h.companyServices.ServiceResource().GetSingle(ctx, &pb.GetSingleServiceResourceReq{ProjectId: crmReviewProject, EnvironmentId: crmReviewEnvironment, ServiceType: pb.ServiceType_BUILDER_SERVICE})
	if err != nil || resource == nil || resource.GetProjectId() != crmReviewProject || resource.GetEnvironmentId() != crmReviewEnvironment || resource.GetResourceId() != crmReviewResource || resource.GetResourceEnvironmentId() != crmReviewResourceEnvironment || resource.GetResourceType() != pb.ResourceType_POSTGRESQL {
		return nil, errors.New("record unavailable")
	}
	return resource, nil
}

func (h *HandlerV2) crmPBXFunction(ctx context.Context, user string, resource *pb.ServiceResourceModel) error {
	service, err := h.GetProjectSrvc(ctx, crmReviewProject, resource.GetNodeType())
	if err != nil {
		return errors.New("record unavailable")
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	for key, value := range map[string]string{"crm-project": crmReviewProject, "crm-environment": crmReviewEnvironment, "crm-resource": crmReviewResourceEnvironment, "crm-user": user, "crm-kind": "user", "crm-purpose": "invoke-professional-pbx"} {
		md.Set(key, value)
	}
	function, err := service.GoObjectBuilderService().Function().GetSingle(metadata.NewOutgoingContext(ctx, md), &nb.FunctionPrimaryKey{ProjectId: crmReviewResourceEnvironment, Path: crmPBXPath})
	if err != nil || function == nil || function.GetId() != crmPBXFunctionID || function.GetPath() != crmPBXPath || function.GetType() != "KNATIVE" || function.GetEnvironmentId() != crmReviewEnvironment || function.GetProjectId() != crmReviewResourceEnvironment {
		return errors.New("record unavailable")
	}
	return nil
}

func forwardCRMPBX(ctx context.Context, host string, data map[string]any, bearer, app string) ([]byte, error) {
	deny := errors.New("record unavailable")
	parts := strings.Fields(bearer)
	if host == "" || strings.ContainsAny(host, "/?#@\\") || len(parts) != 2 || parts[0] != "Bearer" {
		return nil, deny
	}
	bearer = "Bearer " + parts[1]
	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return nil, deny
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+crmPBXPath+"."+host, bytes.NewReader(body))
	if err != nil {
		return nil, deny
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer)
	req.Header.Set("Environment-Id", crmReviewEnvironment)
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect denied") }}
	response, err := client.Do(req)
	if err != nil {
		return nil, deny
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, crmPBXResponseLimit+1))
	var result map[string]any
	if err != nil || len(raw) > crmPBXResponseLimit || response.StatusCode != http.StatusOK || json.Unmarshal(raw, &result) != nil || (result["status"] != "success" && result["status"] != "OK") || !crmPBXSafeResponse(result, []string{app, parts[1]}, 0) {
		return nil, deny
	}
	return raw, nil
}

// Inspect decoded strings as well as keys, so JSON escaping cannot reflect an
// internal app context or original session token back to the browser.
func crmPBXSafeResponse(value any, secrets []string, depth int) bool {
	if depth > 128 {
		return false
	}
	switch v := value.(type) {
	case string:
		for _, secret := range secrets {
			if secret != "" && strings.Contains(v, secret) {
				return false
			}
		}
	case map[string]any:
		for key, item := range v {
			if !crmPBXSafeResponse(key, secrets, depth+1) || !crmPBXSafeResponse(item, secrets, depth+1) {
				return false
			}
		}
	case []any:
		for _, item := range v {
			if !crmPBXSafeResponse(item, secrets, depth+1) {
				return false
			}
		}
	}
	return true
}
