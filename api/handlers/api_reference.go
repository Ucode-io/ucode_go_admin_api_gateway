package handlers

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	ars "ucode/ucode_go_api_gateway/genproto/api_reference_service"
	vcs "ucode/ucode_go_api_gateway/genproto/versioning_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateApiReference godoc
// @Security ApiKeyAuth
// @ID create_api_reference
// @Router /v1/api-reference [POST]
// @Summary Create api reference
// @Description Create api reference
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference body models.CreateApiReferenceModel true "CreateApiReferenceRequestBody"
// @Success 201 {object} status_http.Response{data=models.ApiReference} "Api Reference data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateApiReference(c *gin.Context) {
	var apiReference ars.CreateApiReferenceRequest

	err := c.ShouldBindJSON(&apiReference)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if !util.IsValidUUID(apiReference.ProjectId) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid").Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, apiReference.GetProjectId())
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}

	apiReference.CommitId = commitGuid
	apiReference.VersionId = versionGuid

	// set: commit_id
	resp, err := services.ApiReferenceService().ApiReference().Create(
		context.Background(),
		&apiReference,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetApiReferenceByID godoc
// @Security ApiKeyAuth
// @ID get_api_reference_by_id
// @Router /v1/api-reference/{api_reference_id} [GET]
// @Summary Get api reference by id
// @Description Get api reference by id
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference_id path string true "api_reference_id"
// @Success 200 {object} status_http.Response{data=models.ApiReference} "AppBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetApiReferenceByID(c *gin.Context) {
	id := c.Param("api_reference_id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "api reference id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is not set").Error())
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}
	activeVersion, err := services.VersioningService().Release().GetCurrentActive(
		c.Request.Context(),
		&vcs.GetCurrentReleaseRequest{
			EnvironmentId: environmentId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.ApiReferenceService().ApiReference().Get(
		context.Background(),
		&ars.GetApiReferenceRequest{
			Guid:      id,
			VersionId: activeVersion.GetVersionId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllApiReferences godoc
// @Security ApiKeyAuth
// @ID get_all_api_reference
// @Router /v1/api-reference [GET]
// @Summary Get all apps
// @Description Get all api reference
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param filters query api_reference_service.GetListApiReferenceRequest true "filters"
// @Success 200 {object} status_http.Response{data=models.GetAllApiReferenceResponse} "ApiReferencesBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllApiReferences(c *gin.Context) {
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
	if !util.IsValidUUID(c.Query("project_id")) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid").Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is not set").Error())
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	activeVersion, err := services.VersioningService().Release().GetCurrentActive(
		c.Request.Context(),
		&vcs.GetCurrentReleaseRequest{
			EnvironmentId: environmentId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.ApiReferenceService().ApiReference().GetList(
		context.Background(),
		&ars.GetListApiReferenceRequest{
			Limit:      int64(limit),
			Offset:     int64(offset),
			CategoryId: c.Query("category_id"),
			ProjectId:  c.Query("project_id"),
			VersionId:  activeVersion.GetVersionId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateApiReference godoc
// @Security ApiKeyAuth
// @ID update_reference
// @Router /v1/api-reference [PUT]
// @Summary Update api reference
// @Description Update api reference
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference body models.ApiReference  true "UpdateApiReferenceRequestBody"
// @Success 200 {object} status_http.Response{data=models.ApiReference} "Api Reference data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateApiReference(c *gin.Context) {
	var apiReference ars.ApiReference

	err := c.ShouldBindJSON(&apiReference)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if !util.IsValidUUID(apiReference.GetProjectId()) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid").Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, apiReference.GetProjectId())
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}

	apiReference.CommitId = commitGuid
	apiReference.VersionId = versionGuid

	resp, err := services.ApiReferenceService().ApiReference().Update(
		context.Background(), &apiReference,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteApiReference godoc
// @Security ApiKeyAuth
// @ID delete_api_reference_id
// @Router /v1/api-reference/{project_id}/{api_reference_id} [DELETE]
// @Summary Delete App
// @Description Delete App
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference_id path string true "api_reference_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteApiReference(c *gin.Context) {
	id := c.Param("api_reference_id")
	projectId := c.Param("project_id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	versionGuid, _, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, projectId)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}

	resp, err := services.ApiReferenceService().ApiReference().Delete(
		context.Background(),
		&ars.DeleteApiReferenceRequest{
			Guid:      id,
			VersionId: versionGuid,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// GetApiReferenceChanges godoc
// @Security ApiKeyAuth
// @ID get_api_reference_changes
// @Router /v1/api-reference/history/{project_id}/{api_reference_id} [GET]
// @Summary Get Api Reference Changes
// @Description Get Api Reference Changes
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference_id path string true "api_reference_id"
// @Param project_id path string true "project_id"
// @Param page query int false "page"
// @Param per_page query int false "per_page"
// @Param sort query string false "sort"
// @Param order query string false "order"
// @Success 200 {object} status_http.Response{data=ars.GetListApiReferenceChangesResponse} "Api Reference Changes"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetApiReferenceChanges(c *gin.Context) {
	id := c.Param("api_reference_id")
	project_id := c.Param("project_id")

	if !util.IsValidUUID(id) {
		err := errors.New("api_reference_id is an invalid uuid")
		h.log.Error("api_reference_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(project_id) {
		err := errors.New("project_id is an invalid uuid")
		h.log.Error("project_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.log.Error("error getting limit param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.log.Error("error getting offset param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := services.ApiReferenceService().ApiReference().GetApiReferenceChanges(
		context.Background(),
		&ars.GetListApiReferenceChangesRequest{
			Guid:      id,
			ProjectId: project_id,
			Offset:    int64(offset),
			Limit:     int64(limit),
		},
	)
	if err != nil {
		h.log.Error("error getting api reference changes", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		commitIds  []string
		versionIds []string
	)
	for _, item := range resp.GetApiReferences() {
		commitIds = append(commitIds, item.GetCommitId())
		versionIds = append(versionIds, item.GetVersionId())
	}

	multipleCommitResp, err := services.VersioningService().Commit().GetMultipleCommitInfo(
		c.Request.Context(),
		&vcs.GetMultipleCommitInfoRequest{
			CommitIds: commitIds,
		},
	)
	if err != nil {
		h.log.Error("error getting multiple commit infos", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	multipleVersionResp, err := services.VersioningService().Release().GetMultipleVersionInfo(
		c.Request.Context(),
		&vcs.GetMultipleVersionInfoRequest{
			VersionIds: versionIds,
		},
	)
	if err != nil {
		h.log.Error("error getting multiple version infos", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, item := range resp.GetApiReferences() {
		commitInfo := multipleCommitResp.GetCommits()[item.GetCommitId()]

		item.CommitInfo = &ars.ApiReference_CommitInfo{
			CommitId:   commitInfo.GetCommitId(),
			VersionId:  commitInfo.GetVersionId(),
			ProjectId:  commitInfo.GetProjectId(),
			AuthorId:   commitInfo.GetAuthorId(),
			Name:       commitInfo.GetName(),
			CommitType: commitInfo.GetCommitType(),
			CreatedAt:  commitInfo.GetCreatedAt(),
			UpdatedAt:  commitInfo.GetUpdatedAt(),
		}

		versionInfo := multipleVersionResp.GetVersionInfos()[item.GetVersionId()]

		item.VersionInfo = &ars.ApiReference_VersionInfo{
			VersionId: versionInfo.GetVersionId(),
			Version:   versionInfo.GetVersion(),
			Desc:      versionInfo.GetDesc(),
			IsCurrent: versionInfo.GetIsCurrent(),
			CreatedAt: versionInfo.GetCreatedAt(),
			UpdatedAt: versionInfo.GetUpdatedAt(),
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}
