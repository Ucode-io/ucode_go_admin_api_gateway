package v1

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/fileupload"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	googleDriveOAuthStatePrefix = "google-drive-oauth-state:"
	googleDriveOAuthStateTTL    = 10 * time.Minute
)

type googleDriveOAuthState struct {
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	UserID        string `json:"user_id"`
}

// GoogleDriveConnect godoc
// @Security ApiKeyAuth
// @ID google_drive_connect
// @Router /v1/google-drive/connect [GET]
// @Summary Initiate Google Drive OAuth
// @Tags Google Drive Integration
// @Success 200 {object} status_http.Response{data=map[string]string}
func (h *HandlerV1) GoogleDriveConnect(c *gin.Context) {
	projectID, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectID.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentID, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentID.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return
	}

	oauthConfig, err := fileupload.NewOAuthConfig(h.googleDriveConfig())
	if err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	state, err := newGoogleDriveOAuthState()
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	userID := ""
	if value, ok := c.Get("user_id"); ok {
		userID, _ = value.(string)
	}

	if err := h.storeGoogleDriveOAuthState(c.Request.Context(), state, googleDriveOAuthState{
		ProjectID:     projectID.(string),
		EnvironmentID: environmentID.(string),
		UserID:        userID,
	}); err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	authURL := oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)

	if c.Query("redirect") == "true" {
		c.Redirect(http.StatusTemporaryRedirect, authURL)
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{"auth_url": authURL})
}

// GoogleDriveCallback godoc
// @ID google_drive_callback
// @Router /v1/google-drive/callback [GET]
// @Summary Google Drive OAuth Callback
// @Tags Google Drive Integration
// @Param code query string true "Authorization code from Google"
// @Param state query string true "CSRF state token"
// @Success 307 "Redirect to frontend"
func (h *HandlerV1) GoogleDriveCallback(c *gin.Context) {
	if googleErr := c.Query("error"); googleErr != "" {
		h.redirectGoogleDriveOAuth(c, false, googleErr)
		return
	}

	code := c.Query("code")
	stateToken := c.Query("state")
	if code == "" || stateToken == "" {
		h.redirectGoogleDriveOAuth(c, false, "missing_code_or_state")
		return
	}

	state, err := h.popGoogleDriveOAuthState(c.Request.Context(), stateToken)
	if err != nil {
		h.redirectGoogleDriveOAuth(c, false, "invalid_state")
		return
	}

	config := h.googleDriveConfig()
	oauthConfig, err := fileupload.NewOAuthConfig(config)
	if err != nil {
		h.redirectGoogleDriveOAuth(c, false, "oauth_config_error")
		return
	}

	token, err := oauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		h.redirectGoogleDriveOAuth(c, false, "token_exchange_failed")
		return
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		h.redirectGoogleDriveOAuth(c, false, "refresh_token_missing")
		return
	}

	project, err := h.companyServices.Project().GetById(c.Request.Context(), &pb.GetProjectByIdRequest{
		ProjectId: state.ProjectID,
	})
	if err != nil {
		h.redirectGoogleDriveOAuth(c, false, "project_not_found")
		return
	}

	credentials, err := fileupload.OAuthCredentialsFromRefreshToken(config, token.RefreshToken)
	if err != nil {
		h.redirectGoogleDriveOAuth(c, false, "oauth_credentials_error")
		return
	}

	uploader := fileupload.NewGoogleDriveUploader(h.companyServices.Resource(), config)
	folder, err := uploader.ProvisionFolder(c.Request.Context(), credentials, fileupload.GoogleDriveCreateFolderRequest{
		Name: fileupload.ProjectFolderName(project.GetTitle(), state.ProjectID),
	})
	if err != nil {
		h.redirectGoogleDriveOAuth(c, false, "folder_create_failed")
		return
	}

	settings := &pb.Settings{
		GoogleDrive: &pb.GoogleDriveCredentials{
			AuthType:     "oauth",
			FolderId:     folder.GetFolderID(),
			Visibility:   googleDriveVisibility(config.Visibility),
			RefreshToken: strings.TrimSpace(token.RefreshToken),
		},
	}

	if err := h.upsertGoogleDriveResource(c.Request.Context(), state, settings); err != nil {
		h.redirectGoogleDriveOAuth(c, false, "resource_save_failed")
		return
	}

	h.redirectGoogleDriveOAuth(c, true, "")
}

