package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// GetViewRelation godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID get_single_view_relation
// @Router /v1/view_relation [GET]
// @Summary Get single view relation
// @Description Get single view relation
// @Description get list view relation switch to get single view relation because for one table be one view relation
// @Tags ViewRelation
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllSectionsRequest true "filters"
// @Success 200 {object} http.Response{data=string} "ViewRelationBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetViewRelation(c *gin.Context) {

	// get list view relation switch to get single view relation because for one table be one view relation
	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	resp, err := services.SectionService().GetViewRelation(
		context.Background(),
		&obs.GetAllSectionsRequest{
			TableId:   c.Query("table_id"),
			TableSlug: c.Query("table_slug"),
			RoleId:    tokenInfo.RoleId,
			ProjectId: resourceId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpsertViewRelations godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID upsert_view_relation
// @Router /v1/view_relation [PUT]
// @Summary Upsert view relation
// @Description Upsert view relation
// @Tags ViewRelation
// @Accept json
// @Produce json
// @Param table body object_builder_service.UpsertViewRelationsBody  true "UpsertViewRelationsBody"
// @Success 200 {object} http.Response{data=string} "View Relation data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpsertViewRelations(c *gin.Context) {
	var viewRelation obs.UpsertViewRelationsBody

	err := c.ShouldBindJSON(&viewRelation)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, http.Forbidden, err.Error())
	//	return
	//}
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	viewRelation.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.SectionService().UpsertViewRelations(
		context.Background(),
		&viewRelation,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
