package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strconv"
	"ucode/ucode_go_api_gateway/api/status_http"
	tmp "ucode/ucode_go_api_gateway/genproto/query_service"
	"ucode/ucode_go_api_gateway/pkg/util"
)

// CreateQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_query_request
// @Router /v1/query-request [POST]
// @Summary Create query
// @Description Create query
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query body tmp.CreateQueryReq true "CreateQueryReq"
// @Success 201 {object} status_http.Response{data=tmp.Query} "Query data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateQueryRequest(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		query tmp.CreateQueryReq
	)

	err := c.ShouldBindJSON(&query)
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

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	//
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	//template.ProjectId = resourceEnvironment.GetId()

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query.CommitId = uuID.String()
	query.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	query.ProjectId = projectId
	query.EnvironmentId = environmentId.(string)

	res, err := services.QueryService().Query().CreateQuery(
		context.Background(),
		&query,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_single_query_request
// @Router /v1/query-request/{query-id} [GET]
// @Summary Get single query
// @Description Get single query
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query-id path string true "query-id"
// @Success 200 {object} status_http.Response{data=tmp.Query} "Query"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleQueryRequest(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	queryId := c.Param("query-id")

	if !util.IsValidUUID(queryId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}
	//
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.QueryService().Query().GetSingleQuery(
		context.Background(),
		&tmp.GetSingleQueryReq{
			Id:        queryId,
			ProjectId: projectId,
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_query_request
// @Router /v1/query-request [PUT]
// @Summary Update query
// @Description Update query
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query body tmp.UpdateQueryReq true "UpdateQueryReq"
// @Success 200 {object} status_http.Response{data=tmp.Query} "Query data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateQueryRequest(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		query tmp.UpdateQueryReq
	)

	err := c.ShouldBindJSON(&query)
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

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	//template.ProjectId = resourceEnvironment.GetId()

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query.CommitId = uuID.String()
	query.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	query.ProjectId = projectId
	query.EnvironmentId = environmentId.(string)

	res, err := services.QueryService().Query().UpdateQuery(
		context.Background(),
		&query,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_query_request
// @Router /v1/query-request/{query-id} [DELETE]
// @Summary Delete query
// @Description Delete query
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query-id path string true "query-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteQueryRequest(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	queryId := c.Param("query-id")

	if !util.IsValidUUID(queryId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}
	//
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.QueryService().Query().DeleteQuery(
		context.Background(),
		&tmp.DeleteQueryReq{
			Id:        queryId,
			ProjectId: projectId,
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_query_request
// @Router /v1/query-request [GET]
// @Summary Get List query
// @Description Get List query
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param folder-id query string true "folder-id"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListQueryRes} "GetListQueryRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListQueryRequest(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)

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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}
	//
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.QueryService().Query().GetListQuery(
		context.Background(),
		&tmp.GetListQueryReq{
			ProjectId: projectId,
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			FolderId:  c.DefaultQuery("folder-id", ""),
			Limit:     int32(limit),
			Offset:    int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// QueryRun godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID run_query_request
// @Router /v1/query-request/run [POST]
// @Summary Run query
// @Description Run query
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query body tmp.Query true "Query"
// @Success 201 {object} status_http.Response{data=tmp.RunQueryRes} "RunQueryRes data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) QueryRun(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		query tmp.Query
	)

	err := c.ShouldBindJSON(&query)
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

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}
	//
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	//template.ProjectId = resourceEnvironment.GetId()

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query.CommitId = uuID.String()
	query.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	query.ProjectId = projectId
	query.EnvironmentId = environmentId.(string)

	res, err := services.QueryService().Query().RunQuery(
		context.Background(),
		&query,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}
