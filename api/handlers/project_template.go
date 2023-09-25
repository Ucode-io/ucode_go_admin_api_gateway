package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// CopyProjectTemplate godoc
// @Security ApiKeyAuth
// @ID create_event
// @Router /v2/copy-project [POST]
// @Summary CopyProjectTemplate
// @Description CopyProjectTemplate
// @Tags ProjectTemplate
// @Accept json
// @Produce json
// @Param event body models.CopyProjectTemplateRequest true "CopyFromProjectRequestMessage"
// @Success 201 {object} status_http.Response{data=obs.CommonMessage} "Event data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CopyProjectTemplate(c *gin.Context) {
	var body models.CopyProjectTemplateRequest

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	fromResource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     body.FromProjectId,
			EnvironmentId: body.FromEnvId,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	toResource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     body.ToProjectId,
			EnvironmentId: body.ToEnvId,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.BuilderService().ObjectBuilder().CopyFromProject(
		context.Background(),
		&obs.CopyFromProjectRequestMessage{
			ProjectId:     toResource.ResourceEnvironmentId,
			FromProjectId: fromResource.ResourceEnvironmentId,
			Menus:         body.Menus,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)

}
