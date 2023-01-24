package handlers

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	SUPERADMIN_HOST string = "admin.u-code.io"
	CLIENT_HOST     string = "app.u-code.io"
)

func (h *Handler) NodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Set("namespace", h.cfg.UcodeNamespace)
		c.Next()
	}
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		var (
			res    = &auth_service.V2HasAccessUserRes{}
			ok     bool
			origin = c.GetHeader("Origin")
		)

		bearerToken := c.GetHeader("Authorization")
		strArr := strings.Split(bearerToken, " ")

		if len(strArr) < 1 && (strArr[0] != "Bearer" || strArr[0] != "API-KEY") {
			_ = c.AbortWithError(http.StatusForbidden, errors.New("token error: wrong format"))
			return
		}

		switch strArr[0] {
		case "Bearer":
			if strings.Contains(origin, CLIENT_HOST) {
				res, ok = h.hasAccess(c)
				if !ok {
					c.Abort()
					return
				}

				resourceId := c.GetHeader("Resource-Id")
				environmentId := c.GetHeader("Environment-Id")

				c.Set("resource_id", resourceId)
				c.Set("environment_id", environmentId)
			}

		case "API-KEY":
			// X-API-KEY contains app_id
			// get environment_id by app_id from api_keys table
			// get resource_id from resource_environment table filter by environment_id

			// c.Set("resource_id", resourceId)
			// c.Set("environment_id", environmentId)

			tDec, _ := base64.StdEncoding.DecodeString(bearerToken)
			if string(tDec) != config.API_KEY_SECRET {
				_ = c.AbortWithError(http.StatusForbidden, errors.New("wrong token"))
				return
			}

			app_id := c.GetHeader("X-API-KEY")
			apikeys, err := h.authService.ApiKeyService().GetEnvID(c, &auth_service.GetReq{
				Id: app_id,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			resource, err := h.companyServices.ResourceService().GetResourceByEnvID(c, &company_service.GetResourceByEnvIDRequest{
				EnvId: apikeys.GetEnvironmentId(),
			})
			if err != nil{
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			c.Set("resource_id", resource.GetResource().GetId())
			c.Set("environment_id", apikeys.GetEnvironmentId())

		}

		c.Set("Auth", res)

		c.Set("namespace", h.cfg.UcodeNamespace)
		c.Next()
	}
}

func (h *Handler) hasAccess(c *gin.Context) (*auth_service.V2HasAccessUserRes, bool) {
	bearerToken := c.GetHeader("Authorization")
	projectId := c.DefaultQuery("project_id", "")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) != 2 || strArr[0] != "Bearer" {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		return nil, false
	}
	accessToken := strArr[1]
	resp, err := h.authService.SessionService().V2HasAccessUser(
		c.Request.Context(),
		&auth_service.V2HasAccessUserReq{
			AccessToken: accessToken,
			ProjectId:   projectId,
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

func (h *Handler) GetAuthInfo(c *gin.Context) (result *auth_service.V2HasAccessUserRes, err error) {
	data, ok := c.Get("Auth")

	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}
	accessResponse, ok := data.(*auth_service.V2HasAccessUserRes)
	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}

	return accessResponse, nil
}
