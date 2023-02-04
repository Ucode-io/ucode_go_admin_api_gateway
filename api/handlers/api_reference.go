package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	ars "ucode/ucode_go_api_gateway/genproto/api_reference_service"
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
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid"))
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	commit_id, _ := uuid.NewRandom()
	version_id, _ := uuid.NewRandom()
	apiReference.CommitId = commit_id.String()
	apiReference.VersionId = version_id.String()

	apiReference.CommitId = commit_id.String()
	apiReference.VersionId = "0a4d3e5a-a273-422c-bef3-aebea3f2cec9"

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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	resp, err := services.ApiReferenceService().ApiReference().Get(
		context.Background(),
		&ars.GetApiReferenceRequest{
			Guid:      id,
			VersionId: "0a4d3e5a-a273-422c-bef3-aebea3f2cec9",
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
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid"))
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	resp, err := services.ApiReferenceService().ApiReference().GetList(
		context.Background(),
		&ars.GetListApiReferenceRequest{
			Limit:      int64(limit),
			Offset:     int64(offset),
			CategoryId: c.Query("category_id"),
			ProjectId:  c.Query("project_id"),
			VersionId:  "0a4d3e5a-a273-422c-bef3-aebea3f2cec9",
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

	if !util.IsValidUUID(apiReference.ProjectId) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid"))
		return
	}
	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	// app.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	// attributes, err := helper.ConvertMapToStruct(apiReference.Attributes)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err)
	// 	return
	// }

	commit_id := uuid.NewString()
	version_id := "0a4d3e5a-a273-422c-bef3-aebea3f2cec9"
	apiReference.CommitId = commit_id
	apiReference.VersionId = version_id

	resp, err := services.ApiReferenceService().ApiReference().Update(
		context.Background(), &apiReference,
		// &ars.ApiReference{
		// 	Guid:             apiReference.Guid,
		// 	Title:            apiReference.Title,
		// 	ProjectId:        apiReference.ProjectID,
		// 	AdditionalUrl:    apiReference.AdditionalUrl,
		// 	ExternalUrl:      apiReference.ExternalUrl,
		// 	Desc:             apiReference.Desc,
		// 	Method:           apiReference.Method,
		// 	CategoryId:       apiReference.CategoryID,
		// 	Authentification: apiReference.Authentification,
		// 	NewWindow:        apiReference.NewWindow,
		// 	Attributes:       attributes,
		// },
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
// @Router /v1/api-reference/{api_reference_id} [DELETE]
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

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "app id is an invalid uuid")
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

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	resp, err := services.ApiReferenceService().ApiReference().Delete(
		context.Background(),
		&ars.DeleteApiReferenceRequest{
			Guid: id,
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
// @Router /v1/api-reference/{project_id}/{api_reference_id}/changes [GET]
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
// @Success 200 {object} status_http.Response{data=ars.ApiReferenceChanges} "Api Reference Changes"
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	
	offset, err := h.getLimitParam(c)
	if err != nil {
		h.log.Error("error getting limit param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	limit, err := h.getOffsetParam(c)
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

	h.handleResponse(c, status_http.OK, resp)
}
