package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"
)

// UpdateServiceResource godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_service_resource
// @Router /v1/company/project/service-resource [PUT]
// @Summary Update service resource
// @Description Update service resource
// @Tags Service Resource
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
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

	projectId := c.DefaultQuery("project-id", "")
	if !util.IsValidUUID(projectId) {
		err := errors.New("invalid projectId")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	data.ProjectId = projectId

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
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_service_resource
// @Router /v1/company/project/service-resource [GET]
// @Summary get list service resource
// @Description get list service resource
// @Tags Service Resource
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Success 200 {object} status_http.Response{data=pb.GetListServiceResourceRes} "data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListServiceResource(c *gin.Context) {

	projectId := c.DefaultQuery("project-id", "")
	if !util.IsValidUUID(projectId) {
		err := errors.New("invalid projectId")
		h.handleResponse(c, status_http.BadRequest, err.Error())
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
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
