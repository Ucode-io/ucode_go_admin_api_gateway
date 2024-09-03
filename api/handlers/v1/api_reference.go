package v1

import (
	"context"
	"errors"
	"fmt"
	"time"
	_ "ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	ars "ucode/ucode_go_api_gateway/genproto/api_reference_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
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
// @Param api_reference body ars.CreateApiReferenceRequest true "CreateApiReferenceRequestBody"
// @Success 201 {object} status_http.Response{data=ars.ApiReference} "Api Reference data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateApiReference(c *gin.Context) {
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	apiReference.CommitInfo = &ars.CommitInfo{
		Guid:       "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Create api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  projectId.(string),
	}
	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	apiReference.ResourceId = resource.GetResourceEnvironmentId()
	apiReference.ProjectId = projectId.(string)
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
// @Param project-id query string false "project-id"
// @Success 200 {object} status_http.Response{data=ars.ApiReference} "AppBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetApiReferenceByID(c *gin.Context) {
	id := c.Param("api_reference_id")
	commit_id := c.Query("commit_id")
	version_id := c.Query("version_id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "api reference id is an invalid uuid")
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ApiReferenceService().ApiReference().Get(
		c.Request.Context(),
		&ars.GetApiReferenceRequest{
			Guid:       id,
			VersionId:  version_id,
			CommitId:   commit_id,
			ResourceId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	versions, err := services.VersioningService().Release().GetMultipleVersionInfo(context.Background(), &vcs.GetMultipleVersionInfoRequest{
		VersionIds: resp.CommitInfo.VersionIds,
		ProjectId:  resp.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	versionInfos := make([]*ars.VersionInfo, 0, len(resp.GetCommitInfo().GetVersionIds()))
	for _, id := range resp.CommitInfo.VersionIds {
		versionInfo, ok := versions.VersionInfos[id]
		if ok {
			versionInfos = append(versionInfos, &ars.VersionInfo{
				AuthorId:  versionInfo.AuthorId,
				CreatedAt: versionInfo.CreatedAt,
				UpdatedAt: versionInfo.UpdatedAt,
				Desc:      versionInfo.Desc,
				IsCurrent: versionInfo.IsCurrent,
				Version:   versionInfo.Version,
				VersionId: versionInfo.VersionId,
			})
		}
	}
	resp.CommitInfo.VersionInfos = versionInfos
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
// @Param filters query ars.GetListApiReferenceRequest true "filters"
// @Success 200 {object} status_http.Response{data=ars.GetListApiReferenceResponse} "ApiReferencesBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllApiReferences(c *gin.Context) {
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
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is not set").Error())
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ApiReferenceService().ApiReference().GetList(
		context.Background(),
		&ars.GetListApiReferenceRequest{
			Limit:      int64(limit),
			Offset:     int64(offset),
			CategoryId: c.Query("category_id"),
			ProjectId:  projectId.(string),
			ResourceId: resource.GetResourceEnvironmentId(),
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
// @Param api_reference body ars.ApiReference  true "UpdateApiReferenceRequestBody"
// @Success 200 {object} status_http.Response{data=ars.ApiReference} "Api Reference data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateApiReference(c *gin.Context) {
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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
		Guid:       "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Update api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  projectId.(string),
	}

	apiReference.ResourceId = resource.ResourceEnvironmentId
	apiReference.ProjectId = projectId.(string)
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
func (h *HandlerV1) DeleteApiReference(c *gin.Context) {
	id := c.Param("api_reference_id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
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
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	// versionGuid, _, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, projectId)
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }
	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ApiReferenceService().ApiReference().Delete(
		c.Request.Context(),
		&ars.DeleteApiReferenceRequest{
			Guid:       id,
			VersionId:  uuid.NewString(),
			ResourceId: resource.ResourceEnvironmentId,
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
// @Param page query int false "page"
// @Param per_page query int false "per_page"
// @Param sort query string false "sort"
// @Param order query string false "order"
// @Success 200 {object} status_http.Response{data=ars.GetListApiReferenceChangesResponse} "Api Reference Changes"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetApiReferenceChanges(c *gin.Context) {
	id := c.Param("api_reference_id")

	if !util.IsValidUUID(id) {
		err := errors.New("api_reference_id is an invalid uuid")
		h.log.Error("api_reference_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "api_reference_id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
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
	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ApiReferenceService().ApiReference().GetApiReferenceChanges(
		context.Background(),
		&ars.GetListApiReferenceChangesRequest{
			Guid:       id,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			Offset:     int64(offset),
			Limit:      int64(limit),
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
		versionIds = append(versionIds, item.GetCommitInfo().GetVersionIds()...)
	}

	multipleVersionResp, err := services.VersioningService().Release().GetMultipleVersionInfo(
		c.Request.Context(),
		&vcs.GetMultipleVersionInfoRequest{
			VersionIds: versionIds,
			ProjectId:  projectId.(string),
		},
	)
	if err != nil {
		h.log.Error("error getting multiple version infos", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, item := range resp.GetApiReferences() {
		for key := range item.GetVersionInfos() {

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
// @Param revert_api_reference body ars.ApiRevertApiReferenceRequest true "Request Body"
// @Success 200 {object} status_http.Response{data=ars.ApiReference} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) RevertApiReference(c *gin.Context) {

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	// versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, environmentId.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }
	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	resp, err := services.ApiReferenceService().ApiReference().RevertApiReference(
		context.Background(),
		&ars.RevertApiReferenceRequest{
			Guid:        id,
			VersionId:   uuid.NewString(),
			OldcommitId: body.GetCommitId(),
			NewcommitId: uuid.NewString(),
			ResourceId:  resource.ResourceEnvironmentId,
			ProjectId:   projectId.(string),
		},
	)
	if err != nil {
		h.log.Error("error reverting api reference", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// InsertManyVersionForApiReference godoc
// @Security ApiKeyAuth
// @ID insert_many_api_reference
// @Router /v1/api-reference/select-versions/{api_reference_id} [POST]
// @Summary Select Api Reference
// @Description Select Api Reference
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference_id path string true "api_reference_id"
// @Param body body ars.ApiManyVersions true "Request Body"
// @Success 200 {object} status_http.Response{data=ars.ApiReference} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) InsertManyVersionForApiReference(c *gin.Context) {

	body := ars.ManyVersions{}
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	body.EnvironmentId = environmentID.(string)
	body.Guid = api_reference_id
	// _, commitId, err := h.CreateAutoCommitForAdminChange(c, environmentID.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }
	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: body.EnvironmentId,
			ServiceType:   pb.ServiceType_API_REF_SERVICE,
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

	body.ResourceId = resource.ResourceEnvironmentId
	body.ProjectId = projectId.(string)
	resp, err := services.ApiReferenceService().ApiReference().CreateManyApiReference(c.Request.Context(), &body)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())

		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
