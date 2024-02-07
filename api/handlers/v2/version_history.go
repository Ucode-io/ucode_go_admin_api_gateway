package v2

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV2) GetAllVersionHistory(c *gin.Context) {

	var (
		resp             *obs.ListVersionHistory
		fromDate, toDate string
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

	environmentId := c.Param("environment_id")
	if !util.IsValidUUID(environmentId) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	currentEnvironmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(currentEnvironmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	// envId := ""
	// queryEnvId := ""
	// if strings.ToUpper(c.Param("type")) == "DOWN" {
	// 	envId = currentEnvironmentId.(string)
	// } else {
	// 	envId = environmentId
	// 	queryEnvId = currentEnvironmentId.(string)
	// }

	fmt.Println("\n\n\n history env", environmentId, projectId)

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: currentEnvironmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).VersionHistory().GatAll(
			context.Background(),
			&obs.GetAllRquest{
				Type:      strings.ToUpper(c.Query("type")),
				ProjectId: resource.ResourceEnvironmentId,
				EnvId:     currentEnvironmentId.(string),
				ApiKey:    apiKey,
				Offset:    int32(offset),
				Limit:     int32(limit),
				FromDate:  fromDate,
				ToDate:    toDate,
			},
		)
		fmt.Println("\n\n\n\n ~~~~~~> ENV_ID ", c.DefaultQuery("env_id", ""), resp)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		// resp, err = services.PostgresBuilderService().View().GetList(
		// 	context.Background(),
		// 	&obs.GetAllViewsRequest{
		// 		TableSlug: c.Param("collection"),
		// 		ProjectId: resource.ResourceEnvironmentId,
		// 		RoleId:    roleId,
		// 	},
		// )

		// if err != nil {
		// 	h.handleResponse(c, status_http.GRPCError, err.Error())
		// 	return
		// }
	}

	h.handleResponse(c, status_http.OK, resp)
}
