package v2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/services"
)

type crmPBXAppKeys struct {
	as.ApiKeysClient
	get func(context.Context, *as.GetReq) (*as.GetRes, error)
}

func (f crmPBXAppKeys) Get(ctx context.Context, req *as.GetReq, _ ...grpc.CallOption) (*as.GetRes, error) {
	return f.get(ctx, req)
}

type crmPBXAppAuth struct {
	services.AuthServiceManagerI
	keys as.ApiKeysClient
}

func (f crmPBXAppAuth) ApiKey() as.ApiKeysClient { return f.keys }

func TestCRMPBXExistingFunctionKeyLifecycle(t *testing.T) {
	const recordID = "4613c637-b70a-11f1-a0b5-36f82065c03b"
	for _, name := range []string{"protected-function", "get-omits-disable", "inactive", "deleted", "service-error", "nil-record", "different-record", "different-project", "different-environment", "different-role", "different-client", "public-key-name", "missing-platform", "different-platform", "missing-status", "missing-app", "oversized-app", "invalid-reference"} {
		t.Run(name, func(t *testing.T) {
			key := &as.GetRes{Id: recordID, Name: "Function", ProjectId: crmReviewProject, EnvironmentId: crmReviewEnvironment, RoleId: crmReviewRole, ClientTypeId: crmReviewClientType, Status: "ACTIVE", Disable: true, ClientPlatform: &as.ClientPlatform{Id: crmPBXAppPlatform}, AppId: "synthetic-existing-app"}
			var lookupErr error
			configuredID := recordID
			switch name {
			case "get-omits-disable":
				key.Disable = false
			case "inactive":
				key.Status = "INACTIVE"
			case "deleted", "service-error":
				key, lookupErr = nil, errors.New("synthetic lookup unavailable")
			case "nil-record":
				key = nil
			case "different-record":
				key.Id = "55555555-5555-4555-8555-555555555555"
			case "different-project":
				key.ProjectId = "other"
			case "different-environment":
				key.EnvironmentId = "other"
			case "different-role":
				key.RoleId = crmPBXOperatorRole
			case "different-client":
				key.ClientTypeId = crmPBXOperatorClient
			case "public-key-name":
				key.Name = "Browser"
			case "missing-platform":
				key.ClientPlatform = nil
			case "different-platform":
				key.ClientPlatform.Id = "other"
			case "missing-status":
				key.Status = ""
			case "missing-app":
				key.AppId = ""
			case "oversized-app":
				key.AppId = strings.Repeat("x", 8193)
			case "invalid-reference":
				configuredID = "not-a-record-id"
			}
			calls := 0
			h := &HandlerV2{baseConf: config.BaseConfig{CRMNative: config.CRMNativeConfig{ServerScopeID: configuredID}}, authService: crmPBXAppAuth{keys: crmPBXAppKeys{get: func(_ context.Context, req *as.GetReq) (*as.GetRes, error) {
				calls++
				if req.Id != recordID {
					t.Fatal("configured record selection changed")
				}
				return key, lookupErr
			}}}}
			app, err := h.crmPBXApp(context.Background())
			allowed := name == "protected-function" || name == "get-omits-disable"
			if allowed && (err != nil || app != "synthetic-existing-app" || calls != 1) {
				t.Fatal("existing protected Function context rejected")
			}
			if !allowed && (err == nil || app != "") {
				t.Fatal("invalid Function context accepted")
			}
			if name == "invalid-reference" && calls != 0 {
				t.Fatal("invalid record reference reached AuthService")
			}
		})
	}
}

type crmPBXRoundTrip func(*http.Request) (*http.Response, error)

