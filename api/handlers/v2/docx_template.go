package v2

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"strconv"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/template_service"
	"ucode/ucode_go_api_gateway/pkg/util"
)

// CreateDocxTemplate godoc
// @Security ApiKeyAuth
// @ID create_docx_template
// @Router /v2/docx-template [POST]
// @Summary Create docx template
// @Description Create docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param template body tmp.CreateDocxTemplateReq true "CreateDocxTemplateReq"
// @Success 201 {object} status_http.Response{data=tmp.DocxTemplate} "DocxTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateDocxTemplate(c *gin.Context) {
	var (
		docxTemplate tmp.CreateDocxTemplateReq
	)

	if err := c.ShouldBindJSON(&docxTemplate); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	docxTemplate.ProjectId = projectId.(string)
	docxTemplate.ResourceId = resource.ResourceEnvironmentId

	docxTemplate.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().DocxTemplate().CreateDocxTemplate(
		context.Background(),
		&docxTemplate,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleDocxTemplate godoc
// @Security ApiKeyAuth
// @ID get_single_docx_template
// @Router /v2/docx-template/{docx-template-id} [GET]
// @Summary Get single docx template
// @Description Get single docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param docx-template-id path string true "docx-template-id"
// @Success 200 {object} status_http.Response{data=tmp.DocxTemplate} "DocxTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetSingleDocxTemplate(c *gin.Context) {
	docxTemplateId := c.Param("docx-template-id")

	if !util.IsValidUUID(docxTemplateId) {
		h.handleResponse(c, status_http.InvalidArgument, "docx template id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().DocxTemplate().GetSingleDocxTemplate(
		context.Background(),
		&tmp.GetSingleDocxTemplateReq{
			Id:         docxTemplateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateDocxTemplate godoc
// @Security ApiKeyAuth
// @ID update_docx_template
// @Router /v2/docx-template [PUT]
// @Summary Update docx template
// @Description Update docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param docx_template body tmp.UpdateDocxTemplateReq true "UpdateDocxTemplateReqBody"
// @Success 200 {object} status_http.Response{data=tmp.DocxTemplate} "DocxTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateDocxTemplate(c *gin.Context) {
	var (
		docxTemplate tmp.UpdateDocxTemplateReq
	)

	if err := c.ShouldBindJSON(&docxTemplate); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	docxTemplate.ProjectId = projectId.(string)
	docxTemplate.ResourceId = resource.ResourceEnvironmentId
	docxTemplate.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().DocxTemplate().UpdateDocxTemplate(
		context.Background(),
		&docxTemplate,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteDocxTemplate godoc
// @Security ApiKeyAuth
// @ID delete_docx_template
// @Router /v2/docx-template/{docx-template-id} [DELETE]
// @Summary Delete docx template
// @Description Delete docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param docx-template-id path string true "docx-template-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteDocxTemplate(c *gin.Context) {
	docxTemplateId := c.Param("docx-template-id")

	if !util.IsValidUUID(docxTemplateId) {
		h.handleResponse(c, status_http.InvalidArgument, "docx template id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}
	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().DocxTemplate().DeleteDocxTemplate(
		context.Background(),
		&tmp.DeleteDocxTemplateReq{
			Id:         docxTemplateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListDocxTemplate godoc
// @Security ApiKeyAuth
// @ID get_list_docx_template
// @Router /v2/docx-template [GET]
// @Summary Get List docx template
// @Description Get List docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param folder-id query string true "folder-id"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListDocxTemplateRes} "DocxTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetListDocxTemplate(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().DocxTemplate().GetListDocxTemplate(
		context.Background(),
		&tmp.GetListDocxTemplateReq{
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			FolderId:   c.DefaultQuery("folder-id", ""),
			Limit:      int32(limit),
			Offset:     int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
