package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	as "ucode/ucode_go_api_gateway/genproto/analytics_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/status_http"
)

// GetQueryRows godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_query_rows
// @Router /v1/query [POST]
// @Summary Get all query rows
// @Description Get all query rows
// @Tags Query
// @Accept json
// @Produce json
// @Param object body models.CommonInput true "GetAllQueryRowsRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonInput} "QueryBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetQueryRows(c *gin.Context) {
	var queryReq models.CommonInput

	err := c.ShouldBindJSON(&queryReq)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(queryReq.Data)
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

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.QueryService().GetQueryRows(
		context.Background(),
		&as.CommonInput{
			Data:      structData,
			Query:     queryReq.Query,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
