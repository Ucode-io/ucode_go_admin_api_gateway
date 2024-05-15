package v2

import (
	"errors"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// const (
// 	SUPERADMIN_HOST string = "test.admin.u-code.io"
// 	CLIENT_HOST     string = "test.app.u-code.io"
// )
 
func (h *HandlerV2) NodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Set("namespace", h.baseConf.UcodeNamespace)
		c.Next()
	} 
}

func (h *HandlerV2) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			res = &auth_service.V2HasAccessUserRes{}
			ok  bool
			//platformType = c.GetHeader("Platform-Type")
		)

		bearerToken := c.GetHeader("Authorization")
		strArr := strings.Split(bearerToken, " ")

		if len(strArr) < 1 && (strArr[0] != "Bearer" && strArr[0] != "API-KEY") {
			h.log.Error("---ERR->Unexpected token format")
			_ = c.AbortWithError(http.StatusForbidden, errors.New("token error: wrong format"))
			return
		}
		switch strArr[0] {
		case "Bearer":
			//if platformType != cfg.PlatformType {
			res, ok = h.hasAccess(c)
			if !ok {
				h.log.Error("---ERR->AuthMiddleware->hasNotAccess-->")
				c.Abort()
				return
			}
			//}
			resourceId := c.GetHeader("Resource-Id")
			environmentId := c.GetHeader("Environment-Id")
			projectId := c.Query("Project-Id")
			userId := c.Query("User-Id")

			if res.ProjectId != "" {
				projectId = res.ProjectId
			}
			if res.EnvId != "" {
				environmentId = res.EnvId
			}
			if res.UserId != "" {
				userId = res.UserId
			}

			c.Set("resource_id", resourceId)
			c.Set("environment_id", environmentId)
			c.Set("project_id", projectId)
			c.Set("user_id", userId)

		case "API-KEY":
			app_id := c.GetHeader("X-API-KEY")

			if app_id == "" {
				err := errors.New("error invalid api-key method")
				h.log.Error("--AuthMiddleware--", logger.Error(err))
				c.JSON(401, struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{
					Code:    401,
					Message: "The request requires an user authentication.",
				})
				c.Abort()
				return
			}

			apikeys, err := h.authService.ApiKey().GetEnvID(
				c.Request.Context(),
				&auth_service.GetReq{
					Id: app_id,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, err.Error())
				c.Abort()
				return
			}

			resource, err := h.companyServices.Resource().GetResourceByEnvID(
				c.Request.Context(),
				&company_service.GetResourceByEnvIDRequest{
					EnvId: apikeys.GetEnvironmentId(),
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, err.Error())
				c.Abort()
				return
			}

			c.Set("resource_id", resource.GetResource().GetId())
			c.Set("environment_id", apikeys.GetEnvironmentId())
			c.Set("project_id", apikeys.GetProjectId())
		default:
			if !strings.Contains(c.Request.URL.Path, "api") {
				err := errors.New("error invalid authorization method")
				h.log.Error("--AuthMiddleware--", logger.Error(err))
				h.handleResponse(c, status_http.BadRequest, err.Error())
				c.Abort()
			} else {
				err := errors.New("error invalid authorization method")
				h.log.Error("--AuthMiddleware--", logger.Error(err))
				c.JSON(401, struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{
					Code:    401,
					Message: "The request requires an user authentication.",
				})
				c.Abort()
			}

		}
		c.Set("Auth", res)
		c.Set("namespace", h.baseConf.UcodeNamespace)
		c.Next()

	}
}

func (h *HandlerV2) GlobalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		var (
			res = &auth_service.V2HasAccessUserRes{}
			ok  bool
		)

		bearerToken := c.GetHeader("Authorization")
		strArr := strings.Split(bearerToken, " ")

		if len(strArr) < 1 && (strArr[0] != "Bearer" && strArr[0] != "API-KEY") {
			h.log.Error("---ERR->Unexpected token format")
			_ = c.AbortWithError(http.StatusForbidden, errors.New("token error: wrong format"))
			return
		}
		switch strArr[0] {
		case "Bearer":
			//if platformType != cfg.PlatformType {
			res, ok = h.hasAccess(c)
			if !ok {
				h.log.Error("---ERR->AuthMiddleware->hasNotAccess-->")
				c.Abort()
				return
			}

		case "API-KEY":
			app_id := c.GetHeader("X-API-KEY")
			_, err := h.authService.ApiKey().GetEnvID(
				c.Request.Context(),
				&auth_service.GetReq{
					Id: app_id,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, err.Error())
				c.Abort()
				return
			}

		default:
			err := errors.New("error invalid authorization method")
			h.log.Error("--AuthMiddleware--", logger.Error(err))
			h.handleResponse(c, status_http.BadRequest, err.Error())
			c.Abort()
		}
		c.Set("resource_id", c.Query("resource-id"))
		c.Set("environment_id", c.Query("environment-id"))
		c.Set("project_id", c.Query("project-id"))

		c.Set("Auth", res)

		c.Next()

	}
}
