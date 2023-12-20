package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// const (
// 	SUPERADMIN_HOST string = "test.admin.u-code.io"
// 	CLIENT_HOST     string = "test.app.u-code.io"
// )

var (
	waitApiResourceMap = caching.NewConcurrentMap()
)

func (h *HandlerV1) NodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Set(h.baseConf.UcodeNamespace, h.baseConf.UcodeNamespace)
		c.Next()
	}
}

func (h *HandlerV1) AuthMiddleware(cfg config.BaseConfig) gin.HandlerFunc {
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

			res, ok = h.hasAccess(c)
			if !ok {
				h.log.Error("---ERR->AuthMiddleware->hasNotAccess-->")
				c.Abort()
				return
			}

			resourceId := c.GetHeader("Resource-Id")
			environmentId := c.GetHeader("Environment-Id")
			projectId := c.Query("project-id")

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

			// apikeysTime := time.Now()

			var (
				appIdKey, resourceAppIdKey = app_id, app_id + "resource"

				err      error
				apiJson  []byte
				apikeys  = &auth_service.GetRes{}
				resource = &company_service.GetResourceByEnvIDResponse{}
			)

			var appWaitkey = config.CACHE_WAIT + "-appID"
			_, appIdOk := h.cache.Get(appWaitkey)
			if !appIdOk {
				h.cache.Add(appWaitkey, []byte(appWaitkey), config.REDIS_KEY_TIMEOUT)
			}

			if appIdOk {
				ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
				defer cancel()

				for {
					appIdBody, ok := h.cache.Get(appIdKey)
					if ok {
						apiJson = appIdBody
						err = json.Unmarshal(appIdBody, &apikeys)
						if err != nil {
							h.handleResponse(c, status_http.BadRequest, "cant get auth info")
							c.Abort()
							return
						}
					}

					if apikeys.AppId != "" {
						break
					}

					if ctx.Err() == context.DeadlineExceeded {
						break
					}

					time.Sleep(config.REDIS_SLEEP)
				}
			}

			if apikeys.AppId == "" {
				apikeys, err = h.authService.ApiKey().GetEnvID(
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

				apiJson, err = json.Marshal(apikeys)
				if err != nil {
					h.handleResponse(c, status_http.BadRequest, "cant get auth info")
					c.Abort()
					return
				}

				go func() {
					h.cache.Add(appIdKey, apiJson, config.REDIS_TIMEOUT)
				}()
			}

			var resourceWaitKey = config.CACHE_WAIT + "-resource"
			_, resourceOk := h.cache.Get(resourceWaitKey)
			if !resourceOk {
				h.cache.Add(resourceWaitKey, []byte(resourceWaitKey), config.REDIS_KEY_TIMEOUT)
			}

			if resourceOk {
				ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
				defer cancel()

				for {
					resourceBody, ok := h.cache.Get(resourceAppIdKey)
					if ok {
						err = json.Unmarshal(resourceBody, &resource)
						if err != nil {
							h.handleResponse(c, status_http.BadRequest, "cant get auth info")
							c.Abort()
							return
						}
					}

					if resource.Resource != nil {
						break
					}

					if ctx.Err() == context.DeadlineExceeded {
						break
					}

					time.Sleep(config.REDIS_SLEEP)
				}
			}

			if resource.Resource == nil {
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

				go func() {
					resourceBody, err := json.Marshal(resource)
					if err != nil {
						h.handleResponse(c, status_http.BadRequest, "cant get auth info")
						return
					}
					h.cache.Add(resourceAppIdKey, resourceBody, config.REDIS_TIMEOUT)
				}()
			}

			data := make(map[string]interface{})
			err = json.Unmarshal(apiJson, &data)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, "cant get auth info")
				c.Abort()
				return
			}

			// fmt.Println("\n\n >>>> api key ", apikeys, "\n\n")
			c.Set("auth", models.AuthData{Type: "API-KEY", Data: data})
			c.Set("resource_id", resource.GetResource().GetId())
			c.Set("environment_id", apikeys.GetEnvironmentId())
			c.Set("project_id", apikeys.GetProjectId())

			// fmt.Println(">>>>>>>>>>>>>>>apikeysTime:", time.Since(apikeysTime))
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
		// c.Set("namespace", h.cfg.UcodeNamespace)

		c.Next()
	}
}

