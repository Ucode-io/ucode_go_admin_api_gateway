package v2

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// CreateView godoc
// @Security ApiKeyAuth
// @ID v2_create_view
// @Router /v2/views/{collection} [POST]
// @Summary Create view
// @Description Create view
// @Tags V2_View
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param view body obs.CreateViewRequest true "CreateViewRequestBody"
// @Success 201 {object} status_http.Response{data=obs.View} "View data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateView(c *gin.Context) {
	var (
		view obs.CreateViewRequest
	)

	err := c.ShouldBindJSON(&view)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
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

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "VIEW",
			ActionType:   "CREATE VIEW",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			Request:   &view,
			TableSlug: c.Param("collection"),
		}
	)

	view.ProjectId = resource.ResourceEnvironmentId
	view.EnvId = resource.EnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).View().Create(
			context.Background(),
			&view,
		)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Current = resp
			logReq.Response = resp
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistory(c, logReq)
	case pb.ResourceType_POSTGRESQL:
		newReq := nb.CreateViewRequest{}
		err = helper.MarshalToStruct(&view, &newReq)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resp, err := services.GoObjectBuilderService().View().Create(
			context.Background(),
			&newReq,
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.Created, resp)
	}
}

// GetSingleView godoc
// @Security ApiKeyAuth
// @ID v2_get_view_by_id
// @Router /v2/views/{collection}/{id} [GET]
// @Summary Get single view
// @Description Get single view
// @Tags V2_View
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=obs.View} "ViewBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetSingleView(c *gin.Context) {
	viewID := c.Param("id")

	if !util.IsValidUUID(viewID) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.BuilderService().View().GetSingle(
			context.Background(),
			&obs.ViewPrimaryKey{
				Id:        viewID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().View().GetSingle(
			context.Background(),
			&nb.ViewPrimaryKey{
				Id:        viewID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

// UpdateView godoc
// @Security ApiKeyAuth
// @ID v2_update_view
// @Router /v2/views/{collection} [PUT]
// @Summary Update view
// @Description Update view
// @Tags V2_View
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param view body obs.View true "UpdateViewRequestBody"
// @Success 200 {object} status_http.Response{data=obs.View} "View data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateView(c *gin.Context) {
	var (
		view obs.View
		resp *obs.View
	)

	err := c.ShouldBindJSON(&view)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
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

	var (
		oldView = &obs.View{}
		logReq  = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "VIEW",
			ActionType:   "UPDATE VIEW",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			Request:   &view,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		logReq.Previous = oldView
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	oldView, err = services.GetBuilderServiceByType(resource.NodeType).View().GetSingle(
		context.Background(),
		&obs.ViewPrimaryKey{
			Id:        view.Id,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}

	view.ProjectId = resource.ResourceEnvironmentId
	view.EnvId = resource.EnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		oldView, err = services.GetBuilderServiceByType(resource.NodeType).View().GetSingle(
			context.Background(),
			&obs.ViewPrimaryKey{
				Id:        view.Id,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		logReq.Previous = oldView

		resp, err := services.GetBuilderServiceByType(resource.NodeType).View().Update(
			context.Background(),
			&view,
		)
		if err != nil {
			return
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	case pb.ResourceType_POSTGRESQL:
		newReq := nb.View{}
		err = helper.MarshalToStruct(&view, &newReq)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resp, err := services.GoObjectBuilderService().View().Update(
			context.Background(),
			&newReq,
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

// DeleteView godoc
// @Security ApiKeyAuth
// @ID v2_delete_view
// @Router /v2/views/{collection}/{id} [DELETE]
// @Summary Delete view
// @Description Delete view
// @Tags V2_View
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteView(c *gin.Context) {
	viewID := c.Param("id")

	if !util.IsValidUUID(viewID) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
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

	var (
		oldView = &obs.View{}
		logReq  = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "VIEW",
			ActionType:   "DELETE VIEW",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			TableSlug: c.Param("collection"),
		}
	)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		oldView, err = services.GetBuilderServiceByType(resource.NodeType).View().GetSingle(
			context.Background(),
			&obs.ViewPrimaryKey{
				Id:        viewID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		logReq.Previous = oldView
		resp, err := services.GetBuilderServiceByType(resource.NodeType).View().Delete(
			context.Background(),
			&obs.ViewPrimaryKey{
				Id:        viewID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.NoContent, resp)
			return
		}
		go h.versionHistory(c, logReq)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().View().Delete(
			context.Background(),
			&nb.ViewPrimaryKey{
				Id:        viewID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.NoContent, resp)
	}
}

// GetAllViews godoc
// @Security ApiKeyAuth
// @ID v2_get_view_list
// @Router /v2/views/{collection} [GET]
// @Summary Get view list
// @Description Get view list
// @Tags V2_View
// @Accept json
// @Produce json
// @Param filters query obs.GetAllViewsRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllViewsResponse} "ViewBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllViews(c *gin.Context) {

	var roleId string

	if c.Param("collection") == "" {
		h.handleResponse(c, status_http.BadRequest, "collection is required")
		return
	}
	autoInfo, _ := h.GetAuthInfo(c)
	if autoInfo != nil {
		if autoInfo.GetRoleId() != "" {
			roleId = autoInfo.GetRoleId()
		}
	}
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.BuilderService().View().GetList(
			context.Background(),
			&obs.GetAllViewsRequest{
				TableSlug: c.Param("collection"),
				ProjectId: resource.ResourceEnvironmentId,
				RoleId:    roleId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().View().GetList(
			context.Background(),
			&nb.GetAllViewsRequest{
				TableSlug: c.Param("collection"),
				ProjectId: resource.ResourceEnvironmentId,
				RoleId:    roleId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

// UpdateViewOrder godoc
// @Security ApiKeyAuth
// @ID v2_update_view_order
// @Router /v2/views/{collection}/update-order [PUT]
// @Summary Update view order
// @Description Update view order
// @Tags V2_View
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param view body obs.UpdateViewOrderRequest true "UpdateViewOrderRequestBody"
// @Success 200 {object} status_http.Response{data=string} "View data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateViewOrder(c *gin.Context) {
	var (
		view obs.UpdateViewOrderRequest
	)

	err := c.ShouldBindJSON(&view)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	view.ProjectId = resource.ResourceEnvironmentId

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	view.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.BuilderService().View().UpdateViewOrder(
			context.Background(),
			&view,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		newReq := nb.UpdateViewOrderRequest{}
		err = helper.MarshalToStruct(&view, &newReq)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resp, err := services.GoObjectBuilderService().View().UpdateViewOrder(
			context.Background(),
			&newReq,
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}
