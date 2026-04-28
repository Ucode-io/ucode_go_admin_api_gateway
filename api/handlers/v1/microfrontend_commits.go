package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// GetMicrofrontendCommits godoc
// @Security ApiKeyAuth
// @ID get_microfrontend_commits
// @Router /v2/functions/micro-frontend/commits [GET]
// @Summary Get commit history of a microfrontend repo
// @Description Returns the list of commits on the master branch (or the branch specified) for a microfrontend GitLab repository.
// @Tags Functions
// @Accept json
// @Produce json
// @Param repo_id  query string true  "GitLab numeric project ID"
// @Param branch   query string false "Branch name (default: master)"
// @Param limit    query int    false "Number of commits per page (default: 20, max: 100)"
// @Param page     query int    false "Page number (default: 1)"
// @Success 200 {object} status_http.Response{data=[]models.GitlabCommit} "Commit list"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetMicrofrontendCommits(c *gin.Context) {
	repoID := c.Query("repo_id")
	if repoID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "repo_id is required")
		return
	}

	branch := c.DefaultQuery("branch", "master")
	limit := cast.ToInt(c.DefaultQuery("limit", "20"))
	page := cast.ToInt(c.DefaultQuery("page", "1"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	commits, err := h.fetchGitlabCommits(repoID, branch, limit, page)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, commits)
}

// RevertMicrofrontendToCommit godoc
// @Security ApiKeyAuth
// @ID revert_microfrontend_to_commit
// @Router /v2/functions/micro-frontend/revert [POST]
// @Summary Revert a microfrontend to a specific commit
// @Description Compares the target commit's file tree with the current master HEAD and pushes a single atomic commit that creates, updates, and deletes files to exactly restore that snapshot.
// @Tags Functions
// @Accept json
// @Produce json
// @Param body body models.RevertMicrofrontendRequest true "repo_id and commit_sha"
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
	if req.CommitSHA == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "commit_sha is required")
		return
	}

	// 1. File tree at the target commit (what we want to restore).
	targetPaths, err := h.fetchGitlabTreeAtCommit(req.RepoID, req.CommitSHA)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to fetch target file tree: "+err.Error())
		return
	}
	if len(targetPaths) == 0 {
		h.HandleResponse(c, status_http.InvalidArgument, "no files found at the given commit")
		return
	}

	// 2. File tree at current master HEAD (what is live right now).
	currentPaths, err := h.fetchGitlabTreeAtCommit(req.RepoID, "master")
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to fetch current file tree: "+err.Error())
		return
	}

	// 3. Build a lookup set for the current master files.
	currentSet := make(map[string]struct{}, len(currentPaths))
	for _, p := range currentPaths {
		currentSet[p] = struct{}{}
	}

	targetSet := make(map[string]struct{}, len(targetPaths))
	for _, p := range targetPaths {
		targetSet[p] = struct{}{}
	}

	// 4. Build GitLab commit actions.
	var actions []gitlabCommitAction

	// create or update every file that belongs to the target snapshot
	for _, path := range targetPaths {
		content, err := h.fetchGitlabFileAtCommit(req.RepoID, path, req.CommitSHA)
		if err != nil {
			h.HandleResponse(c, status_http.InternalServerError, fmt.Sprintf("failed to fetch file %s: %v", path, err))
			return
		}
		action := "update"
		if _, exists := currentSet[path]; !exists {
			action = "create"
		}
		actions = append(actions, gitlabCommitAction{
			Action:   action,
			FilePath: path,
			Content:  content,
		})
	}

	// delete files that exist on master but are absent from the target snapshot
	for _, path := range currentPaths {
		if _, exists := targetSet[path]; !exists {
			actions = append(actions, gitlabCommitAction{
				Action:   "delete",
				FilePath: path,
			})
		}
	}

	// 5. Push one atomic commit to master.
	if err = h.pushGitlabAtomicCommit(req.RepoID, req.CommitSHA, actions); err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to push revert commit: "+err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{
		"message":    "Microfrontend successfully reverted to commit " + req.CommitSHA,
		"commit_sha": req.CommitSHA,
		"files":      len(targetPaths),
	})
}

