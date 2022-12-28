package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateView godoc
// @Security ApiKeyAuth
// @ID create_view
// @Router /v1/view [POST]
// @Summary Create view
// @Description Create view
// @Tags View
// @Accept json
// @Produce json
// @Param view body object_builder_service.CreateViewRequest true "CreateViewRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.View} "View data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateView(c *gin.Context) {
	var view obs.CreateViewRequest

	err := c.ShouldBindJSON(&view)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}
	view.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ViewService().Create(
		context.Background(),
		&view,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetSingleView godoc
// @Security ApiKeyAuth
// @ID get_view_by_id
// @Router /v1/view/{view_id} [GET]
// @Summary Get single view
// @Description Get single view
// @Tags View
// @Accept json
// @Produce json
// @Param view_id path string true "view_id"
// @Success 200 {object} http.Response{data=object_builder_service.View} "ViewBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleView(c *gin.Context) {
	viewID := c.Param("view_id")

	if !util.IsValidUUID(viewID) {
		h.handleResponse(c, http.InvalidArgument, "view id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.ViewService().GetSingle(
		context.Background(),
		&obs.ViewPrimaryKey{
			Id:        viewID,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateView godoc
// @Security ApiKeyAuth
// @ID update_view
// @Router /v1/view [PUT]
// @Summary Update view
// @Description Update view
// @Tags View
// @Accept json
// @Produce json
// @Param view body object_builder_service.View true "UpdateViewRequestBody"
// @Success 200 {object} http.Response{data=object_builder_service.View} "View data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateView(c *gin.Context) {
	var view obs.View

	err := c.ShouldBindJSON(&view)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}
	view.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ViewService().Update(
		context.Background(),
		&view,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteView godoc
// @Security ApiKeyAuth
// @ID delete_view
// @Router /v1/view/{view_id} [DELETE]
// @Summary Delete view
// @Description Delete view
// @Tags View
// @Accept json
// @Produce json
// @Param view_id path string true "view_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteView(c *gin.Context) {
	viewID := c.Param("view_id")

	if !util.IsValidUUID(viewID) {
		h.handleResponse(c, http.InvalidArgument, "view id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.ViewService().Delete(
		context.Background(),
		&obs.ViewPrimaryKey{
			Id:        viewID,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetViewLists godoc
// @Security ApiKeyAuth
// @ID get_view_list
// @Router /v1/view [GET]
// @Summary Get view list
// @Description Get view list
// @Tags View
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllViewsRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetAllViewsResponse} "ViewBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetViewList(c *gin.Context) {

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.ViewService().GetList(
		context.Background(),
		&obs.GetAllViewsRequest{
			TableSlug: c.Query("table_slug"),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// ConvertHtmlToPdf godoc
// @Security ApiKeyAuth
// @ID convert_html_to_pdf
// @Router /v1/html-to-pdf [POST]
// @Summary Convert html to pdf
// @Description Convert html to pdf
// @Tags View
// @Accept json
// @Produce json
// @Param view body models.HtmlBody true "HtmlBody"
// @Success 201 {object} http.Response{data=object_builder_service.PdfBody} "PdfBody data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ConvertHtmlToPdf(c *gin.Context) {
	var html models.HtmlBody

	err := c.ShouldBindJSON(&html)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	structData, err := helper.ConvertMapToStruct(html.Data)

	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.ViewService().ConvertHtmlToPdf(
		context.Background(),
		&obs.HtmlBody{
			Data:      structData,
			Html:      html.Html,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// ConvertTemplateToHtml godoc
// @Security ApiKeyAuth
// @ID convert_template_to_html
// @Router /v1/template-to-html [POST]
// @Summary Convert template to html
// @Description Convert template to html
// @Tags View
// @Accept json
// @Produce json
// @Param view body models.HtmlBody true "TemplateBody"
// @Success 201 {object} http.Response{data=models.HtmlBody} "HtmlBody data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ConvertTemplateToHtml(c *gin.Context) {
	var html models.HtmlBody

	err := c.ShouldBindJSON(&html)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(html.Data)

	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.ViewService().ConvertTemplateToHtml(
		context.Background(),
		&obs.HtmlBody{
			Data:      structData,
			Html:      html.Html,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}
