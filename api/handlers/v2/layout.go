package v2

import (
	"context"
	"errors"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/cast"
)

func (h *HandlerV2) GetSingleLayout(c *gin.Context) {
	tableSlug := c.Param("collection")
	menuId := c.Param("menu_id")

	if tableSlug == "" && menuId == "" {
		h.handleResponse(c, status_http.BadRequest, "table-slug or table-id is required")
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

	authInfo, _ := h.GetAuthInfo(c)

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Layout().GetSingleLayout(
		context.Background(),
		&object_builder_service.GetSingleLayoutRequest{
			ProjectId: resource.ResourceEnvironmentId,
			MenuId:    menuId,
			TableSlug: tableSlug,
			RoleId:    authInfo.RoleId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.handleResponse(c, status_http.OK, resp)
}

// GetListLayouts godoc
// @Security ApiKeyAuth
// @ID get_list_layouts
// @Router /v1/layout [GET]
// @Summary Get list layouts
// @Description Get list layouts
// @Tags Layout
// @Accept json
// @Produce json
// @Param table-id query string false "table-id"
// @Param table-slug query string false "table-slug"
// @Param language_setting query string false "language_setting"
// @Success 200 {object} status_http.Response{data=object_builder_service.GetListLayoutResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetListLayouts(c *gin.Context) {
	tableSlug := c.Query("table-slug")
	tableId := c.Query("table-id")
	if tableSlug == "" && tableId == "" {
		h.handleResponse(c, status_http.BadRequest, "table-slug or table-id is required")
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

	var resourceEnvironmentId string
	var nodeType string

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	nodeType = resource.NodeType

	var isDefault = false
	var languageSettings = ""
	if c.Query("is_defualt") == "true" {
		isDefault = true
	}
	if c.Query("language_setting") != "" {
		languageSettings = c.Query("language_setting")
	}
	authInfo, _ := h.GetAuthInfo(c)

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)

	resp, err := services.GetBuilderServiceByType(nodeType).Layout().GetAll(
		context.Background(),
		&object_builder_service.GetListLayoutRequest{
			TableSlug:       tableSlug,
			TableId:         tableId,
			ProjectId:       resourceEnvironmentId,
			IsDefualt:       isDefault,
			RoleId:          authInfo.GetRoleId(),
			LanguageSetting: languageSettings,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateLayout godoc
// @Security ApiKeyAuth
// @ID update_layout
// @Router /v1/layout [PUT]
// @Summary Update layouts
// @Description Update layouts
// @Tags Layout
// @Accept json
// @Produce json
// @Param table body object_builder_service.UpdateLayoutRequest true "UpdateLayoutRequest"
// @Success 200 {object} status_http.Response{data=string} "Layout data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateLayout(c *gin.Context) {

	var (
		input object_builder_service.LayoutRequest
		resp  *object_builder_service.LayoutResponse
	)

	err := c.ShouldBindJSON(&input)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if input.TableId == "" {
		h.handleResponse(c, status_http.BadRequest, errors.New("table id is required"))
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

	var resourceEnvironmentId string
	var nodeType string
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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	input.ProjectId = resourceEnvironmentId

	var (
		oldLayout = &object_builder_service.LayoutResponse{}
		logReq    = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "LAYOUT",
			ActionType:   "UPDATE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &input,
			TableSlug: "",
		}
	)

	defer func() {
		logReq.Previous = oldLayout
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	oldLayout, err = services.GetBuilderServiceByType(nodeType).Layout().GetSingleLayout(
		context.Background(),
		&object_builder_service.GetSingleLayoutRequest{
			ProjectId: input.ProjectId,
			TableId:   input.TableId,
			MenuId:    input.MenuId,
			TableSlug: input.TableId,
		},
	)
	if err != nil {
		return
	}

	resp, err = services.GetBuilderServiceByType(nodeType).Layout().Update(
		context.Background(),
		&input,
	)
	if err != nil {
		return
	}
}

func (h *HandlerV2) DeleteLayout(c *gin.Context) {

	var (
		resp = &empty.Empty{}
	)

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

	var resourceEnvironmentId string
	var nodeType string
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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
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
			ActionSource: "LAYOUT",
			ActionType:   "DELETE",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			TableSlug: "",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	resp, err = services.GetBuilderServiceByType(nodeType).Layout().RemoveLayout(
		context.Background(),
		&object_builder_service.LayoutPrimaryKey{
			Id:        c.Param("id"),
			ProjectId: resourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}
}
