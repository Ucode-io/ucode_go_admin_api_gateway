package handlers

import (
	"strings"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		res, ok := h.hasAccess(c)
		if !ok {
			c.Abort()
			return
		}

		c.Set("Auth", res)
		c.Set("namespace", h.cfg.UcodeNamespace)
		c.Next()
	}
}

func (h *Handler) hasAccess(c *gin.Context) (*auth_service.HasAccessResponse, bool) {
	bearerToken := c.GetHeader("Authorization")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) != 2 || strArr[0] != "Bearer" {
		h.handleResponse(c, http.Forbidden, "token error: wrong format")
		return nil, false
	}
	accessToken := strArr[1]

	resp, err := h.authService.SessionService().V2HasAccess(
		c.Request.Context(),
		&auth_service.HasAccessRequest{
			AccessToken:      accessToken,
			ProjectId:        "80cc11d9-2ee6-494a-a09d-40150d151145",
			ClientPlatformId: "3f6320a6-b6ed-4f5f-ad90-14a154c95ed3",
			Path:             helper.GetURLWithTableSlug(c),
			Method:           c.Request.Method,
		},
	)

	if err != nil {
		errr := status.Error(codes.PermissionDenied, "Permission denied")
		if errr.Error() == err.Error() {
			h.handleResponse(c, http.BadRequest, err.Error())
			return nil, false
		}
		errr = status.Error(codes.InvalidArgument, "User has been expired")
		if errr.Error() == err.Error() {
			h.handleResponse(c, http.Forbidden, err.Error())
			return nil, false
		}
		h.handleResponse(c, http.Unauthorized, err.Error())
		return nil, false
	}

	return resp, true
}

func (h *Handler) GetAuthInfo(c *gin.Context) (result *auth_service.HasAccessResponse) {
	data, ok := c.Get("Auth")

	if !ok {
		h.handleResponse(c, http.Forbidden, "token error: wrong format")
		c.Abort()
		return
	}
	accessResponse, ok := data.(*auth_service.HasAccessResponse)
	if !ok {
		h.handleResponse(c, http.Forbidden, "token error: wrong format")
		c.Abort()
		return
	}

	return accessResponse
}
