package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
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
// @Summary Get snapshot history of a microfrontend
// @Description Returns DB snapshots saved after each AI edit, newest first.
// @Tags Functions
// @Accept json
// @Produce json
// @Param microfrontend_id query string true  "Microfrontend UUID"
// @Param limit            query int    false "Page size (default 20)"
// @Param offset           query int    false "Offset (default 0)"
// @Success 200 {object} status_http.Response "Snapshot list"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetMicrofrontendCommits(c *gin.Context) {
	microfrontendID := c.Query("microfrontend_id")
	if microfrontendID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "microfrontend_id is required")
		return
	}

	limit := cast.ToInt(c.DefaultQuery("limit", "20"))
	offset := cast.ToInt(c.DefaultQuery("offset", "0"))

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	filterData, err := helper.ConvertMapToStruct(map[string]any{
		"microfrontend_id": microfrontendID,
		"limit":            limit,
		"offset":           offset,
	})
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to build query: "+err.Error())
		return
	}

	resp, err := service.BuilderService().ItemsService().GetList(
		c.Request.Context(),
		&obs.CommonMessage{
			TableSlug: "microfrontend_versions",
			Data:      filterData,
			ProjectId: resourceEnvId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, resp)
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
// @Summary Revert a microfrontend to a saved snapshot
// @Description Restores u-gen to the file state saved in a DB snapshot. Does NOT create a new snapshot.
// @Tags Functions
// @Accept json
// @Produce json
// @Param body body models.RevertMicrofrontendRequest true "repo_id and snapshot_id"
// @Success 200 {object} status_http.Response "Reverted successfully"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) RevertMicrofrontendToCommit(c *gin.Context) {
	var req models.RevertMicrofrontendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if req.RepoID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "repo_id is required")
		return
	}
	if req.SnapshotID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "snapshot_id is required")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	// 1. Fetch snapshot from DB.
	queryData, err := helper.ConvertMapToStruct(map[string]any{"guid": req.SnapshotID})
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to build query: "+err.Error())
		return
	}

	snapshotResp, err := service.GoObjectBuilderService().Items().GetSingle(
		c.Request.Context(),
		&nb.CommonMessage{
			TableSlug: "microfrontend_versions",
			Data:      queryData,
			ProjectId: resourceEnvId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, "failed to get snapshot: "+err.Error())
		return
	}

	snapshotMap := snapshotResp.GetData().AsMap()
	filesJSON, _ := snapshotMap["files"].(string)
	if filesJSON == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "snapshot has no files")
		return
	}

	var files []models.GitlabFileChange
	if err = json.Unmarshal([]byte(filesJSON), &files); err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to parse snapshot files: "+err.Error())
		return
	}

	// 2. Push files to u-gen via function service (no new snapshot saved).
	type pushReq struct {
		RepoID        int                       `json:"repo_id"`
		Files         []models.GitlabFileChange `json:"files"`
		CommitMessage string                    `json:"commit_message"`
	}

	repoIDInt := cast.ToInt(req.RepoID)
	commitMsg := fmt.Sprintf("revert: restore snapshot %s", req.SnapshotID)
	if msg, ok := snapshotMap["commit_message"].(string); ok && msg != "" {
		commitMsg = fmt.Sprintf("revert to: %s", msg)
	}

	bodyBytes, err := json.Marshal(pushReq{RepoID: repoIDInt, Files: files, CommitMessage: commitMsg})
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to build push request: "+err.Error())
		return
	}

	url := h.baseConf.GoFunctionServiceHost + h.baseConf.GoFunctionServiceHTTPPort +
		"/v2/functions/micro-frontend/push-changes"

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to build http request: "+err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.GetHeader("Authorization"))

	client := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "push-changes call failed: "+err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(httpResp.Body)
		h.HandleResponse(c, status_http.InternalServerError, fmt.Sprintf("push-changes returned %d: %s", httpResp.StatusCode, string(respBytes)))
		return
	}

	// Set is_current=false on all versions, then is_current=true on the reverted one.
	microfrontendID, _ := snapshotMap["microfrontend_id"].(string)
	if microfrontendID != "" {
		clearData, _ := helper.ConvertMapToStruct(map[string]any{
			"microfrontend_id": microfrontendID,
			"is_current":       false,
		})
		_, _ = service.GoObjectBuilderService().Items().MultipleUpdate(
			c.Request.Context(),
			&nb.CommonMessage{
				TableSlug: "microfrontend_versions",
				Data:      clearData,
				ProjectId: resourceEnvId,
			},
		)

		setCurrentData, _ := helper.ConvertMapToStruct(map[string]any{
			"guid":       req.SnapshotID,
			"is_current": true,
		})
		_, _ = service.GoObjectBuilderService().Items().Update(
			c.Request.Context(),
			&nb.CommonMessage{
				TableSlug: "microfrontend_versions",
				Data:      setCurrentData,
				ProjectId: resourceEnvId,
			},
		)
	}

	h.HandleResponse(c, status_http.OK, gin.H{
		"message":     fmt.Sprintf("Microfrontend reverted to version %s. Publish to go live.", req.SnapshotID),
		"snapshot_id": req.SnapshotID,
		"files":       len(files),
	})
}
