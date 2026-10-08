package crmguard

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/storage"
)

func TestTrustedScopeReplacesSpoofedClaims(t *testing.T) {
	cfg := config.CRMNativeConfig{Enabled: true, Project: "11111111-1111-4111-8111-111111111111", Environment: "22222222-2222-4222-8222-222222222222", ResourceEnvironment: "33333333-3333-4333-8333-333333333333", ServerScopeID: "44444444-4444-4444-8444-444444444444"}
	actor := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	for _, tc := range []struct {
		name, path                                                                   string
		disabled, otherEnv, otherProject, unverified, missingActor, key, approvedKey bool
		expected                                                                     int
	}{
		{name: "verified login", path: "/v2/items/deals", expected: 200},
		{name: "unverified target header fallback", path: "/v2/items/deals", unverified: true, expected: 403},
		{name: "actor missing", path: "/v2/items/deals", missingActor: true, expected: 403},
		{name: "API key cannot impersonate operator", path: "/v2/items/deals", key: true, expected: 403},
		{name: "existing reviewed server scope", path: "/v2/items/deals", key: true, approvedKey: true, expected: 200},
		{name: "raw aggregate", path: "/v2/items/aggregation", expected: 403},
		{name: "proxy", path: "/api/query", expected: 403},
		{name: "redirect proxy", path: "/x-api/query", expected: 403},
		{name: "other environment unchanged", path: "/api/query", otherEnv: true, expected: 200},
		{name: "other project unchanged", path: "/api/query", otherProject: true, expected: 200},
		{name: "default off unchanged", path: "/api/query", disabled: true, unverified: true, expected: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := cfg
			local.Enabled = !tc.disabled
			project := cfg.Project
			if tc.otherProject {
				project = "66666666-6666-4666-8666-666666666666"
			}
			env := cfg.Environment
			if tc.otherEnv {
				env = "55555555-5555-4555-8555-555555555555"
			}
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("project_id", project)
				c.Set("environment_id", env)
				if tc.key {
					id := "not-approved"
					if tc.approvedKey {
						id = cfg.ServerScopeID
					}
					c.Set("auth", models.AuthData{Type: "API-KEY", Data: map[string]any{"project_id": project, "environment_id": env, "id": id}})
				} else if !tc.unverified {
					user := actor
					if tc.missingActor {
						user = ""
					}
					c.Set("Auth", &auth.V2HasAccessUserRes{ProjectId: project, EnvId: env, UserId: user})
				}
				c.Next()
			})
			r.Use(Middleware(local))
			r.GET(tc.path, func(c *gin.Context) {
				if (tc.otherEnv || tc.otherProject || tc.disabled) && Scoped(c.Request.Context()) {
					t.Fatal("outside scope changed")
				}
				if local.Enabled && !tc.otherEnv && !tc.otherProject {
					md, _ := metadata.FromOutgoingContext(c.Request.Context())
					kind := "user"
					if tc.approvedKey {
						kind = "server"
					}
					if len(md.Get("crm-kind")) != 1 || md.Get("crm-kind")[0] != kind || !Scoped(c.Request.Context()) {
						t.Fatal("spoofed scope retained")
					}
					if !tc.key && md.Get("crm-user")[0] != actor {
						t.Fatal("actor not pinned")
					}
					if c.Writer.Header().Get("Cache-Control") != "no-store" {
						t.Fatal("scoped response cacheable")
					}
				}
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest("GET", tc.path, nil)
			req.Header.Set("crm-user", "spoof")
			req.Header.Set("crm-kind", "server")
			req = req.WithContext(metadata.NewOutgoingContext(req.Context(), metadata.Pairs("crm-user", "spoof", "crm-kind", "server")))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.expected {
				t.Fatalf("status=%d, body=%s", w.Code, w.Body.String())
			}
		})
	}
}

type cacheProbe struct {
	storage.RedisStorageI
	reads int
}

func (p *cacheProbe) Get(context.Context, string, string, string) (string, error) {
	p.reads++
	return "old actor data", nil
}
func TestScopedCacheCannotReturnActorlessEntry(t *testing.T) {
	cfg := config.CRMNativeConfig{Enabled: true, Project: "project", ResourceEnvironment: "resource"}
	base := &cacheProbe{}
	cache := Cache(base, cfg)
	for _, key := range []string{"deals-resource", base64.StdEncoding.EncodeToString([]byte("deals-resource"))} {
		if _, err := cache.Get(context.Background(), key, cfg.Project, ""); err == nil {
			t.Fatal("actorless cache admitted")
		}
	}
	if base.reads != 0 {
		t.Fatal("protected cache read")
	}
	if _, err := cache.Get(context.Background(), "deals-other", "project", ""); err != nil || base.reads != 1 {
		t.Fatal("other environment changed")
	}
	if _, err := cache.Get(context.WithValue(context.Background(), contextKey{}, true), "any", cfg.Project, ""); err == nil {
		t.Fatal("request scope ignored")
	}
}

