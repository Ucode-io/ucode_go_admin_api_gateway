package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateHtmlTemplate godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_html_template
// @Router /v1/html-template [POST]
// @Summary Create htmlTemplate
// @Description Create htmlTemplate
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template body object_builder_service.CreateHtmlTemplateRequest true "CreateHtmlTemplateRequestBody"
// @Success 201 {object} status_http.Response{data=object_builder_service.HtmlTemplate} "Html Template data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateHtmlTemplate(c *gin.Context) {
	var htmlTemplate obs.CreateHtmlTemplateRequest

	err := c.ShouldBindJSON(&htmlTemplate)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	htmlTemplate.ProjectId = resourceEnvironment.GetId()

	resp, err := services.HtmlTemplateService().Create(
		context.Background(),
		&htmlTemplate,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetSingleHtmlTemplate godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_html_template_by_id
// @Router /v1/html-template/{html_template_id} [GET]
// @Summary Get single html template
// @Description Get single html template
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template_id path string true "html_template_id"
// @Success 200 {object} status_http.Response{data=object_builder_service.HtmlTemplate} "HtmlTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleHtmlTemplate(c *gin.Context) {
	htmlTemplateID := c.Param("html_template_id")
	if !util.IsValidUUID(htmlTemplateID) {
		h.handleResponse(c, status_http.InvalidArgument, "html template id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.HtmlTemplateService().GetSingle(
		context.Background(),
		&obs.HtmlTemplatePrimaryKey{
			Id:        htmlTemplateID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateHtmlTemplate godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_html_template
// @Router /v1/html-template [PUT]
// @Summary Update html template
// @Description Update html template
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template body object_builder_service.HtmlTemplate true "UpdateHtmlTemplateRequestBody"
// @Success 200 {object} status_http.Response{data=object_builder_service.HtmlTemplate} "HtmlTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateHtmlTemplate(c *gin.Context) {
	var htmlTemplate obs.HtmlTemplate

	err := c.ShouldBindJSON(&htmlTemplate)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	htmlTemplate.ProjectId = resourceEnvironment.GetId()

	resp, err := services.HtmlTemplateService().Update(
		context.Background(),
		&htmlTemplate,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteHtmlTemplate godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_html_template_id
// @Router /v1/html-template/{html_template_id} [DELETE]
// @Summary Delete html template
// @Description Delete html template
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param html_template_id path string true "html_template_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteHtmlTemplate(c *gin.Context) {
	htmlTemplateID := c.Param("html_template_id")

	if !util.IsValidUUID(htmlTemplateID) {
		h.handleResponse(c, status_http.InvalidArgument, "html template id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.HtmlTemplateService().Delete(
		context.Background(),
		&obs.HtmlTemplatePrimaryKey{
			Id:        htmlTemplateID,
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// GetHtmlTemplateList godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_html_template_list
// @Router /v1/html-template [GET]
// @Summary Get html template list
// @Description Get html template list
// @Tags HtmlTemplate
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllHtmlTemplateRequest true "filters"
// @Success 200 {object} status_http.Response{data=object_builder_service.GetAllHtmlTemplateResponse} "HtmlTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetHtmlTemplateList(c *gin.Context) {

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.HtmlTemplateService().GetList(
		context.Background(),
		&obs.GetAllHtmlTemplateRequest{
			TableSlug: c.Query("table_slug"),
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
