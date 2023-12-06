package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	as "ucode/ucode_go_api_gateway/genproto/analytics_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// GetQueryRows godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) GetQueryRows(c *gin.Context) {
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
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_ANALYTICS_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.AnalyticsService().Query().GetQueryRows(
		context.Background(),
		&as.CommonInput{
			Data:      structData,
			Query:     queryReq.Query,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
