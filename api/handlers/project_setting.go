package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"ucode/ucode_go_api_gateway/api/status_http"
	obs "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

// GetAllSettings godoc
// @Security ApiKeyAuth
// @ID get_list_setting
// @Router /v1/project/setting [GET]
// @Summary Get List settings
// @Description Get List settings
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project-id query string false "project-id"
// @Param search query string false "search"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Param type query string false "type"
// @Success 200 {object} status_http.Response{data=obs.Setting} "Setting"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllSettings(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		settingType obs.SettingType
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
		settingType = obs.SettingType_LANGUAGE
	case "CURRENCY":
		settingType = obs.SettingType_CURRENCY
	case "TIMEZONE":
		settingType = obs.SettingType_TIMEZONE
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

	projectId := c.DefaultQuery("project-id", "")

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
	//
	//environmentId, ok := c.Get("environment_id")
	//if !ok {
	//	err = errors.New("error getting environment id")
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
	//	return
	//}
	//
	//if util.IsValidUUID(resourceId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ProjectId:     projectId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.CompanyService().Project().GetListSetting(
		context.Background(),
		&obs.GetListSettingReq{
			Type:      settingType,
			ProjectId: projectId,
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
