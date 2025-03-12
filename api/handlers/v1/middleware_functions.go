package v1

import (
	"errors"
	"strings"
	"ucode/ucode_go_api_gateway/api/status_http"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *HandlerV1) hasAccess(c *gin.Context) (*auth.V2HasAccessUserRes, bool) {
	var (
		bearerToken = c.GetHeader("Authorization")
		strArr      = strings.Split(bearerToken, " ")
	)

	if len(strArr) != 2 || strArr[0] != "Bearer" {
		h.log.Error("---ERR->HasAccess->Unexpected token format")
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		return nil, false
	}

	accessToken := strArr[1]
	service, conn, err := h.authService.Session(c)
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	resp, err := service.V2HasAccessUser(
		c.Request.Context(), &auth.V2HasAccessUserReq{
			AccessToken:   accessToken,
			Path:          helper.GetURLWithTableSlug(c),
			Method:        c.Request.Method,
			ProjectId:     c.Query("project-id"),
			EnvironmentId: c.GetHeader("Environment-Id"),
		},
	)
	if err != nil {
		errr := status.Error(codes.PermissionDenied, "Permission denied")
		if errr.Error() == err.Error() {
			h.log.Error("---ERR->HasAccess->Permission--->", logger.Error(err))
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return nil, false
		}
		errr = status.Error(codes.InvalidArgument, "User has been expired")
		if errr.Error() == err.Error() {
			h.log.Error("---ERR->HasAccess->User Expired-->")
			h.handleResponse(c, status_http.Forbidden, err.Error())
			return nil, false
		}
		errr = status.Error(codes.Unavailable, "User not access environment")
		if errr.Error() == err.Error() {
			h.log.Error("---ERR->HasAccess->User not access environment-->")
			h.handleResponse(c, status_http.Unauthorized, err.Error())
			return nil, false
		}

		h.log.Error("---ERR->HasAccess->Session->V2HasAccessUser--->", logger.Error(err))
		h.handleResponse(c, status_http.Unauthorized, err.Error())
		return nil, false
	}

	return resp, true
}

func (h *HandlerV1) GetAuthInfo(c *gin.Context) (result *auth.V2HasAccessUserRes, err error) {
	data, ok := c.Get("Auth")
	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}

	accessResponse, ok := data.(*auth.V2HasAccessUserRes)
	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}

	return accessResponse, nil
}

func (h *HandlerV1) GetAuthAdminInfo(c *gin.Context) (result *auth.HasAccessSuperAdminRes, err error) {
	data, ok := c.Get("Auth_Admin")
	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}

	accessResponse, ok := data.(*auth.HasAccessSuperAdminRes)
	if !ok {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		c.Abort()
		return nil, errors.New("token error: wrong format")
	}

	return accessResponse, nil
}
