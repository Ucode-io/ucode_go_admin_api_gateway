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

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return nil, false
	}

	resp, err := services.SessionService().V2HasAccess(
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
	return data.(*auth_service.HasAccessResponse)
}

func (h *Handler) ProjectsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		host := strings.Split(origin, "://")
		var namespace string

		if len(host) != 2 {
			h.handleResponse(c, http.BadRequest, "wrong Origin format")
			c.Abort()
			return
		}

		namespace = host[1]
		// namespace := c.GetHeader("namespace")

		// if len(namespace) == 0 {
		// 	h.handleResponse(c, http.Forbidden, "namespace required")
		// 	c.Abort()
		// 	return
		// }
		ok := h.IsServiceExists(namespace)
		if !ok {
			h.handleResponse(c, http.Forbidden, "namespace not existing")
			c.Abort()
			return
		}

		c.Set("namespace", namespace)
		c.Next()
	}
}

// func (h *ProjectsHandler) AuthMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		res, ok := h.hasAccess(c)
// 		if !ok {
// 			c.Abort()
// 			return
// 		}

// 		c.Set("Auth", res)

// 		c.Next()
// 	}
// }

// func (h *ProjectsHandler) hasAccess(c *gin.Context) (*auth_service.HasAccessResponse, bool) {
// 	bearerToken := c.GetHeader("Authorization")
// 	strArr := strings.Split(bearerToken, " ")
// 	if len(strArr) != 2 || strArr[0] != "Bearer" {
// 		h.handleResponse(c, http.Forbidden, "token error: wrong format")
// 		return nil, false
// 	}
// 	accessToken := strArr[1]
// 	namespace := c.GetHeader("namespace")

// 	h.services.Mu.Lock()
// 	services, ok := h.services.Services[namespace]
// 	if !ok {
// 		h.handleResponse(c, http.Forbidden, "nil service")
// 		return nil, false
// 	}
// 	h.services.Mu.Unlock()

// 	resp, err := services.SessionService().V2HasAccess(
// 		c.Request.Context(),
// 		&auth_service.HasAccessRequest{
// 			AccessToken:      accessToken,
// 			ProjectId:        "80cc11d9-2ee6-494a-a09d-40150d151145",
// 			ClientPlatformId: "3f6320a6-b6ed-4f5f-ad90-14a154c95ed3",
// 			Path:             helper.GetURLWithTableSlug(c),
// 			Method:           c.Request.Method,
// 		},
// 	)
// 	if err != nil {
// 		errr := status.Error(codes.PermissionDenied, "Permission denied")
// 		if errr.Error() == err.Error() {
// 			h.handleResponse(c, http.BadRequest, err.Error())
// 			return nil, false
// 		}
// 		errr = status.Error(codes.InvalidArgument, "User has been expired")
// 		if errr.Error() == err.Error() {
// 			h.handleResponse(c, http.Forbidden, err.Error())
// 			return nil, false
// 		}
// 		h.handleResponse(c, http.Unauthorized, err.Error())
// 		return nil, false
// 	}

// 	return resp, true
// }

// func (h *ProjectsHandler) GetAuthInfo(c *gin.Context) (result *auth_service.HasAccessResponse) {
// 	data, ok := c.Get("Auth")

// 	if !ok {
// 		h.handleResponse(c, http.Forbidden, "token error: wrong format")
// 		c.Abort()
// 		return
// 	}
// 	return data.(*auth_service.HasAccessResponse)
// }
