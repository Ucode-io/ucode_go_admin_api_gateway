package v2

import (
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/pkg/gitlab"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV2) GithubLogin(c *gin.Context) {
	var (
		githubLoginRequest models.GithubLogin

		accessTokenUrl string = "https://github.com/login/oauth/access_token"
	)

	err := c.ShouldBindJSON(&githubLoginRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	param := map[string]interface{}{
		"client_id":     h.baseConf.GithubClientId,
		"client_secret": h.baseConf.GithubClientSecret,
		"code":          githubLoginRequest.Code,
	}

	result, err := gitlab.MakeRequest("POST", accessTokenUrl, "", param)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	if _, ok := result["error"]; ok {
		h.handleResponse(c, status_http.InvalidArgument, result["error_description"])
		return
	}

	h.handleResponse(c, status_http.OK, result)
}
