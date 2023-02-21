package handlers

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		res, ok := h.adminHasAccess(c)
		fmt.Println(res)
		if !ok {
			_ = c.AbortWithError(401, errors.New("unauthorized"))
			return
		}
		log.Printf("AUTH OBJECT: %+v", res)
		c.Set("Auth", res)
		c.Set("namespace", h.cfg.UcodeNamespace)
		c.Set("environment_id", c.GetHeader("Environment-Id"))
		c.Set("resource_id", c.GetHeader("Resource-Id"))
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
	data, ok := c.Get("Auth")

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
