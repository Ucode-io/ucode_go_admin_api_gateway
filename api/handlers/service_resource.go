package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// UpdateServiceResource godoc
// @Security ApiKeyAuth
// @ID update_service_resource
// @Router /v1/company/project/service-resource [PUT]
// @Summary Update service resource
// @Description Update service resource
// @Tags Service Resource
// @Accept json
// @Produce json
// @Param data body pb.UpdateServiceResourceReq true "data"
// @Success 200 {object} status_http.Response{data=pb.UpdateServiceResourceRes} "data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateServiceResource(c *gin.Context) {
	var (
		data pb.UpdateServiceResourceReq
	)

	if err := c.ShouldBindJSON(&data); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}
	data.ProjectId = projectId.(string)

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	data.EnvironmentId = environmentId.(string)

	resp, err := h.companyServices.CompanyService().ServiceResource().Update(
		context.Background(),
		&data,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListServiceResource godoc
// @Security ApiKeyAuth
// @ID get_list_service_resource
// @Router /v1/company/project/service-resource [GET]
// @Summary get list service resource
// @Description get list service resource
// @Tags Service Resource
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=pb.GetListServiceResourceRes} "data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListServiceResource(c *gin.Context) {

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().ServiceResource().GetList(
		context.Background(),
		&pb.GetListServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
