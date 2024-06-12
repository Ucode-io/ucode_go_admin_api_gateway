package v2

import (
	"context"
	"errors"
	"reflect"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// GetMenuSettingByUserID godoc
// @Security ApiKeyAuth
// @ID get_menu_settings_by_user_id
// @Router /v2/user/{id}/menu-settings [GET]
// @Summary Get menu settings by user-id
// @Description Get menu settings by user-id
// @Tags V2_User
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param template_id query string true "template_id"
// @Success 200 {object} status_http.Response{data=obs.MenuSettings} "MenuSettingsBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetMenuSettingByUserID(c *gin.Context) {
	ID := c.Param("id")
	if !util.IsValidUUID(ID) {
		h.handleResponse(c, status_http.InvalidArgument, "menu id is an invalid uuid")
		return
	}

	templateId := c.Query("template_id")

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

	var userId string
	authInfo, _ := h.GetAuthInfo(c)
	if authInfo != nil {
		userId = authInfo.GetUserId()
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Menu().GetByIDMenuSettings(
			context.Background(),
			&obs.MenuSettingPrimaryKey{
				Id:         ID,
				ProjectId:  resource.ResourceEnvironmentId,
				TemplateId: templateId,
				UserId:     userId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		if IsEmptyStruct(resp.MenuTemplate) {
			resp.MenuTemplate, err = helper.GetMenuTemplateById(templateId, services)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			resp.MenuTemplateId = templateId
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		pgResp, err := services.GoObjectBuilderService().Menu().GetByIDMenuSettings(
			context.Background(),
			&nb.MenuSettingPrimaryKey{
				Id:         ID,
				ProjectId:  resource.ResourceEnvironmentId,
				TemplateId: templateId,
				UserId:     userId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		if IsEmptyStruct(pgResp.MenuTemplate) {
			pgResp.MenuTemplate, err = helper.PgGetMenuTemplateById(templateId, services)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			pgResp.MenuTemplateId = templateId
		}
		h.handleResponse(c, status_http.OK, pgResp)
	}

}

func IsEmptyStruct(s interface{}) bool {
	return reflect.DeepEqual(s, reflect.Zero(reflect.TypeOf(s)).Interface())
}