func (h *HandlerV1) googleDriveConfig() fileupload.GoogleDriveConfig {
	return fileupload.GoogleDriveConfig{
		ClientID:           h.baseConf.GoogleDriveClientID,
		ClientSecret:       h.baseConf.GoogleDriveClientSecret,
		RedirectURI:        h.baseConf.GoogleDriveRedirectURI,
		ServiceAccountJSON: h.baseConf.GoogleDriveServiceAccountJSON,
		ParentFolderID:     h.baseConf.GoogleDriveParentFolderID,
		Visibility:         h.baseConf.GoogleDriveVisibility,
	}
}

func (h *HandlerV1) upsertGoogleDriveResource(ctx context.Context, state googleDriveOAuthState, settings *pb.Settings) error {
	list, err := h.companyServices.Resource().GetProjectResourceList(ctx, &pb.GetProjectResourceListRequest{
		ProjectId:     state.ProjectID,
		EnvironmentId: state.EnvironmentID,
		Type:          pb.ResourceType_GOOGLE_DRIVE,
	})
	if err != nil {
		return err
	}
	if len(list.GetResources()) > 1 {
		return errors.New("multiple google drive resources configured for project environment")
	}

	if len(list.GetResources()) == 1 {
		current := list.GetResources()[0]
		name := current.GetName()
		if name == "" {
			name = "Google Drive"
		}
		_, err = h.companyServices.Resource().UpdateProjectResource(ctx, &pb.ProjectResource{
			Id:            current.GetId(),
			ProjectId:     state.ProjectID,
			EnvironmentId: state.EnvironmentID,
			Name:          name,
			Type:          pb.ResourceType_GOOGLE_DRIVE.String(),
			ResourceType:  int32(pb.ResourceType_GOOGLE_DRIVE),
			Settings:      settings,
		})
		return err
	}

	_, err = h.companyServices.Resource().AddResourceToProject(ctx, &pb.AddResourceToProjectRequest{
		Name:          "Google Drive",
		ProjectId:     state.ProjectID,
		EnvironmentId: state.EnvironmentID,
		Type:          pb.ResourceType_GOOGLE_DRIVE,
		Settings:      settings,
	})
	return err
}

func (h *HandlerV1) storeGoogleDriveOAuthState(ctx context.Context, state string, payload googleDriveOAuthState) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	key := googleDriveOAuthStatePrefix + state
	if h.centralRedis != nil {
		return h.centralRedis.Set(ctx, key, body, googleDriveOAuthStateTTL).Err()
	}
	if h.cache != nil {
		h.cache.Add(key, body, googleDriveOAuthStateTTL)
		return nil
	}
	return errors.New("oauth state storage is not configured")
}

func (h *HandlerV1) popGoogleDriveOAuthState(ctx context.Context, state string) (googleDriveOAuthState, error) {
	var payload googleDriveOAuthState
	key := googleDriveOAuthStatePrefix + state

	var body []byte
	if h.centralRedis != nil {
		value, err := h.centralRedis.Get(ctx, key).Result()
		if err != nil {
			return payload, err
		}
		body = []byte(value)
		_ = h.centralRedis.Del(ctx, key).Err()
	} else if h.cache != nil {
		value, ok := h.cache.Get(key)
		if !ok {
			return payload, errors.New("oauth state not found")
		}
		body = value
	} else {
		return payload, errors.New("oauth state storage is not configured")
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, err
	}
	if payload.ProjectID == "" || payload.EnvironmentID == "" {
		return payload, errors.New("oauth state payload is incomplete")
	}

	return payload, nil
}

func (h *HandlerV1) redirectGoogleDriveOAuth(c *gin.Context, success bool, reason string) {
	target := h.baseConf.GoogleDriveFrontendErrorURL
	if success {
		target = h.baseConf.GoogleDriveFrontendSuccessURL
	}
	if target == "" {
		if success {
			target = "/?google_drive=success"
		} else {
			target = "/?google_drive=error"
		}
	}

	u, err := url.Parse(target)
	if err == nil {
		q := u.Query()
		if success {
			q.Set("google_drive", "success")
		} else {
			q.Set("google_drive", "error")
			if reason != "" {
				q.Set("reason", reason)
			}
		}
		u.RawQuery = q.Encode()
		target = u.String()
	}

	c.Redirect(http.StatusTemporaryRedirect, target)
}

func newGoogleDriveOAuthState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func googleDriveVisibility(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "private"
	}
	return value
}
