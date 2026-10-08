package v2

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/structpb"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
)

// Reject key/tenant overrides before the existing fresh session middleware runs.
func (h *HandlerV2) CRMNativeReviewGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if !h.baseConf.CRMNativeReviewEnabled {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		parts := strings.Fields(c.GetHeader("Authorization"))
		query := c.Request.URL.Query()
		if len(c.Request.Header.Values("Authorization")) != 1 || len(parts) != 2 || parts[0] != "Bearer" || len(parts[1]) > 8192 || c.GetHeader("X-API-KEY") != "" || len(query) != 1 || len(query["project-id"]) != 1 || query.Get("project-id") != crmReviewProject || len(c.Request.Header.Values("Environment-Id")) != 1 || c.GetHeader("Environment-Id") != crmReviewEnvironment {
			denyCRMNativeReview(c)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func denyCRMNativeReview(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "error", "data": gin.H{"message": "native review unavailable"}})
}

func crmNativeReviewActor(c *gin.Context) bool {
	value, exists := c.Get("Auth")
	actor, ok := value.(*as.V2HasAccessUserRes)
	if !exists || !ok || actor == nil || actor.GetId() == "" || actor.GetProjectId() != crmReviewProject || actor.GetEnvId() != crmReviewEnvironment || c.GetString("project_id") != crmReviewProject || c.GetString("environment_id") != crmReviewEnvironment || actor.GetRoleId() != crmReviewRole || actor.GetClientTypeId() != crmReviewClientType {
		return false
	}
	switch actor.GetUserIdAuth() {
	case "5f73ff09-69a9-458d-8868-5009e6c8291c", "eeae89f7-c111-4fb1-bea0-02c38586ceb1", "5e7f763e-120b-4b9a-a451-04775cee55a7":
		return true
	}
	return false
}

// The caller supplies no query, actor or SDK context. Auth came from middleware.
func (h *HandlerV2) CRMNativeReview(c *gin.Context) {
	if !h.baseConf.CRMNativeReviewEnabled || !crmNativeReviewActor(c) {
		denyCRMNativeReview(c)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1025))
	var object map[string]any
	if err != nil || len(body) > 1024 || (len(strings.TrimSpace(string(body))) > 0 && (json.Unmarshal(body, &object) != nil || object == nil || len(object) != 0)) {
		denyCRMNativeReview(c)
		return
	}
	ctx := c.Request.Context()
	resource, err := h.companyServices.ServiceResource().GetSingle(ctx, &pb.GetSingleServiceResourceReq{ProjectId: crmReviewProject, EnvironmentId: crmReviewEnvironment, ServiceType: pb.ServiceType_BUILDER_SERVICE})
	if err != nil || resource == nil || resource.GetProjectId() != crmReviewProject || resource.GetEnvironmentId() != crmReviewEnvironment || resource.GetResourceId() != crmReviewResource || resource.GetResourceEnvironmentId() != crmReviewResourceEnvironment || resource.GetResourceType() != pb.ResourceType_POSTGRESQL {
		denyCRMNativeReview(c)
		return
	}
	service, err := h.GetProjectSrvc(ctx, crmReviewProject, resource.GetNodeType())
	if err != nil {
		denyCRMNativeReview(c)
		return
	}
	data, _ := structpb.NewStruct(map[string]any{"operation": "SELECT", "table": "(SELECT 1) AS native_review", "columns": []any{crmReviewSQL}, "limit": 1})
	response, err := service.GoObjectBuilderService().ObjectBuilder().GetListAggregation(ctx, &nb.CommonMessage{ProjectId: crmReviewResourceEnvironment, TableSlug: "deals", CompanyProjectId: crmReviewProject, EnvId: crmReviewEnvironment, Data: data})
	if err != nil || response == nil || response.GetData() == nil || response.GetBlockedBuilder() || response.GetBlockedLoginTable() {
		denyCRMNativeReview(c)
		return
	}
	report, ok := crmNativeReviewReport(response.GetData().AsMap())
	if !ok {
		denyCRMNativeReview(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "OK", "data": report})
}

func crmNativeReviewReport(data map[string]any) (map[string]any, bool) {
	rows, ok := data["data"].([]any)
	if !ok || len(rows) != 1 {
		return nil, false
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		return nil, false
	}
	raw, ok := row["result"].(map[string]any)
	if !ok || raw["deals_table_id"] != crmReviewDealsTable || raw["contacts_table_id"] != crmReviewContactsTable {
		return nil, false
	}
	report := map[string]any{"deals_table_id": crmReviewDealsTable, "contacts_table_id": crmReviewContactsTable, "safe_to_activate": false, "privilege_subject": "builder_database_executor"}
	for _, key := range []string{"native_schema", "executor_role"} {
		value, ok := raw[key].(string)
		if !ok || value == "" || len(value) > 128 {
			return nil, false
		}
		report[key] = value
	}
	for _, key := range []string{"executor_bypasses_rls", "can_insert_deals", "can_update_deals", "can_insert_contacts"} {
		value, ok := raw[key].(bool)
		if !ok {
			return nil, false
		}
		report[key] = value
	}
	columns, ok := raw["columns"].([]any)
	if !ok || len(columns) == 0 || len(columns) > 256 {
		return nil, false
	}
	safeColumns := make([]map[string]any, 0, len(columns))
	for _, value := range columns {
		column, ok := value.(map[string]any)
		if !ok || (column["table"] != "deals" && column["table"] != "contacts") {
			return nil, false
		}
		safe := map[string]any{"table": column["table"]}
		for _, key := range []string{"column", "type"} {
			text, ok := column[key].(string)
			if !ok || text == "" || len(text) > 256 {
				return nil, false
			}
			safe[key] = text
		}
		for _, key := range []string{"nullable", "default_present"} {
			flag, ok := column[key].(bool)
			if !ok {
				return nil, false
			}
			safe[key] = flag
		}
		safeColumns = append(safeColumns, safe)
	}
	counts, ok := raw["hook_counts"].(map[string]any)
	if !ok {
		return nil, false
	}
	safeCounts, hooksClear := map[string]any{}, true
	for _, key := range []string{"deals_create", "deals_update", "contacts_create", "contacts_update"} {
		count, ok := counts[key].(float64)
		if !ok || math.IsNaN(count) || math.IsInf(count, 0) || count < 0 || count > 1e6 || count != math.Trunc(count) {
			return nil, false
		}
		safeCounts[key] = count
		hooksClear = hooksClear && count == 0
	}
	report["columns"], report["hook_counts"], report["hooks_clear"] = safeColumns, safeCounts, hooksClear
	return report, true
}
