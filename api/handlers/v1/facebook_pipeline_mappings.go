package v1

import (
	"strings"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

type facebookPipelineMappingRequest struct {
	PipelineValue      string            `json:"pipeline_value" binding:"required"`
	PipelineStageField string            `json:"pipeline_stage_field" binding:"required"`
	StageValue         string            `json:"stage_value" binding:"required"`
	PageIDs            []string          `json:"page_ids"`
	PageStageValues    map[string]string `json:"page_stage_values"`
}

type facebookPagePipelineMappingRequest struct {
	PipelineValue      string `json:"pipeline_value"`
	PipelineStageField string `json:"pipeline_stage_field"`
	StageValue         string `json:"stage_value"`
}

func facebookAssignedPipeline(mapping *pb.FacebookCrmMapping) string {
	value := mapping.GetPipelineValue()
	if value == disabledFacebookPipeline || (value == "Udevs" && !strings.HasPrefix(mapping.GetStageField(), "pipeline_")) {
		return ""
	}
	return value
}

func (h *HandlerV1) FacebookPipelineMappings(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	list, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), &pb.GetProjectResourceListRequest{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Type: pb.ResourceType_META_LEADS})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	pages := make([]gin.H, 0, len(list.GetResources()))
	for _, resource := range list.GetResources() {
		mapping := resource.GetSettings().GetFacebookLeads().GetCrmMapping()
		pipeline, stage := facebookAssignedPipeline(mapping), ""
		if pipeline != "" {
			stage = mapping.GetStageValue()
		}
		pages = append(pages, gin.H{"page_id": resource.GetExternalId(), "page_name": resource.GetName(), "pipeline_value": pipeline, "stage_value": stage, "status": resource.GetSettings().GetFacebookLeads().GetStatus()})
	}
	h.HandleResponse(c, status_http.OK, gin.H{"pages": pages})
}

func (h *HandlerV1) SaveFacebookPipelineMappings(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	var req facebookPipelineMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.PipelineValue, req.StageValue = strings.TrimSpace(req.PipelineValue), strings.TrimSpace(req.StageValue)
	req.PipelineStageField = strings.TrimSpace(req.PipelineStageField)
	if req.PipelineValue == "" || req.StageValue == "" || req.PipelineValue == disabledFacebookPipeline || !strings.HasPrefix(req.PipelineStageField, "pipeline_") || strings.ContainsAny(req.PipelineStageField, " /\\") {
		h.HandleResponse(c, status_http.BadRequest, "invalid pipeline or stage")
		return
	}
	list, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), &pb.GetProjectResourceListRequest{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Type: pb.ResourceType_META_LEADS})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	resources := list.GetResources()
	selected := make(map[string]bool, len(req.PageIDs))
	for _, id := range req.PageIDs {
		selected[strings.TrimSpace(id)] = true
	}
	for id := range selected {
		found := false
		for _, resource := range resources {
			if resource.GetExternalId() == id {
				found = true
				break
			}
		}
		if !found {
			h.HandleResponse(c, status_http.BadRequest, "selected page is not connected to this project")
			return
		}
	}
	for _, resource := range resources {
		credentials := resource.GetSettings().GetFacebookLeads()
		if credentials == nil {
			continue
		}
		current := credentials.GetCrmMapping()
		assigned := facebookAssignedPipeline(current)
		if selected[resource.GetExternalId()] && assigned != "" && assigned != req.PipelineValue {
			h.HandleResponse(c, status_http.BadRequest, "page is already assigned to another pipeline")
			return
		}
	}
	for _, resource := range resources {
		credentials := resource.GetSettings().GetFacebookLeads()
		if credentials == nil {
			continue
		}
		current := credentials.GetCrmMapping()
		assigned := facebookAssignedPipeline(current)
		if !selected[resource.GetExternalId()] && ((assigned != "" && assigned != req.PipelineValue) || current.GetPipelineValue() == disabledFacebookPipeline) {
			continue
		}
		mapping := current
		if mapping == nil {
			mapping = crmMappingRequestToProto(models.FacebookCrmMapping{})
		}
		if selected[resource.GetExternalId()] {
			stage := req.PageStageValues[resource.GetExternalId()]
			if stage == "" {
				stage = req.StageValue
			}
			mapping.PipelineValue, mapping.StageValue, mapping.StageField = req.PipelineValue, stage, req.PipelineStageField
		} else {
			mapping.PipelineValue = disabledFacebookPipeline
		}
		credentials.CrmMapping = mapping
		_, err := h.companyServices.Resource().UpdateProjectResource(c.Request.Context(), &pb.ProjectResource{
			Id: resource.GetId(), ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId,
			Name: resource.GetName(), Type: pb.ResourceType_META_LEADS.String(), ResourceType: int32(pb.ResourceType_META_LEADS),
			ExternalId: resource.GetExternalId(), Settings: &pb.Settings{FacebookLeads: credentials},
		})
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	h.HandleResponse(c, status_http.OK, gin.H{"page_ids": req.PageIDs})
}

// SaveFacebookPagePipelineMapping configures one connected Page from its
// integration card without changing the mappings of other Pages.
func (h *HandlerV1) SaveFacebookPagePipelineMapping(c *gin.Context) {
	state, ok := h.authContext(c)
	if !ok {
		return
	}
	var req facebookPagePipelineMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.PipelineValue = strings.TrimSpace(req.PipelineValue)
	req.PipelineStageField = strings.TrimSpace(req.PipelineStageField)
	req.StageValue = strings.TrimSpace(req.StageValue)
	if req.PipelineValue != "" && (req.StageValue == "" || !strings.HasPrefix(req.PipelineStageField, "pipeline_") || strings.ContainsAny(req.PipelineStageField, " /\\")) {
		h.HandleResponse(c, status_http.BadRequest, "invalid pipeline or stage")
		return
	}
	list, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), &pb.GetProjectResourceListRequest{ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Type: pb.ResourceType_META_LEADS})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	for _, resource := range list.GetResources() {
		if resource.GetExternalId() != strings.TrimSpace(c.Param("page_id")) {
			continue
		}
		credentials := resource.GetSettings().GetFacebookLeads()
		if credentials == nil {
			break
		}
		mapping := credentials.GetCrmMapping()
		if mapping == nil {
			mapping = &pb.FacebookCrmMapping{}
		}
		if req.PipelineValue == "" {
			mapping.PipelineValue = disabledFacebookPipeline
			mapping.StageValue = ""
			mapping.StageField = ""
		} else {
			mapping.PipelineValue, mapping.StageValue, mapping.StageField = req.PipelineValue, req.StageValue, req.PipelineStageField
		}
		credentials.CrmMapping = mapping
		_, err := h.companyServices.Resource().UpdateProjectResource(c.Request.Context(), &pb.ProjectResource{Id: resource.GetId(), ProjectId: state.ProjectId, EnvironmentId: state.EnvironmentId, Name: resource.GetName(), Type: pb.ResourceType_META_LEADS.String(), ResourceType: int32(pb.ResourceType_META_LEADS), ExternalId: resource.GetExternalId(), Settings: &pb.Settings{FacebookLeads: credentials}})
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.HandleResponse(c, status_http.OK, gin.H{"page_id": resource.GetExternalId(), "pipeline_value": req.PipelineValue, "stage_value": req.StageValue})
		return
	}
	h.HandleResponse(c, status_http.NotFound, "Facebook Page is not connected to this project")
}