func (f crmPBXRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCRMPBXSharedProxyKeepsOtherCRMExactly(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	for _, tc := range []struct {
		name, project, environment, authorization string
		disabled                                  bool
	}{
		{name: "other project same env", project: "66666666-6666-4666-8666-666666666666", environment: crmReviewEnvironment, authorization: "API-KEY"},
		{name: "same project other env", project: crmReviewProject, environment: "55555555-5555-4555-8555-555555555555", authorization: "Bearer existing-other-session"},
		{name: "missing context", authorization: "API-KEY"},
		{name: "scope off target", project: crmReviewProject, environment: crmReviewEnvironment, authorization: "API-KEY", disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"data":{"method":"crm_deals_page","object_data":{"value":"original"}}}`
			calls := 0
			http.DefaultTransport = crmPBXRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				actual, err := io.ReadAll(r.Body)
				if err != nil || string(actual) != body || r.Header.Get("Authorization") != tc.authorization || r.Header.Get("X-API-KEY") != "existing-other-key" || r.Header.Get("Environment-Id") != tc.environment || r.URL.Query().Get("project-id") != tc.project || r.URL.Host != "legacy-fixture.invalid" || r.URL.Path != crmPBXRoute {
					t.Fatal("old transport/body/headers changed")
				}
				return &http.Response{StatusCode: 202, Header: http.Header{"Cache-Control": []string{"legacy-cache-policy"}, "X-Legacy": []string{"original"}}, Body: io.NopCloser(strings.NewReader("legacy-response"))}, nil
			})
			// Nil auth/company/builder would panic if a non-target request touched
			// the new authorization path rather than the existing proxy.
			h := &HandlerV2{baseConf: config.BaseConfig{GoFunctionServiceHost: "http://legacy-fixture.invalid", CRMNative: config.CRMNativeConfig{Enabled: !tc.disabled, Project: crmReviewProject, Environment: crmReviewEnvironment, ResourceEnvironment: crmReviewResourceEnvironment}}}
			r := gin.New()
			r.POST("/v2/invoke_function/:function-path", h.InvokeFunctionByPath)
			req := httptest.NewRequest(http.MethodPost, crmPBXRoute+"?project-id="+tc.project, strings.NewReader(body))
			req.Header.Set("Authorization", tc.authorization)
			req.Header.Set("X-API-KEY", "existing-other-key")
			if tc.environment != "" {
				req.Header.Set("Environment-Id", tc.environment)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != 202 || w.Body.String() != "legacy-response" || w.Header().Get("Cache-Control") != "legacy-cache-policy" || w.Header().Get("X-Legacy") != "original" || calls != 1 {
				t.Fatalf("original proxy result changed: %d %s calls%d", w.Code, w.Body.String(), calls)
			}
		})
	}
}

func TestCRMPBXActorCannotCrossApprovedMappings(t *testing.T) {
	a := &as.V2HasAccessUserRes{Id: "synthetic-session", ProjectId: crmReviewProject, EnvId: crmReviewEnvironment, UserId: "76fad06c-7c1a-4967-8037-f698279c4335", UserIdAuth: "50d88de6-c51e-429a-bb6c-0a284273a3d1", RoleId: crmPBXOperatorRole, ClientTypeId: crmPBXOperatorClient}
	if !crmPBXActor(a) {
		t.Fatal("verified operator rejected")
	}
	a.UserId = "99543ad4-9db0-48b8-8da6-39e15c3efd4c"
	if crmPBXActor(a) {
		t.Fatal("different CRM actor accepted")
	}
	a.UserId = "76fad06c-7c1a-4967-8037-f698279c4335"
	a.RoleId = crmReviewRole
	if crmPBXActor(a) {
		t.Fatal("unapproved role accepted")
	}
	a.RoleId = crmPBXOperatorRole
	a.EnvId = "55555555-5555-4555-8555-555555555555"
	if crmPBXActor(a) {
		t.Fatal("other env accepted")
	}
}

func TestCRMPBXDirectResponseCannotReflectEscapedCredentials(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	for _, tc := range []struct {
		name, body, bearer string
		allowed            bool
	}{
		{name: "ordinary protected response", body: `{"status":"success","data":{"rows":[],"total":0}}`, allowed: true},
		{name: "app unicode reflection", body: `{"status":"success","data":{"value":"fixture\u002dprivate-context"}}`},
		{name: "session unicode reflection", body: `{"status":"success","data":{"rows":[{"value":"fixture\u002dsession-token"}]}}`},
		{name: "tab bearer reflection", bearer: "Bearer\tfixture-session-token", body: `{"status":"success","data":{"value":"fixture-session-token"}}`},
		{name: "multiple space bearer reflection", bearer: "Bearer    fixture-session-token", body: `{"status":"success","data":{"value":"fixture-session-token"}}`},
		{name: "escaped secret key", body: `{"status":"success","data":{"fixture\u002dprivate-context":true}}`},
		{name: "application error", body: `{"status":"error","data":{"message":"not available"}}`},
		{name: "oversized body", body: `{"status":"success","data":{"value":"` + strings.Repeat("x", crmPBXResponseLimit) + `"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = crmPBXRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != crmPBXPath+".fixture.invalid" || r.Header.Get("Authorization") != "Bearer fixture-session-token" {
					t.Fatal("fixed authenticated transport changed")
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			bearer := tc.bearer
			if bearer == "" {
				bearer = "Bearer fixture-session-token"
			}
			result, err := forwardCRMPBX(context.Background(), "fixture.invalid", map[string]any{"method": "crm_deals_page"}, bearer, "fixture-private-context")
			if tc.allowed && (err != nil || string(result) != tc.body) {
				t.Fatal("ordinary response changed")
			}
			if !tc.allowed && (err == nil || len(result) != 0) {
				t.Fatal("private response accepted")
			}
		})
	}
}

func TestCRMPBXTargetProtectedLegacyNeverUsesFrozenProxyUnsigned(t *testing.T) {
	for _, method := range []string{"pbx_get_recording", "pbx_save_stereo_recording", "pbx_save_transcript", "pbx_list_untranscribed", "pbx_debug"} {
		t.Run(method, func(t *testing.T) {
			h := &HandlerV2{baseConf: config.BaseConfig{CRMNative: config.CRMNativeConfig{Enabled: true, Project: crmReviewProject, Environment: crmReviewEnvironment, ResourceEnvironment: crmReviewResourceEnvironment}}}
			r := gin.New()
			r.POST("/v2/invoke_function/:function-path", h.InvokeFunctionByPath)
			req := httptest.NewRequest(http.MethodPost, crmPBXRoute+"?project-id="+crmReviewProject, strings.NewReader(`{"data":{"method":"`+method+`","object_data":{}}}`))
			req.Header.Set("Environment-Id", crmReviewEnvironment)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("unsigned protected legacy route escaped direct bridge")
			}
		})
	}
}
