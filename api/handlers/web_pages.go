package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateWebPage godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_web_page
// @Router /v3/web_pages [POST]
// @Summary Create Web Page
// @Description Create Web Page
// @Tags Web Page
// @Accept json
// @Produce json
// @Param table body models.CreateWebPageRequest true "CreateWebPageRequest"
// @Success 201 {object} status_http.Response{data=models.QueryFolder} "QueryFolder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateWebPage(c *gin.Context) {
	var webPage models.CreateWebPageRequest

	err := c.ShouldBindJSON(&webPage)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	components, err := helper.ConvertMapToStruct(webPage.Components)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := h.companyServices.WebPageService().Create(
		context.Background(),
		&obs.CreateWebPageRequest{
			Title:      webPage.Title,
			Components: components,
			ProjectId:  resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetWebPagesById godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_web_page_by_id
// @Router /v3/web_pages/{guid} [GET]
// @Summary Get Web Page By Id
// @Description Get Web Page By Id
// @Tags Web Page
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Success 200 {object} status_http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetWebPagesById(c *gin.Context) {

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err := errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.WebPageService().GetById(
		context.Background(),
		&obs.WebPageId{
			Id:        c.Param("guid"),
			ProjectId: resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetWebPagesList godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_web_page_list
// @Router /v3/web_pages [GET]
// @Summary Get Web Page List
// @Description Get Web Page List
// @Tags Web Page
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllWebPagesRequest true "filters"
// @Success 200 {object} status_http.Response{data=models.GetAllFieldsResponse} "WebPageBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetWebPagesList(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := h.companyServices.WebPageService().GetAll(
		context.Background(),
		&obs.GetAllWebPagesRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateWebPage godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_web_page
// @Router /v3/web_pages/{guid} [PUT]
// @Summary Update Web Page
// @Description Update Web Page
// @Tags Web Page
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Param web_page body models.CreateWebPageRequest  true "UpdateWebPageRequestBody"
// @Success 200 {object} status_http.Response{data=models.WebPage} "WebPage data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateWebPage(c *gin.Context) {
	var updateWebPage models.CreateWebPageRequest
	guid := c.Param("guid")

	err := c.ShouldBindJSON(&updateWebPage)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	components, err := helper.ConvertMapToStruct(updateWebPage.Components)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := h.companyServices.WebPageService().Update(
		context.Background(),
		&obs.WebPage{
			Id:         guid,
			Title:      updateWebPage.Title,
			Components: components,
			ProjectId:  resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteWebPage godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_web_page
// @Router /v3/web_pages/{guid} [DELETE]
// @Summary Delete Query Folder
// @Description Delete Web Page
// @Tags Web Page
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteWebPage(c *gin.Context) {
	webPageId := c.Param("guid")

	if !util.IsValidUUID(webPageId) {
		h.handleResponse(c, status_http.InvalidArgument, "field id is an invalid uuid")
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err := errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.WebPageService().Delete(
		context.Background(),
		&obs.WebPageId{
			Id:        webPageId,
			ProjectId: resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
