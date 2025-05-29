package v1

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
func (h *HandlerV1) GetAllSettings(c *gin.Context) {
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

	projectId := c.DefaultQuery("project-id", "")

	res, err := h.companyServices.Project().GetListSetting(
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