func TestNativeWritesCloseBeforeGatewayHooks(t *testing.T) {
	for _, path := range []string{"/v2/items/deals", "/v2/items/deals/multiple-insert", "/v3/table/deals/items", "/v1/object/deals", "/v1/object-upsert/deals"} {
		if !nativeDataWrite(http.MethodPost, path) {
			t.Fatalf("write %s admitted", path)
		}
	}
	for _, path := range []string{"/v1/object/get-list/deals", "/v3/table/deals/items/list", "/v2/items/deals/filter", "/v2/invoke_function"} {
		if nativeDataWrite(http.MethodPost, path) {
			t.Fatalf("read/facade %s denied", path)
		}
	}
}

func TestUnscopedQueryRoutesDeniedBeforeHandler(t *testing.T) {
	cfg := config.CRMNativeConfig{Enabled: true, Project: "11111111-1111-4111-8111-111111111111", Environment: "22222222-2222-4222-8222-222222222222", ResourceEnvironment: "33333333-3333-4333-8333-333333333333"}
	for _, path := range []string{"/v1/object/get-list-aggregate/deals", "/v2/items/deals/filter", "/v2/items/deals/aggregation", "/v1/custom-endpoints/exec-query"} {
		for _, outside := range []bool{false, true} {
			t.Run(path+map[bool]string{false: "/target", true: "/other-project"}[outside], func(t *testing.T) {
				project := cfg.Project
				if outside {
					project = "66666666-6666-4666-8666-666666666666"
				}
				handlerReads := 0
				r := gin.New()
				r.Use(func(c *gin.Context) {
					c.Set("project_id", project)
					c.Set("environment_id", cfg.Environment)
					c.Set("Auth", &auth.V2HasAccessUserRes{ProjectId: project, EnvId: cfg.Environment, UserId: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"})
				})
				r.Use(Middleware(cfg))
				r.POST(path, func(c *gin.Context) {
					handlerReads++ // Actorless LRU cache and raw SQL are reached here.
					c.Status(http.StatusOK)
				})
				w := httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
				if outside {
					if w.Code != http.StatusOK || handlerReads != 1 {
						t.Fatalf("other project changed: status=%d reads=%d", w.Code, handlerReads)
					}
				} else if w.Code != http.StatusForbidden || handlerReads != 0 {
					t.Fatalf("protected handler reached: status=%d reads=%d", w.Code, handlerReads)
				}
			})
		}
	}
}

func TestPreflightRunsBeforeHooksWithPinnedActor(t *testing.T) {
	for _, reject := range []bool{false, true} {
		cfg := config.CRMNativeConfig{Enabled: true, NativeWritersReady: true, Project: "11111111-1111-4111-8111-111111111111", Environment: "22222222-2222-4222-8222-222222222222", ResourceEnvironment: "33333333-3333-4333-8333-333333333333"}
		actor := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
		preflight, hooks := 0, 0
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("project_id", cfg.Project)
			c.Set("environment_id", cfg.Environment)
			c.Set("Auth", &auth.V2HasAccessUserRes{ProjectId: cfg.Project, EnvId: cfg.Environment, UserId: actor})
		})
		r.Use(Middleware(cfg, func(c *gin.Context) error {
			preflight++
			md, _ := metadata.FromOutgoingContext(c.Request.Context())
			if md.Get("crm-user")[0] != actor || md.Get("crm-purpose")[0] != "" {
				t.Fatal("preflight trusted spoofed claims")
			}
			if reject {
				return errors.New("hidden row")
			}
			return nil
		}))
		r.POST("/v2/items/:collection", func(c *gin.Context) {
			hooks++
			md, _ := metadata.FromOutgoingContext(c.Request.Context())
			if preflight != 1 || md.Get("crm-purpose")[0] != "native-write" {
				t.Fatal("hooks preceded preflight")
			}
			c.Status(201)
		})
		req := httptest.NewRequest("POST", "/v2/items/deals", nil)
		req = req.WithContext(metadata.NewOutgoingContext(req.Context(), metadata.Pairs("crm-kind", "server", "crm-user", "spoof", "crm-purpose", "native-write")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if preflight != 1 || reject && (hooks != 0 || w.Code != 403) || !reject && (hooks != 1 || w.Code != 201) {
			t.Fatalf("preflight=%d hooks=%d status=%d", preflight, hooks, w.Code)
		}
	}
}
