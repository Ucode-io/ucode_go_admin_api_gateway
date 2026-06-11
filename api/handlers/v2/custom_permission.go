package v2

import (
	"strings"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

type customPermissionNavFlags struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

type customPermissionNavMapResponse struct {
	Permissions map[string]customPermissionNavFlags `json:"permissions"`
}

func (h *HandlerV2) GetCustomPermissionNavMap(c *gin.Context) {
	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	if authInfo.GetRoleId() == "" || authInfo.GetClientTypeId() == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "role_id and client_type_id are required")
		return
	}

	service, resourceEnvId, err := h.getCustomPermissionServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().CustomPermission().GetAllCustomPermissionAccesses(
		c.Request.Context(), &pbo.GetAllCustomPermissionAccessesRequest{
			ProjectId:    resourceEnvId,
			RoleId:       authInfo.GetRoleId(),
			ClientTypeId: authInfo.GetClientTypeId(),
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	permissions := make(map[string]customPermissionNavFlags)
	for _, item := range response.GetPermissions() {
		navPath := customPermissionNavPath(item)
		if navPath == "" {
			continue
		}

		permissions[navPath] = customPermissionNavFlags{
			Read:   customPermissionAccessEnabled(item.GetRead()),
			Write:  customPermissionAccessEnabled(item.GetWrite()),
			Update: customPermissionAccessEnabled(item.GetUpdate()),
			Delete: customPermissionAccessEnabled(item.GetDelete()),
		}
	}

	h.HandleResponse(c, status_http.OK, customPermissionNavMapResponse{Permissions: permissions})
}

func customPermissionNavPath(item *pbo.CustomPermissionWithAccess) string {
	if item == nil || item.GetAttributes() == nil {
		return ""
	}

	value, ok := item.GetAttributes().GetFields()["nav_path"]
	if !ok {
		return ""
	}

	return strings.TrimSpace(value.GetStringValue())
}

func customPermissionAccessEnabled(value string) bool {
	return strings.EqualFold(value, "yes") || strings.EqualFold(value, "true")
}

func (h *HandlerV2) getCustomPermissionServices(c *gin.Context) (services.ServiceManagerI, string, error) {
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return nil, "", config.ErrProjectIdValid
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return nil, "", config.ErrEnvironmentIdValid
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
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return nil, "", err
	}

	if resource.ResourceType != pb.ResourceType_POSTGRESQL {
		h.HandleResponse(c, status_http.InvalidArgument, "resource type not supported")
		return nil, "", config.ErrProjectIdValid
	}

	service, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return nil, "", err
	}

	return service, resource.ResourceEnvironmentId, nil
}
