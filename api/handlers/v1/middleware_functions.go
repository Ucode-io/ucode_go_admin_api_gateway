package v1

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/versioning_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *HandlerV1) hasAccess(c *gin.Context) (*auth_service.V2HasAccessUserRes, bool) {
	bearerToken := c.GetHeader("Authorization")
	// projectId := c.DefaultQuery("project_id", "")

	strArr := strings.Split(bearerToken, " ")

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
		c.Request.Context(),
		&auth_service.V2HasAccessUserReq{
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

func (h *HandlerV1) GetAuthInfo(c *gin.Context) (result *auth_service.V2HasAccessUserRes, err error) {
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

func (h *HandlerV1) GetAuthAdminInfo(c *gin.Context) (result *auth_service.HasAccessSuperAdminRes, err error) {
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

func (h *HandlerV1) CreateAutoCommit(c *gin.Context, environmentID, commitType string) (versionId, commitGuid string, err error) {
	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return "", "", err
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	if !util.IsValidUUID(authInfo.GetUserId()) {
		err := errors.New("invalid or missing user id")
		h.log.Error("--CreateAutoCommit--", logger.Error(err))
		return "", "", err
	}

	if !util.IsValidUUID(authInfo.GetProjectId()) {
		err := errors.New("invalid or missing project id")
		h.log.Error("--CreateAutoCommit--", logger.Error(err))
		return "", "", err
	}

	if !util.IsValidUUID(environmentID) {
		err := errors.New("invalid or missing environment id")
		h.log.Error("--CreateAutoCommit--", logger.Error(err))
		return "", "", err
	}

	commit, err := services.VersioningService().Commit().Insert(
		c.Request.Context(),
		&versioning_service.CreateCommitRequest{
			AuthorId:      authInfo.GetUserId(),
			ProjectId:     authInfo.GetProjectId(),
			EnvironmentId: environmentID,
			CommitType:    config.COMMIT_TYPE_APP,
			Name:          fmt.Sprintf("Auto Created Commit - %s", time.Now().Format(time.RFC1123)),
		},
	)
	if err != nil {
		return "", "", err
	}

	return commit.GetVersionId(), commit.GetCommitId(), nil
}

func (h *HandlerV1) CreateAutoCommitForAdminChange(c *gin.Context, environmentID, commitType string, project_id string) (versionId, commitGuid string, err error) {
	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		return "", "", err
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	if !util.IsValidUUID(authInfo.GetUserId()) {
		err := errors.New("invalid or missing user id")
		h.log.Error("--CreateAutoCommit--", logger.Error(err))
		return "", "", err
	}

	if !util.IsValidUUID(project_id) {
		err := errors.New("invalid or missing project id")
		h.log.Error("--CreateAutoCommit--", logger.Error(err))
		return "", "", err
	}

	if !util.IsValidUUID(environmentID) {
		err := errors.New("invalid or missing environment id")
		h.log.Error("--CreateAutoCommit--", logger.Error(err))
		return "", "", err
	}

	commit, err := services.VersioningService().Commit().Insert(
		c.Request.Context(),
		&versioning_service.CreateCommitRequest{
			AuthorId:      authInfo.GetUserId(),
			ProjectId:     project_id,
			EnvironmentId: environmentID,
			CommitType:    config.COMMIT_TYPE_APP,
			Name:          fmt.Sprintf("Auto Created Commit - %s", time.Now().Format(time.RFC1123)),
		},
	)
	if err != nil {
		return "", "", err
	}

	return commit.GetVersionId(), commit.GetCommitId(), nil
}
