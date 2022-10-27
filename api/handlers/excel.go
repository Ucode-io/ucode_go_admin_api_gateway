package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
)

// ExcelReader godoc
// @Security ApiKeyAuth
// @ID excel_reader
// @Router /v1/excel/{excel_id} [GET]
// @Summary Get excel writer
// @Description Get excel writer
// @Tags Excel
// @Accept json
// @Produce json
// @Param excel_id path string true "excel_id"
// @Success 200 {object} http.Response{data=object_builder_service.ExcelReadResponse} "ExcelReadResponse"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ExcelReader(c *gin.Context) {
	excelId := c.Param("excel_id")
	res, err := h.services.ExcelService().ExcelRead(context.Background(), &object_builder_service.ExcelReadRequest{Id: excelId})
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	h.handleResponse(c, http.OK, res)
}

// ExcelReader godoc
// @Security ApiKeyAuth
// @ID excel_to_db
// @Router /v1/excel/excel_to_db/{excel_id} [POST]
// @Summary Post excel writer
// @Description Post excel writer
// @Tags Excel
// @Accept json
// @Produce json
// @Param excel_id path string true "excel_id"
// @Param table body models.ExcelToDbRequest true "ExcelToDbRequest"
// @Success 200 {object} http.Response{data=models.ExcelToDbResponse} "ExcelToDbResponse"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ExcelToDb(c *gin.Context) {
	var excelRequest models.ExcelToDbRequest

	err := c.ShouldBindJSON(&excelRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	data, err := helper.ConvertMapToStruct(excelRequest.Data)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	_, err = h.services.ExcelService().ExcelToDb(context.Background(), &object_builder_service.ExcelToDbRequest{
		Id:        c.Param("excel_id"),
		TableSlug: excelRequest.TableSlug,
		Data:      data,
	})
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	h.handleResponse(c, http.Created, models.ExcelToDbResponse{
		Message: "Success!",
	})
}
