package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cast"
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
func (h *HandlerV1) CreateRedirectUrl(c *gin.Context) {
	var (
		data pb.RedirectUrl
		res  = &pb.RedirectUrl{}
	)

	err := c.ShouldBindJSON(&data)
	if err != nil {
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
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	data.ProjectId = projectId.(string)
	data.EnvId = environmentId.(string)

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "CREATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo: cast.ToString(userId),
			Request:  &data,
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = res
			h.handleResponse(c, status_http.Created, res)
		}
		go h.versionHistory(c, logReq)
	}()

	res, err = h.companyServices.Redirect().Create(
		context.Background(),
		&data,
	)
	if err != nil {
		return
	}
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
func (h *HandlerV1) GetSingleRedirectUrl(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	id := c.Param("redirect-url-id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "app id is an invalid uuid")
		return
	}

	// namespace := c.GetString("namespace")
	// services, err := h.GetService(namespace)
	// if err != nil {
	// 	h.handleResponse(c, status_http.Forbidden, err)
	// 	return
	// }

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

	res, err := h.companyServices.Redirect().GetSingle(
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
func (h *HandlerV1) UpdateRedirectUrl(c *gin.Context) {
	var (
		data pb.RedirectUrl
		res  = &pb.RedirectUrl{}
	)

	err := c.ShouldBindJSON(&data)
	if err != nil {
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
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	data.ProjectId = projectId.(string)
	data.EnvId = environmentId.(string)

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "UPDATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo: cast.ToString(userId),
			Request:  &data,
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = res
			h.handleResponse(c, status_http.OK, res)
		}
		go h.versionHistory(c, logReq)
	}()

	res, err = h.companyServices.Redirect().Update(
		context.Background(),
		&data,
	)
	if err != nil {
		return
	}
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
func (h *HandlerV1) DeleteRedirectUrl(c *gin.Context) {
	var (
		res = &empty.Empty{}
	)
	id := c.Param("redirect-url-id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "DELETE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo: cast.ToString(userId),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.NoContent, res)
		}
		go h.versionHistory(c, logReq)
	}()

	res, err = h.companyServices.Redirect().Delete(
		context.Background(),
		&pb.DeleteRedirectUrlReq{
			Id: id,
		},
	)

	if err != nil {
		return
	}
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
func (h *HandlerV1) GetListRedirectUrl(c *gin.Context) {
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

	// namespace := c.GetString("namespace")
	// services, err := h.GetService(namespace)
	// if err != nil {
	// 	h.handleResponse(c, status_http.Forbidden, err)
	// 	return
	// }

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

	res, err := h.companyServices.Redirect().GetList(
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
func (h *HandlerV1) UpdateRedirectUrlOrder(c *gin.Context) {
	var (
		data pb.UpdateOrderRedirectUrlReq
	)

	err := c.ShouldBindJSON(&data)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	// namespace := c.GetString("namespace")
	// services, err := h.GetService(namespace)
	// if err != nil {
	// 	h.handleResponse(c, status_http.Forbidden, err)
	// 	return
	// }

	res, err := h.companyServices.Redirect().UpdateOrder(
		context.Background(),
		&data,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
