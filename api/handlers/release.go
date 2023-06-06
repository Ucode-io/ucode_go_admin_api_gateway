package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	obs "ucode/ucode_go_api_gateway/genproto/versioning_service"
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
// @Param release body obs.ApiCreateReleaseRequest true "Request Body"
// @Success 201 {object} status_http.Response{data=string} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateRelease(c *gin.Context) {
	var release obs.ApiCreateReleaseRequest

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	if !util.IsValidUUID(authInfo.GetUserId()) {
		h.handleResponse(c, status_http.InvalidArgument, "user_id is an invalid uuid")
		return
	}

	err = c.ShouldBindJSON(&release)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()).Error())
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	if !util.IsValidUUID(release.GetProjectId()) {
		h.handleResponse(c, status_http.BadRequest, errors.New("invalid project id"))
		return
	}

	resp, err := h.companyServices.VersioningService().Release().Create(
		c.Request.Context(),
		&obs.CreateReleaseRequest{
			ProjectId:     release.GetProjectId(),
			EnvironmentId: environmentId.(string),
			ReleaseType:   release.GetReleaseType(),
			AuthorId:      authInfo.GetUserId(),
			Description:   release.GetDescription(),
			IsCurrent:     release.GetIsCurrent(),
		},
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
// @Router /v1/release/{project_id}/{version_id} [GET]
// @Summary Get release by id
// @Description Get release by id
// @Tags Release
// @Accept json
// @Produce json
// @Param version_id path string true "version_id"
// @Param project_id path string true "project_id"
// @Success 200 {object} status_http.Response{data=obs.ReleaseWithCommit} "ReleaseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetReleaseByID(c *gin.Context) {
	versionID := c.Param("version_id")

	if !util.IsValidUUID(versionID) {
		h.handleResponse(c, status_http.InvalidArgument, "version_id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()).Error())
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	resp, err := h.companyServices.VersioningService().Release().GetByID(
		c.Request.Context(),
		&obs.ReleasePrimaryKey{
			Id:            versionID,
			EnvironmentId: environmentId.(string),
			ProjectId:     projectId.(string),
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
// @Router /v1/release/{project_id} [GET]
// @Summary Get all releases
// @Description Get all releases
// @Tags Release
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Param offset query int false "offset"
// @Param limit query int false "limit"
// @Success 200 {object} status_http.Response{data=obs.GetReleaseListResponse} "Response Body"
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()).Error())
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	resp, err := h.companyServices.VersioningService().Release().GetList(
		c.Request.Context(),
		&obs.GetReleaseListRequest{
			Limit:         int32(limit),
			Offset:        int32(offset),
			Search:        c.Query("search"),
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

// UpdateRelease godoc
// @Security ApiKeyAuth
// @ID update_release
// @Router /v1/release/{version_id} [PUT]
// @Summary Update release
// @Description Update release
// @Tags Release
// @Accept json
// @Produce json
// @Param version_id path string true "version_id"
// @Param release body obs.ApiUpdateReleaseRequest  true "Request Body"
// @Success 200 {object} status_http.Response{data=obs.ReleaseWithCommit} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateRelease(c *gin.Context) {
	var release obs.UpdateReleaseRequest

	err := c.ShouldBindJSON(&release)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	version_id := c.Param("version_id")
	if !util.IsValidUUID(version_id) {
		h.handleResponse(c, status_http.InvalidArgument, "version_id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(release.GetProjectId()) {
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	environmentID, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()).Error())
		return
	}
	if !util.IsValidUUID(environmentID.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	release.EnvironmentId = environmentID.(string)
	release.Id = version_id
	resp, err := h.companyServices.VersioningService().Release().Update(
		c.Request.Context(),
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
// @Router /v1/release/{version_id} [DELETE]
// @Summary Delete Release
// @Description Delete Release
// @Tags Release
// @Accept json
// @Produce json
// @Param version_id path string true "version_id"
// @Param release body obs.ApiDeleteReleaseRequest true "Request Body"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteRelease(c *gin.Context) {
	versionID := c.Param("version_id")

	body := &obs.ApiDeleteReleaseRequest{}

	err := c.ShouldBindJSON(body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if !util.IsValidUUID(versionID) {
		h.handleResponse(c, status_http.InvalidArgument, "version_id is an invalid uuid")
		return
	}

	environmentID, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()).Error())
		return
	}
	if !util.IsValidUUID(environmentID.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	resp, err := h.companyServices.VersioningService().Release().Delete(
		context.Background(),
		&obs.ReleasePrimaryKey{
			Id:            versionID,
			ProjectId:     body.GetProjectId(),
			EnvironmentId: environmentID.(string),
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
// @Param release body obs.ApiSetCurrentReleaseRequest  true "Request Body"
// @Success 200 {object} status_http.Response{data=string} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) SetCurrentRelease(c *gin.Context) {
	var release obs.SetCurrentReleaseRequest

	err := c.ShouldBindJSON(&release)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentID, ok := c.Get("environment_id")
	if !ok {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()).Error())
		return
	}
	if !util.IsValidUUID(environmentID.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	if !util.IsValidUUID(release.GetProjectId()) {
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(release.GetVersionId()) {
		h.handleResponse(c, status_http.InvalidArgument, "version_id is an invalid uuid")
		return
	}

	release.EnvironmentId = environmentID.(string)

	resp, err := h.companyServices.VersioningService().Release().SetCurrentActive(
		c.Request.Context(),
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
// @Success 200 {object} status_http.Response{data=obs.GetCurrentReleaseResponse} "GetReleaseBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCurrentRelease(c *gin.Context) {
	environmentId := c.Param("environment-id")

	if !util.IsValidUUID(environmentId) {
		h.handleResponse(c, status_http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.VersioningService().Release().GetCurrentActive(
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
