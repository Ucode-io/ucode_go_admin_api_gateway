package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateField godoc
// @Security ApiKeyAuth
// @ID create_field
// @Router /v1/field [POST]
// @Summary Create field
// @Description Create field
// @Tags Field
// @Accept json
// @Produce json
// @Param table body models.CreateFieldRequest true "CreateFieldRequestBody"
// @Success 201 {object} http.Response{data=models.Field} "Field data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateField(c *gin.Context) {
	var fieldRequest models.CreateFieldRequest

	err := c.ShouldBindJSON(&fieldRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(fieldRequest.Attributes)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	var field = obs.CreateFieldRequest{
		Id:            fieldRequest.ID,
		Default:       fieldRequest.Default,
		Type:          fieldRequest.Type,
		Index:         fieldRequest.Index,
		Label:         fieldRequest.Label,
		Slug:          fieldRequest.Slug,
		TableId:       fieldRequest.TableID,
		Attributes:    attributes,
		IsVisible:     fieldRequest.IsVisible,
		AutofillTable: fieldRequest.AutoFillTable,
		AutofillField: fieldRequest.AutoFillField,
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}
	field.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FieldService().Create(
		context.Background(),
		&field,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetAllFields godoc
// @Security ApiKeyAuth
// @ID get_all_fields
// @Router /v1/field [GET]
// @Summary Get all fields
// @Description Get all fields
// @Tags Field
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllFieldsRequest true "filters"
// @Success 200 {object} http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllFields(c *gin.Context) {
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

	var withManyRelation, withOneRelation = false, false
	if c.Query("with_many_relation") == "true" {
		withManyRelation = true
	}

	if c.Query("with_one_relation") == "true" {
		withOneRelation = true
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.FieldService().GetAll(
		context.Background(),
		&obs.GetAllFieldsRequest{
			Limit:            int32(limit),
			Offset:           int32(offset),
			Search:           c.DefaultQuery("search", ""),
			TableId:          c.DefaultQuery("table_id", ""),
			TableSlug:        c.DefaultQuery("table_slug", ""),
			WithManyRelation: withManyRelation,
			WithOneRelation:  withOneRelation,
			ProjectId:        authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateField godoc
// @Security ApiKeyAuth
// @ID update_field
// @Router /v1/field [PUT]
// @Summary Update field
// @Description Update field
// @Tags Field
// @Accept json
// @Produce json
// @Param relation body models.Field  true "UpdateFieldRequestBody"
// @Success 200 {object} http.Response{data=models.Field} "Field data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateField(c *gin.Context) {
	var fieldRequest models.Field

	err := c.ShouldBindJSON(&fieldRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(fieldRequest.Attributes)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	var field = obs.Field{
		Id:            fieldRequest.ID,
		Default:       fieldRequest.Default,
		Type:          fieldRequest.Type,
		Index:         fieldRequest.Index,
		Label:         fieldRequest.Label,
		Slug:          fieldRequest.Slug,
		TableId:       fieldRequest.TableID,
		Required:      fieldRequest.Required,
		Attributes:    attributes,
		IsVisible:     fieldRequest.IsVisible,
		AutofillField: fieldRequest.AutoFillField,
		AutofillTable: fieldRequest.AutoFillTable,
		RelationId:    fieldRequest.RelationId,
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}
	field.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FieldService().Update(
		context.Background(),
		&field,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteField godoc
// @Security ApiKeyAuth
// @ID delete_field
// @Router /v1/field/{field_id} [DELETE]
// @Summary Delete Field
// @Description Delete Field
// @Tags Field
// @Accept json
// @Produce json
// @Param field_id path string true "field_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteField(c *gin.Context) {
	fieldID := c.Param("field_id")

	if !util.IsValidUUID(fieldID) {
		h.handleResponse(c, http.InvalidArgument, "field id is an invalid uuid")
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
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	resp, err := services.FieldService().Delete(
		context.Background(),
		&obs.FieldPrimaryKey{
			Id:        fieldID,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
