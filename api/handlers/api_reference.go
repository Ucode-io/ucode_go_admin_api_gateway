package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	ars "ucode/ucode_go_api_gateway/genproto/api_reference_service"
	vcs "ucode/ucode_go_api_gateway/genproto/versioning_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, apiReference.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
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

	apiReference.VersionId = activeVersion.GetVersionId()
	apiReference.CommitInfo = &ars.CommitInfo{
		CommitId:   "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Create api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  apiReference.ProjectId,
	}

	// set: commit_id
	resp, err := services.ApiReferenceService().ApiReference().Create(
		c.Request.Context(),
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
// @Param commit_id query string false "commit_id"
// @Param version_id query string false "version_id"
// @Success 200 {object} status_http.Response{data=models.ApiReference} "AppBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetApiReferenceByID(c *gin.Context) {
	id := c.Param("api_reference_id")
	commit_id := c.Query("commit_id")
	version_id := c.Query("version_id")

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

	if _, err := uuid.Parse(version_id); err != nil {
		// If version_id is not a valid uuid, then we need to get the active version
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
		version_id = activeVersion.GetVersionId()
	}

	resp, err := services.ApiReferenceService().ApiReference().Get(
		c.Request.Context(),
		&ars.GetApiReferenceRequest{
			Guid:      id,
			VersionId: version_id,
			CommitId:  commit_id,
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

	// activeVersion, err := services.VersioningService().Release().GetCurrentActive(
	// 	c.Request.Context(),
	// 	&vcs.GetCurrentReleaseRequest{
	// 		EnvironmentId: environmentId.(string),
	// 	},
	// )
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	resp, err := services.ApiReferenceService().ApiReference().GetList(
		context.Background(),
		&ars.GetListApiReferenceRequest{
			Limit:      int64(limit),
			Offset:     int64(offset),
			CategoryId: c.Query("category_id"),
			ProjectId:  c.Query("project_id"),
			VersionId:  "",
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

	// versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, apiReference.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
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

	apiReference.VersionId = activeVersion.GetVersionId()
	apiReference.CommitInfo = &ars.CommitInfo{
		CommitId:   "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Update api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  apiReference.ProjectId,
	}
	resp, err := services.ApiReferenceService().ApiReference().Update(
		c.Request.Context(), &apiReference,
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
		c.Request.Context(),
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
		versionIds []string
		// autherGuids []string
	)
	for _, item := range resp.GetApiReferences() {
		versionIds = append(versionIds, item.GetCommitInfo().GetVersionId())
	}

	multipleVersionResp, err := services.VersioningService().Release().GetMultipleVersionInfo(
		c.Request.Context(),
		&vcs.GetMultipleVersionInfoRequest{
			VersionIds: versionIds,
			ProjectId:  project_id,
		},
	)
	if err != nil {
		h.log.Error("error getting multiple version infos", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, item := range resp.GetApiReferences() {
		for key, _ := range item.GetVersionInfos() {

			versionInfoData := multipleVersionResp.GetVersionInfos()[key]

			item.VersionInfos[key] = &ars.VersionInfo{
				VersionId: versionInfoData.GetVersionId(),
				AuthorId:  versionInfoData.GetAuthorId(),
				Version:   versionInfoData.GetVersion(),
				Desc:      versionInfoData.GetDesc(),
				CreatedAt: versionInfoData.GetCreatedAt(),
				UpdatedAt: versionInfoData.GetUpdatedAt(),
				IsCurrent: versionInfoData.GetIsCurrent(),
			}
		}
	}
	// reqAutherGuids, err := helper.ConvertMapToStruct(map[string]interface{}{
	// 	"guid": []string{},
	// })
	// if err != nil {
	// 	h.log.Error("error converting map to struct", logger.Error(err))
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	// // Get User Data
	// respAuthersData, err := services.BuilderService().ObjectBuilder().GetList(
	// 	c.Request.Context(),

	// 	&builder.CommonMessage{
	// 		TableSlug: "users",
	// 		Data:      reqAutherGuids,
	// 	},
	// )
	// if err != nil {
	// 	h.log.Error("error getting user data", logger.Error(err))
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	h.handleResponse(c, status_http.OK, resp)
}

// RevertApiReference godoc
// @Security ApiKeyAuth
// @ID revert_api_reference
// @Router /v1/api-reference/revert/{api_reference_id} [POST]
// @Summary Revert Api Reference
// @Description Revert Api Reference
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference_id path string true "api_reference_id"
// @Param page query int false "page"
// @Param per_page query int false "per_page"
// @Param sort query string false "sort"
// @Param order query string false "order"
// @Param Environment-Id header string true "Environment-Id"
// @Param revert_api_reference body ars.ApiRevertApiReferenceRequest true "Request Body"
// @Success 200 {object} status_http.Response{data=ars.ApiReference} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RevertApiReference(c *gin.Context) {

	body := &ars.ApiRevertApiReferenceRequest{}

	err := c.ShouldBindJSON(body)
	if err != nil {
		h.log.Error("error binding json", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	id := c.Param("api_reference_id")

	if !util.IsValidUUID(id) {
		err := errors.New("api_reference_id is an invalid uuid")
		h.log.Error("api_reference_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(body.GetProjectId()) {
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

	versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}

	resp, err := services.ApiReferenceService().ApiReference().RevertApiReference(
		context.Background(),
		&ars.RevertApiReferenceRequest{
			Guid:        id,
			VersionId:   versionGuid,
			OldcommitId: body.GetCommitId(),
			NewcommitId: commitGuid,
		},
	)
	if err != nil {
		h.log.Error("error reverting api reference", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// InsertManyVersionForApiRef godoc
// @Security ApiKeyAuth
// @ID insert_many_api_reference
// @Router /v1/api-reference/select-versions/{api_reference_id} [POST]
// @Summary Select Api Reference
// @Description Select Api Reference
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference_id path string true "api_reference_id"
// @Param Environment-Id header string true "Environment-Id"
// @Param body body ars.ApiManyVersions true "Request Body"
// @Success 200 {object} status_http.Response{data=ars.ApiReference} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) InsertManyVersionForApiReference(c *gin.Context) {

	body := ars.ManyVersions{}
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	log.Printf("API->body: %+v", body)

	environmentID, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}

	if !util.IsValidUUID(environmentID.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	api_reference_id := c.Param("api_reference_id")
	if !util.IsValidUUID(api_reference_id) {
		err := errors.New("api_reference_id is an invalid uuid")
		h.log.Error("api_reference_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(body.GetProjectId()) {
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

	body.EnvironmentId = environmentID.(string)
	body.Guid = api_reference_id

	// _, commitId, err := h.CreateAutoCommitForAdminChange(c, environmentID.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }

	resp, err := services.ApiReferenceService().ApiReference().CreateManyApiReference(c.Request.Context(), &body)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())

		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
