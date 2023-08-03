package handlers

import (
	"context"
	"errors"
	"reflect"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateMenu godoc
// @Security ApiKeyAuth
// @ID create_menu
// @Router /v1/menu [POST]
// @Summary Create menu
// @Description Create menu
// @Tags Menu
// @Accept json
// @Produce json
// @Param menu body models.CreateMenuRequest true "MenuRequest"
// @Success 201 {object} status_http.Response{data=models.Menu} "Menu data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateMenu(c *gin.Context) {
	var (
		menu models.CreateMenuRequest
		resp *obs.Menu
	)

	err := c.ShouldBindJSON(&menu)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	attributes, err := helper.ConvertMapToStruct(menu.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().Create(
			context.Background(),
			&obs.CreateMenuRequest{
				Label:           menu.Label,
				Icon:            menu.Icon,
				TableId:         menu.TableId,
				LayoutId:        menu.LayoutId,
				ParentId:        menu.ParentId,
				Type:            menu.Type,
				ProjectId:       resource.ResourceEnvironmentId,
				MicrofrontendId: menu.MicrofrontendId,
				WebpageId:       menu.WebpageId,
				Attributes:      attributes,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().Create(
			context.Background(),
			&obs.CreateMenuRequest{
				Label:           menu.Label,
				Icon:            menu.Icon,
				TableId:         menu.TableId,
				LayoutId:        menu.LayoutId,
				ParentId:        menu.ParentId,
				Type:            menu.Type,
				ProjectId:       resource.ResourceEnvironmentId,
				MicrofrontendId: menu.MicrofrontendId,
				WebpageId:       menu.WebpageId,
				Attributes:      attributes,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetMenuByID godoc
// @Security ApiKeyAuth
// @ID get_menu_by_id
// @Router /v1/menu/{menu_id} [GET]
// @Summary Get menu by id
// @Description Get menu by id
// @Tags Menu
// @Accept json
// @Produce json
// @Param menu_id path string true "menu_id"
// @Success 200 {object} status_http.Response{data=obs.Menu} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetMenuByID(c *gin.Context) {
	menuID := c.Param("menu_id")
	var (
		resp *obs.Menu
	)

	if !util.IsValidUUID(menuID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
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

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().GetByID(
			context.Background(),
			&obs.MenuPrimaryKey{
				Id:        menuID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetByID(
			context.Background(),
			&obs.MenuPrimaryKey{
				Id:        menuID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllMenus godoc
// @Security ApiKeyAuth
// @ID get_all_menus
// @Router /v1/menu [GET]
// @Summary Get all menu
// @Description Get all menu
// @Tags Menu
// @Accept json
// @Produce json
// @Param X-API-KEY header string false "API key for the endpoint"
// @Param filters query obs.GetAllMenusRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllMenusResponse} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllMenus(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	var (
		resp *obs.GetAllMenusResponse
	)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	authInfo, _ := h.GetAuthInfo(c)
	limit = 100

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().GetAll(
			context.Background(),
			&obs.GetAllMenusRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				Search:    c.DefaultQuery("search", ""),
				ProjectId: resource.ResourceEnvironmentId,
				ParentId:  c.DefaultQuery("parent_id", ""),
				RoleId:    authInfo.GetRoleId(),
			},
		)
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetAll(
			context.Background(),
			&obs.GetAllMenusRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				Search:    c.DefaultQuery("search", ""),
				ProjectId: resource.ResourceEnvironmentId,
				ParentId:  c.DefaultQuery("parent_id", ""),
				RoleId:    authInfo.GetRoleId(),
			},
		)
	}
	h.handleResponse(c, status_http.OK, resp)
}

// UpdateMenu godoc
// @Security ApiKeyAuth
// @ID update_menu
// @Router /v1/menu [PUT]
// @Summary Update menu
// @Description Update menu
// @Tags Menu
// @Accept json
// @Produce json
// @Param menu body obs.Menu  true "UpdateMenuRequestBody"
// @Success 200 {object} status_http.Response{data=obs.Menu} "App data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateMenu(c *gin.Context) {
	var (
		menu models.Menu
		resp *emptypb.Empty
	)

	err := c.ShouldBindJSON(&menu)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	attributes, err := helper.ConvertMapToStruct(menu.Attributes)
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
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().Update(
			context.Background(),
			&obs.Menu{
				Id:              menu.Id,
				Icon:            menu.Icon,
				TableId:         menu.TableId,
				LayoutId:        menu.LayoutId,
				Label:           menu.Label,
				ParentId:        menu.ParentId,
				Type:            menu.Type,
				ProjectId:       resource.ResourceEnvironmentId,
				MicrofrontendId: menu.MicrofrontendId,
				WebpageId:       menu.WebpageId,
				Attributes:      attributes,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().Update(
			context.Background(),
			&obs.Menu{
				Id:              menu.Id,
				Icon:            menu.Icon,
				TableId:         menu.TableId,
				LayoutId:        menu.LayoutId,
				Label:           menu.Label,
				ParentId:        menu.ParentId,
				Type:            menu.Type,
				ProjectId:       resource.ResourceEnvironmentId,
				MicrofrontendId: menu.MicrofrontendId,
				WebpageId:       menu.WebpageId,
				Attributes:      attributes,
			},
		)
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteMenu godoc
// @Security ApiKeyAuth
// @ID delete_menu
// @Router /v1/menu/{menu_id} [DELETE]
// @Summary Delete menu
// @Description Delete menu
// @Tags Menu
// @Accept json
// @Produce json
// @Param menu_id path string true "menu_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteMenu(c *gin.Context) {
	menuID := c.Param("menu_id")
	var (
		resp *emptypb.Empty
	)

	if !util.IsValidUUID(menuID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().Delete(
			context.Background(),
			&obs.MenuPrimaryKey{
				Id:        menuID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().Delete(
			context.Background(),
			&obs.MenuPrimaryKey{
				Id:        menuID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// UpdateMenuOrder godoc
// @Security ApiKeyAuth
// @ID update_menu_order
// @Router /v1/menu/menu-order/ [PUT]
// @Summary Delete menu
// @Description Delete menu
// @Tags Menu
// @Accept json
// @Produce json
// @Param menu body obs.UpdateMenuOrderRequest true "menu"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateMenuOrder(c *gin.Context) {
	var (
		resp  *emptypb.Empty
		menus obs.UpdateMenuOrderRequest
	)
	err := c.ShouldBindJSON(&menus)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().UpdateMenuOrder(
			context.Background(),
			&obs.UpdateMenuOrderRequest{
				Menus:     menus.GetMenus(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().UpdateMenuOrder(
			context.Background(),
			&obs.UpdateMenuOrderRequest{
				Menus:     menus.GetMenus(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

//// >>>>>>>>  Menu settings

// CreateMenuSettings godoc
// @Security ApiKeyAuth
// @ID create_menu_settings
// @Router /v1/menu-settings [POST]
// @Summary Create menu settings
// @Description Create menu settings
// @Tags Menu settings
// @Accept json
// @Produce json
// @Param menu body obs.CreateMenuSettingsRequest true "CreateMenuSettingsRequest"
// @Success 201 {object} status_http.Response{data=obs.MenuSettings} "Menu data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateMenuSettings(c *gin.Context) {
	var (
		menu obs.CreateMenuSettingsRequest
		resp *obs.MenuSettings
	)

	err := c.ShouldBindJSON(&menu)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	menu.ProjectId = resource.ResourceEnvironmentId

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().CreateMenuSettings(
			context.Background(),
			&menu,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().CreateMenuSettings(
			context.Background(),
			&menu,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetAllMenuSettings godoc
// @Security ApiKeyAuth
// @ID get_all_menu_settings
// @Router /v1/menu-settings [GET]
// @Summary Get all menu settings
// @Description Get all menu settings
// @Tags Menu settings
// @Accept json
// @Produce json
// @Param X-API-KEY header string false "API key for the endpoint"
// @Param filters query obs.GetAllMenuSettingsRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllMenuSettingsResponse} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllMenuSettings(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	var (
		resp *obs.GetAllMenuSettingsResponse
	)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().GetAllMenuSettings(
			context.Background(),
			&obs.GetAllMenuSettingsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetAllMenuSettings(
			context.Background(),
			&obs.GetAllMenuSettingsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
	}
	h.handleResponse(c, status_http.OK, resp)
}

// GetMenuSettingsByID godoc
// @Security ApiKeyAuth
// @ID get_menu_settings_by_id
// @Router /v1/menu-settings/{id} [GET]
// @Summary Get menu settings by id
// @Description Get menu settings by id
// @Tags Menu settings
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param template_id query string true "TemplateId"
// @Success 200 {object} status_http.Response{data=obs.MenuSettings} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetMenuSettingByID(c *gin.Context) {
	ID := c.Param("id")
	var (
		resp *obs.MenuSettings
	)
	if !util.IsValidUUID(ID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
		return
	}
	template_id := c.Query("template_id")

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().GetByIDMenuSettings(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:         ID,
				ProjectId:  resource.ResourceEnvironmentId,
				TemplateId: template_id,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetByIDMenuSettings(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	if IsEmptyStruct(resp.MenuTemplate) {
		resp.MenuTemplate, err = GetMenuTemplateById(template_id, services)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		resp.MenuTemplateId = template_id
	}

	h.handleResponse(c, status_http.OK, resp)
}

func IsEmptyStruct(s interface{}) bool {
	return reflect.DeepEqual(s, reflect.Zero(reflect.TypeOf(s)).Interface())
}

// UpdateMenuSettings godoc
// @Security ApiKeyAuth
// @ID update_menu_settings
// @Router /v1/menu-settings [PUT]
// @Summary Update menu settings
// @Description Update menu settings
// @Tags Menu settings
// @Accept json
// @Produce json
// @Param menu body obs.UpdateMenuSettingsRequest  true "UpdateMenuSettingsRequest"
// @Success 200 {object} status_http.Response{data=obs.Menu} "App data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateMenuSettings(c *gin.Context) {
	var (
		menu obs.UpdateMenuSettingsRequest
		resp *emptypb.Empty
	)

	err := c.ShouldBindJSON(&menu)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	menu.ProjectId = resource.ResourceEnvironmentId

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().UpdateMenuSettings(
			context.Background(),
			&menu,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().UpdateMenuSettings(
			context.Background(),
			&menu,
		)
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteMenuSetting godoc
// @Security ApiKeyAuth
// @ID delete_menu_settings
// @Router /v1/menu-settings/{id} [DELETE]
// @Summary Delete menu setting
// @Description Delete menu setting
// @Tags Menu setting
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteMenuSettings(c *gin.Context) {
	ID := c.Param("id")
	var (
		resp *emptypb.Empty
	)

	if !util.IsValidUUID(ID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().DeleteMenuSettings(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().DeleteMenuSettings(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}

	h.handleResponse(c, status_http.NoContent, resp)
}

//// >> Menu templates

// CreateMenuTemplate godoc
// @Security ApiKeyAuth
// @ID create_menu_template
// @Router /v1/menu-template [POST]
// @Summary Create menu template
// @Description Create menu template
// @Tags Menu template
// @Accept json
// @Produce json
// @Param menu body obs.CreateMenuTemplateRequest true "CreateMenuSettingsRequest"
// @Success 201 {object} status_http.Response{data=obs.MenuTemplate} "Menu data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateMenuTemplate(c *gin.Context) {
	var (
		menu obs.CreateMenuTemplateRequest
		resp *obs.MenuTemplate
	)

	err := c.ShouldBindJSON(&menu)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	menu.ProjectId = resource.ResourceEnvironmentId

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().CreateMenuTemplate(
			context.Background(),
			&menu,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().CreateMenuTemplate(
			context.Background(),
			&menu,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetAllMenuTemplate godoc
// @Security ApiKeyAuth
// @ID get_all_menu_template
// @Router /v1/menu-template [GET]
// @Summary Get all menu template
// @Description Get all menu template
// @Tags Menu template
// @Accept json
// @Produce json
// @Param X-API-KEY header string false "API key for the endpoint"
// @Param filters query obs.GetAllMenuSettingsRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GatAllMenuTemplateResponse} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllMenuTemplates(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	var (
		resp *obs.GatAllMenuTemplateResponse
	)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().GetAllMenuTemplate(
			context.Background(),
			&obs.GetAllMenuSettingsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetAllMenuTemplate(
			context.Background(),
			&obs.GetAllMenuSettingsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	globalMenus, err := services.CompanyService().Company().GetAllMenuTemplate(context.Background(), &emptypb.Empty{})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	for _, value := range globalMenus.GetMenuTemplates() {
		resp.MenuTemplates = append(resp.MenuTemplates, &obs.MenuTemplate{
			Id:               value.GetId(),
			Background:       value.GetBackground(),
			ActiveBackground: value.GetActiveBackground(),
			Text:             value.GetText(),
			ActiveText:       value.GetActiveText(),
			Title:            value.GetTitle(),
		})
	}
	resp.Count += int32(len(globalMenus.GetMenuTemplates()))
	h.handleResponse(c, status_http.OK, resp)
}

// GetMenuTemplateByID godoc
// @Security ApiKeyAuth
// @ID get_menu_template_by_id
// @Router /v1/menu-template/{id} [GET]
// @Summary Get menu template by id
// @Description Get menu template by id
// @Tags Menu template
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=obs.MenuTemplate} "MenuBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetMenuTemplateByID(c *gin.Context) {
	ID := c.Param("id")
	var (
		resp *obs.MenuTemplate
	)

	if !util.IsValidUUID(ID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().GetByIDMenuTemplate(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().GetByIDMenuTemplate(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	if resp == (&obs.MenuTemplate{}) {
		resp, err = GetMenuTemplateById(ID, services)
		if err != nil {
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

func GetMenuTemplateById(id string, services services.ServiceManagerI) (*obs.MenuTemplate, error) {
	var resp *obs.MenuTemplate
	global, err := services.CompanyService().Company().GetMenuTemplateById(context.Background(), &pb.GetMenuTemplateRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}
	resp = &obs.MenuTemplate{
		Id:               global.GetId(),
		Background:       global.GetBackground(),
		ActiveBackground: global.GetActiveBackground(),
		Text:             global.GetText(),
		ActiveText:       global.GetActiveText(),
		Title:            global.GetTitle(),
	}
	return resp, nil
}

// UpdateMenuTemplate godoc
// @Security ApiKeyAuth
// @ID update_menu_template
// @Router /v1/menu-template [PUT]
// @Summary Update menu template
// @Description Update menu template
// @Tags Menu template
// @Accept json
// @Produce json
// @Param menu body obs.UpdateMenuTemplateRequest  true "UpdateMenuTemplateRequest"
// @Success 200 {object} status_http.Response{data=obs.Menu} "App data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateMenuTemplate(c *gin.Context) {
	var (
		menu obs.UpdateMenuTemplateRequest
		resp *emptypb.Empty
	)

	err := c.ShouldBindJSON(&menu)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	menu.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().UpdateMenuTemplate(
			context.Background(),
			&menu,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().UpdateMenuTemplate(
			context.Background(),
			&menu,
		)
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteMenuTemplate godoc
// @Security ApiKeyAuth
// @ID delete_menu_template
// @Router /v1/menu-template/{id} [DELETE]
// @Summary Delete menu template
// @Description Delete menu template
// @Tags Menu template
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteMenuTemplate(c *gin.Context) {
	ID := c.Param("id")
	var (
		resp *emptypb.Empty
	)

	if !util.IsValidUUID(ID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Menu().DeleteMenuTemplate(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Menu().DeleteMenuTemplate(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:        ID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}

	h.handleResponse(c, status_http.NoContent, resp)
}
