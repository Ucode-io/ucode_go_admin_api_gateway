package v2

import (
	"context"
	"errors"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func (h *HandlerV2) GetSingleLayout(c *gin.Context) {
	collection := c.Param("collection")
	menuId := c.Param("menu_id")

	if collection == "" && menuId == "" {
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
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	authInfo, _ := h.GetAuthInfo(c)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Layout().GetSingleLayout(
			context.Background(),
			&obs.GetSingleLayoutRequest{
				TableSlug: collection,
				ProjectId: resource.ResourceEnvironmentId,
				MenuId:    menuId,
				RoleId:    authInfo.GetRoleId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Layout().GetSingleLayout(
			context.Background(),
			&nb.GetSingleLayoutRequest{
				TableSlug: collection,
				ProjectId: resource.ResourceEnvironmentId,
				MenuId:    menuId,
				RoleId:    authInfo.GetRoleId(),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
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
// @Success 200 {object} status_http.Response{data=obs.GetListLayoutResponse} "TableBody"
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
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Layout().GetAll(
			context.Background(),
			&obs.GetListLayoutRequest{
				TableSlug:       tableSlug,
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

	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Layout().GetAll(
			context.Background(),
			&nb.GetListLayoutRequest{
				TableSlug:       tableSlug,
				ProjectId:       resourceEnvironmentId,
				IsDefault:       isDefault,
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
}

// UpdateLayout godoc
// @Security ApiKeyAuth
// @ID update_layout_v2
// @Router /v2/collections/{collection}/layout [PUT]
// @Summary Update layouts
// @Description Update layouts
// @Tags Layout
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param layout body obs.LayoutRequest true "LayoutRequest"
// @Success 200 {object} status_http.Response{data=obs.LayoutResponse} "Layout data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateLayout(c *gin.Context) {

	var (
		input obs.LayoutRequest
		resp  *obs.LayoutResponse
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

	if input.Id == "" {
		input.Id = uuid.NewString()
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "LAYOUT",
			ActionType:   "UPDATE LAYOUT",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			Request:   &input,
			TableSlug: c.Param("collection"),
		}
	)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Layout().Update(
			context.Background(),
			&input,
		)
		logReq.Previous = &input
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(logReq)
		h.handleResponse(c, status_http.OK, resp)
		return
	case pb.ResourceType_POSTGRESQL:
		newInput := nb.LayoutRequest{}
		err = helper.MarshalToStruct(&input, &newInput)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		resp2, err := services.GoObjectBuilderService().Layout().Update(
			context.Background(),
			&newInput,
		)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, resp2)
			return
		}

		err = helper.MarshalToStruct(resp2, &resp)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		logReq.Response = resp
		go h.versionHistoryGo(c, logReq)

		h.handleResponse(c, status_http.OK, resp)
		return
	}
}

// DeleteLayout godoc
// @Security ApiKeyAuth
// @ID delete_layout_v2
// @Router /v2/collections/{collection}/layout/{id} [DELETE]
// @Summary Delete layout
// @Description Delete layouts
// @Tags Layout
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
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
		oldLayout = &obs.LayoutResponse{}
		logReq    = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "LAYOUT",
			ActionType:   "DELETE LAYOUT",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			TableSlug: c.Param("collection"),
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
		switch resource.ResourceType {
		case pb.ResourceType_MONGODB:
			go h.versionHistory(logReq)
		case pb.ResourceType_POSTGRESQL:
			go h.versionHistoryGo(c, logReq)
		}
	}()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		oldLayout, err = services.GetBuilderServiceByType(nodeType).Layout().GetByID(
			context.Background(),
			&obs.LayoutPrimaryKey{
				Id:        c.Param("id"),
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			return
		}
		resp, err = services.GetBuilderServiceByType(nodeType).Layout().RemoveLayout(
			context.Background(),
			&obs.LayoutPrimaryKey{
				Id:        c.Param("id"),
				ProjectId: resourceEnvironmentId,
				EnvId:     environmentId.(string),
			},
		)
		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.GoObjectBuilderService().Layout().RemoveLayout(
			context.Background(),
			&nb.LayoutPrimaryKey{
				Id:        c.Param("id"),
				ProjectId: resourceEnvironmentId,
				EnvId:     environmentId.(string),
			},
		)
		if err != nil {
			return
		}
	}
}
