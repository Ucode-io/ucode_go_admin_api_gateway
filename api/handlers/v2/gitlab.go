package v2

import (
	_ "ucode/ucode_go_api_gateway/api/models"
	_ "ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// Gitlab godoc
// @ID gitlab_get_branches
// @Router /gitlab/branches [GET]
// @Summary Gitlab Branches
// @Description Gitlab Branches
// @Tags Gitlab
// @Accept json
// @Produce json
// @Param token query string true "token"
// @Param project_id query string true "project_id"
// @Success 201 {object} status_http.Response{data=models.GitlabBranch} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GitlabGetBranches(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// Gitlab godoc
// @ID gitlab_get_repos
// @Router /gitlab/repos [GET]
// @Summary Gitlab Repo
// @Description Gitlab Repo
// @Tags Gitlab
// @Accept json
// @Produce json
// @Param token query string false "token"
// @Success 201 {object} status_http.Response{data=models.GitlabProjectResponse} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GitlabGetRepos(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// Gitlab godoc
// @ID gitlab_get_user
// @Router /gitlab/user [GET]
// @Summary Gitlab User
// @Description Gitlab User
// @Tags Gitlab
// @Accept json
// @Produce json
// @Param token query string false "token"
// @Success 201 {object} status_http.Response{data=models.GitlabUser} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GitlabGetUser(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// Gitlab godoc
// @ID gitlab_login
// @Router /gitlab/login [GET]
// @Summary Gitlab Login
// @Description Gitlab Login
// @Tags Gitlab
// @Accept json
// @Produce json
// @Param code query string false "code"
// @Success 201 {object} status_http.Response{data=string} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GitlabLogin(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}
