package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	grpcpool "github.com/processout/grpc-go-pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "ucode/ucode_go_api_gateway/api/handlers/v1"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
)

type reviewResource struct {
	pb.MicroserviceResourceClient
	value *pb.ServiceResourceModel
	calls *int
}

func (f reviewResource) GetSingle(_ context.Context, req *pb.GetSingleServiceResourceReq, _ ...grpc.CallOption) (*pb.ServiceResourceModel, error) {
	*f.calls++
	if req.ProjectId != crmReviewProject || req.EnvironmentId != crmReviewEnvironment || req.ServiceType != pb.ServiceType_BUILDER_SERVICE {
		return nil, errors.New("unexpected resource lookup")
	}
	return f.value, nil
}

type reviewCompany struct {
	services.CompanyServiceI
	resource pb.MicroserviceResourceClient
}

func (f reviewCompany) ServiceResource() pb.MicroserviceResourceClient { return f.resource }

type reviewNodes struct {
	services.ServiceNodesI
	service services.ServiceManagerI
}

func (f reviewNodes) Get(string) (services.ServiceManagerI, error) { return f.service, nil }

type reviewServices struct {
	services.ServiceManagerI
	builder services.GoBuilderServiceI
}

func (f reviewServices) GoObjectBuilderService() services.GoBuilderServiceI { return f.builder }

type reviewBuilder struct {
	services.GoBuilderServiceI
	object nb.ObjectBuilderServiceClient
}

func (f reviewBuilder) ObjectBuilder() nb.ObjectBuilderServiceClient { return f.object }

type reviewObject struct {
	nb.ObjectBuilderServiceClient
	run func(context.Context, *nb.CommonMessage) (*nb.CommonMessage, error)
}

func (f reviewObject) GetListAggregation(ctx context.Context, req *nb.CommonMessage, _ ...grpc.CallOption) (*nb.CommonMessage, error) {
	return f.run(ctx, req)
}

type reviewSession struct {
	as.SessionServiceClient
	run func(context.Context, *as.V2HasAccessUserReq) (*as.V2HasAccessUserRes, error)
}

func (f reviewSession) V2HasAccessUser(ctx context.Context, req *as.V2HasAccessUserReq, _ ...grpc.CallOption) (*as.V2HasAccessUserRes, error) {
	return f.run(ctx, req)
}

type reviewAuth struct {
	services.AuthServiceManagerI
	pool    *grpcpool.Pool
	session as.SessionServiceClient
}

func (f reviewAuth) Session(ctx context.Context) (as.SessionServiceClient, *grpcpool.ClientConn, error) {
	conn, err := f.pool.Get(ctx)
	return f.session, conn, err
}

func reviewFixtureReport() map[string]any {
	return map[string]any{
		"deals_table_id": crmReviewDealsTable, "contacts_table_id": crmReviewContactsTable, "native_schema": "public", "executor_role": "fixture_builder",
		"executor_bypasses_rls": false, "can_insert_deals": true, "can_update_deals": true, "can_insert_contacts": true,
		"login_compatibility":  reviewFixtureLoginMetadata(),
		"native_catalog_count": 4.,
		"native_catalog": []any{
			map[string]any{"id": crmReviewDealsTable, "slug": "deals", "is_login_table": false, "is_system": false},
			map[string]any{"id": crmReviewContactsTable, "slug": "contacts", "is_login_table": false, "is_system": false},
			map[string]any{"id": "11111111-1111-4111-8111-111111111111", "slug": "users", "is_login_table": true, "is_system": false},
			map[string]any{"id": "22222222-2222-4222-8222-222222222222", "slug": "hidden_system", "is_login_table": true, "is_system": true, "api_key": "secret-fixture", "default": "secret-fixture", "url": "https://secret-fixture.invalid"},
		},
		"columns":     []any{map[string]any{"table": "deals", "column": "pbx_branch", "type": "text[]", "nullable": true, "default_present": false, "default_expression": "secret-fixture", "attributes": "secret-fixture"}},
		"hook_counts": map[string]any{"deals_create": 0., "deals_update": 0., "contacts_create": 0., "contacts_update": 0.}, "token": "secret-fixture",
	}
}

