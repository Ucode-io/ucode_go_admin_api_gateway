package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strconv"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	tmp "ucode/ucode_go_api_gateway/genproto/query_service"
	vcs "ucode/ucode_go_api_gateway/genproto/versioning_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
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

// GetQueryHistory godoc
// @Security ApiKeyAuth
// @ID get_query_history
// @Router /v1/query-request/{query-id}/history [GET]
// @Summary Get Api query history
// @Description Get query history
// @Tags Query
// @Accept json
// @Produce json
// @Param query-id path string true "query-id"
// @Param project-id query string true "project-id"
// @Param limit query string true "limit"
// @Param offset query string true "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetQueryHistoryRes} "QueryHistory"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetQueryHistory(c *gin.Context) {
	id := c.Param("query-id")
	projectId := c.Query("project-id")

	if !util.IsValidUUID(id) {
		err := errors.New("query is an invalid uuid")
		h.log.Error("query is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "query is an invalid uuid")
		return
	}

	if !util.IsValidUUID(projectId) {
		err := errors.New("project_id is an invalid uuid")
		h.log.Error("project_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.log.Error("error getting limit param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.log.Error("error getting offset param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := services.QueryService().Query().GetQueryHistory(
		context.Background(),
		&tmp.GetQueryHistoryReq{
			Id:        id,
			ProjectId: projectId,
			Offset:    int64(offset),
			Limit:     int64(limit),
		},
	)
	if err != nil {
		h.log.Error("error getting query history", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		versionIds []string
		// autherGuids []string
	)
	for _, item := range resp.GetQueries() {
		versionIds = append(versionIds, item.GetCommitInfo().GetVersionIds()...)
	}

	multipleVersionResp, err := services.VersioningService().Release().GetMultipleVersionInfo(
		c.Request.Context(),
		&vcs.GetMultipleVersionInfoRequest{
			VersionIds: versionIds,
			ProjectId:  projectId,
		},
	)
	if err != nil {
		h.log.Error("error getting multiple version infos", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, item := range resp.GetQueries() {
		for key := range item.GetVersionInfos() {

			versionInfoData := multipleVersionResp.GetVersionInfos()[key]

			item.VersionInfos[key] = &tmp.VersionInfo{
				VersionId: versionInfoData.GetVersionId(),
				AuthorId:  versionInfoData.GetAuthorId(),
				Version:   versionInfoData.GetVersion(),
				Desc:      versionInfoData.GetDesc(),
				CreatedAt: versionInfoData.GetCreatedAt(),
				UpdatedAt: versionInfoData.GetUpdatedAt(),
				IsCurrent: versionInfoData.GetIsCurrent(),
			}
		}
	}
	// reqAutherGuids, err := helper.ConvertMapToStruct(map[string]interface{}{
	// 	"guid": []string{},
	// })
	// if err != nil {
	// 	h.log.Error("error converting map to struct", logger.Error(err))
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	// // Get User Data
	// respAuthersData, err := services.BuilderService().ObjectBuilder().GetList(
	// 	c.Request.Context(),

	// 	&builder.CommonMessage{
	// 		TableSlug: "users",
	// 		Data:      reqAutherGuids,
	// 	},
	// )
	// if err != nil {
	// 	h.log.Error("error getting user data", logger.Error(err))
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	h.handleResponse(c, status_http.OK, resp)
}

// RevertQuery godoc
// @Security ApiKeyAuth
// @Param Environment-Id header string true "Environment-Id"
// @ID revert_query
// @Router /v1/query-request/{query-id}/revert [POST]
// @Summary Revert query
// @Description Revert query
// @Tags Query
// @Accept json
// @Produce json
// @Param query-id path string true "query-id"
// @Param project-id query string true "project-id"
// @Param RevertQueryReq body tmp.RevertQueryReq true "Request Body"
// @Success 200 {object} status_http.Response{data=tmp.Query} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RevertQuery(c *gin.Context) {

	var body tmp.RevertQueryReq

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("error binding json", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	id := c.Param("query-id")

	if !util.IsValidUUID(id) {
		err := errors.New("query id is an invalid uuid")
		h.log.Error("query is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "query is an invalid uuid")
		return
	}

	if !util.IsValidUUID(body.GetProjectId()) {
		err := errors.New("project_id is an invalid uuid")
		h.log.Error("project_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}

	resp, err := services.QueryService().Query().RevertQuery(
		context.Background(),
		&tmp.RevertQueryReq{
			Id:          id,
			VersionId:   versionGuid,
			OldCommitId: body.GetOldCommitId(),
			NewCommitId: commitGuid,
		},
	)
	if err != nil {
		h.log.Error("error reverting query", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

//// InsertManyVersionForApiReference godoc
//// @Security ApiKeyAuth
//// @ID insert_many_api_reference
//// @Router /v1/api-reference/select-versions/{api_reference_id} [POST]
//// @Summary Select Api Reference
//// @Description Select Api Reference
//// @Tags ApiReference
//// @Accept json
//// @Produce json
//// @Param api_reference_id path string true "api_reference_id"
//// @Param Environment-Id header string true "Environment-Id"
//// @Param body body api_reference_service.ApiManyVersions true "Request Body"
//// @Success 200 {object} status_http.Response{data=ars.ApiReference} "Response Body"
//// @Response 400 {object} status_http.Response{data=string} "Bad Request"
//// @Failure 500 {object} status_http.Response{data=string} "Server Error"
//func (h *Handler) InsertManyVersionForApiReference(c *gin.Context) {
//
//	body := ars.ManyVersions{}
//	err := c.ShouldBindJSON(&body)
//	if err != nil {
//		h.handleResponse(c, status_http.BadRequest, err.Error())
//		return
//	}
//
//	log.Printf("API->body: %+v", body)
//
//	environmentID, ok := c.Get("environment_id")
//	if !ok {
//		err = errors.New("error getting environment id")
//		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
//		return
//	}
//
//	if !util.IsValidUUID(environmentID.(string)) {
//		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
//		return
//	}
//
//	api_reference_id := c.Param("api_reference_id")
//	if !util.IsValidUUID(api_reference_id) {
//		err := errors.New("api_reference_id is an invalid uuid")
//		h.log.Error("api_reference_id is an invalid uuid", logger.Error(err))
//		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
//		return
//	}
//
//	if !util.IsValidUUID(body.GetProjectId()) {
//		err := errors.New("project_id is an invalid uuid")
//		h.log.Error("project_id is an invalid uuid", logger.Error(err))
//		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
//		return
//	}
//
//	namespace := c.GetString("namespace")
//	services, err := h.GetService(namespace)
//	if err != nil {
//		h.log.Error("error getting service", logger.Error(err))
//		h.handleResponse(c, status_http.Forbidden, err)
//		return
//	}
//
//	body.EnvironmentId = environmentID.(string)
//	body.Guid = api_reference_id
//
//	// _, commitId, err := h.CreateAutoCommitForAdminChange(c, environmentID.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
//	// if err != nil {
//	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
//	// 	return
//	// }
//
//	resp, err := services.ApiReferenceService().ApiReference().CreateManyApiReference(c.Request.Context(), &body)
//	if err != nil {
//		h.handleResponse(c, status_http.GRPCError, err.Error())
//
//		return
//	}
//
//	h.handleResponse(c, status_http.OK, resp)
//}

// GetSingleQueryLog godoc
// @Security ApiKeyAuth
// @Param Environment-Id header string true "Environment-Id"
// @ID get_single_query_log
// @Router /v1/query-request/{query-id}/log/{log-id} [GET]
// @Summary get single log
// @Description get single log
// @Tags Query
// @Accept json
// @Produce json
// @Param query-id path string true "query-id"
// @Param log-id path string true "log-id"
// @Param project-id query string true "project-id"
// @Success 200 {object} status_http.Response{data=tmp.Log} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleQueryLog(c *gin.Context) {

	queryId := c.Param("query-id")
	if !util.IsValidUUID(queryId) {
		err := errors.New("query id is an invalid uuid")
		h.log.Error("query is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "query is an invalid uuid")
		return
	}

	logId := c.Param("log-id")
	if !util.IsValidUUID(logId) {
		err := errors.New("log id is an invalid uuid")
		h.log.Error("log is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "log is an invalid uuid")
		return
	}

	projectId := c.DefaultQuery("project-id", "")
	if !util.IsValidUUID(projectId) {
		err := errors.New("project_id is an invalid uuid")
		h.log.Error("project_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	resp, err := services.QueryService().Log().GetSingleLog(
		context.Background(),
		&tmp.GetSingleLogReq{
			Id:            logId,
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
		},
	)
	if err != nil {
		h.log.Error("error reverting query", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListQueryLog godoc
// @Security ApiKeyAuth
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_query_log
// @Router /v1/query-request/{query-id}/log [GET]
// @Summary get list log
// @Description get list log
// @Tags Query
// @Accept json
// @Produce json
// @Param query-id path string true "query-id"
// @Param project-id query string true "project-id"
// @Param limit query string true "limit"
// @Param offset query string true "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListLogRes} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListQueryLog(c *gin.Context) {

	queryId := c.Param("query-id")
	if !util.IsValidUUID(queryId) {
		err := errors.New("query id is an invalid uuid")
		h.log.Error("query is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "query is an invalid uuid")
		return
	}

	projectId := c.DefaultQuery("project-id", "")
	if !util.IsValidUUID(projectId) {
		err := errors.New("project_id is an invalid uuid")
		h.log.Error("project_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.log.Error("error getting limit param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.log.Error("error getting offset param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	resp, err := services.QueryService().Log().GetListLog(
		context.Background(),
		&tmp.GetListLogReq{
			QueryId:       queryId,
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			Limit:         int64(limit),
			Offset:        int64(offset),
		},
	)
	if err != nil {
		h.log.Error("error reverting query", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
