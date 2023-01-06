package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// Create Query godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID create_query
// @Router /v3/query [POST]
// @Summary Create Query
// @Description Create Query
// @Tags Queries
// @Accept json
// @Produce json
// @Param table body models.CreateQueryRequest true "CreateQueryRequest"
// @Success 201 {object} http.Response{data=models.Queries} "Queries data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateQuery(c *gin.Context) {
	var query models.CreateQueryRequest

	err := c.ShouldBindJSON(&query)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(query.Attributes)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.QueriesService().Create(
		context.Background(),
		&obs.CreateQueryRequest{
			Title:         query.Title,
			QueryFolderId: query.QueryFolderId,
			Attributes:    attributes,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetQueryById godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_query_item
// @Router /v3/query/{guid} [GET]
// @Summary Get Query By Id
// @Description Get Query By Id
// @Tags Queries
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Success 200 {object} http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetQueryByID(c *gin.Context) {
	guid := c.Param("guid")
	resp, err := h.companyServices.QueriesService().GetById(
		context.Background(),
		&obs.QueryId{
			Id: guid,
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
// @Param Resource-Id header string true "Resource-Id"
// @ID get_query_list
// @Router /v3/query [GET]
// @Summary Get Query Folder List
// @Description Get Query Folder Lit
// @Tags Queries
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllQueriesRequest true "filters"
// @Success 200 {object} http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetQueryList(c *gin.Context) {
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.QueriesService().GetAll(
		context.Background(),
		&obs.GetAllQueriesRequest{
			Limit:         int32(limit),
			Offset:        int32(offset),
			Search:        c.DefaultQuery("search", ""),
			QueryFolderId: c.DefaultQuery("query_folder_id", ""),
			ProjectId:     resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateQuery godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID update_field
// @Router /v3/query/{guid} [PUT]
// @Summary Update Query
// @Description Update Query
// @Tags Queries
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Param relation body models.CreateQueryRequest  true "UpdateQueryRequestBody"
// @Success 200 {object} http.Response{data=models.Queries} "Queries data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateQuery(c *gin.Context) {
	var query models.CreateQueryRequest
	guid := c.Param("guid")

	err := c.ShouldBindJSON(&query)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(query.Attributes)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.QueriesService().Update(
		context.Background(),
		&obs.Query{
			Id:            guid,
			Title:         query.Title,
			QueryFolderId: query.QueryFolderId,
			Attributes:    attributes,
			ProjectId:     resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteQuery godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID delete_query
// @Router /v3/query/{guid} [DELETE]
// @Summary Delete Query
// @Description Delete Query
// @Tags Queries
// @Accept json
// @Produce json
// @Param guid path string true "guid"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteQuery(c *gin.Context) {
	queryId := c.Param("guid")

	if !util.IsValidUUID(queryId) {
		h.handleResponse(c, http.InvalidArgument, "field id is an invalid uuid")
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err := errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.QueryFolderService().Delete(
		context.Background(),
		&obs.QueryFolderId{
			Id:        queryId,
			ProjectId: resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
