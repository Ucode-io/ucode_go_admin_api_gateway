package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/status_http"
	obs "ucode/ucode_go_api_gateway/genproto/versioning_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateCommit godoc
// @Security ApiKeyAuth
// @ID create_commit
// @Router /v1/commit [POST]
// @Summary Create commit
// @Description Create commit
// @Tags Commit
// @Accept json
// @Produce json
// @Param commit body obs.CreateCommitRequest true "CreateCommitRequestBody"
// @Success 201 {object} status_http.Response{data=obs.CommitWithRelease} "Commit data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateCommit(c *gin.Context) {
	var commit obs.CreateCommitRequest

	err := c.ShouldBindJSON(&commit)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.VersioningService().Commit().Create(
		context.Background(),
		&commit,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetCommitByID godoc
// @Security ApiKeyAuth
// @ID get_commit_by_id
// @Router /v1/commit/{id} [GET]
// @Summary Get commit by id
// @Description Get commit by id
// @Tags Commit
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=obs.CommitWithRelease} "CommitBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCommitByID(c *gin.Context) {
	commitID := c.Param("id")

	if !util.IsValidUUID(commitID) {
		h.handleResponse(c, status_http.InvalidArgument, "commit id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.VersioningService().Commit().GetByID(
		context.Background(),
		&obs.CommitPrimaryKey{
			Id: commitID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllCommits godoc
// @Security ApiKeyAuth
// @ID get_all_commits
// @Router /v1/commit [GET]
// @Summary Get all commits
// @Description Get all commits
// @Tags Commit
// @Accept json
// @Produce json
// @Param filters query obs.GetCommitListRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetCommitListResponse} "CommitBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllCommits(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resp, err := services.VersioningService().Commit().GetList(
		context.Background(),
		&obs.GetCommitListRequest{
			Limit:         int32(limit),
			Offset:        int32(offset),
			Search:        c.Query("search"),
			ProjectId:     projectId.(string),
			EnvironmentId: c.Query("environment_id"),
			CommitType:    c.Query("commit_type"),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
