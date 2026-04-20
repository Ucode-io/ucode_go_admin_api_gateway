package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	githubAuthURL     = "https://github.com/login/oauth/authorize"
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubAPIUserURL  = "https://api.github.com/user"
	githubAPIReposURL = "https://api.github.com/user/repos"
	githubStatePrefix = "github:state:"
	githubStateTTL    = 15 * time.Minute
)

// githubStatePayload is stored in Redis during the OAuth flow to carry context across the redirect.
type githubStatePayload struct {
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	UserID        string `json:"user_id"`
}

// getProjectAndEnv is a shared helper that reads project_id and environment_id
// from the gin context (set by AuthMiddleware from the Bearer token / API key).
func getProjectAndEnv(c *gin.Context) (projectID, environmentID string, ok bool) {
	pid, pidOk := c.Get("project_id")
	eid, eidOk := c.Get("environment_id")
	if !pidOk || !eidOk {
		return "", "", false
	}
	projectID, _ = pid.(string)
	environmentID, _ = eid.(string)
	return projectID, environmentID, projectID != "" && environmentID != ""
}

// GithubConnect initiates the GitHub OAuth flow.
// Reads project_id and environment_id from the auth context (set by AuthMiddleware),
// stores a CSRF state in Redis, then redirects the user to GitHub.
//
// @Security ApiKeyAuth
// @ID github_connect
// @Router /v1/github/connect [GET]
// @Summary Initiate GitHub OAuth
// @Tags GitHub Integration
// @Success 302 "Redirect to GitHub"
// @Failure 400 {object} status_http.Response{data=string}
func (h *HandlerV1) GithubConnect(c *gin.Context) {
	projectID, environmentID, ok := getProjectAndEnv(c)
	if !ok {
		h.HandleResponse(c, status_http.InvalidArgument, "project_id and environment_id not found in token")
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	payload := githubStatePayload{
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		UserID:        userIDStr,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to encode state")
		return
	}

	state := uuid.NewString()
	redisKey := githubStatePrefix + state
	if err := h.redis.SetX(c.Request.Context(), redisKey, string(payloadBytes), githubStateTTL, "", ""); err != nil {
		h.HandleResponse(c, status_http.InternalServerError, "failed to store OAuth state")
		return
	}

	params := url.Values{}
	params.Set("client_id", h.baseConf.GithubClientID)
	params.Set("redirect_uri", h.baseConf.GithubRedirectURI)
	params.Set("state", state)
	params.Set("scope", "repo read:user user:email")

	c.JSON(http.StatusCreated, githubAuthURL+"?"+params.Encode())
}

// GithubCallback handles the OAuth callback from GitHub.
// This endpoint is public (no auth middleware) — GitHub calls it after user grants access.
// It validates the CSRF state, exchanges the code for a token, fetches the GitHub user,
// saves the integration, then redirects the user back to the frontend.
//
// @ID github_callback
// @Router /v1/github/callback [GET]
// @Summary GitHub OAuth Callback
// @Tags GitHub Integration
// @Param code query string true "OAuth code from GitHub"
// @Param state query string true "CSRF state token"
// @Success 302 "Redirect to frontend success/error page"
func (h *HandlerV1) GithubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	errorURL := h.baseConf.GithubFrontendErrorURL

	if code == "" || state == "" {
		c.Redirect(http.StatusTemporaryRedirect, errorURL+"?reason=missing_params")
		return
	}

	// Validate CSRF state — must exist in Redis
	redisKey := githubStatePrefix + state
	payloadStr, err := h.redis.Get(c.Request.Context(), redisKey, "", "")
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, errorURL+"?reason=invalid_state")
		return
	}

	// One-time use — delete immediately after reading
	_ = h.redis.Del(c.Request.Context(), redisKey, "", "")

	var stateData githubStatePayload
	if err := json.Unmarshal([]byte(payloadStr), &stateData); err != nil {
		c.Redirect(http.StatusTemporaryRedirect, errorURL+"?reason=state_parse_error")
		return
	}

	token, err := h.exchangeGithubCode(c.Request.Context(), code)
	if err != nil {
		h.log.Error("github: token exchange failed: " + err.Error())
		c.Redirect(http.StatusTemporaryRedirect, errorURL+"?reason=token_exchange_failed")
		return
	}

	ghUser, err := h.fetchGithubUser(c.Request.Context(), token)
	if err != nil {
		h.log.Error("github: fetch user failed: " + err.Error())
		c.Redirect(http.StatusTemporaryRedirect, errorURL+"?reason=fetch_user_failed")
		return
	}

	integrationID, err := h.upsertGithubIntegration(c.Request.Context(), token, ghUser, stateData)
	if err != nil {
		h.log.Error("github: save integration failed: " + err.Error())
		c.Redirect(http.StatusTemporaryRedirect, errorURL+"?reason=save_failed")
		return
	}

	successURL := fmt.Sprintf("%s?integration_id=%s&username=%s",
		h.baseConf.GithubFrontendSuccessURL, integrationID, ghUser.Login)
	c.Redirect(http.StatusTemporaryRedirect, successURL)
}

