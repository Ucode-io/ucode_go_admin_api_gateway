package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/query_service"
	vcs "ucode/ucode_go_api_gateway/genproto/versioning_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
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

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

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

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	query.ProjectId = projectId
	query.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query.CommitId = uuID.String()
	query.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	query.EnvironmentId = environmentId.(string)

	query.CommitInfo = &tmp.CommitInfo{
		Id:         "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Create api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  query.GetProjectId(),
	}

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
// @Param Resource-Id header string false "Resource-Id"
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
// @Param commit-id query string false "commit-id"
// @Success 200 {object} status_http.Response{data=tmp.Query} "Query"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleQueryRequest(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	queryId := c.Param("query-id")
	commitId := c.Query("commit_id")
	versionId := c.Query("version_id")

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

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
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
			Id:         queryId,
			ProjectId:  projectId,
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  versionId,
			CommitId:   commitId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	versions, err := services.VersioningService().Release().GetMultipleVersionInfo(context.Background(), &vcs.GetMultipleVersionInfoRequest{
		VersionIds: res.CommitInfo.VersionIds,
		ProjectId:  res.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	versionInfos := make([]*tmp.VersionInfo, 0, len(res.GetCommitInfo().GetVersionIds()))
	for _, id := range res.CommitInfo.VersionIds {
		versionInfo, ok := versions.VersionInfos[id]
		if ok {
			versionInfos = append(versionInfos, &tmp.VersionInfo{
				AuthorId:  versionInfo.AuthorId,
				CreatedAt: versionInfo.CreatedAt,
				UpdatedAt: versionInfo.UpdatedAt,
				Desc:      versionInfo.Desc,
				IsCurrent: versionInfo.IsCurrent,
				Version:   versionInfo.Version,
				VersionId: versionInfo.VersionId,
			})
		}
	}
	res.CommitInfo.VersionInfos = versionInfos

	h.handleResponse(c, status_http.OK, res)
}

// UpdateQueryRequest godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
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

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	query.ProjectId = projectId
	query.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query.CommitId = uuID.String()
	query.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	query.EnvironmentId = environmentId.(string)
	//activeVersion, err := services.VersioningService().Release().GetCurrentActive(
	//	c.Request.Context(),
	//	&vcs.GetCurrentReleaseRequest{
	//		EnvironmentId: environmentId.(string),
	//	},
	//)
	//if err != nil {
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	//query.VersionId = activeVersion.GetVersionId()
	query.CommitInfo = &tmp.CommitInfo{
		Id:         "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Update api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  query.GetProjectId(),
	}

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
// @Param Resource-Id header string false "Resource-Id"
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
		h.handleResponse(c, status_http.InvalidArgument, "query id is an invalid uuid")
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

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
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
			Id:         queryId,
			ProjectId:  projectId,
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  uuid.NewString(),
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
// @Param Resource-Id header string false "Resource-Id"
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

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
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
			ProjectId:  projectId,
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "",
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

// QueryRun godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
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

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	query.ResourceId = resource.ResourceEnvironmentId
	query.ProjectId = projectId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query.CommitId = uuID.String()
	query.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
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
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	resp, err := services.QueryService().Query().GetQueryHistory(
		context.Background(),
		&tmp.GetQueryHistoryReq{
			Id:         id,
			ProjectId:  projectId,
			ResourceId: resource.ResourceEnvironmentId,
			Offset:     int64(offset),
			Limit:      int64(limit),
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
// @Param Resource-Id header string false "Resource-Id"
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
// @Param RevertQueryReq body models.QueryRevertRequest true "Request Body"
// @Success 200 {object} status_http.Response{data=tmp.Query} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RevertQuery(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		body models.QueryRevertRequest
	)

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			ProjectId:  body.ProjectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, body.ProjectId)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}

	resp, err := services.QueryService().Query().RevertQuery(
		context.Background(),
		&tmp.RevertQueryReq{
			Id:          id,
			VersionId:   versionGuid,
			OldCommitId: body.CommitId,
			NewCommitId: commitGuid,
			ProjectId:   projectId,
			ResourceId:  resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.log.Error("error reverting query", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// InsertManyVersionForQueryService godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID insert_many_query_reference
// @Router /v1/query-request/select-versions/{query-id} [POST]
// @Summary Insert Many query
// @Description Insert Many query
// @Tags Query
// @Accept json
// @Produce json
// @Param query-id path string true "query-id"
// @Param project-id query string true "project-id"
// @Param data body tmp.QueryManyVersions true "Request Body"
// @Success 200 {object} status_http.Response{data=string} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) InsertManyVersionForQueryService(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	body := tmp.ManyVersions{}
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	queryId := c.Param("query-id")
	if !util.IsValidUUID(queryId) {
		err := errors.New("query-id is an invalid uuid")
		h.log.Error("query-id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "query-id is an invalid uuid")
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			ProjectId:  body.ProjectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	body.ProjectId = projectId
	body.ResourceId = resource.ResourceEnvironmentId
	body.EnvironmentId = environmentId.(string)
	body.Id = queryId

	// _, commitId, err := h.CreateAutoCommitForAdminChange(c, environmentID.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }

	resp, err := services.QueryService().Query().CreateManyQuery(c.Request.Context(), &body)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetSingleQueryLog godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
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
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	resp, err := services.QueryService().Log().GetSingleLog(
		context.Background(),
		&tmp.GetSingleLogReq{
			Id:            logId,
			ProjectId:     projectId,
			ResourceId:    resource.ResourceEnvironmentId,
			EnvironmentId: environmentId.(string),
		},
	)
	if err != nil {
		h.log.Error("error get single log query", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListQueryLog godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
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
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_QUERY_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	resp, err := services.QueryService().Log().GetListLog(
		context.Background(),
		&tmp.GetListLogReq{
			QueryId:       queryId,
			ProjectId:     projectId,
			ResourceId:    resource.ResourceEnvironmentId,
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
