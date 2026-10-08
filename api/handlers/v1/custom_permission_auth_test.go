package v1

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	grpcpool "github.com/processout/grpc-go-pool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"ucode/ucode_go_api_gateway/api/handlers/crmguard"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/services"
)

type permissionSession struct {
	as.SessionServiceClient
	login                  *as.V2HasAccessUserRes
	loginCalls, adminCalls int
}

func (s *permissionSession) V2HasAccessUser(_ context.Context, req *as.V2HasAccessUserReq, _ ...grpc.CallOption) (*as.V2HasAccessUserRes, error) {
	s.loginCalls++
	if req.Method != http.MethodGet || req.Path != "/v1/custom-permission/accesses" || req.AccessToken != "fixture-token" {
		return nil, context.Canceled
	}
	return s.login, nil
}
func (s *permissionSession) HasAccessSuperAdmin(context.Context, *as.HasAccessSuperAdminReq, ...grpc.CallOption) (*as.HasAccessSuperAdminRes, error) {
	s.adminCalls++
	return &as.HasAccessSuperAdminRes{UserId: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ProjectId: s.login.ProjectId, EnvId: s.login.EnvId}, nil
}

type permissionAuth struct {
	services.AuthServiceManagerI
	session *permissionSession
	pool    *grpcpool.Pool
}

func (s permissionAuth) Session(ctx context.Context) (as.SessionServiceClient, *grpcpool.ClientConn, error) {
	conn, err := s.pool.Get(ctx)
	return s.session, conn, err
}

type permissionResource struct {
	pb.MicroserviceResourceClient
	resource *pb.ServiceResourceModel
	reads    int
}

func (s *permissionResource) GetSingle(_ context.Context, req *pb.GetSingleServiceResourceReq, _ ...grpc.CallOption) (*pb.ServiceResourceModel, error) {
	s.reads++
	if req.ProjectId != s.resource.ProjectId || req.EnvironmentId != s.resource.EnvironmentId {
		return nil, context.Canceled
	}
	return s.resource, nil
}

type permissionCompany struct {
	services.CompanyServiceI
	resource *permissionResource
}

func (s permissionCompany) ServiceResource() pb.MicroserviceResourceClient { return s.resource }

type permissionNodes struct {
	services.ServiceNodesI
	service services.ServiceManagerI
}

func (s permissionNodes) Get(string) (services.ServiceManagerI, error) { return s.service, nil }

type permissionService struct {
	services.ServiceManagerI
	builder services.GoBuilderServiceI
}

func (s permissionService) GoObjectBuilderService() services.GoBuilderServiceI { return s.builder }

type permissionBuilder struct {
	services.GoBuilderServiceI
	permissions nb.CustomPermissionsServiceClient
}

func (s permissionBuilder) CustomPermission() nb.CustomPermissionsServiceClient { return s.permissions }

type permissionRPC struct {
	nb.CustomPermissionsServiceClient
	run func(context.Context, *nb.GetCustomPermissionAccessesRequest) (*nb.GetCustomPermissionAccessesResponse, error)
}

func (s permissionRPC) GetCustomPermissionAccesses(ctx context.Context, req *nb.GetCustomPermissionAccessesRequest, _ ...grpc.CallOption) (*nb.GetCustomPermissionAccessesResponse, error) {
	return s.run(ctx, req)
}