// GithubGetIntegration returns the stored GitHub integration for the current project/environment.
// project_id and environment_id are taken from the auth context.
//
// @Security ApiKeyAuth
// @ID github_get_integration
// @Router /v1/github/integration [GET]
// @Summary Get GitHub Integration
// @Tags GitHub Integration
// @Success 200 {object} status_http.Response{data=models.GithubIntegration}
// @Failure 404 {object} status_http.Response{data=string}
func (h *HandlerV1) GithubGetIntegration(c *gin.Context) {
	projectID, environmentID, ok := getProjectAndEnv(c)
	if !ok {
		h.HandleResponse(c, status_http.InvalidArgument, "project_id and environment_id not found in token")
		return
	}

	resp, err := h.companyServices.IntegrationResource().GetIntegrationResourceList(
		c.Request.Context(),
		&pb.GetListIntegrationResourceRequest{
			ProjectId:     projectID,
			EnvironmentId: environmentID,
			Type:          pb.ResourceType_GITHUB,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	integrations := resp.GetIntegrationResources()
	if len(integrations) == 0 {
		h.HandleResponse(c, status_http.NotFound, "no GitHub integration found")
		return
	}

	ir := integrations[0]
	h.HandleResponse(c, status_http.OK, models.GithubIntegration{
		ID:            ir.GetId(),
		Username:      ir.GetUsername(),
		Name:          ir.GetName(),
		ProjectID:     ir.GetProjectId(),
		EnvironmentID: ir.GetEnvironmentId(),
	})
}

// GithubDeleteIntegration removes a GitHub integration by its ID.
//
// @Security ApiKeyAuth
// @ID github_delete_integration
// @Router /v1/github/integration/:id [DELETE]
// @Summary Delete GitHub Integration
// @Tags GitHub Integration
// @Param id path string true "Integration ID"
// @Success 200 {object} status_http.Response{data=string}
func (h *HandlerV1) GithubDeleteIntegration(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "integration id is required")
		return
	}

	_, err := h.companyServices.IntegrationResource().DeleteIntegrationResource(
		c.Request.Context(),
		&pb.IntegrationResourcePrimaryKey{Id: id},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, "GitHub integration deleted")
}

// GithubCreateRepo creates a new repository on the user's GitHub account.
// Uses the stored token for the current project/environment.
//
// @Security ApiKeyAuth
// @ID github_create_repo
// @Router /v1/github/repo [POST]
// @Summary Create GitHub Repository
// @Tags GitHub Integration
// @Accept json
// @Produce json
// @Param body body models.GithubCreateRepoRequest true "Repo details"
// @Success 201 {object} status_http.Response{data=models.GithubRepo}
// @Failure 400 {object} status_http.Response{data=string}
func (h *HandlerV1) GithubCreateRepo(c *gin.Context) {
	var req models.GithubCreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	projectID, environmentID, ok := getProjectAndEnv(c)
	if !ok {
		h.HandleResponse(c, status_http.InvalidArgument, "project_id and environment_id not found in token")
		return
	}

	token, err := h.getGithubToken(c.Request.Context(), projectID, environmentID)
	if err != nil {
		h.HandleResponse(c, status_http.NotFound, "GitHub integration not found: "+err.Error())
		return
	}

	repo, err := h.createGithubRepo(c.Request.Context(), token, req)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.Created, repo)
}

