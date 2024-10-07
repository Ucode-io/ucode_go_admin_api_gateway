package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *HandlerV1) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cast.ToBool(c.GetHeader("redirect")) {
			var authData = models.AuthData{}
			err := json.Unmarshal([]byte(c.GetHeader("auth")), &authData)
			if err != nil {
				h.handleResponse(c, status_http.BadRequest, "cant get auth info")
				c.Abort()
				return
			}

			c.Set("auth", authData)
			c.Set("resource_id", c.GetHeader("resource_id"))
			c.Set("environment_id", c.GetHeader("environment_id"))
			c.Set("project_id", c.GetHeader("project_id"))
		} else {

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
				projectId := c.DefaultQuery("project-id", "")

				if res.ProjectId != "" {
					projectId = res.ProjectId
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
				c.Set("project_id", projectId)
			case "API-KEY":
				appId := c.GetHeader("X-API-KEY")
				apiKey, err := h.authService.ApiKey().GetEnvID(
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
				apiJson, err := json.Marshal(apiKey)
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
					Type: "API-KEY",
					Data: data,
				})
				c.Set("environment_id", apiKey.GetEnvironmentId())
				c.Set("project_id", apiKey.GetProjectId())
			default:
				err := errors.New("error invalid authorization method")
				h.log.Error("--AuthMiddleware--", logger.Error(err))
				h.handleResponse(c, status_http.BadRequest, err.Error())
				c.Abort()
			}

			c.Set("Auth_Admin", res)
			// c.Set("namespace", h.cfg.UcodeNamespace)
		}

		c.Next()
	}
}

func (h *HandlerV1) adminHasAccess(c *gin.Context) (*auth_service.HasAccessSuperAdminRes, bool) {
	bearerToken := c.GetHeader("Authorization")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) != 2 || strArr[0] != "Bearer" {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		return nil, false
	}
	accessToken := strArr[1]
	service, conn, err := h.authService.Session(c)
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	resp, err := service.HasAccessSuperAdmin(
		c.Request.Context(),
		&auth_service.HasAccessSuperAdminReq{
			AccessToken: accessToken,
			Path:        helper.GetURLWithTableSlug(c),
			Method:      c.Request.Method,
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

func (h *HandlerV1) adminAuthInfo(c *gin.Context) (result *auth_service.HasAccessSuperAdminRes, err error) {
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
