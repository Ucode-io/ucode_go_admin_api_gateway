package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// Create Query Folder godoc
// @Security ApiKeyAuth
// @ID create_query_folder
// @Router /v3/query_folder [POST]
// @Summary Create Query Folder
// @Description Create Query Folder
// @Tags Query Folder
// @Accept json
// @Produce json
// @Param table body models.CreateQueryFolderRequest true "CreateQueryFolderRequest"
// @Success 201 {object} http.Response{data=models.QueryFolder} "QueryFolder data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateQueryFolder(c *gin.Context) {
	var queryfolder models.CreateQueryFolderRequest

	err := c.ShouldBindJSON(&queryfolder)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := h.companyServices.QueryFolderService().Create(
		context.Background(),
		&obs.CreateQueryFolderRequest{
			Title:     queryfolder.Title,
			ParentId:  queryfolder.ParentId,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetQueryFolderById godoc
// @Security ApiKeyAuth
// @ID get_query_folder
// @Router /v3/query_folder/{guid} [GET]
// @Summary Get Query Folder By Id
// @Description Get Query Folder By Id
// @Tags Query Folder
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Success 200 {object} http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetQueryFolderByID(c *gin.Context) {

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := h.companyServices.QueryFolderService().GetById(
		context.Background(),
		&obs.QueryFolderId{
			Id:        c.Param("guid"),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetQueryFolderList godoc
// @Security ApiKeyAuth
// @ID get_query_folder_list
// @Router /v3/query_folder [GET]
// @Summary Get Query Folder List
// @Description Get Query Folder Lit
// @Tags Query Folder
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllQueryFolderRequest true "filters"
// @Success 200 {object} http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetQueryFolderList(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := h.companyServices.QueryFolderService().GetAll(
		context.Background(),
		&obs.GetAllQueryFolderRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			ParentId:  c.DefaultQuery("parent_id", ""),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateQueryFolder godoc
// @Security ApiKeyAuth
// @ID update_query_folder
// @Router /v3/query_folder/{guid} [PUT]
// @Summary Update Query Folder
// @Description Update Query Folder
// @Tags Query Folder
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Param relation body models.CreateQueryFolderRequest  true "UpdateQueryFolderRequestBody"
// @Success 200 {object} http.Response{data=models.QueryFolder} "QueryFolder data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateQueryFolder(c *gin.Context) {
	var queryFolder models.CreateQueryFolderRequest
	guid := c.Param("guid")

	err := c.ShouldBindJSON(&queryFolder)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := h.companyServices.QueryFolderService().Update(
		context.Background(),
		&obs.QueryFolder{
			Id:        guid,
			Title:     queryFolder.Title,
			ParentId:  queryFolder.ParentId,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteQueryFolder godoc
// @Security ApiKeyAuth
// @ID delete_query_folder
// @Router /v3/query_folder/{guid} [DELETE]
// @Summary Delete Query Folder
// @Description Delete Query Folder
// @Tags Query Folder
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteQueryFolder(c *gin.Context) {
	queryFolderId := c.Param("guid")

	if !util.IsValidUUID(queryFolderId) {
		h.handleResponse(c, http.InvalidArgument, "field id is an invalid uuid")
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := h.companyServices.QueryFolderService().Delete(
		context.Background(),
		&obs.QueryFolderId{
			Id:        queryFolderId,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
