package handlers

import (
	"context"
	"fmt"
	"ucode/ucode_go_admin_api_gateway/api/http"
	obs "ucode/ucode_go_admin_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_admin_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateHtmlTemplate godoc
// @Security ApiKeyAuth
// @ID create_html_template
// @Router /v1/html-template [POST]
// @Summary Create htmlTemplate
// @Description Create htmlTemplate
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template body object_builder_service.CreateHtmlTemplateRequest true "CreateHtmlTemplateRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.HtmlTemplate} "Html Template data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateHtmlTemplate(c *gin.Context) {
	var htmlTemplate obs.CreateHtmlTemplateRequest

	err := c.ShouldBindJSON(&htmlTemplate)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.services.HtmlTemplateService().Create(
		context.Background(),
		&htmlTemplate,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetSingleHtmlTemplate godoc
// @Security ApiKeyAuth
// @ID get_html_template_by_id
// @Router /v1/html-template/{html_template_id} [GET]
// @Summary Get single html template
// @Description Get single html template
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template_id path string true "html_template_id"
// @Success 200 {object} http.Response{data=object_builder_service.HtmlTemplate} "HtmlTemplateBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleHtmlTemplate(c *gin.Context) {
	htmlTemplateID := c.Param("html_template_id")
	fmt.Println(htmlTemplateID)
	if !util.IsValidUUID(htmlTemplateID) {
		h.handleResponse(c, http.InvalidArgument, "html template id is an invalid uuid")
		return
	}
	resp, err := h.services.HtmlTemplateService().GetSingle(
		context.Background(),
		&obs.HtmlTemplatePrimaryKey{
			Id: htmlTemplateID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateHtmlTemplate godoc
// @Security ApiKeyAuth
// @ID update_html_template
// @Router /v1/html-template [PUT]
// @Summary Update html template
// @Description Update html template
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template body object_builder_service.HtmlTemplate true "UpdateHtmlTemplateRequestBody"
// @Success 200 {object} http.Response{data=object_builder_service.HtmlTemplate} "HtmlTemplate data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateHtmlTemplate(c *gin.Context) {
	var htmlTemplate obs.HtmlTemplate

	err := c.ShouldBindJSON(&htmlTemplate)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	resp, err := h.services.HtmlTemplateService().Update(
		context.Background(),
		&htmlTemplate,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteHtmlTemplate godoc
// @Security ApiKeyAuth
// @ID delete_html_template_id
// @Router /v1/html-template/{html_template_id} [DELETE]
// @Summary Delete html template
// @Description Delete html template
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template_id path string true "html_template_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteHtmlTemplate(c *gin.Context) {
	htmlTemplateID := c.Param("html_template_id")

	if !util.IsValidUUID(htmlTemplateID) {
		h.handleResponse(c, http.InvalidArgument, "html template id is an invalid uuid")
		return
	}

	resp, err := h.services.HtmlTemplateService().Delete(
		context.Background(),
		&obs.HtmlTemplatePrimaryKey{
			Id: htmlTemplateID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetHtmlTemplateList godoc
// @Security ApiKeyAuth
// @ID get_html_template_list
// @Router /v1/html-template [GET]
// @Summary Get html template list
// @Description Get html template list
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllHtmlTemplateRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetAllHtmlTemplateResponse} "HtmlTemplateBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetHtmlTemplateList(c *gin.Context) {

	resp, err := h.services.HtmlTemplateService().GetList(
		context.Background(),
		&obs.GetAllHtmlTemplateRequest{
			TableSlug: c.Query("table_slug"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
