package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/status_http"
	obs "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateRelease godoc
// @Security ApiKeyAuth
// @ID create_release
// @Router /v1/release [POST]
// @Summary Create release
// @Description Create release
// @Tags Release
// @Accept json
// @Produce json
// @Param release body company_service.CreateReleaseRequest true "CreateReleaseRequestBody"
// @Success 201 {object} status_http.Response{data=company_service.ReleaseWithCommit} "Release data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateRelease(c *gin.Context) {
	var release obs.CreateReleaseRequest

	err := c.ShouldBindJSON(&release)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ReleaseService().Create(
		context.Background(),
		&release,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetReleaseByID godoc
// @Security ApiKeyAuth
// @ID get_release_by_id
// @Router /v1/release/{id} [GET]
// @Summary Get release by id
// @Description Get release by id
// @Tags Release
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=company_service.ReleaseWithCommit} "ReleaseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetReleaseByID(c *gin.Context) {
	releaseID := c.Param("id")

	if !util.IsValidUUID(releaseID) {
		h.handleResponse(c, status_http.InvalidArgument, "release id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.ReleaseService().GetByID(
		context.Background(),
		&obs.ReleasePrimaryKey{
			Id: releaseID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllReleases godoc
// @Security ApiKeyAuth
// @ID get_all_releases
// @Router /v1/release [GET]
// @Summary Get all releases
// @Description Get all releases
// @Tags Release
// @Accept json
// @Produce json
// @Param filters query company_service.GetReleaseListRequest true "filters"
// @Success 200 {object} status_http.Response{data=company_service.GetReleaseListResponse} "ReleaseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllReleases(c *gin.Context) {
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

	resp, err := h.companyServices.ReleaseService().GetList(
		context.Background(),
		&obs.GetReleaseListRequest{
			Limit:         int32(limit),
			Offset:        int32(offset),
			Search:        c.Query("search"),
			ProjectId:     c.Query("project-id"),
			EnvironmentId: c.Query("environment-id"),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateRelease godoc
// @Security ApiKeyAuth
// @ID update_release
// @Router /v1/release/{id} [PUT]
// @Summary Update release
// @Description Update release
// @Tags Release
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param release body company_service.UpdateReleaseRequest  true "UpdateReleaseRequestBody"
// @Success 200 {object} status_http.Response{data=company_service.ReleaseWithCommit} "Release data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateRelease(c *gin.Context) {
	var release obs.UpdateReleaseRequest

	err := c.ShouldBindJSON(&release)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	release.Id = c.Param("id")

	resp, err := h.companyServices.ReleaseService().Update(
		context.Background(),
		&release,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteRelease godoc
// @Security ApiKeyAuth
// @ID delete_release
// @Router /v1/release/{id} [DELETE]
// @Summary Delete Release
// @Description Delete Release
// @Tags Release
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteRelease(c *gin.Context) {
	releaseID := c.Param("id")

	if !util.IsValidUUID(releaseID) {
		h.handleResponse(c, status_http.InvalidArgument, "release id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.ReleaseService().Delete(
		context.Background(),
		&obs.ReleasePrimaryKey{
			Id: releaseID,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// SetCurrentRelease godoc
// @Security ApiKeyAuth
// @ID set_current_release
// @Router /v1/release/current [POST]
// @Summary SetCurrent release
// @Description SetCurrent release
// @Tags Release
// @Accept json
// @Produce json
// @Param release body company_service.SetCurrentReleaseRequest  true "SetCurrentReleaseRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Release data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) SetCurrentRelease(c *gin.Context) {
	var release obs.SetCurrentReleaseRequest

	err := c.ShouldBindJSON(&release)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ReleaseService().SetCurrentActive(
		context.Background(),
		&release,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetGetRelease godoc
// @Security ApiKeyAuth
// @ID get_current_release
// @Router /v1/release/current/{environment-id} [GET]
// @Summary Get release by id
// @Description Get release by id
// @Tags GetRelease
// @Accept json
// @Produce json
// @Param environment-id path string true "environment-id"
// @Success 200 {object} status_http.Response{data=company_service.GetCurrentReleaseResponse} "GetReleaseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCurrentRelease(c *gin.Context) {
	environmentId := c.Param("environment-id")

	if !util.IsValidUUID(environmentId) {
		h.handleResponse(c, status_http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.ReleaseService().GetCurrentActive(
		context.Background(),
		&obs.GetCurrentReleaseRequest{
			EnvironmentId: environmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
