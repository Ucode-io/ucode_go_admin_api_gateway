package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/services"
)

func TestResolveCRMMappingUsesSelectedPipelineStageField(t *testing.T) {
	resource := &pb.ProjectResource{Settings: &pb.Settings{FacebookLeads: &pb.FacebookLeadsCredentials{
		CrmMapping: &pb.FacebookCrmMapping{PipelineValue: "Gisht Voronkasi", StageValue: "Yangi Lid", StageField: "pipeline_gisht_voronkasi"},
	}}}
	mapping := (&HandlerV1{}).resolveCRMMapping(resource)
	if mapping.PipelineValue != "Gisht Voronkasi" || mapping.StageValue != "Yangi Lid" || mapping.StageField != "stage" || mapping.PipelineStageField != "pipeline_gisht_voronkasi" {
		t.Fatalf("wrong page-specific routing: %+v", mapping)
	}
}

type pipelineMappingResources struct {
	pb.ResourceServiceClient
	resources []*pb.ProjectResource
	updated   []*pb.ProjectResource
	request   *pb.GetProjectResourceListRequest
}

func (r *pipelineMappingResources) GetProjectResourceList(ctx context.Context, request *pb.GetProjectResourceListRequest, opts ...grpc.CallOption) (*pb.ListProjectResource, error) {
	r.request = request
	return &pb.ListProjectResource{Resources: r.resources}, nil
}

func (r *pipelineMappingResources) UpdateProjectResource(ctx context.Context, resource *pb.ProjectResource, opts ...grpc.CallOption) (*pb.Empty, error) {
	r.updated = append(r.updated, resource)
	return &pb.Empty{}, nil
}

type pipelineMappingCompany struct {
	services.CompanyServiceI
	resources *pipelineMappingResources
}

func (c *pipelineMappingCompany) Resource() pb.ResourceServiceClient { return c.resources }

func pipelineMappingResource(id, pipeline string) *pb.ProjectResource {
	return &pb.ProjectResource{Id: id, ExternalId: id, Name: id, Settings: &pb.Settings{FacebookLeads: &pb.FacebookLeadsCredentials{
		PageId: id, PageAccessToken: "test-page-token", CrmMapping: &pb.FacebookCrmMapping{
			PipelineValue: pipeline, StageValue: "Old stage", StageField: "pipeline_test", ContactsTable: "custom_contacts",
		},
	}}}
}

