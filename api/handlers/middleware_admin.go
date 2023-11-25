package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var waitKeyMap = map[string]models.WaitKey{}

func (h *Handler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			res = &auth_service.HasAccessSuperAdminRes{}
			ok  bool
		)

		data := make(map[string]interface{})

		bearerToken := c.GetHeader("Authorization")
		strArr := strings.Split(bearerToken, " ")

		if len(strArr) < 1 && (strArr[0] != "Bearer" && strArr[0] != "API-KEY") {
			h.log.Error("---ERR->Unexpected token format")
			_ = c.AbortWithError(http.StatusForbidden, errors.New("token error: wrong format"))
			return
		}

		switch strArr[0] {
		case "Bearer":
			res, ok = h.adminHasAccess(c)
			if !ok {
				_ = c.AbortWithError(401, errors.New("unauthorized"))
				return
			}

			resourceId := c.GetHeader("Resource-Id")
			environmentId := c.GetHeader("Environment-Id")
			if res.ProjectId != "" {
				c.Set("project_id", res.ProjectId)
			}
			if res.EnvId != "" {
				environmentId = res.EnvId
			}

			apiJson, err := json.Marshal(res)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, "cant get auth info")
				c.Abort()
				return
			}

			err = json.Unmarshal(apiJson, &data)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, "cant get auth info")
				c.Abort()
				return
			}

			c.Set("auth", models.AuthData{
				Type: "BEARER",
				Data: data,
			})

			c.Set("environment_id", environmentId)
			c.Set("resource_id", resourceId)
		case "API-KEY":
			appId := c.GetHeader("X-API-KEY")

			// apikeysTime := time.Now()

			var (
				appIdWaitKey, appIdKey = appId + "X-API-KEY", appId

				apiJson []byte
				apiKey  = &auth_service.GetRes{}
				lock    = sync.RWMutex{}
				err     error
			)

			if _, ok := waitKeyMap[appIdWaitKey]; ok {

				if waitKeyMap[appIdWaitKey].Timeout.Err() == context.DeadlineExceeded {
					delete(waitKeyMap, appIdWaitKey)
				}

				if waitKeyMap[appIdWaitKey].Value == "WAIT" {
					waitTimeoutCtx, _ := context.WithTimeout(context.Background(), time.Second*20)
					for {
						redisAppId, err := h.redis.Get(context.Background(), appIdKey, h.baseConf.UcodeNamespace, config.LOW_NODE_TYPE)
						if err == nil {
							apiJson = []byte(redisAppId)
							err = json.Unmarshal([]byte(redisAppId), &apiKey)
							if err != nil {
								h.handleResponse(c, status_http.BadRequest, "cant get auth info")
								c.Abort()
								return
							}

							break
						}

						if waitTimeoutCtx.Err() == context.DeadlineExceeded {
							break
						}
						time.Sleep(time.Millisecond * 1)
					}
				}
			} else {

				lock.Lock()
				ctxWait, _ := context.WithTimeout(context.Background(), time.Second*280)
				waitKeyMap[appIdWaitKey] = models.WaitKey{Value: "WAIT", Timeout: ctxWait}
				lock.Unlock()

				apiKey, err = h.authService.ApiKey().GetEnvID(
					c.Request.Context(),
					&auth_service.GetReq{
						Id: appId,
					},
				)
				if err != nil {
					h.handleResponse(c, status_http.BadRequest, err.Error())
					c.Abort()
					return
				}

				apiJson, err = json.Marshal(apiKey)
				if err != nil {
					h.handleResponse(c, status_http.BadRequest, "cant get auth info")
					c.Abort()
					return
				}

				err = h.redis.SetX(context.Background(), appIdKey, string(apiJson), 5*time.Minute, h.baseConf.UcodeNamespace, config.LOW_NODE_TYPE)
				if err != nil {
					h.log.Error("Error while setting redis", logger.Error(err))
				}
			}

			err = json.Unmarshal(apiJson, &data)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, "cant get auth info")
				c.Abort()
				return
			}

			c.Set("auth", models.AuthData{Type: "API-KEY", Data: data})
			c.Set("environment_id", apiKey.GetEnvironmentId())
			c.Set("project_id", apiKey.GetProjectId())

			// fmt.Println("::::::apikeysTime:", time.Since(apikeysTime))
		default:
			err := errors.New("error invalid authorization method")
			h.log.Error("--AuthMiddleware--", logger.Error(err))
			h.handleResponse(c, status_http.BadRequest, err.Error())
			c.Abort()
		}
		c.Set("Auth_Admin", res)
		// c.Set("namespace", h.cfg.UcodeNamespace)
		c.Next()
	}
}

func (h *Handler) adminHasAccess(c *gin.Context) (*auth_service.HasAccessSuperAdminRes, bool) {
	bearerToken := c.GetHeader("Authorization")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) != 2 || strArr[0] != "Bearer" {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		return nil, false
	}
	accessToken := strArr[1]
	resp, err := h.authService.Session().HasAccessSuperAdmin(
		c.Request.Context(),
		&auth_service.HasAccessSuperAdminReq{
			AccessToken: accessToken,
			// ClientPlatformId: "3f6320a6-b6ed-4f5f-ad90-14a154c95ed3",
			Path:   helper.GetURLWithTableSlug(c),
			Method: c.Request.Method,
		},
	)
	if err != nil {
		errr := status.Error(codes.PermissionDenied, "Permission denied")
		if errr.Error() == err.Error() {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return nil, false
		}
		errr = status.Error(codes.InvalidArgument, "User has been expired")
		if errr.Error() == err.Error() {
			h.handleResponse(c, status_http.Forbidden, err.Error())
			return nil, false
		}
		h.handleResponse(c, status_http.Unauthorized, err.Error())
		return nil, false
	}

	return resp, true
}

func (h *Handler) adminAuthInfo(c *gin.Context) (result *auth_service.HasAccessSuperAdminRes, err error) {
	data, ok := c.Get("Auth_Admin")

	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}
	accessResponse, ok := data.(*auth_service.HasAccessSuperAdminRes)
	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}

	return accessResponse, nil
}
