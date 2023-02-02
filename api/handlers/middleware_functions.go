package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) hasAccess(c *gin.Context) (*auth_service.V2HasAccessUserRes, bool) {
	bearerToken := c.GetHeader("Authorization")
	projectId := c.DefaultQuery("project_id", "")

	strArr := strings.Split(bearerToken, " ")
	if len(strArr) != 2 || strArr[0] != "Bearer" {
		h.handleResponse(c, status_http.Forbidden, "token error: wrong format")
		return nil, false
	}

	accessToken := strArr[1]
	resp, err := h.authService.Session().V2HasAccessUser(
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

func (h *Handler) CreateAutoCommit(c *gin.Context, environmentID, commitType string) (commitID int64, commitGuid string, err error) {
	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return 0, "", err
	}

	fmt.Println("auethInfo.GetUsrId()", authInfo.GetUserId())
	fmt.Println("authInfo.GetProjectId()", authInfo.GetProjectId())
	fmt.Println("environmentID", environmentID)

	commit, err := h.companyServices.CompanyService().Commit().Insert(
		c.Request.Context(),
		&company_service.CreateCommitRequest{
			AuthorId:      authInfo.GetUserId(),
			ProjectId:     authInfo.GetProjectId(),
			EnvironmentId: environmentID,
			CommitType:    config.COMMIT_TYPE_APP,
			Name:          fmt.Sprintf("Auto Created Commit - %s", time.Now().Format(time.RFC1123)),
		},
	)
	if err != nil {
		return 0, "", err
	}

	return commit.GetCommitId(), commit.GetId(), nil
}
