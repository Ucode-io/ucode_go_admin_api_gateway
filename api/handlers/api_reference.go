package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/status_http"
	ars "ucode/ucode_go_api_gateway/genproto/api_reference_service"
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
// @Param app body models.CreateApiReference true "CreateApiReferenceRequestBody"
// @Success 201 {object} status_http.Response{data=models.ApiReference} "Api Reference data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateApiReference(c *gin.Context) {
	var apiRefence ars.CreateApiReferenceRequest

	err := c.ShouldBindJSON(&apiRefence)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	resp, err := services.ApiReferenceService().Create(
		context.Background(),
		&apiRefence,
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

	resp, err := services.ApiReferenceService().Get(
		context.Background(),
		&ars.GetApiReferenceRequest{
			Guid: id,
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

	resp, err := services.ApiReferenceService().GetList(
		context.Background(),
		&ars.GetListApiReferenceRequest{
			Limit:      int64(limit),
			Offset:     int64(offset),
			CategoryId: c.Query("category_id"),
			ProjectId:  c.Query("project_id"),
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

	resp, err := services.ApiReferenceService().Update(
		context.Background(),
		&apiReference,
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

	resp, err := services.ApiReferenceService().Delete(
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
