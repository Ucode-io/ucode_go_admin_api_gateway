package v1

import (
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/genproto/transcoder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) TranscoderWebhook(c *gin.Context) {
	var (
		req models.TranscoderWebhook
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if len(req.KeyId) == 0 {
		return
	}

	structDate, err := helper.ConvertMapToStruct(map[string]any{
		"guid":        req.KeyId,
		req.FieldSlug: fmt.Sprintf("https://%v/movies/%v/master.m3u8", h.baseConf.MinioEndpoint, req.OutputKey),
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	resourceEnvironment, err := h.companyServices.Resource().GetResourceEnvironment(c.Request.Context(), &pb.GetResourceEnvironmentReq{
		Id: req.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resourceEnvironment.GetProjectId(),
		resourceEnvironment.GetNodeType(),
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceEnvironment.ResourceType {
	case 3:
		_, err = services.GoObjectBuilderService().Items().Update(
			c.Request.Context(), &nb.CommonMessage{
				TableSlug:        req.TableSlug,
				Data:             structDate,
				ProjectId:        resourceEnvironment.GetId(),
				BlockedBuilder:   false,
				EnvId:            resourceEnvironment.GetEnvironmentId(),
				CompanyProjectId: resourceEnvironment.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
}

func (h *HandlerV1) GetListPipeline(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleError(c, status_http.BadRequest, err)
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleError(c, status_http.BadRequest, err)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(), &pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.GetNodeType(),
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	response, err := services.TranscoderService().Pipeline().GetList(c, &transcoder_service.GetListPipelineRequest{
		Page:      int32(offset),
		Limit:     int32(limit),
		ProjectId: resource.ResourceEnvironmentId,
		OrderBy:   c.Query("order_by"),
		Order:     c.Query("order"),
		FromDate:  c.Query("from_date"),
		ToDate:    c.Query("to_date"),
		Search:    c.Query("search"),
	})
	if err != nil {
		h.handleError(c, status_http.InternalServerError, err)
		return
	}

	h.handleResponse(c, status_http.OK, response)
}
