package v2

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"ucode/ucode_go_api_gateway/api/handlers/crmguard"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/services"
)

type preflightResource struct {
	pb.MicroserviceResourceClient
	resource *pb.ServiceResourceModel
}

func (f preflightResource) GetSingle(context.Context, *pb.GetSingleServiceResourceReq, ...grpc.CallOption) (*pb.ServiceResourceModel, error) {
	return f.resource, nil
}

type preflightCompany struct {
	services.CompanyServiceI
	resource pb.MicroserviceResourceClient
}

func (f preflightCompany) ServiceResource() pb.MicroserviceResourceClient { return f.resource }

type preflightNodes struct {
	services.ServiceNodesI
	service services.ServiceManagerI
}

func (f preflightNodes) Get(string) (services.ServiceManagerI, error) { return f.service, nil }

type preflightService struct {
	services.ServiceManagerI
	builder services.GoBuilderServiceI
}

func (f preflightService) GoObjectBuilderService() services.GoBuilderServiceI { return f.builder }

type preflightBuilder struct {
	services.GoBuilderServiceI
	items  nb.ItemsServiceClient
	events nb.CustomEventServiceClient
}

func (f preflightBuilder) Items() nb.ItemsServiceClient             { return f.items }
func (f preflightBuilder) CustomEvent() nb.CustomEventServiceClient { return f.events }

type preflightItems struct {
	nb.ItemsServiceClient
	run func(context.Context, *nb.CommonMessage) (*nb.CommonMessage, error)
}

func (f preflightItems) GetSingle(ctx context.Context, req *nb.CommonMessage, _ ...grpc.CallOption) (*nb.CommonMessage, error) {
	return f.run(ctx, req)
}

type preflightEvents struct {
	nb.CustomEventServiceClient
	run func(context.Context, *nb.GetCustomEventsListRequest) (*nb.GetCustomEventsListResponse, error)
}

func (f preflightEvents) GetList(ctx context.Context, req *nb.GetCustomEventsListRequest, _ ...grpc.CallOption) (*nb.GetCustomEventsListResponse, error) {
	return f.run(ctx, req)
}

func TestHTTPNativePreflightRestoresBodyAndRejectsHooksBeforeWrite(t *testing.T) {
	for _, tc := range []struct {
		name          string
		allowed, hook bool
		status        int
	}{{name: "allowed native write", allowed: true, status: 201}, {name: "hidden row", status: 403}, {name: "before or after hook requires binding", allowed: true, hook: true, status: 403}} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.CRMNativeConfig{Enabled: true, NativeWritersReady: true, Project: "11111111-1111-4111-8111-111111111111", Environment: "22222222-2222-4222-8222-222222222222", ResourceEnvironment: "33333333-3333-4333-8333-333333333333"}
			actor := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			writes, reads, events := 0, 0, 0
			items := preflightItems{run: func(ctx context.Context, req *nb.CommonMessage) (*nb.CommonMessage, error) {
				reads++
				md, _ := metadata.FromOutgoingContext(ctx)
				data := req.Data.AsMap()
				if md.Get("crm-user")[0] != actor || md.Get("crm-purpose")[0] != "native-preflight" || req.ProjectId != cfg.ResourceEnvironment || data["operation"] != "create" || data["object"].(map[string]any)["name"] != "fixture" {
					t.Fatal("untrusted preflight request")
				}
				result, _ := structpb.NewStruct(map[string]any{"allowed": tc.allowed})
				return &nb.CommonMessage{Data: result}, nil
			}}
			custom := preflightEvents{run: func(ctx context.Context, req *nb.GetCustomEventsListRequest) (*nb.GetCustomEventsListResponse, error) {
				events++
				md, _ := metadata.FromOutgoingContext(ctx)
				if md.Get("crm-purpose")[0] != "native-write" || req.Method != "CREATE" {
					t.Fatal("untrusted hook discovery")
				}
				res := &nb.GetCustomEventsListResponse{}
				if tc.hook {
					res.CustomEvents = []*nb.CustomEvent{{}}
				}
				return res, nil
			}}
			builder := preflightBuilder{items: items, events: custom}
			service := preflightService{builder: builder}
			h := &HandlerV2{baseConf: config.BaseConfig{CRMNative: cfg}, services: preflightNodes{service: service}, companyServices: preflightCompany{resource: preflightResource{resource: &pb.ServiceResourceModel{ResourceType: pb.ResourceType_POSTGRESQL, ResourceEnvironmentId: cfg.ResourceEnvironment, ProjectId: cfg.Project, EnvironmentId: cfg.Environment}}}}
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("project_id", cfg.Project)
				c.Set("environment_id", cfg.Environment)
				c.Set("Auth", &as.V2HasAccessUserRes{ProjectId: cfg.Project, EnvId: cfg.Environment, UserId: actor})
			})
			r.Use(crmguard.Middleware(cfg, h.CRMNativePreflight))
			body := `{"data":{"name":"fixture"}}`
			r.POST("/v2/items/:collection", func(c *gin.Context) {
				writes++
				b, err := io.ReadAll(c.Request.Body)
				if err != nil || string(b) != body {
					t.Fatal("native body changed")
				}
				c.Status(201)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("POST", "/v2/items/deals", strings.NewReader(body)))
			if w.Code != tc.status || reads != 1 || !tc.allowed && events != 0 || tc.allowed && events != 1 || tc.status == 403 && writes != 0 {
				t.Fatalf("status=%d reads=%d events=%d writes=%d", w.Code, reads, events, writes)
			}
		})
	}
}