func TestCRMNativeReviewCompleteCatalog(t *testing.T) {
	raw := reviewFixtureReport()
	report, ok := crmNativeReviewReport(map[string]any{"data": []any{map[string]any{"result": raw}}})
	if !ok || report["native_catalog_count"] != 4 {
		t.Fatal("complete catalog rejected")
	}
	rows := report["native_catalog"].([]map[string]any)
	if len(rows) != 4 || rows[3]["slug"] != "hidden_system" || rows[3]["is_system"] != true || rows[3]["is_login_table"] != true {
		t.Fatal("system/login tables omitted from catalog")
	}
	for _, row := range rows {
		if len(row) != 4 {
			t.Fatal("catalog contains unapproved fields")
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil || strings.Contains(string(encoded), "secret-fixture") || strings.Contains(string(encoded), "https://") {
		t.Fatal("unapproved catalog metadata leaked")
	}
	if !strings.Contains(crmReviewSQL, "'native_catalog_count',(SELECT count(*) FROM public.\"table\")") || !strings.Contains(crmReviewSQL, "FROM public.\"table\" ORDER BY slug,id LIMIT 1025") {
		t.Fatal("catalog is not the bounded unfiltered public catalog")
	}
}

func TestCRMNativeReviewCatalogRejectsMalformed(t *testing.T) {
	for _, name := range []string{"missing-catalog", "missing-count", "string-count", "fractional-count", "negative-count", "nan-count", "infinite-count", "count-mismatch", "oversize-count", "malformed-row", "invalid-id", "nil-id", "duplicate-id", "duplicate-slug", "invalid-slug", "long-slug", "invalid-login", "invalid-system", "missing-flag", "wrong-core-id", "missing-core-table"} {
		t.Run(name, func(t *testing.T) {
			raw := reviewFixtureReport()
			rows := raw["native_catalog"].([]any)
			row := rows[3].(map[string]any)
			switch name {
			case "missing-catalog":
				delete(raw, "native_catalog")
			case "missing-count":
				delete(raw, "native_catalog_count")
			case "string-count":
				raw["native_catalog_count"] = "4"
			case "fractional-count":
				raw["native_catalog_count"] = 4.5
			case "negative-count":
				raw["native_catalog_count"] = -1.
			case "nan-count":
				raw["native_catalog_count"] = math.NaN()
			case "infinite-count":
				raw["native_catalog_count"] = math.Inf(1)
			case "count-mismatch":
				raw["native_catalog_count"] = 3.
			case "oversize-count":
				raw["native_catalog_count"] = float64(crmReviewCatalogLimit + 1)
				raw["native_catalog"] = make([]any, crmReviewCatalogLimit+1)
			case "malformed-row":
				rows[3] = "invalid"
			case "invalid-id":
				row["id"] = "not-a-uuid"
			case "nil-id":
				row["id"] = "00000000-0000-0000-0000-000000000000"
			case "duplicate-id":
				row["id"] = strings.ToUpper(crmReviewDealsTable)
			case "duplicate-slug":
				row["slug"] = "deals"
			case "invalid-slug":
				row["slug"] = "https://injected.invalid"
			case "long-slug":
				row["slug"] = strings.Repeat("x", 64)
			case "invalid-login":
				row["is_login_table"] = "true"
			case "invalid-system":
				row["is_system"] = 1.
			case "missing-flag":
				delete(row, "is_system")
			case "wrong-core-id":
				rows[0].(map[string]any)["id"] = "33333333-3333-4333-8333-333333333333"
			case "missing-core-table":
				rows[0].(map[string]any)["slug"] = "other"
			}
			if _, ok := crmNativeReviewReport(map[string]any{"data": []any{map[string]any{"result": raw}}}); ok {
				t.Fatal("malformed catalog accepted")
			}
		})
	}
}

func TestCRMNativeReviewCatalogLimit(t *testing.T) {
	raw := reviewFixtureReport()
	rows := raw["native_catalog"].([]any)
	for i := len(rows); i < crmReviewCatalogLimit; i++ {
		rows = append(rows, map[string]any{"id": fmt.Sprintf("33333333-3333-4333-8333-%012x", i), "slug": fmt.Sprintf("table_%d", i), "is_login_table": false, "is_system": true})
	}
	raw["native_catalog"], raw["native_catalog_count"] = rows, float64(len(rows))
	if _, count, ok := crmNativeReviewCatalog(raw); !ok || count != crmReviewCatalogLimit {
		t.Fatal("complete catalog at supported limit rejected")
	}
}

func TestCRMNativeReviewMountedFreshMiddleware(t *testing.T) {
	// The real middleware runs; only remote auth/company/builder service replies
	// are fixtures. No cached access RPC, API-key lookup or write RPC is implemented.
	pool, err := grpcpool.New(func() (*grpc.ClientConn, error) {
		return grpc.NewClient("passthrough:///fixture", grpc.WithTransportCredentials(insecure.NewCredentials()))
	}, 1, 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, name := range []string{"approved1", "approved2", "approved3", "disabled", "api-key", "missing-token", "duplicate-auth", "duplicate-environment", "bearer-with-key", "wrong-query-project", "duplicate-project", "wrong-header-environment", "query-override", "expired", "missing-session", "wrong-user", "wrong-auth-project", "wrong-auth-environment", "wrong-role", "wrong-client-type", "spoofed-json", "sql-override", "oversized-body", "wrong-resource", "wrong-resource-project", "wrong-resource-environment", "wrong-resource-type", "wrong-resource-db", "wrong-table", "builder-denied", "malformed-result", "missing-count", "fractional-count", "hooks-present"} {
		t.Run(name, func(t *testing.T) {
			authCalls, resourceCalls, queries := 0, 0, 0
			actor := &as.V2HasAccessUserRes{Id: "fixture-session", ProjectId: crmReviewProject, EnvId: crmReviewEnvironment, UserIdAuth: "5f73ff09-69a9-458d-8868-5009e6c8291c", RoleId: crmReviewRole, ClientTypeId: crmReviewClientType}
			switch name {
			case "approved2":
				actor.UserIdAuth = "eeae89f7-c111-4fb1-bea0-02c38586ceb1"
			case "approved3":
				actor.UserIdAuth = "5e7f763e-120b-4b9a-a451-04775cee55a7"
			case "missing-session":
				actor.Id = ""
			case "wrong-user":
				actor.UserIdAuth = "unapproved"
			case "wrong-auth-project":
				actor.ProjectId = "other"
			case "wrong-auth-environment":
				actor.EnvId = "other"
			case "wrong-role":
				actor.RoleId = "other"
			case "wrong-client-type":
				actor.ClientTypeId = "other"
			}
			resource := &pb.ServiceResourceModel{ProjectId: crmReviewProject, EnvironmentId: crmReviewEnvironment, ResourceId: crmReviewResource, ResourceEnvironmentId: crmReviewResourceEnvironment, ResourceType: pb.ResourceType_POSTGRESQL}
			switch name {
			case "wrong-resource":
				resource.ResourceId = "other"
			case "wrong-resource-project":
				resource.ProjectId = "other"
			case "wrong-resource-environment":
				resource.EnvironmentId = "other"
			case "wrong-resource-type":
				resource.ResourceType = pb.ResourceType_MONGODB
			case "wrong-resource-db":
				resource.ResourceEnvironmentId = "other"
			}
			fixture := reviewFixtureReport()
			switch name {
			case "wrong-table":
				fixture["deals_table_id"] = "other"
			case "missing-count":
				delete(fixture["hook_counts"].(map[string]any), "contacts_update")
			case "fractional-count":
				fixture["hook_counts"].(map[string]any)["deals_create"] = 0.5
			case "hooks-present":
				fixture["hook_counts"].(map[string]any)["deals_create"] = 1.
			}
			object := reviewObject{run: func(ctx context.Context, req *nb.CommonMessage) (*nb.CommonMessage, error) {
				queries++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 30*time.Second || req.ProjectId != crmReviewResourceEnvironment || req.CompanyProjectId != crmReviewProject || req.EnvId != crmReviewEnvironment || req.TableSlug != "deals" || req.IsCached {
					t.Fatal("unbounded/wrong scope")
				}
				data := req.Data.AsMap()
				if data["operation"] != "SELECT" || data["table"] != "(SELECT 1) AS native_review" || data["limit"] != 1. || len(data) != 4 || len(data["columns"].([]any)) != 1 || data["columns"].([]any)[0] != crmReviewSQL {
					t.Fatal("caller-controlled/read-write query")
				}
				if name == "builder-denied" {
					return nil, errors.New("secret-fixture")
				}
				rows := []any{map[string]any{"result": fixture}}
				if name == "malformed-result" {
					rows = nil
				}
				response, err := structpb.NewStruct(map[string]any{"data": rows})
				return &nb.CommonMessage{Data: response}, err
			}}
			cfg := config.BaseConfig{CRMNativeReviewEnabled: name != "disabled"}
			h := &HandlerV2{baseConf: cfg, companyServices: reviewCompany{resource: reviewResource{value: resource, calls: &resourceCalls}}, services: reviewNodes{service: reviewServices{builder: reviewBuilder{object: object}}}}
			auth := reviewAuth{pool: pool, session: reviewSession{run: func(ctx context.Context, req *as.V2HasAccessUserReq) (*as.V2HasAccessUserRes, error) {
				authCalls++
				if req.AccessToken != "fixture-token" || req.ProjectId != crmReviewProject || req.EnvironmentId != crmReviewEnvironment || req.Method != "POST" || req.Path != "/v1/crm-pbx/native-review" {
					t.Fatal("fresh auth scope mismatch", req.Path)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("auth not bounded")
				}
				if name == "expired" {
					return nil, status.Error(codes.InvalidArgument, "User has been expired")
				}
				return actor, nil
			}}}
			authHandler := v1.NewHandlerV1(cfg, nil, logger.NewLogger("crm-review-fixture", logger.LevelError), nil, nil, auth, nil, nil, nil, nil, nil)
			r := gin.New()
			r.POST("/v1/crm-pbx/native-review", h.CRMNativeReviewGuard(), authHandler.AuthMiddleware(cfg), h.CRMNativeReview)
			url, body := "/v1/crm-pbx/native-review?project-id="+crmReviewProject, "{}"
			if name == "wrong-query-project" {
				url = "/v1/crm-pbx/native-review?project-id=other"
			}
			if name == "duplicate-project" {
				url += "&project-id=" + crmReviewProject
			}
			if name == "query-override" {
				url += "&operation=UPDATE"
			}
			if name == "spoofed-json" {
				body = `{"user_id_auth":"5f73ff09-69a9-458d-8868-5009e6c8291c","app_id":"spoof"}`
			}
			if name == "sql-override" {
				body = `{"data":{"operation":"UPDATE","table":"deals"}}`
			}
			if name == "oversized-body" {
				body = strings.Repeat(" ", 1025)
			}
			req := httptest.NewRequest("POST", url, strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer fixture-token")
			req.Header.Set("Environment-Id", crmReviewEnvironment)
			req.Header.Set("X-CRM-User", "untrusted")
			if name == "api-key" {
				req.Header.Set("Authorization", "API-KEY")
				req.Header.Set("X-API-KEY", "fixture-key")
			}
			if name == "missing-token" {
				req.Header.Del("Authorization")
			}
			if name == "duplicate-auth" {
				req.Header.Add("Authorization", "Bearer spoof")
			}
			if name == "duplicate-environment" {
				req.Header.Add("Environment-Id", crmReviewEnvironment)
			}
			if name == "bearer-with-key" {
				req.Header.Set("X-API-KEY", "fixture-key")
			}
			if name == "wrong-header-environment" {
				req.Header.Set("Environment-Id", "other")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			allowed := strings.HasPrefix(name, "approved") || name == "hooks-present"
			if allowed {
				var response struct {
					Data map[string]any `json:"data"`
				}
				if w.Code != 200 || authCalls != 1 || resourceCalls != 1 || queries != 1 || json.Unmarshal(w.Body.Bytes(), &response) != nil || response.Data["safe_to_activate"] != false || response.Data["privilege_subject"] != "builder_database_executor" {
					t.Fatal("invalid review", w.Code, w.Body.String())
				}
				if response.Data["hooks_clear"] != (name != "hooks-present") {
					t.Fatal("hooks presence lost")
				}
			} else if w.Code < 400 {
				t.Fatal("unauthorized accepted", w.Body.String())
			}
			if strings.Contains(w.Body.String(), "secret-fixture") || strings.Contains(w.Body.String(), "fixture-token") || strings.Contains(w.Body.String(), "fixture-session") {
				t.Fatal("secret/request leaked")
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cacheable metadata")
			}
			if strings.HasPrefix(name, "wrong-auth") || name == "wrong-user" || name == "wrong-role" || name == "wrong-client-type" || name == "missing-session" || name == "expired" || name == "spoofed-json" || name == "sql-override" || name == "oversized-body" {
				if resourceCalls != 0 || queries != 0 {
					t.Fatal("invalid actor/body reached resource")
				}
			}
			if name == "disabled" || name == "api-key" || name == "missing-token" || name == "duplicate-auth" || name == "duplicate-environment" || name == "bearer-with-key" || name == "wrong-query-project" || name == "duplicate-project" || name == "wrong-header-environment" || name == "query-override" {
				if authCalls != 0 || resourceCalls != 0 || queries != 0 {
					t.Fatal("guard failed before auth")
				}
			}
			if strings.HasPrefix(name, "wrong-resource") && queries != 0 {
				t.Fatal("wrong resource reached query")
			}
		})
	}
}