func (h *HandlerV1) ResEnvMiddleware() gin.HandlerFunc {
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

		_, err := h.GetService(namespace)
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

		resourceEnvironment, err := h.companyServices.Resource().GetResourceEnvironment(
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

func (h *HandlerV1) GlobalAuthMiddleware(cfg config.BaseConfig) gin.HandlerFunc {
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
		fmt.Println("\n\nquery", c.Request.URL.Query(), c.Query("environment-id"), c.Query("project-id"))
		c.Set("resource_id", c.Query("resource-id"))
		c.Set("environment_id", c.Query("environment-id"))
		c.Set("project_id", c.Query("project-id"))

		c.Set("Auth", res)
		// c.Set("namespace", h.cfg.UcodeNamespace)

		c.Next()

	}
}

func (h *HandlerV1) RedirectAuthMiddleware(cfg config.BaseConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			res = &auth_service.V2HasAccessUserRes{}
			//platformType = c.GetHeader("Platform-Type")
		)
		fmt.Println("\n\n\n ~~~~~~~> RedirectAuthMiddleware")
		app_id := c.DefaultQuery("x-api-key", "")
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

		var (
			appIdKey, resourceAppIdKey = app_id, app_id + "resource"

			err      error
			apiJson  []byte
			apikeys  = &auth_service.GetRes{}
			resource = &company_service.GetResourceByEnvIDResponse{}
		)

		var appWaitkey = config.CACHE_WAIT + "-appID"
		_, appIdOk := h.cache.Get(appWaitkey)
		if !appIdOk {
			h.cache.Add(appWaitkey, []byte(appWaitkey), config.REDIS_KEY_TIMEOUT)
		}

		if appIdOk {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				appIdBody, ok := h.cache.Get(appIdKey)
				if ok {
					apiJson = appIdBody
					err = json.Unmarshal(appIdBody, &apikeys)
					if err != nil {
						h.handleResponse(c, status_http.BadRequest, "cant get auth info")
						c.Abort()
						return
					}
				}

				if apikeys.AppId != "" {
					break
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		}

		if apikeys.AppId == "" {
			apikeys, err = h.authService.ApiKey().GetEnvID(
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

			apiJson, err = json.Marshal(apikeys)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, "cant get auth info")
				c.Abort()
				return
			}

			go func() {
				h.cache.Add(appIdKey, apiJson, config.REDIS_TIMEOUT)
			}()
		}

		var resourceWaitKey = config.CACHE_WAIT + "-resource"
		_, resourceOk := h.cache.Get(resourceWaitKey)
		if !resourceOk {
			h.cache.Add(resourceWaitKey, []byte(resourceWaitKey), config.REDIS_KEY_TIMEOUT)
		}

		if resourceOk {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				resourceBody, ok := h.cache.Get(resourceAppIdKey)
				if ok {
					err = json.Unmarshal(resourceBody, &resource)
					if err != nil {
						h.handleResponse(c, status_http.BadRequest, "cant get auth info")
						c.Abort()
						return
					}
				}

				if resource.Resource != nil {
					break
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		}

		if resource.Resource == nil {
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

			go func() {
				resourceBody, err := json.Marshal(resource)
				if err != nil {
					h.handleResponse(c, status_http.BadRequest, "cant get auth info")
					return
				}
				h.cache.Add(resourceAppIdKey, resourceBody, config.REDIS_TIMEOUT)
			}()
		}

		data := make(map[string]interface{})
		err = json.Unmarshal(apiJson, &data)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, "cant get auth info")
			c.Abort()
			return
		}

		// fmt.Println("\n\n >>>> api key ", apikeys, "\n\n")
		c.Set("auth", models.AuthData{Type: "API-KEY", Data: data})
		c.Set("resource_id", resource.GetResource().GetId())
		c.Set("environment_id", apikeys.GetEnvironmentId())
		c.Set("project_id", apikeys.GetProjectId())

		c.Set("Auth", res)
		// c.Set("namespace", h.cfg.UcodeNamespace)

		c.Next()
	}
}