func TestMountedCustomPermissionReadUsesFreshOwnCRMLogin(t *testing.T) {
	cfg := config.CRMNativeConfig{Enabled: true, Project: "11111111-1111-4111-8111-111111111111", Environment: "22222222-2222-4222-8222-222222222222", ResourceEnvironment: "33333333-3333-4333-8333-333333333333"}
	actor, role, client := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	listener := bufconn.Listen(1024)
	server := grpc.NewServer()
	go server.Serve(listener)
	t.Cleanup(func() { server.Stop(); listener.Close() })
	pool, err := grpcpool.New(func() (*grpc.ClientConn, error) {
		return grpc.DialContext(context.Background(), "passthrough:///permission-fixture", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	}, 1, 1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for _, tc := range []struct {
		name                                                                                                                string
		query, body, authorization                                                                                          string
		otherProject, off, badEnv, missingActor, otherIdentity, badResource, redirect, duplicateEnv, duplicateAuthorization bool
		allowed                                                                                                             bool
	}{
		{name: "own role fresh twice", allowed: true},
		{name: "own child menu", query: "parent_id=dddddddd-dddd-4ddd-8ddd-dddddddddddd", allowed: true},
		{name: "missing bearer", authorization: "missing"},
		{name: "API key", authorization: "API-KEY"},
		{name: "redirect injection", redirect: true},
		{name: "other role", query: "role_id=dddddddd-dddd-4ddd-8ddd-dddddddddddd"},
		{name: "other client type", query: "client_type_id=dddddddd-dddd-4ddd-8ddd-dddddddddddd"},
		{name: "missing role", query: "role_id="},
		{name: "duplicate role", query: "role_id=" + role + "&role_id=" + role},
		{name: "arbitrary query", query: "data=secret"},
		{name: "invalid parent", query: "parent_id=' OR 1=1"},
		{name: "body injection", body: `{"role_id":"other"}`},
		{name: "missing CRM actor", missingActor: true},
		{name: "other authenticated tenant", otherIdentity: true},
		{name: "environment mismatch", badEnv: true},
		{name: "duplicate environment", duplicateEnv: true},
		{name: "duplicate authorization", duplicateAuthorization: true},
		{name: "actual resource mismatch", badResource: true},
		{name: "unrelated CRM original admin path", otherProject: true, query: "role_id=another-role", body: `{}`, allowed: true},
		{name: "scope off original admin path", off: true, query: "role_id=another-role", body: `{}`, allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := cfg
			local.Enabled = !tc.off
			project := cfg.Project
			if tc.otherProject {
				project = "66666666-6666-4666-8666-666666666666"
			}
			login := &as.V2HasAccessUserRes{ProjectId: project, EnvId: cfg.Environment, UserId: actor, UserIdAuth: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", RoleId: role, ClientTypeId: client}
			if tc.missingActor {
				login.UserId = ""
			}
			if tc.otherIdentity {
				login.ProjectId = "66666666-6666-4666-8666-666666666666"
			}
			session := &permissionSession{login: login}
			resourceID := cfg.ResourceEnvironment
			if tc.badResource {
				resourceID = "77777777-7777-4777-8777-777777777777"
			}
			resource := &permissionResource{resource: &pb.ServiceResourceModel{ProjectId: project, EnvironmentId: cfg.Environment, ResourceType: pb.ResourceType_POSTGRESQL, ResourceEnvironmentId: resourceID}}
			rpcCalls := 0
			rpc := permissionRPC{run: func(ctx context.Context, req *nb.GetCustomPermissionAccessesRequest) (*nb.GetCustomPermissionAccessesResponse, error) {
				rpcCalls++
				if req.ProjectId != resourceID {
					t.Fatal("resource not selected from official mapping")
				}
				if local.Enabled && !tc.otherProject {
					md, _ := metadata.FromOutgoingContext(ctx)
					if req.RoleId != role || req.ClientTypeId != client || len(md.Get("crm-role")) != 1 || md.Get("crm-role")[0] != role || md.Get("crm-client-type")[0] != client || md.Get("crm-user")[0] != actor || md.Get("crm-purpose")[0] != "own-navigation-permissions" {
						t.Fatal("caller claims retained")
					}
				} else if crmguard.Scoped(ctx) {
					t.Fatal("unrelated/disabled request changed")
				}
				return &nb.GetCustomPermissionAccessesResponse{}, nil
			}}
			h := &HandlerV1{baseConf: config.BaseConfig{CRMNative: local}, log: zap.NewNop(), authService: permissionAuth{session: session, pool: pool}, companyServices: permissionCompany{resource: resource}, services: permissionNodes{service: permissionService{builder: permissionBuilder{permissions: rpc}}}}
			r := gin.New()
			r.GET("/v1/custom-permission/accesses", h.CustomPermissionAccessAuthMiddleware(h.baseConf), crmguard.Middleware(local), crmguard.CustomPermissionAccesses(local), h.GetCustomPermissionAccesses)
			query := url.Values{"project-id": {project}, "role_id": {role}, "client_type_id": {client}}
			if tc.query != "" {
				extra, _ := url.ParseQuery(tc.query)
				for key, values := range extra {
					query[key] = values
				}
			}
			repetitions := 1
			if tc.name == "own role fresh twice" {
				repetitions = 2
			}
			for range repetitions {
				req := httptest.NewRequest(http.MethodGet, "/v1/custom-permission/accesses?"+query.Encode(), strings.NewReader(tc.body))
				authorization := "Bearer fixture-token"
				if tc.authorization != "" {
					authorization = tc.authorization
				}
				if authorization != "missing" {
					req.Header.Set("Authorization", authorization)
				}
				req.Header.Set("Environment-Id", cfg.Environment)
				if tc.badEnv {
					req.Header.Set("Environment-Id", "88888888-8888-4888-8888-888888888888")
				}
				if tc.duplicateEnv {
					req.Header.Add("Environment-Id", cfg.Environment)
				}
				if tc.duplicateAuthorization {
					req.Header.Add("Authorization", "Bearer fixture-token")
				}
				if tc.redirect {
					req.Header.Set("redirect", "true")
				}
				req = req.WithContext(metadata.NewOutgoingContext(req.Context(), metadata.Pairs("crm-role", "spoof", "crm-client-type", "spoof", "crm-user", "spoof", "crm-kind", "server")))
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if (w.Code == http.StatusOK) != tc.allowed {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
			}
			if tc.allowed && rpcCalls != repetitions || !tc.allowed && rpcCalls != 0 {
				t.Fatalf("RPC calls=%d", rpcCalls)
			}
			if !tc.allowed && !tc.badResource && resource.reads != 0 {
				t.Fatal("invalid identity/selectors reached resource discovery")
			}
			if tc.otherProject || tc.off {
				if session.adminCalls != 1 || session.loginCalls != 0 {
					t.Fatal("original authentication changed")
				}
			} else if tc.allowed && (session.loginCalls != repetitions || session.adminCalls != 0) {
				t.Fatal("fresh CRM session not rechecked")
			}
		})
	}
}
