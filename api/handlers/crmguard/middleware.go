package crmguard

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
)

type contextKey struct{}

func Scoped(ctx context.Context) bool { value, _ := ctx.Value(contextKey{}).(bool); return value }
func validID(id string) bool          { _, err := uuid.Parse(id); return err == nil && len(id) == 36 }

// Runs after the established auth middleware. Never consumes HTTP identity headers.
func Middleware(cfg config.CRMNativeConfig, authorize ...func(*gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !executableEnabled(cfg) {
			c.Next()
			return
		}
		project, _ := c.Get("project_id")
		environment, _ := c.Get("environment_id")
		var actor, kind, verifiedProject, verifiedEnvironment string
		if value, ok := c.Get("auth"); ok {
			if data, ok := value.(models.AuthData); ok && data.Type == "API-KEY" {
				verifiedProject, _ = data.Data["project_id"].(string)
				verifiedEnvironment, _ = data.Data["environment_id"].(string)
				id, _ := data.Data["id"].(string)
				if id != "" && id == cfg.ServerScopeID {
					kind = "server"
				}
			}
		}
		if kind == "" {
			if value, ok := c.Get("Auth"); ok {
				if data, ok := value.(*auth.V2HasAccessUserRes); ok && data != nil {
					verifiedProject = data.GetProjectId()
					verifiedEnvironment = data.GetEnvId()
					actor = data.GetUserId()
					kind = "user"
				}
			}
		}
		// Outside the exact authenticated scope, preserve the original route. A
		// legacy auth fallback selecting the target without verified IDs is denied.
		selected := project == cfg.Project && environment == cfg.Environment
		verified := verifiedProject == cfg.Project && verifiedEnvironment == cfg.Environment
		if !selected && !verified {
			c.Next()
			return
		}
		deny := func() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "error", "message": "record unavailable"})
		}
		// This closes authenticated target aliases even when their original
		// handler selected another tenant or held a privileged managed key.
		if executableAlias(c.Request.URL.Path) && c.Request.URL.Path != canonicalPBXInvocation {
			denyExecutable(c)
			return
		}
		if !cfg.Enabled {
			c.Next()
			return
		}
		if !validID(cfg.Project) || !validID(cfg.Environment) || !validID(cfg.ResourceEnvironment) || !verified || !selected || kind == "" || (kind == "user" && !validID(actor)) {
			deny()
			return
		}
		write := nativeDataWrite(c.Request.Method, c.Request.URL.Path)
		if kind == "user" && (dangerous(c.Request.URL.Path) || (write && (!cfg.NativeWritersReady || len(authorize) != 1))) {
			deny()
			return
		}
		c.Header("Cache-Control", "no-store")
		ctx := context.WithValue(c.Request.Context(), contextKey{}, true)
		outgoing, _ := metadata.FromOutgoingContext(ctx)
		outgoing = outgoing.Copy()
		// Replace, rather than append, claims: HTTP/header supplied duplicates cannot win.
		for key, value := range map[string]string{"crm-project": cfg.Project, "crm-environment": cfg.Environment, "crm-resource": cfg.ResourceEnvironment, "crm-user": actor, "crm-kind": kind, "crm-purpose": ""} {
			outgoing.Set(key, value)
		}
		c.Request = c.Request.WithContext(metadata.NewOutgoingContext(ctx, outgoing))
		if kind == "user" && write {
			if err := authorize[0](c); err != nil {
				deny()
				return
			}
			c.Request = c.Request.WithContext(Purpose(c.Request.Context(), "native-write"))
		}
		c.Next()
	}
}

func Purpose(ctx context.Context, purpose string) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set("crm-purpose", purpose)
	return metadata.NewOutgoingContext(ctx, md)
}
func dangerous(path string) bool {
	path = strings.ToLower(path)
	if executableAlias(path) && path != canonicalPBXInvocation {
		return true
	}
	// These handlers build raw queries or can serve actorless in-memory entries
	// before the builder's row policy sees the request.
	if strings.Contains(path, "/items/") && strings.HasSuffix(path, "/filter") {
		return true
	}
	for _, term := range []string{"aggregation", "get-list-aggregate", "execute-sql", "execute_sql", "exec-query", "custom-endpoint", "custom_endpoint", "/api/", "/x-api/", "/agents/", "/agent/", "/ai-chat"} {
		if strings.Contains(path, term) {
			return true
		}
	}
	return false
}

// Native pre-write hooks run in the gateway before the builder RPC. Close the
// write route here until hook visibility/default/dedupe equivalence is bound.
func nativeDataWrite(method, path string) bool {
	if method == http.MethodGet || method == http.MethodHead {
		return false
	}
	path = strings.ToLower(path)
	native := strings.Contains(path, "/items") || strings.Contains(path, "/object/") || strings.Contains(path, "/object-upsert/")
	if !native {
		return false
	}
	if method == http.MethodPost && (strings.Contains(path, "/object/get-list/") || strings.HasSuffix(path, "/list") || strings.HasSuffix(path, "/filter")) {
		return false
	}
	return true
}
