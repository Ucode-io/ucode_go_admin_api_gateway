package handlers

import (
	"errors"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/config"
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

func (h *Handler) NodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Set("namespace", h.cfg.UcodeNamespace)
		c.Next()
	}
}

func (h *Handler) AuthMiddleware(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {

		var (
			res    = &auth_service.V2HasAccessUserRes{}
			ok     bool
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

			if res.ProjectId != "" {
				projectId = res.ProjectId
			}
			if res.EnvId != "" {
				environmentId = res.EnvId
			}

			c.Set("resource_id", resourceId)
			c.Set("environment_id", environmentId)
			c.Set("project_id", projectId)

		case "API-KEY":
			app_id := c.GetHeader("X-API-KEY")
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

			resource, err := h.companyServices.CompanyService().Resource().GetResourceByEnvID(
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

			// fmt.Println("\n\n >>>> api key ", apikeys, "\n\n")
			c.Set("resource_id", resource.GetResource().GetId())
			c.Set("environment_id", apikeys.GetEnvironmentId())
			c.Set("project_id", apikeys.GetProjectId())
		default:
			err := errors.New("error invalid authorization method")
			h.log.Error("--AuthMiddleware--", logger.Error(err))
			h.handleResponse(c, status_http.BadRequest, err.Error())
			c.Abort()
		}
		c.Set("Auth", res)
		c.Set("namespace", h.cfg.UcodeNamespace)

		c.Next()

	}
}

func (h *Handler) ResEnvMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		namespaceVal, ok := c.Get("namespace")
		if !ok {
			err := errors.New("error getting namespace")
			h.handleResponse(c, status_http.Forbidden, err.Error())
			return
		}

		namespace, ok := namespaceVal.(string)
		if !ok {
			err := errors.New("error namespace not ok")
			h.handleResponse(c, status_http.Forbidden, err.Error())
			return
		}

		services, err := h.GetService(namespace)
		if err != nil {
			h.handleResponse(c, status_http.Forbidden, err)
			return
		}

		resourceIDVal, ok := c.Get("resource_id")
		if !ok {
			err = errors.New("error getting resource id")
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		resourceID, ok := resourceIDVal.(string)
		if !ok {
			err = errors.New("error resource id not ok")
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		environmentIDVal, ok := c.Get("environment_id")
		if !ok {
			err = errors.New("error getting environment id")
			h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
			return
		}

		environmentID, ok := environmentIDVal.(string)
		if !ok {
			err = errors.New("error environment id not ok")
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&company_service.GetResourceEnvironmentReq{
				EnvironmentId: environmentID,
				ResourceId:    resourceID,
			},
		)
		if err != nil {
			err = errors.New("error getting resource environment id")
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		c.Set("resource_environment_id", resourceEnvironment.GetId())

		c.Next()
	}
}