// GithubGetRepoList returns all repositories for the authenticated GitHub user.
// Uses the stored token for the current project/environment.
//
// @Security ApiKeyAuth
// @ID github_list_repos
// @Router /v1/github/repos [GET]
// @Summary List GitHub Repositories
// @Tags GitHub Integration
// @Success 200 {object} status_http.Response{data=[]models.GithubRepo}
func (h *HandlerV1) GithubGetRepoList(c *gin.Context) {
	projectID, environmentID, ok := getProjectAndEnv(c)
	if !ok {
		h.HandleResponse(c, status_http.InvalidArgument, "project_id and environment_id not found in token")
		return
	}

	token, err := h.getGithubToken(c.Request.Context(), projectID, environmentID)
	if err != nil {
		h.HandleResponse(c, status_http.NotFound, "GitHub integration not found: "+err.Error())
		return
	}

	repos, err := h.listGithubRepos(c.Request.Context(), token)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, repos)
}

// ─── private helpers ──────────────────────────────────────────────────────────

// exchangeGithubCode exchanges the OAuth authorization code for an access token.
func (h *HandlerV1) exchangeGithubCode(ctx context.Context, code string) (string, error) {
	body := url.Values{}
	body.Set("client_id", h.baseConf.GithubClientID)
	body.Set("client_secret", h.baseConf.GithubClientSecret)
	body.Set("code", code)
	body.Set("redirect_uri", h.baseConf.GithubRedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, bytes.NewBufferString(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp models.GithubTokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("github: %s — %s", tokenResp.Error, tokenResp.ErrorDescription)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("github returned empty access token")
	}
	return tokenResp.AccessToken, nil
}

// fetchGithubUser calls the GitHub API to get the authenticated user's profile.
func (h *HandlerV1) fetchGithubUser(ctx context.Context, token string) (*models.GithubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user API: status %d — %s", resp.StatusCode, string(b))
	}

	var user models.GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// upsertGithubIntegration saves the GitHub token to the company-service.
// If an integration already exists for this project/env, it is deleted first.
func (h *HandlerV1) upsertGithubIntegration(ctx context.Context, token string, user *models.GithubUser, state githubStatePayload) (string, error) {
	existing, err := h.companyServices.IntegrationResource().GetIntegrationResourceList(ctx, &pb.GetListIntegrationResourceRequest{
		ProjectId:     state.ProjectID,
		EnvironmentId: state.EnvironmentID,
		Type:          pb.ResourceType_GITHUB,
	})
	if err == nil {
		for _, ir := range existing.GetIntegrationResources() {
			_, _ = h.companyServices.IntegrationResource().DeleteIntegrationResource(ctx, &pb.IntegrationResourcePrimaryKey{Id: ir.GetId()})
		}
	}

	displayName := user.Name
	if displayName == "" {
		displayName = user.Login
	}

	created, err := h.companyServices.IntegrationResource().CreateIntegrationResource(ctx, &pb.CreateIntegrationResourceRequest{
		Token:         token,
		ProjectId:     state.ProjectID,
		EnvironmentId: state.EnvironmentID,
		Username:      user.Login,
		Name:          displayName,
	})
	if err != nil {
		return "", err
	}
	return created.GetId(), nil
}

// getGithubToken retrieves the stored GitHub access token for a given project/environment.
func (h *HandlerV1) getGithubToken(ctx context.Context, projectID, environmentID string) (string, error) {
	list, err := h.companyServices.IntegrationResource().GetIntegrationResourceList(ctx, &pb.GetListIntegrationResourceRequest{
		ProjectId:     projectID,
		EnvironmentId: environmentID,
		Type:          pb.ResourceType_GITHUB,
	})
	if err != nil {
		return "", err
	}

	integrations := list.GetIntegrationResources()
	if len(integrations) == 0 {
		return "", fmt.Errorf("no GitHub integration found for this project/environment")
	}
	return integrations[0].GetToken(), nil
}

// createGithubRepo calls the GitHub API to create a new repository.
func (h *HandlerV1) createGithubRepo(ctx context.Context, token string, req models.GithubCreateRepoRequest) (*models.GithubRepo, error) {
	bodyBytes, err := json.Marshal(map[string]any{
		"name":        req.Name,
		"description": req.Description,
		"private":     req.Private,
		"auto_init":   req.AutoInit,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/user/repos", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github create repo: status %d — %s", resp.StatusCode, string(b))
	}

	var repo models.GithubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// listGithubRepos returns all repositories for the authenticated GitHub user.
func (h *HandlerV1) listGithubRepos(ctx context.Context, token string) ([]models.GithubRepo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIReposURL+"?per_page=100&sort=updated", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github list repos: status %d — %s", resp.StatusCode, string(b))
	}

	var repos []models.GithubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}
