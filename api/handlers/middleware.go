package handlers

import (
	"errors"
	"fmt"
	"strings"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"ucode/ucode_go_api_gateway/api/status_http"
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
			res *auth_service.V2HasAccessUserRes
			ok  bool
		)
		//host := c.Request.Host
		origin := c.GetHeader("Origin")
		fmt.Println("HOST", origin)
		if strings.Contains(origin, CLIENT_HOST) {
			fmt.Println("HOST::::2", origin)
			res, ok = h.hasAccess(c)
			if !ok {
				c.Abort()
				return
			}
		}

		resourceId := c.GetHeader("Resource-Id")
		environmentId := c.GetHeader("Environment-Id")
		fmt.Println(res)

		c.Set("Auth", res)
		c.Set("resource_id", resourceId)
		c.Set("environment_id", environmentId)
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
	fmt.Println("resp", resp)
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
