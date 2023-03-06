package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ucode/ucode_go_api_gateway/api/status_http"
	tmp "ucode/ucode_go_api_gateway/genproto/query_service"
	"ucode/ucode_go_api_gateway/pkg/util"
)

// CreateQueryRequestFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_query_request_folder
// @Router /v1/query-folder [POST]
// @Summary Create query request folder
// @Description Create query request folder
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query_folder body tmp.CreateFolderReq true "CreateFolderReq"
// @Success 201 {object} status_http.Response{data=tmp.Folder} "Query folder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateQueryRequestFolder(c *gin.Context) {
	var (
		folder tmp.CreateFolderReq
		//resourceEnvironment *obs.ResourceEnvironment
	)

	err := c.ShouldBindJSON(&folder)
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
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}

	//if util.IsValidUUID(environmentId.(string)) {
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
	//folder.ProjectId = resourceEnvironment.GetId()

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	folder.CommitId = uuID.String()
	folder.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	folder.ProjectId = projectId

	res, err := services.QueryService().Folder().CreateFolder(
		context.Background(),
		&folder,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleQueryRequestFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_single_query_request_folder
// @Router /v1/query-folder/{query-folder-id} [GET]
// @Summary Get single query request folder
// @Description Get single query request folder
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query-folder-id path string true "query-folder-id"
// @Success 200 {object} status_http.Response{data=tmp.GetSingleFolderRes} "GetSingleFolderRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleQueryRequestFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	folderId := c.Param("query-folder-id")

	if !util.IsValidUUID(folderId) {
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
	//if util.IsValidUUID(environmentId.(string)) {
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

	res, err := services.QueryService().Folder().GetSingleFolder(
		context.Background(),
		&tmp.GetSingleFolderReq{
			Id:        folderId,
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

// UpdateQueryRequestFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_query_request_folder
// @Router /v1/query-folder [PUT]
// @Summary Update query request folder
// @Description Update query folder
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param folder body tmp.UpdateFolderReq true "UpdateFolderReq"
// @Success 200 {object} status_http.Response{data=tmp.Folder} "Folder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateQueryRequestFolder(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		folder tmp.UpdateFolderReq
	)

	err := c.ShouldBindJSON(&folder)
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
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}
	//
	//if util.IsValidUUID(environmentId.(string)) {
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
	//folder.ProjectId = resourceEnvironment.GetId()

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	folder.CommitId = uuID.String()
	folder.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"
	folder.ProjectId = projectId

	res, err := services.QueryService().Folder().UpdateFolder(
		context.Background(),
		&folder,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteQueryRequestFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_query_request_folder
// @Router /v1/query-folder/{query-folder-id} [DELETE]
// @Summary Delete query folder
// @Description Delete query folder
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param query-folder-id path string true "query-folder-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteQueryRequestFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	folderId := c.Param("query-folder-id")

	if !util.IsValidUUID(folderId) {
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
	//if util.IsValidUUID(environmentId.(string)) {
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

	res, err := services.QueryService().Folder().DeleteFolder(
		context.Background(),
		&tmp.DeleteFolderReq{
			Id:        folderId,
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

// GetListQueryRequestFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_query_request_folder
// @Router /v1/query-folder [GET]
// @Summary Get List query folder
// @Description Get List query folder
// @Tags Query
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Success 200 {object} status_http.Response{data=tmp.GetListFolderRes} "FolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListQueryRequestFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)

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
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}
	//
	//if util.IsValidUUID(environmentId.(string)) {
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

	res, err := services.QueryService().Folder().GetListFolder(
		context.Background(),
		&tmp.GetListFolderReq{
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
