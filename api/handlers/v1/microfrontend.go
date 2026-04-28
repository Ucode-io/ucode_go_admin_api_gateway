package v1

import (
	_ "ucode/ucode_go_api_gateway/genproto/new_function_service"
	_ "ucode/ucode_go_api_gateway/api/models"
	_ "ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// CreateMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID create_micro_frontend
// @Router /v2/functions/micro-frontend [POST]
// @Summary Create Micro Frontend
// @Description Create Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param MicroFrontend body models.CreateFunctionRequest true "MicroFrontend"
// @Success 201 {object} status_http.Response{data=new_function_service.Function} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateMicroFrontEnd(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// GetMicroFrontEndByID godoc
// @Security ApiKeyAuth
// @ID get_micro_frontend_by_id
// @Router /v2/functions/micro-frontend/{micro-frontend-id} [GET]
// @Summary Get Micro Frontend By Id
// @Description Get Micro Frontend By Id
// @Tags Functions
// @Accept json
// @Produce json
// @Param micro-frontend-id path string true "micro-frontend-id"
// @Success 200 {object} status_http.Response{data=new_function_service.Function} "Data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetMicroFrontEndByID(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// GetAllMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID get_all_micro_frontend
// @Router /v2/functions/micro-frontend [GET]
// @Summary Get All Micro Frontend
// @Description Get All Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "Data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllMicroFrontEnd(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// UpdateMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID update_micro_frontend
// @Router /v2/functions/micro-frontend [PUT]
// @Summary Update Micro Frontend
// @Description Update Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param Data body models.Function  true "Data"
// @Success 200 {object} status_http.Response{data=models.Function} "Data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateMicroFrontEnd(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// DeleteMicroFrontEnd godoc
// @Security ApiKeyAuth
// @ID delete_micro_frontend
// @Router /v2/functions/micro-frontend/{micro-frontend-id} [DELETE]
// @Summary Delete Micro Frontend
// @Description Delete Micro Frontend
// @Tags Functions
// @Accept json
// @Produce json
// @Param micro-frontend-id path string true "micro-frontend-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteMicroFrontEnd(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// PromoteMicrofrontendToMaster godoc
// @Security ApiKeyAuth
// @ID promote_microfrontend_to_master
// @Router /v2/functions/micro-frontend/promote [POST]
// @Summary Promote u-gen branch to master
// @Description Syncs all files from the u-gen branch to master, triggering the CI/CD pipeline.
// @Tags Functions
// @Accept json
// @Produce json
// @Param body body models.PushMicrofrontendChangesRequest true "repo_id required"
// @Success 200 {object} status_http.Response "OK"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) PromoteMicrofrontendToMaster(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// GetMicrofrontendCommits godoc
// @Security ApiKeyAuth
// @ID get_microfrontend_commits
// @Router /v2/functions/micro-frontend/commits [GET]
// @Summary Get commit history of a microfrontend repo
// @Description Returns commits from the master branch (published versions).
// @Tags Functions
// @Accept json
// @Produce json
// @Param repo_id query string true  "GitLab numeric project ID"
// @Param limit   query int    false "Number of commits per page (default: 20, max: 100)"
// @Param page    query int    false "Page number (default: 1)"
// @Success 200 {object} status_http.Response "Commit list"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetMicrofrontendCommits(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// GetMicrofrontendFilesAtCommit godoc
// @Security ApiKeyAuth
// @ID get_microfrontend_files_at_commit
// @Router /v2/functions/micro-frontend/files-at-commit [GET]
// @Summary Get all file contents of a microfrontend at a specific commit
// @Description Fetches the full file tree and each file's raw content at the given commit SHA for previewing a historical version.
// @Tags Functions
// @Accept json
// @Produce json
// @Param repo_id    query string true "GitLab numeric project ID"
// @Param commit_sha query string true "Commit SHA to fetch files from"
// @Success 200 {object} status_http.Response "File list with contents"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetMicrofrontendFilesAtCommit(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// RevertMicrofrontendToCommit godoc
// @Security ApiKeyAuth
// @ID revert_microfrontend_to_commit
// @Router /v2/functions/micro-frontend/revert [POST]
// @Summary Revert a microfrontend to a specific commit
// @Description Restores the snapshot of the chosen master commit to the u-gen branch. The user then publishes to go live.
// @Tags Functions
// @Accept json
// @Produce json
// @Param body body models.RevertMicrofrontendRequest true "repo_id and commit_sha"
// @Success 200 {object} status_http.Response "Reverted successfully"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) RevertMicrofrontendToCommit(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}
