package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"
	"github.com/gin-gonic/gin"
)

// CreateTable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID create_table
// @Router /v1/table [POST]
// @Summary Create table
// @Description Create table
// @Tags Table
// @Accept json
// @Produce json
// @Param table body models.CreateTableRequest true "CreateTableRequestBody"
// @Success 201 {object} status_http.Response{data=object_builder_service.Table} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateTable(c *gin.Context) {
	var tableRequest models.CreateTableRequest

	err := c.ShouldBindJSON(&tableRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	var fields []*obs.CreateFieldsRequest
	for _, field := range tableRequest.Fields {
		attributes, err := helper.ConvertMapToStruct(field.Attributes)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
		var tempField = obs.CreateFieldsRequest{
			Id:         field.ID,
			Default:    field.Default,
			Type:       field.Type,
			Index:      field.Index,
			Label:      field.Label,
			Slug:       field.Slug,
			Attributes: attributes,
			IsVisible:  field.IsVisible,
			Unique:     field.Unique,
			Automatic:  field.Automatic,
		}

		tempField.ProjectId = resourceId.(string)

		fields = append(fields, &tempField)
	}

	var table = obs.CreateTableRequest{
		Label:             tableRequest.Label,
		Description:       tableRequest.Description,
		Slug:              tableRequest.Slug,
		ShowInMenu:        tableRequest.ShowInMeny,
		Icon:              tableRequest.Icon,
		Fields:            fields,
		SubtitleFieldSlug: tableRequest.SubtitleFieldSlug,
		Sections:          tableRequest.Sections,
		AppId:             tableRequest.AppID,
		IncrementId: &obs.IncrementID{
			WithIncrementId: tableRequest.IncrementID.WithIncrementID,
			DigitNumber:     tableRequest.IncrementID.DigitNumber,
			Prefix:          tableRequest.IncrementID.Prefix,
		},
	}

	table.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.TableService().Create(
		context.Background(),
		&table,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetTableByID godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_table_by_id
// @Router /v1/table/{table_id} [GET]
// @Summary Get table by id
// @Description Get table by id
// @Tags Table
// @Accept json
// @Produce json
// @Param table_id path string true "table_id"
// @Success 200 {object} status_http.Response{data=models.CreateTableResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTableByID(c *gin.Context) {
	tableID := c.Param("table_id")

	if !util.IsValidUUID(tableID) {
		h.handleResponse(c, status_http.InvalidArgument, "table id is an invalid uuid")
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

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := services.TableService().GetByID(
		context.Background(),
		&obs.TablePrimaryKey{
			Id:        tableID,
			ProjectId: resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllTables godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_all_tables
// @Router /v1/table [GET]
// @Summary Get all tables
// @Description Get all tables
// @Tags Table
// @Accept json
// @Produce json
// @Param resource_Id query string true "resource_Id"
// @Param filters query object_builder_service.GetAllTablesRequest true "filters"
// @Success 200 {object} status_http.Response{data=object_builder_service.GetAllTablesResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllTables(c *gin.Context) {
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

	//_, err = h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get resource_id"))
	//	return
	//}
	//fmt.Println("resourceID:::::::", resourceId.(string))

	resp, err := services.TableService().GetAll(
		context.Background(),
		&obs.GetAllTablesRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: c.DefaultQuery("resource_Id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateTable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID update_table
// @Router /v1/table [PUT]
// @Summary Update table
// @Description Update table
// @Tags Table
// @Accept json
// @Produce json
// @Param table body object_builder_service.Table  true "UpdateTableRequestBody"
// @Success 200 {object} status_http.Response{data=object_builder_service.Table} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateTable(c *gin.Context) {
	var table obs.Table

	err := c.ShouldBindJSON(&table)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	table.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.TableService().Update(
		context.Background(),
		&table,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteTable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID delete_table
// @Router /v1/table/{table_id} [DELETE]
// @Summary Delete Table
// @Description Delete Table
// @Tags Table
// @Accept json
// @Produce json
// @Param table_id path string true "table_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteTable(c *gin.Context) {
	tableID := c.Param("table_id")

	if !util.IsValidUUID(tableID) {
		h.handleResponse(c, status_http.InvalidArgument, "table id is an invalid uuid")
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

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := services.TableService().Delete(
		context.Background(),
		&obs.TablePrimaryKey{
			Id:        tableID,
			ProjectId: resourceId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