func runPipelineMappingRequest(resources *pipelineMappingResources, method, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("project_id", "00000000-0000-4000-8000-000000000001")
	c.Set("environment_id", "00000000-0000-4000-8000-000000000002")
	c.Request = httptest.NewRequest(method, "/v1/facebook/pipeline-mappings", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler := &HandlerV1{companyServices: &pipelineMappingCompany{resources: resources}, log: zap.NewNop()}
	if method == http.MethodGet {
		handler.FacebookPipelineMappings(c)
	} else {
		handler.SaveFacebookPipelineMappings(c)
	}
	return recorder
}

func TestFacebookPipelineMappingsSavePreservesCredentialsAndOtherPipelines(t *testing.T) {
	resources := &pipelineMappingResources{resources: []*pb.ProjectResource{
		pipelineMappingResource("selected", "Gisht Voronkasi"),
		pipelineMappingResource("removed", "Gisht Voronkasi"),
		pipelineMappingResource("other", "Generators"),
	}}
	response := runPipelineMappingRequest(resources, http.MethodPut, `{"pipeline_value":"Gisht Voronkasi","pipeline_stage_field":"pipeline_gisht_voronkasi","stage_value":"Yangi Lid","page_ids":["selected"]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("save failed: %d %s", response.Code, response.Body.String())
	}
	if len(resources.updated) != 2 {
		t.Fatalf("expected only this pipeline's pages updated, got %d", len(resources.updated))
	}
	selected := resources.updated[0].GetSettings().GetFacebookLeads()
	if selected.GetPageAccessToken() != "test-page-token" || selected.GetCrmMapping().GetContactsTable() != "custom_contacts" || selected.GetCrmMapping().GetStageValue() != "Yangi Lid" {
		t.Fatal("mapping update lost credentials, custom schema, or selected entry stage")
	}
	if resources.updated[1].GetSettings().GetFacebookLeads().GetCrmMapping().GetPipelineValue() != disabledFacebookPipeline {
		t.Fatal("removed page must stop routing leads")
	}
	if resources.request.GetProjectId() != "00000000-0000-4000-8000-000000000001" || resources.request.GetEnvironmentId() != "00000000-0000-4000-8000-000000000002" || resources.request.GetType() != pb.ResourceType_META_LEADS {
		t.Fatal("resources must be scoped to the authenticated project and environment")
	}
}

func TestFacebookPipelineMappingsPreservesPageSpecificStages(t *testing.T) {
	resources := &pipelineMappingResources{resources: []*pb.ProjectResource{
		pipelineMappingResource("first", "Gisht Voronkasi"),
		pipelineMappingResource("second", "Gisht Voronkasi"),
	}}
	response := runPipelineMappingRequest(resources, http.MethodPut, `{"pipeline_value":"Gisht Voronkasi","pipeline_stage_field":"pipeline_gisht_voronkasi","stage_value":"Yangi Lid","page_ids":["first","second"],"page_stage_values":{"first":"Old stage","second":"Other stage"}}`)
	if response.Code != http.StatusOK || len(resources.updated) != 2 {
		t.Fatalf("save failed: %d %s", response.Code, response.Body.String())
	}
	for i, stage := range []string{"Old stage", "Other stage"} {
		if got := resources.updated[i].GetSettings().GetFacebookLeads().GetCrmMapping().GetStageValue(); got != stage {
			t.Fatalf("page %d stage = %q, want %q", i, got, stage)
		}
	}
}

func TestFacebookPagePipelineMappingUpdatesOnlySelectedPage(t *testing.T) {
	resources := &pipelineMappingResources{resources: []*pb.ProjectResource{pipelineMappingResource("first", "Gisht Voronkasi"), pipelineMappingResource("second", "Generators")}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("project_id", "00000000-0000-4000-8000-000000000001")
	c.Set("environment_id", "00000000-0000-4000-8000-000000000002")
	c.Params = gin.Params{{Key: "page_id", Value: "first"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/v1/facebook/pipeline-mappings/first", strings.NewReader(`{"pipeline_value":"Gisht Voronkasi","pipeline_stage_field":"pipeline_gisht_voronkasi","stage_value":"Yangi Lid"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	(&HandlerV1{companyServices: &pipelineMappingCompany{resources: resources}, log: zap.NewNop()}).SaveFacebookPagePipelineMapping(c)
	if recorder.Code != http.StatusOK || len(resources.updated) != 1 {
		t.Fatalf("page update failed: %d %s", recorder.Code, recorder.Body.String())
	}
	got := resources.updated[0].GetSettings().GetFacebookLeads()
	if got.GetCrmMapping().GetStageValue() != "Yangi Lid" || got.GetPageAccessToken() != "test-page-token" || resources.updated[0].GetExternalId() != "first" {
		t.Fatalf("selected page mapping or credentials changed unexpectedly: %+v", got)
	}
}

func TestFacebookPipelineMappingsRejectBeforeWriting(t *testing.T) {
	for _, pageID := range []string{"unknown", "other"} {
		t.Run(pageID, func(t *testing.T) {
			resources := &pipelineMappingResources{resources: []*pb.ProjectResource{pipelineMappingResource("selected", "Gisht Voronkasi"), pipelineMappingResource("other", "Generators")}}
			response := runPipelineMappingRequest(resources, http.MethodPut, `{"pipeline_value":"Gisht Voronkasi","pipeline_stage_field":"pipeline_gisht_voronkasi","stage_value":"Yangi Lid","page_ids":["selected","`+pageID+`"]}`)
			if response.Code != http.StatusBadRequest || len(resources.updated) != 0 {
				t.Fatalf("invalid selection must not write any mappings: status=%d writes=%d", response.Code, len(resources.updated))
			}
		})
	}
}

func TestFacebookPipelineMappingsDoesNotExposeTokens(t *testing.T) {
	resources := &pipelineMappingResources{resources: []*pb.ProjectResource{pipelineMappingResource("selected", "Udevs")}}
	response := runPipelineMappingRequest(resources, http.MethodGet, "")
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "test-page-token") || !strings.Contains(response.Body.String(), `"pipeline_value":"Udevs"`) {
		t.Fatalf("unexpected public mapping response: %d %s", response.Code, response.Body.String())
	}
}

func TestResolveCRMMappingKeepsLegacyCustomStage(t *testing.T) {
	resource := pipelineMappingResource("legacy", "Custom")
	resource.Settings.FacebookLeads.CrmMapping.StageField = "custom_stage"
	mapping := (&HandlerV1{}).resolveCRMMapping(resource)
	if mapping.StageField != "custom_stage" {
		t.Fatal("legacy custom stage field changed")
	}
}

func TestResolveCRMMappingKeepsLegacyDefault(t *testing.T) {
	resource := &pb.ProjectResource{Settings: &pb.Settings{FacebookLeads: &pb.FacebookLeadsCredentials{}}}
	mapping := (&HandlerV1{}).resolveCRMMapping(resource)
	if mapping.PipelineValue != "Udevs" || mapping.StageField != "stage" || mapping.PipelineStageField != "pipeline_udevs" {
		t.Fatalf("legacy mapping changed: %+v", mapping)
	}
}
