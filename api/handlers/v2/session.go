package v2

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV2) V2Login(c *gin.Context) {
	var login auth_service.V2LoginRequest

	if err := c.ShouldBindJSON(&login); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if login.GetClientType() == "" {
		h.HandleResponse(c, status_http.BadRequest, "client_type is required")
		return
	}

	if login.GetProjectId() == "" {
		h.HandleResponse(c, status_http.BadRequest, "project_id is required")
		return
	}

	environmentId := c.GetHeader("Environment-Id")
	if environmentId == "" {
		environmentId = login.GetEnvironmentId()
	}
	if environmentId == "" {
		h.HandleResponse(c, status_http.BadRequest, "Environment-Id is required")
		return
	}

	resourceEnvironment, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(), &pb.GetSingleServiceResourceReq{
			EnvironmentId: environmentId,
			ProjectId:     login.GetProjectId(),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	login.ResourceEnvironmentId = resourceEnvironment.GetResourceEnvironmentId()
	login.ResourceType = int32(resourceEnvironment.GetResourceType())
	login.EnvironmentId = resourceEnvironment.GetEnvironmentId()
	login.NodeType = resourceEnvironment.GetNodeType()
	login.ClientIp = c.RemoteIP()
	login.UserAgent = c.Request.UserAgent()

	service, conn, err := h.authService.Session(c)
	if err != nil {
		h.HandleResponse(c, status_http.BadEnvironment, err.Error())
		return
	}
	defer conn.Close()

	response, err := service.V2Login(c.Request.Context(), &login)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.Created, response)
}
