package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"ucode/ucode_go_api_gateway/api/status_http"
	obs "ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"
)

// SetDefaultSettings godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID set_default_settings
// @Router /v1/project/{project-id}/setting [PUT]
// @Summary Set Default settings
// @Description Set Default settings
// @Tags Setting
// @Accept json
// @Produce json
// @Param project-id path string true "project-id"
// @Param setting body tmp.SetDefaultSettingsReq true "SetDefaultSettingsReq"
// @Success 200 {object} status_http.Response{data=tmp.Setting} "Setting data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) SetDefaultSettings(c *gin.Context) {
	var (
		resourceEnvironment *obs.ResourceEnvironment
		setting             tmp.SetDefaultSettingsReq
	)

	err := c.ShouldBindJSON(&setting)
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

	projectId := c.Param("project-id")
	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
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

	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     projectId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	setting.ProjectId = resourceEnvironment.GetId()

	res, err := services.BuilderService().Setting().SetDefaultSettings(
		context.Background(),
		&setting,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// GetAllSettings godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_setting
// @Router /v1/project/{project-id}/setting-all [GET]
// @Summary Get List settings
// @Description Get List settings
// @Tags Setting
// @Accept json
// @Produce json
// @Param project-id path string true "project-id"
// @Param search query string false "search"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Param type query string false "type"
// @Success 200 {object} status_http.Response{data=tmp.Setting} "Setting"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllSettings(c *gin.Context) {
	var (
		resourceEnvironment *obs.ResourceEnvironment
		settingType         tmp.SettingType
	)

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	stype := c.DefaultQuery("type", "LANGUAGE")

	switch strings.ToUpper(stype) {
	case "LANGUAGE":
		settingType = tmp.SettingType_LANGUAGE
	case "CURRENCY":
		settingType = tmp.SettingType_CURRENCY
	case "TIMEZONE":
		settingType = tmp.SettingType_TIMEZONE
	default:
		err = errors.New("not valid type")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId := c.Param("project-id")
	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
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

	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     projectId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	res, err := services.BuilderService().Setting().GetAll(
		context.Background(),
		&tmp.GetAllReq{
			Type:      settingType,
			ProjectId: resourceEnvironment.GetId(),
			Search:    c.DefaultQuery("search", ""),
			Limit:     int32(limit),
			Offset:    int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// GetDefaultSettings godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_default_settings
// @Router /v1/project/{project-id}/setting [GET]
// @Summary Get Default Setting
// @Description Get Default Setting
// @Tags Setting
// @Accept json
// @Produce json
// @Param project-id path string true "project-id"
// @Success 200 {object} status_http.Response{data=tmp.GetDefaultSettingsRes} "GetDefaultSettingsRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetDefaultSettings(c *gin.Context) {
	var (
		resourceEnvironment *obs.ResourceEnvironment
	)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId := c.Param("project-id")
	if !util.IsValidUUID(projectId) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
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

	if util.IsValidUUID(resourceId.(string)) {
		resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&obs.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	} else {
		resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
			c.Request.Context(),
			&obs.GetDefaultResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ProjectId:     projectId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	res, err := services.BuilderService().Setting().GetDefaultSettings(
		context.Background(),
		&tmp.GetDefaultSettingsReq{
			ProjectId: resourceEnvironment.GetId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
