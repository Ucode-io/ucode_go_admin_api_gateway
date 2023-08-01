package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateRedirectUrl godoc
// @Security ApiKeyAuth
// @ID create_redirect_url
// @Router /v1/redirect-url [POST]
// @Summary Create redirect url
// @Description Create redirect url
// @Tags RedirectUrl
// @Accept json
// @Produce json
// @Param data body pb.RedirectUrl true "CreateRedirectUrl"
// @Success 201 {object} status_http.Response{data=pb.RedirectUrl} "Redirect Url response"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateRedirectUrl(c *gin.Context) {
	var (
		data pb.RedirectUrl
		//resourceEnvironment *obs.ResourceEnvironment
	)

	err := c.ShouldBindJSON(&data)
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

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//
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
	data.ProjectId = projectId.(string)
	data.EnvId = environmentId.(string)

	res, err := services.CompanyService().Redirect().Create(
		context.Background(),
		&data,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleRedirectUrl godoc
// @Security ApiKeyAuth
// @ID get_single_redirect_url
// @Router /v1/redirect-url/{redirect-url-id} [GET]
// @Summary Get single redirect url
// @Description Get single redirect url
// @Tags RedirectUrl
// @Accept json
// @Produce json
// @Param redirect-url-id path string true "redirect-url-id"
// @Success 200 {object} status_http.Response{data=pb.RedirectUrl} "GetSingleRedirectUrl response"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleRedirectUrl(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	id := c.Param("redirect-url-id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "app id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}
	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	res, err := services.CompanyService().Redirect().GetSingle(
		context.Background(),
		&pb.GetSingleRedirectUrlReq{
			Id: id,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateRedirectUrl godoc
// @Security ApiKeyAuth
// @ID update_redirect_url
// @Router /v1/redirect-url [PUT]
// @Summary Update redirect url
// @Description Update redirect url
// @Tags RedirectUrl
// @Accept json
// @Produce json
// @Param data body pb.RedirectUrl true "UpdateRedirectUrl"
// @Success 200 {object} status_http.Response{data=pb.RedirectUrl} "RedirectUrl response"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateRedirectUrl(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		data pb.RedirectUrl
	)

	err := c.ShouldBindJSON(&data)
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

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//
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

	data.ProjectId = projectId.(string)
	data.EnvId = environmentId.(string)

	res, err := services.CompanyService().Redirect().Update(
		context.Background(),
		&data,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteRedirectUrl godoc
// @Security ApiKeyAuth
// @ID delete_redirect_url
// @Router /v1/redirect-url/{redirect-url-id} [DELETE]
// @Summary Delete redirect url
// @Description Delete redirect url
// @Tags RedirectUrl
// @Accept json
// @Produce json
// @Param redirect-url-id path string true "redirect-url-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteRedirectUrl(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	id := c.Param("redirect-url-id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}
	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	res, err := services.CompanyService().Redirect().Delete(
		context.Background(),
		&pb.DeleteRedirectUrlReq{
			Id: id,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListRedirectUrl godoc
// @Security ApiKeyAuth
// @ID get_list_redirect_url
// @Router /v1/redirect-url [GET]
// @Summary Get List redirect url
// @Description Get List redirect url
// @Tags RedirectUrl
// @Accept json
// @Produce json
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=pb.GetListRedirectUrlRes} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListRedirectUrl(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//
	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	res, err := services.CompanyService().Redirect().GetList(
		context.Background(),
		&pb.GetListRedirectUrlReq{
			ProjectId: projectId.(string),
			EnvId:     environmentId.(string),
			Offset:    int32(offset),
			Limit:     int32(limit),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateRedirectUrlOrder godoc
// @Security ApiKeyAuth
// @ID update_redirect_url_order
// @Router /v1/redirect-url/re-order [PUT]
// @Summary Update redirect url order
// @Description Update redirect url order
// @Tags RedirectUrl
// @Accept json
// @Produce json
// @Param data body pb.UpdateOrderRedirectUrlReq true "UpdateRedirectUrlOrder"
// @Success 200 {object} status_http.Response{data=string} "Update RedirectUrl Order response"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateRedirectUrlOrder(c *gin.Context) {
	var (
		data pb.UpdateOrderRedirectUrlReq
	)

	err := c.ShouldBindJSON(&data)
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

	res, err := services.CompanyService().Redirect().UpdateOrder(
		context.Background(),
		&data,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
