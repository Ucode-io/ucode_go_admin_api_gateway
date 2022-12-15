package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateRelation godoc
// @ID create_relation
// @Router /v1/relation [POST]
// @Summary Create relation
// @Description Create relation
// @Tags Relation
// @Accept json
// @Produce json
// @Param table body object_builder_service.CreateRelationRequest true "CreateRelationRequestBody"
// @Success 201 {object} http.Response{data=string} "Relation data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateRelation(c *gin.Context) {
	var relation obs.CreateRelationRequest

	err := c.ShouldBindJSON(&relation)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo := h.GetAuthInfo(c)
	relation.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.RelationService().Create(
		context.Background(),
		&relation,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetAllRelationss godoc
// @Security ApiKeyAuth
// @ID get_all_relations
// @Router /v1/relation [GET]
// @Summary Get all relations
// @Description Get all relations
// @Tags Relation
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllRelationsRequest true "filters"
// @Success 200 {object} http.Response{data=string} "RelationBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllRelations(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	resp, err := services.RelationService().GetAll(
		context.Background(),
		&obs.GetAllRelationsRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			TableSlug: c.DefaultQuery("table_slug", ""),
			TableId:   c.DefaultQuery("table_id", ""),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateRelation godoc
// @Security ApiKeyAuth
// @ID update_relation
// @Router /v1/relation [PUT]
// @Summary Update relation
// @Description Update relation
// @Tags Relation
// @Accept json
// @Produce json
// @Param relation body object_builder_service.UpdateRelationRequest  true "UpdateRelationRequestBody"
// @Success 200 {object} http.Response{data=string} "Relation data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateRelation(c *gin.Context) {
	var relation obs.UpdateRelationRequest

	err := c.ShouldBindJSON(&relation)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo := h.GetAuthInfo(c)
	relation.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.RelationService().Update(
		context.Background(),
		&relation,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteRelation godoc
// @Security ApiKeyAuth
// @ID delete_relation
// @Router /v1/relation/{relation_id} [DELETE]
// @Summary Delete Relation
// @Description Delete Relation
// @Tags Relation
// @Accept json
// @Produce json
// @Param relation_id path string true "relation_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteRelation(c *gin.Context) {
	relationID := c.Param("relation_id")

	if !util.IsValidUUID(relationID) {
		h.handleResponse(c, http.InvalidArgument, "relation id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	resp, err := services.RelationService().Delete(
		context.Background(),
		&obs.RelationPrimaryKey{
			Id:        relationID,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
