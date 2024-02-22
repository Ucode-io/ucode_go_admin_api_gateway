package v2

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// CreateVersion godoc
// @Security ApiKeyAuth
// @ID create_version
// @Router /v2/version [POST]
// @Summary Create version
// @Description Create version
// @Tags Version
// @Accept json
// @Produce json
// @Param version body obs.CreateVersionRequest true "CreateVersionRequest"
// @Success 201 {object} status_http.Response{data=obs.Version} "Version data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateVersion(c *gin.Context) {
	var (
		version obs.CreateVersionRequest
		resp    *obs.Version
	)

	err := c.ShouldBindJSON(&version)
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
			ActionSource: "VERSION",
			ActionType:   "CREATE VERSION",
			UserInfo:     cast.ToString(userId),
			Request:      &version,
			TableSlug:    "version",
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Current = resp
			logReq.Response = resp
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	version.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Version().Create(
			context.Background(),
			&version,
		)
		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:

	}
}

// GeVersionList godoc
// @Security ApiKeyAuth
// @ID get_version_list
// @Router /v2/version [GET]
// @Summary Get version list
// @Description Get version list
// @Tags Version
// @Accept json
// @Produce json
// @Param filters query obs.GetVersionListRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetVersionListResponse} "VersionList"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetVersionList(c *gin.Context) {

	var (
		resp *obs.GetVersionListResponse
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

	fmt.Println("\n\n\n history env", environmentId, projectId)

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
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Version().GetList(
			context.Background(),
			&obs.GetVersionListRequest{
				ProjectId: resource.ResourceEnvironmentId,
				Offset:    int32(offset),
				Limit:     int32(limit),
			},
		)
		fmt.Println("\n\n\n\n ~~~~~~> ENV_ID ", c.DefaultQuery("env_id", ""), resp)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:

	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateVersion godoc
// @Security ApiKeyAuth
// @ID update_version
// @Router /v2/version [PUT]
// @Summary Update version
// @Description Update version
// @Tags Version
// @Accept json
// @Produce json
// @Param version body obs.Version true "Version"
// @Success 200 {object} status_http.Response{data=obs.Version} "UpdateVersionRequest"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateVersion(c *gin.Context) {
	var (
		version obs.Version
		resp    *obs.Version
	)

	err := c.ShouldBindJSON(&version)
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

	userId, _ := c.Get("user_id")

	var (
		oldView = &obs.View{}
		logReq  = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "VERSION",
			ActionType:   "UPDATE VERSION",
			UserInfo:     cast.ToString(userId),
			Request:      &version,
			TableSlug:    "version",
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
			// h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	version.ProjectId = resource.ResourceEnvironmentId

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		_, err = services.GetBuilderServiceByType(resource.NodeType).Version().Update(
			context.Background(),
			&version,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:

	}

	h.handleResponse(c, status_http.OK, nil)
}
