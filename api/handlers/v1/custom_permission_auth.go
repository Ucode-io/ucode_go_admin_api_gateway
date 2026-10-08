package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/config"
)

// Only this existing navigation read uses CRM login authentication when scope is on.
func (h *HandlerV1) CustomPermissionAccessAuthMiddleware(cfg config.BaseConfig) gin.HandlerFunc {
	return customPermissionAccessAuth(cfg.CRMNative, h.AuthMiddleware(cfg), h.AdminAuthMiddleware())
}

func customPermissionAccessAuth(cfg config.CRMNativeConfig, login, admin gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled || c.Query("project-id") != cfg.Project {
			admin(c)
			return
		}
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(c.Request.Header.Values("Authorization")) != 1 || len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" || c.GetHeader("redirect") != "" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		login(c)
	}
}
