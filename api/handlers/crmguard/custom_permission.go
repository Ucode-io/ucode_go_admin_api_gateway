package crmguard

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"ucode/ucode_go_api_gateway/config"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
)

// The role selectors describe the fresh login, never a role supplied by the caller.
func CustomPermissionAccesses(cfg config.CRMNativeConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Scoped(c.Request.Context()) {
			if cfg.Enabled && c.Query("project-id") == cfg.Project {
				denyExecutable(c)
				return
			}
			c.Next()
			return
		}
		deny := func() { denyExecutable(c) }
		value, _ := c.Get("Auth")
		login, ok := value.(*auth.V2HasAccessUserRes)
		query, err := url.ParseQuery(c.Request.URL.RawQuery)
		parts := strings.Fields(c.GetHeader("Authorization"))
		if !ok || login == nil || err != nil || len(parts) != 2 || parts[0] != "Bearer" || !validID(login.GetUserId()) || !validID(login.GetRoleId()) || !validID(login.GetClientTypeId()) || c.Request.Method != http.MethodGet || c.Request.URL.Path != "/v1/custom-permission/accesses" {
			deny()
			return
		}
		for key, values := range query {
			if len(values) != 1 || (key != "project-id" && key != "role_id" && key != "client_type_id" && key != "parent_id") {
				deny()
				return
			}
		}
		if query.Get("project-id") != login.GetProjectId() || query.Get("role_id") != login.GetRoleId() || query.Get("client_type_id") != login.GetClientTypeId() || (query.Get("parent_id") != "" && !validID(query.Get("parent_id"))) || len(c.Request.Header.Values("Environment-Id")) != 1 || c.GetHeader("Environment-Id") != login.GetEnvId() || len(c.Request.Header.Values("Authorization")) != 1 || c.GetHeader("redirect") != "" {
			deny()
			return
		}
		if c.Request.Body != nil {
			var byteBody [1]byte
			if count, err := c.Request.Body.Read(byteBody[:]); count != 0 || err != io.EOF {
				deny()
				return
			}
		}
		ctx := Purpose(c.Request.Context(), "own-navigation-permissions")
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set("crm-role", login.GetRoleId())
		md.Set("crm-client-type", login.GetClientTypeId())
		c.Request = c.Request.WithContext(metadata.NewOutgoingContext(ctx, md))
		c.Next()
	}
}
