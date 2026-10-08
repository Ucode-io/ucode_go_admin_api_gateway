package crmguard

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
)

func executableFixtureConfig() config.CRMNativeConfig {
	return config.CRMNativeConfig{Enabled: true, Project: "11111111-1111-4111-8111-111111111111", Environment: "22222222-2222-4222-8222-222222222222", ResourceEnvironment: "33333333-3333-4333-8333-333333333333"}
}

func TestExecutableBoundaryClosesBeforeKeysCacheRPCAndTransport(t *testing.T) {
	cfg := executableFixtureConfig()
	aliases := []string{"/v1/invoke_function/other", "/v2/invoke_function/other", "/v1/knative/other", "/v1/knative/other/without-auth", "/v1/knative/other/without-data", "/v1/knative/other/proxy/noauth", "/v1/functions/other/run", "/v2/functions/other/run", "/v2/functions/other/invoke", "/api/query", "/x-api/query", "/v1/crm-ai/chat", "/v2/ai-builder/messages"}
	for _, path := range aliases {
		for _, source := range []string{"query", "underscore", "headers", "body"} {
			t.Run(path+"/"+source, func(t *testing.T) {
				keys, cachedKeys, rpc, transport := 0, 0, 0, 0
				r := gin.New()
				r.Use(ExecutableBoundary(cfg))
				r.POST(path, func(c *gin.Context) { keys++; cachedKeys++; rpc++; transport++; c.Status(200) })
				url, body := path, "{}"
				if source == "query" {
					url += "?project-id=" + cfg.Project + "&environment-id=" + cfg.Environment
				}
				if source == "underscore" {
					url += "?project_id=" + cfg.Project + "&environment_id=" + cfg.Environment
				}
				if source == "body" {
					body = `{"data":{"project_id":"` + cfg.Project + `","environment_id":"` + cfg.Environment + `"}}`
				}
				req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
				if source == "headers" {
					req.Header.Set("Project-Id", cfg.Project)
					req.Header.Set("Environment-Id", cfg.Environment)
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if w.Code != 403 || keys+cachedKeys+rpc+transport != 0 {
					t.Fatalf("status=%d reached=%d", w.Code, keys+cachedKeys+rpc+transport)
				}
			})
		}
	}
}

func TestExecutableBoundaryDoesNotInferTenantFromFunctionName(t *testing.T) {
	cfg := executableFixtureConfig()
	for _, path := range []string{"/v1/knative/professional-crm-pbx-integration-call/without-data", canonicalPBXInvocation + "/suffix", "/v1/functions/2b0f8e70-06b8-4fa9-98a4-aaea2a22c348/run", canonicalPBXInvocation} {
		r := gin.New()
		reached := 0
		r.Use(ExecutableBoundary(cfg))
		r.POST(path, func(c *gin.Context) { reached++; c.Status(200) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}")))
		if w.Code != 200 || reached != 1 {
			t.Fatalf("missing tuple/function name alone denied: %s", path)
		}
	}
}

func TestExecutableBoundaryPreservesOffOtherProjectAndInput(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		cfg := executableFixtureConfig()
		cfg.Enabled = !disabled
		body := `{"data":{"project_id":"66666666-6666-4666-8666-666666666666","environment_id":"55555555-5555-4555-8555-555555555555","object_data":{"value":"unchanged"}}}`
		path := "/v1/knative/other/without-data"
		if disabled {
			path = "/v1/knative/professional-crm-pbx-integration-call/without-data"
		}
		r := gin.New()
		reached := 0
		r.Use(ExecutableBoundary(cfg))
		r.POST(path, func(c *gin.Context) {
			reached++
			actual, err := io.ReadAll(c.Request.Body)
			if err != nil || string(actual) != body {
				t.Fatal("input changed")
			}
			c.Status(200)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		if w.Code != 200 || reached != 1 {
			t.Fatalf("original other/off path changed: %d", w.Code)
		}
	}
}

func TestPostAuthExecutableTargetCannotEscapeOrUseServerKey(t *testing.T) {
	for _, inboundOnly := range []bool{false, true} {
		for _, identity := range []string{"target-session-spoofed-query", "wrong-key-target-selection", "approved-server-key"} {
			cfg := executableFixtureConfig()
			cfg.Enabled = !inboundOnly
			cfg.InboundEnabled = inboundOnly
			cfg.ServerScopeID = "44444444-4444-4444-8444-444444444444"
			r := gin.New()
			reached := 0
			r.Use(func(c *gin.Context) {
				if identity == "target-session-spoofed-query" {
					c.Set("project_id", "66666666-6666-4666-8666-666666666666")
					c.Set("environment_id", cfg.Environment)
					c.Set("Auth", &as.V2HasAccessUserRes{ProjectId: cfg.Project, EnvId: cfg.Environment, UserId: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"})
				} else {
					c.Set("project_id", cfg.Project)
					c.Set("environment_id", cfg.Environment)
					keyProject, key := "66666666-6666-4666-8666-666666666666", "not-approved"
					if identity == "approved-server-key" {
						keyProject, key = cfg.Project, cfg.ServerScopeID
					}
					c.Set("auth", models.AuthData{Type: "API-KEY", Data: map[string]any{"project_id": keyProject, "environment_id": cfg.Environment, "id": key}})
				}
			})
			r.Use(Middleware(cfg))
			r.POST("/v2/invoke_function/other", func(c *gin.Context) { reached++; c.Status(200) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v2/invoke_function/other?project-id=other", nil))
			if w.Code != 403 || reached != 0 {
				t.Fatalf("%s inbound=%v status=%d reached=%d", identity, inboundOnly, w.Code, reached)
			}
		}
	}
}

func TestExecutableBoundaryPreservesAllNonTargetContexts(t *testing.T) {
	cfg := executableFixtureConfig()
	otherProject, otherEnvironment := "66666666-6666-4666-8666-666666666666", "55555555-5555-4555-8555-555555555555"
	for _, tc := range []struct {
		name, project, environment, body string
		disabled                         bool
	}{
		{name: "other project same environment", project: otherProject, environment: cfg.Environment},
		{name: "same project other environment", project: cfg.Project, environment: otherEnvironment},
		{name: "missing context"},
		{name: "wrong context", project: otherProject, environment: otherEnvironment},
		{name: "off target context", project: cfg.Project, environment: cfg.Environment, disabled: true},
		{name: "malformed unclassified body", body: "{"},
		{name: "oversized unclassified body", body: `{"data":{"padding":"` + strings.Repeat("x", executableBodyLimit) + `","project_id":"` + cfg.Project + `","environment_id":"` + cfg.Environment + `"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := cfg
			local.Enabled = !tc.disabled
			body := tc.body
			if body == "" {
				body = `{"data":{"project_id":"` + tc.project + `","environment_id":"` + tc.environment + `"}}`
			}
			path := "/v1/knative/professional-crm-pbx-integration-call/without-data"
			r := gin.New()
			keys, cache, transport := 0, 0, 0
			r.Use(ExecutableBoundary(local))
			r.Use(func(c *gin.Context) {
				c.Set("project_id", tc.project)
				c.Set("environment_id", tc.environment)
				c.Set("auth", models.AuthData{Type: "API-KEY", Data: map[string]any{"project_id": tc.project, "environment_id": tc.environment, "id": "synthetic-existing-key"}})
				c.Next()
			})
			r.Use(Middleware(local))
			r.POST(path, func(c *gin.Context) {
				keys++
				cache++
				transport++
				actual, err := io.ReadAll(c.Request.Body)
				if err != nil || string(actual) != body {
					t.Fatal("body changed")
				}
				if c.Request.Header.Get("Authorization") != "API-KEY" || c.Request.Header.Get("X-API-KEY") != "synthetic-existing-key" || c.Request.Header.Get("Environment-Id") != tc.environment {
					t.Fatal("auth headers changed")
				}
				c.Header("Cache-Control", "existing-policy")
				c.String(202, "original-response")
			})
			req := httptest.NewRequest(http.MethodPost, path+"?project-id="+tc.project, strings.NewReader(body))
			req.Header.Set("Authorization", "API-KEY")
			req.Header.Set("X-API-KEY", "synthetic-existing-key")
			req.Header.Set("Environment-Id", tc.environment)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != 202 || w.Body.String() != "original-response" || w.Header().Get("Cache-Control") != "existing-policy" || keys != 1 || cache != 1 || transport != 1 {
				t.Fatalf("old behavior changed: status=%d keys=%d cache=%d transport=%d", w.Code, keys, cache, transport)
			}
		})
	}
}

func TestExecutableBoundaryExplicitSelectorsGovernBody(t *testing.T) {
	cfg := executableFixtureConfig()
	otherProject, otherEnvironment := "66666666-6666-4666-8666-666666666666", "55555555-5555-4555-8555-555555555555"
	targetBody := `{"data":{"project_id":"` + cfg.Project + `","environment_id":"` + cfg.Environment + `"}}`
	otherBody := `{"data":{"project_id":"` + otherProject + `","environment_id":"` + cfg.Environment + `"}}`
	for _, tc := range []struct {
		name, query, environment, body string
		status                         int
	}{
		{name: "explicit other project dominates target JSON", query: "project-id=" + otherProject, environment: cfg.Environment, body: targetBody, status: 202},
		{name: "explicit other environment dominates target JSON", query: "project-id=" + cfg.Project, environment: otherEnvironment, body: targetBody, status: 202},
		{name: "explicit target dominates other JSON", query: "project-id=" + cfg.Project, environment: cfg.Environment, body: otherBody, status: 403},
		{name: "partial project suppresses JSON inference", query: "project-id=" + cfg.Project, body: targetBody, status: 202},
		{name: "partial environment suppresses JSON inference", environment: cfg.Environment, body: targetBody, status: 202},
		{name: "empty selector suppresses JSON inference", query: "project-id=", environment: cfg.Environment, body: targetBody, status: 202},
		{name: "duplicate project selectors", query: "project-id=" + cfg.Project + "&project-id=" + otherProject, environment: cfg.Environment, body: targetBody, status: 202},
		{name: "duplicate identical project selectors", query: "project-id=" + cfg.Project + "&project-id=" + cfg.Project, environment: cfg.Environment, body: targetBody, status: 202},
		{name: "conflicting standard underscore selectors", query: "project-id=" + cfg.Project + "&project_id=" + otherProject, environment: cfg.Environment, body: targetBody, status: 202},
		{name: "duplicate query header environment", query: "project-id=" + cfg.Project + "&environment-id=" + cfg.Environment, environment: cfg.Environment, body: targetBody, status: 202},
		{name: "malformed explicit selector", query: "project-id=%ZZ&environment-id=%ZZ", body: targetBody, status: 202},
		{name: "body-only target remains denied", body: targetBody, status: 403},
		{name: "duplicate body selector", body: `{"data":{"project_id":"` + otherProject + `","project_id":"` + cfg.Project + `","environment_id":"` + cfg.Environment + `"}}`, status: 202},
		{name: "duplicate case-insensitive body selector", body: `{"data":{"PROJECT_ID":"` + otherProject + `","project_id":"` + cfg.Project + `","environment_id":"` + cfg.Environment + `"}}`, status: 202},
		{name: "explicit root other suppresses nested target", body: `{"project_id":"` + otherProject + `","environment_id":"` + cfg.Environment + `","data":{"project_id":"` + cfg.Project + `","environment_id":"` + cfg.Environment + `"}}`, status: 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			reached := 0
			r.Use(ExecutableBoundary(cfg))
			path := "/v1/knative/professional-crm-pbx-integration-call/without-data"
			r.POST(path, func(c *gin.Context) {
				reached++
				body, err := io.ReadAll(c.Request.Body)
				if err != nil || string(body) != tc.body || c.Request.URL.RawQuery != tc.query || c.Request.Header.Get("Environment-Id") != tc.environment || c.Request.Header.Get("Authorization") != "API-KEY" || c.Request.Header.Get("X-API-KEY") != "synthetic-existing-key" {
					t.Fatal("original request changed")
				}
				c.Header("Cache-Control", "original-cache")
				c.String(202, "original-response")
			})
			url := path
			if tc.query != "" {
				url += "?" + tc.query
			}
			req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(tc.body))
			if tc.environment != "" {
				req.Header.Set("Environment-Id", tc.environment)
			}
			req.Header.Set("Authorization", "API-KEY")
			req.Header.Set("X-API-KEY", "synthetic-existing-key")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d wanted=%d", w.Code, tc.status)
			}
			if tc.status == 202 && (reached != 1 || w.Body.String() != "original-response" || w.Header().Get("Cache-Control") != "original-cache") {
				t.Fatal("original handler/cache response changed")
			}
			if tc.status == 403 && reached != 0 {
				t.Fatal("protected handler reached")
			}
		})
	}
}