// GetMicrofrontendFilesAtCommit godoc
// @Security ApiKeyAuth
// @ID get_microfrontend_files_at_commit
// @Router /v2/functions/micro-frontend/files-at-commit [GET]
// @Summary Get all file contents of a microfrontend at a specific commit
// @Description Fetches the full file tree and each file's raw content at the given commit SHA. Use this to preview a historical version before reverting.
// @Tags Functions
// @Accept json
// @Produce json
// @Param repo_id    query string true "GitLab numeric project ID"
// @Param commit_sha query string true "Commit SHA to fetch files from"
// @Success 200 {object} status_http.Response{data=[]models.GitlabFileChange} "File list with contents"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetMicrofrontendFilesAtCommit(c *gin.Context) {
	repoID := c.Query("repo_id")
	if repoID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "repo_id is required")
		return
	}

	commitSHA := c.Query("commit_sha")
	if commitSHA == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "commit_sha is required")
		return
	}

	filePaths, err := h.fetchGitlabTreeAtCommit(repoID, commitSHA)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to fetch file tree: "+err.Error())
		return
	}
	if len(filePaths) == 0 {
		h.HandleResponse(c, status_http.InvalidArgument, "no files found at the given commit")
		return
	}

	files := make([]models.GitlabFileChange, 0, len(filePaths))
	for _, path := range filePaths {
		content, err := h.fetchGitlabFileAtCommit(repoID, path, commitSHA)
		if err != nil {
			h.HandleResponse(c, status_http.InternalServerError, fmt.Sprintf("failed to fetch file %s: %v", path, err))
			return
		}
		files = append(files, models.GitlabFileChange{
			FilePath: path,
			Content:  content,
		})
	}

	h.HandleResponse(c, status_http.OK, files)
}

// ─── GitLab API helpers ───────────────────────────────────────────────────────

type gitlabCommitAction struct {
	Action   string `json:"action"`
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
}

// pushGitlabAtomicCommit creates a single commit on master via the GitLab
// Commits API that applies all provided actions (create / update / delete).
func (h *HandlerV1) pushGitlabAtomicCommit(repoID, targetSHA string, actions []gitlabCommitAction) error {
	baseURL := strings.TrimRight(h.baseConf.GitlabBaseURL, "/")
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits", baseURL, url.PathEscape(repoID))

	payload := struct {
		Branch        string               `json:"branch"`
		CommitMessage string               `json:"commit_message"`
		Actions       []gitlabCommitAction `json:"actions"`
	}{
		Branch:        "master",
		CommitMessage: fmt.Sprintf("revert: restore snapshot from commit %s", targetSHA),
		Actions:       actions,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", h.baseConf.GitlabToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab commit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab returned %d: %s", resp.StatusCode, string(respBytes))
	}
	return nil
}

// fetchGitlabCommits calls the GitLab Commits API and returns the list of commits.
func (h *HandlerV1) fetchGitlabCommits(repoID, branch string, perPage, page int) ([]models.GitlabCommit, error) {
	baseURL := strings.TrimRight(h.baseConf.GitlabBaseURL, "/")
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits", baseURL, url.PathEscape(repoID))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	q := req.URL.Query()
	q.Set("ref_name", branch)
	q.Set("per_page", cast.ToString(perPage))
	q.Set("page", cast.ToString(page))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("PRIVATE-TOKEN", h.baseConf.GitlabToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gitlab returned %d: %s", resp.StatusCode, string(body))
	}

	var commits []models.GitlabCommit
	if err = json.Unmarshal(body, &commits); err != nil {
		return nil, fmt.Errorf("parse commits: %w", err)
	}
	return commits, nil
}

// fetchGitlabTreeAtCommit returns all blob (file) paths in the repo at the given ref.
// It pages through the GitLab tree API to handle repos with many files.
func (h *HandlerV1) fetchGitlabTreeAtCommit(repoID, ref string) ([]string, error) {
	baseURL := strings.TrimRight(h.baseConf.GitlabBaseURL, "/")
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/repository/tree", baseURL, url.PathEscape(repoID))

	client := &http.Client{Timeout: 30 * time.Second}
	var allPaths []string
	page := 1

	for {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}

		q := req.URL.Query()
		q.Set("ref", ref)
		q.Set("recursive", "true")
		q.Set("per_page", "100")
		q.Set("page", cast.ToString(page))
		req.URL.RawQuery = q.Encode()
		req.Header.Set("PRIVATE-TOKEN", h.baseConf.GitlabToken)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("gitlab tree request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read tree response: %w", err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("gitlab returned %d: %s", resp.StatusCode, string(body))
		}

		var items []models.GitlabTreeItem
		if err = json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("parse tree: %w", err)
		}

		for _, item := range items {
			if item.Type == "blob" {
				allPaths = append(allPaths, item.Path)
			}
		}

		if resp.Header.Get("X-Next-Page") == "" {
			break
		}
		page++
	}

	return allPaths, nil
}

// fetchGitlabFileAtCommit fetches raw file content at a specific ref from GitLab.
func (h *HandlerV1) fetchGitlabFileAtCommit(repoID, filePath, ref string) (string, error) {
	baseURL := strings.TrimRight(h.baseConf.GitlabBaseURL, "/")
	encodedPath := url.PathEscape(filePath)
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s/raw",
		baseURL, url.PathEscape(repoID), encodedPath)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	q := req.URL.Query()
	q.Set("ref", ref)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("PRIVATE-TOKEN", h.baseConf.GitlabToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gitlab file request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read file response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gitlab returned %d for %s: %s", resp.StatusCode, filePath, string(body))
	}

	return string(body), nil
}
