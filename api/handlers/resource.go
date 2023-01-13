package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

// GetResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_resource_id
// @Router /v1/company/project/resource/{resource_id} [GET]
// @Summary Get Resource by id
// @Description Get Resource by id
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param resource_id path string true "resource_id"
// @Success 200 {object} http.Response{data=company_service.ResourceWithoutPassword} "Resource data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetResource(c *gin.Context) {

	resp, err := h.companyServices.ResourceService().GetResource(
		context.Background(),
		&company_service.GetResourceRequest{
			Id: c.Param("resource_id"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// AddProjectResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID add_project_resource
// @Router /v1/company/project/resource [POST]
// @Summary Add ProjectResource
// @Description Add ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.AddResourceRequest true "ProjectResourceAddRequest"
// @Success 201 {object} http.Response{data=company_service.AddResourceResponse} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) AddProjectResource(c *gin.Context) {
	var company company_service.AddResourceRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().AddResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// ConfigureProjectResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID configure_project_resource
// @Router /v1/company/project/configure-resource [POST]
// @Summary Configure ProjectResource
// @Description Configure ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.ConfigureResourceRequest true "ProjectResourceConfigureRequest"
// @Success 201 {object} http.Response{data=company_service.ConfigureResourceResponse} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ConfigureProjectResource(c *gin.Context) {
	var company company_service.ConfigureResourceRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().ConfigureResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// CreateProjectResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID create_project_resource
// @Router /v1/company/project/create-resource [POST]
// @Summary Create ProjectResource
// @Description Create ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.CreateResourceReq true "ProjectResourceCreateRequest"
// @Success 201 {object} http.Response{data=company_service.CreateResourceRes} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateProjectResource(c *gin.Context) {
	var company company_service.CreateResourceReq

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().CreateResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// RemoveProjectResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID remove_project_resource
// @Router /v1/company/project/resource [DELETE]
// @Summary Remove ProjectResource
// @Description Remove ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.RemoveResourceRequest true "ProjectResourceRemoveRequest"
// @Success 201 {object} http.Response{data=company_service.EmptyProto} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) RemoveProjectResource(c *gin.Context) {
	var company company_service.RemoveResourceRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().RemoveResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// UpdateResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID put_resource_id
// @Router /v1/company/project/resource/{resource_id} [PUT]
// @Summary Update Resource by id
// @Description Update Resource by id
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param UpdateResourceRequestBody body company_service.UpdateResourceRequest  true "UpdateResourceRequestBody"
// @Success 200 {object} http.Response{data=company_service.ResourceWithoutPassword} "Resource data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateResource(c *gin.Context) {
	var resource company_service.UpdateResourceRequest

	err := c.ShouldBindJSON(&resource)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().UpdateResource(
		c.Request.Context(),
		&resource,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetResourceList godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_resource_list
// @Router /v1/company/project/resource [GET]
// @Summary Get all companies
// @Description Get all companies
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param filters query company_service.GetReourceListRequest true "filters"
// @Success 200 {object} http.Response{data=company_service.GetReourceListResponse} "Resource data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetResourceList(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().GetReourceList(
		context.Background(),
		&company_service.GetReourceListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: c.DefaultQuery("project_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// ReconnectProjectResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID reconnect_project_resource
// @Router /v1/company/project/resource/reconnect [POST]
// @Summary Reconnect ProjectResource
// @Description Reconnect ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.ReconnectResourceRequest true "ProjectResourceReconnectRequest"
// @Success 201 {object} http.Response{data=company_service.EmptyProto} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ReconnectProjectResource(c *gin.Context) {
	var company company_service.ReconnectResourceRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ResourceService().ReconnectResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetResourceEnvironment godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_resource_environment_id
// @Router /v1/company/project/resource-environment/{resource_id} [GET]
// @Summary Get Resource Environment by id
// @Description Get Resource Environment by id
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param resource_id path string true "resource_id"
// @Success 200 {object} http.Response{data=company_service.ResourceWithoutPassword} "Resource data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetResourceEnvironment(c *gin.Context) {

	resp, err := h.companyServices.ResourceService().GetResourceByResEnvironId(
		context.Background(),
		&company_service.GetResourceRequest{
			Id: c.Param("resource_id"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
