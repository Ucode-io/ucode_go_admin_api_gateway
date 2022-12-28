package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	as "ucode/ucode_go_api_gateway/genproto/analytics_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
)

// GetAllQueryRows godoc
// @Security ApiKeyAuth
// @ID get_list_query_rows
// @Router /v1/query [POST]
// @Summary Get all query rows
// @Description Get all query rows
// @Tags Query
// @Accept json
// @Produce json
// @Param object body models.CommonInput true "GetAllQueryRowsRequestBody"
// @Success 200 {object} http.Response{data=models.CommonInput} "QueryBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetQueryRows(c *gin.Context) {
	var queryReq models.CommonInput

	err := c.ShouldBindJSON(&queryReq)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(queryReq.Data)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
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

	resp, err := services.QueryService().GetQueryRows(
		context.Background(),
		&as.CommonInput{
			Data:      structData,
			Query:     queryReq.Query,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
