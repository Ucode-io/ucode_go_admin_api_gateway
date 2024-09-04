package v2

import (
	"context"
	"errors"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// GetByUDVersionHistory godoc
// @Security ApiKeyAuth
// @ID get_version_history_by_id
// @Router /v2/version/history/{environment_id}/{id} [GET]
// @Summary Get single version history
// @Description Get single version history
// @Tags VersionHistory
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=obs.VersionHistory} "VersionHistoryBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetVersionHistoryByID(c *gin.Context) {
	id := c.Param("id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "version history id is an invalid uuid")
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.BuilderService().VersionHistory().GetByID(
			context.Background(),
			&obs.VersionHistoryPrimaryKey{
				Id:        id,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().VersionHistory().GetByID(
			context.Background(),
			&nb.VersionHistoryPrimaryKey{
				Id:        id,
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

// GetAllVersionHistory godoc
// @Security ApiKeyAuth
// @ID get_all_version_history
// @Router /v2/version/history/{environment_id} [GET]
// @Summary Get version history list
// @Description Get version history list
// @Tags VersionHistory
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Param filters query obs.GetAllRquest true "filters"
// @Success 200 {object} status_http.Response{data=obs.ListVersionHistory} "ViewBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllVersionHistory(c *gin.Context) {

	var (
		fromDate, toDate string
		orderby          bool
	)
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	apiKey := c.Query("api_key")

	if c.Query("from_date") != "" {
		formatFromDate, err := time.Parse("2006-01-02", c.Query("from_date"))
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		fromDate = formatFromDate.Format("2006-01-02")
	}

	if c.Query("to_date") != "" {
		formatToDate, err := time.Parse("2006-01-02", c.Query("to_date"))
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		toDate = formatToDate.Format("2006-01-02")
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

	// environmentId := c.Param("environment_id")
	// if !util.IsValidUUID(environmentId) {
	// 	err := errors.New("error getting environment id | not valid")
	// 	h.handleResponse(c, status_http.BadRequest, err)
	// 	return
	// }

	currEnvironmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(currEnvironmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: currEnvironmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	tip := strings.ToUpper(c.Query("type"))
	if tip == "UP" {
		orderby = false
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).VersionHistory().GatAll(
			context.Background(),
			&obs.GetAllRquest{
				Type:      tip,
				ProjectId: resource.ResourceEnvironmentId,
				EnvId:     currEnvironmentId.(string),
				ApiKey:    apiKey,
				Offset:    int32(offset),
				Limit:     int32(limit),
				FromDate:  fromDate,
				ToDate:    toDate,
				OrderBy:   orderby,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().VersionHistory().GatAll(
			context.Background(),
			&nb.GetAllRquest{
				Type:      tip,
				ProjectId: resource.ResourceEnvironmentId,
				EnvId:     currEnvironmentId.(string),
				ApiKey:    apiKey,
				Offset:    int32(offset),
				Limit:     int32(limit),
				FromDate:  fromDate,
				ToDate:    toDate,
				OrderBy:   orderby,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	}

}

// UpdateListVersionHistory godoc
// @Security ApiKeyAuth
// @ID update_version_history
// @Router /v2/version/history/{environment_id} [PUT]
// @Summary Update version history
// @Description Update version history
// @Tags VersionHistory
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Param view body obs.UsedForEnvRequest true "UpdateViewRequestBody"
// @Success 200 {object} status_http.Response{data=obs.UsedForEnvRequest} "UsedForEnvRequest data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateVersionHistory(c *gin.Context) {
	var (
		view obs.UsedForEnvRequest
	)

	err := c.ShouldBindJSON(&view)
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

	environmentId := c.Param("environment_id")
	if !util.IsValidUUID(environmentId) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	view.ProjectId = resource.ResourceEnvironmentId

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		_, err = services.GetBuilderServiceByType(resource.NodeType).VersionHistory().Update(
			context.Background(),
			&view,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// resp, err = services.PostgresBuilderService().View().Update(
		// 	context.Background(),
		// 	&view,
		// )

		// if err != nil {
		// 	h.handleResponse(c, status_http.GRPCError, err.Error())
		// 	return
		// }
	}

	h.handleResponse(c, status_http.OK, nil)
}
