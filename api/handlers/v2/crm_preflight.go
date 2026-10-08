package v2

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/structpb"
	"ucode/ucode_go_api_gateway/api/handlers/crmguard"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
)

// Called by the scope middleware before any native before/after hook. The
// original body is restored; existing native defaults and hooks still run.
func (h *HandlerV2) CRMNativePreflight(c *gin.Context) error {
	deny := errors.New("record unavailable")
	path := c.FullPath()
	single := strings.HasSuffix(path, "/object/:collection") || strings.HasSuffix(path, "/object/:collection/:object_id") || strings.HasSuffix(path, "/items/:collection") || strings.HasSuffix(path, "/items/:collection/:id") || strings.HasSuffix(path, "/:collection/items") || strings.HasSuffix(path, "/:collection/items/:id")
	if !single || (c.Request.Method == http.MethodPatch && c.Param("id") == "") || (c.Request.Method == http.MethodDelete && c.Param("id") == "" && c.Param("object_id") == "") {
		return deny
	}
	table := c.Param("collection")
	if table == "" {
		return deny
	}
	operation := ""
	switch c.Request.Method {
	case http.MethodPost:
		operation = "create"
	case http.MethodPut, http.MethodPatch:
		operation = "update"
	case http.MethodDelete:
		operation = "delete"
	default:
		return deny
	}
	if operation != "create" && c.Param("id") == "" && c.Param("object_id") == "" && c.Request.ContentLength == 0 {
		return deny
	}
	bytesBody, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
	if err != nil || len(bytesBody) >= 2<<20 {
		return deny
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bytesBody))
	body := struct {
		Data map[string]any `json:"data"`
	}{}
	if len(bytesBody) > 0 {
		if json.Unmarshal(bytesBody, &body) != nil {
			return deny
		}
	}
	if body.Data == nil {
		body.Data = map[string]any{}
	}
	id := c.Param("id")
	if id == "" {
		id = c.Param("object_id")
	}
	if posted, ok := body.Data["guid"].(string); ok && posted != "" {
		if id != "" && id != posted {
			return deny
		}
		id = posted
	}
	if operation != "create" && id == "" {
		return deny
	}
	for _, key := range []string{"objects", "ids", "guids", "from_auth_service", "already_hashed", "auth_guid", "user_id_auth", "pbx_version"} {
		if _, ok := body.Data[key]; ok {
			return deny
		}
	}
	if operation != "create" {
		if _, ok := body.Data["pbx_branch"]; ok {
			return deny
		}
	}
	resource, err := h.companyServices.ServiceResource().GetSingle(c.Request.Context(), &pb.GetSingleServiceResourceReq{ProjectId: c.GetString("project_id"), EnvironmentId: c.GetString("environment_id"), ServiceType: pb.ServiceType_BUILDER_SERVICE})
	if err != nil || resource.ResourceType != pb.ResourceType_POSTGRESQL || resource.ResourceEnvironmentId != h.baseConf.CRMNative.ResourceEnvironment {
		return deny
	}
	services, err := h.GetProjectSrvc(c.Request.Context(), c.GetString("project_id"), resource.NodeType)
	if err != nil {
		return deny
	}
	data, err := structpb.NewStruct(map[string]any{"operation": operation, "id": id, "object": body.Data})
	if err != nil {
		return deny
	}
	res, err := services.GoObjectBuilderService().Items().GetSingle(crmguard.Purpose(c.Request.Context(), "native-preflight"), &nb.CommonMessage{ProjectId: resource.ResourceEnvironmentId, TableSlug: table, CompanyProjectId: resource.ProjectId, EnvId: resource.EnvironmentId, Data: data})
	if err != nil || res.GetData().GetFields()["allowed"].GetBoolValue() != true {
		return deny
	}
	// Check both BEFORE and AFTER actions before committing a native write. An
	// AFTER action discovered only after commit cannot safely fail the request.
	events, err := services.GoObjectBuilderService().CustomEvent().GetList(crmguard.Purpose(c.Request.Context(), "native-write"), &nb.GetCustomEventsListRequest{ProjectId: resource.ResourceEnvironmentId, TableSlug: table, Method: strings.ToUpper(operation)})
	if err != nil || len(events.GetCustomEvents()) != 0 {
		return deny
	}
	return nil
}
