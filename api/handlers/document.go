package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateDocument godoc
// @Security ApiKeyAuth
// @ID create_document
// @Router /v1/document [POST]
// @Summary Create Document
// @Description Create Document
// @Tags Document
// @Accept json
// @Produce json
// @Param Document body object_builder_service.CreateDocumentRequest true "CreateDocumentRequestBody"
// @Success 201 {object} http.Response{data=object_builder_service.Document} "Document data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateDocument(c *gin.Context) {
	var document obs.CreateDocumentRequest

	err := c.ShouldBindJSON(&document)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}
	document.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.DocumentService().Create(
		context.Background(),
		&document,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetSingleDocument godoc
// @Security ApiKeyAuth
// @ID get_document_by_id
// @Router /v1/document/{document_id} [GET]
// @Summary Get single document
// @Description Get single document
// @Tags Document
// @Accept json
// @Produce json
// @Param document_id path string true "document_id"
// @Success 200 {object} http.Response{data=object_builder_service.Document} "DocumentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleDocument(c *gin.Context) {
	documentID := c.Param("document_id")

	if !util.IsValidUUID(documentID) {
		h.handleResponse(c, http.InvalidArgument, "Document id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.DocumentService().GetSingle(
		context.Background(),
		&obs.DocumentPrimaryKey{
			Id:        documentID,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if resp == nil {
		err := errors.New("Not Found")
		h.handleResponse(c, http.NoContent, err.Error())
		return
	}
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateDocument godoc
// @Security ApiKeyAuth
// @ID update_document
// @Router /v1/document [PUT]
// @Summary Update Document
// @Description Update Document
// @Tags Document
// @Accept json
// @Produce json
// @Param Document body object_builder_service.Document true "UpdateDocumentRequestBody"
// @Success 200 {object} http.Response{data=object_builder_service.Document} "Document data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateDocument(c *gin.Context) {
	var document obs.Document

	err := c.ShouldBindJSON(&document)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}
	document.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.DocumentService().Update(
		context.Background(),
		&document,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteDocument godoc
// @Security ApiKeyAuth
// @ID delete_document
// @Router /v1/document/{document_id} [DELETE]
// @Summary Delete Document
// @Description Delete Document
// @Tags Document
// @Accept json
// @Produce json
// @Param document_id path string true "document_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteDocument(c *gin.Context) {
	documentID := c.Param("document_id")

	if !util.IsValidUUID(documentID) {
		h.handleResponse(c, http.InvalidArgument, "Document id is an invalid uuid")
		return
	}
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.DocumentService().Delete(
		context.Background(),
		&obs.DocumentPrimaryKey{
			Id:        documentID,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetDocumentLists godoc
// @Security ApiKeyAuth
// @ID get_document_list
// @Router /v1/document [GET]
// @Summary Get Document list
// @Description Get Document list
// @Tags Document
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllDocumentsRequest true "filters"
// @Success 200 {object} http.Response{data=object_builder_service.GetAllDocumentsResponse} "DocumentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetDocumentList(c *gin.Context) {

	if c.Query("start_date") > c.Query("end_date") {
		err := errors.New("end date must be bigger than start date")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.DocumentService().GetList(
		context.Background(),
		&obs.GetAllDocumentsRequest{
			ObjectId:  c.Query("object_id"),
			Tags:      c.Query("tags"),
			StartDate: c.Query("start_date"),
			EndDate:   c.Query("end_date"),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
