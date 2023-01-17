package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// GetViewRelation godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_single_view_relation
// @Router /v1/view_relation [GET]
// @Summary Get single view relation
// @Description Get single view relation
// @Description get list view relation switch to get single view relation because for one table be one view relation
// @Tags ViewRelation
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllSectionsRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "ViewRelationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetViewRelation(c *gin.Context) {

	// get list view relation switch to get single view relation because for one table be one view relation
	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
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

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	resp, err := services.SectionService().GetViewRelation(
		context.Background(),
		&obs.GetAllSectionsRequest{
			TableId:   c.Query("table_id"),
			TableSlug: c.Query("table_slug"),
			RoleId:    tokenInfo.RoleId,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpsertViewRelations godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID upsert_view_relation
// @Router /v1/view_relation [PUT]
// @Summary Upsert view relation
// @Description Upsert view relation
// @Tags ViewRelation
// @Accept json
// @Produce json
// @Param table body object_builder_service.UpsertViewRelationsBody  true "UpsertViewRelationsBody"
// @Success 200 {object} status_http.Response{data=string} "View Relation data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpsertViewRelations(c *gin.Context) {
	var viewRelation obs.UpsertViewRelationsBody

	err := c.ShouldBindJSON(&viewRelation)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	viewRelation.ProjectId = resourceEnvironment.GetId()

	resp, err := services.SectionService().UpsertViewRelations(
		context.Background(),
		&viewRelation,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
