package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

var facebookHTTPClient = &http.Client{Timeout: 20 * time.Second}

func (h *HandlerV1) facebookGraphURL(path string) string {
	return fmt.Sprintf("%s/%s/%s",
		h.baseConf.FacebookGraphBaseURL,
		h.baseConf.FacebookGraphAPIVersion,
		strings.TrimPrefix(path, "/"),
	)
}

func (h *HandlerV1) facebookGraphGet(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := h.facebookGraphURL(path)
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build graph request: %w", err)
	}

	return facebookGraphDo(req, out)
}

func (h *HandlerV1) facebookGraphPost(ctx context.Context, path string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.facebookGraphURL(path), strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build graph request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return facebookGraphDo(req, out)
}

func (h *HandlerV1) facebookGraphDelete(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := h.facebookGraphURL(path)
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build graph request: %w", err)
	}

	return facebookGraphDo(req, out)
}

func facebookGraphDo(req *http.Request, out any) error {
	resp, err := facebookHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("graph api call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read graph response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var wrap struct {
			Error *models.FacebookAPIError `json:"error"`
		}
		if json.Unmarshal(body, &wrap) == nil && wrap.Error != nil {
			return wrap.Error
		}
		return fmt.Errorf("graph api %d: %s", resp.StatusCode, string(body))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse graph response: %w", err)
	}
	return nil
}

func (h *HandlerV1) exchangeFacebookCode(ctx context.Context, code string) (string, error) {
	var token models.FacebookTokenResponse

	err := h.facebookGraphGet(
		ctx, "oauth/access_token",
		url.Values{
			"client_id":     {h.baseConf.FacebookAppID},
			"client_secret": {h.baseConf.FacebookAppSecret},
			"redirect_uri":  {h.baseConf.FacebookRedirectURI},
			"code":          {code},
		},
		&token,
	)
	if err != nil {
		return "", err
	}

	if token.AccessToken == "" {
		return "", fmt.Errorf("empty access token in code exchange")
	}

	return token.AccessToken, nil
}

func (h *HandlerV1) exchangeFacebookLongLivedToken(ctx context.Context, shortLived string) (string, error) {
	var token models.FacebookTokenResponse

	err := h.facebookGraphGet(
		ctx, "oauth/access_token",
		url.Values{
			"grant_type":        {"fb_exchange_token"},
			"client_id":         {h.baseConf.FacebookAppID},
			"client_secret":     {h.baseConf.FacebookAppSecret},
			"fb_exchange_token": {shortLived},
		},
		&token,
	)
	if err != nil {
		return "", err
	}

	if token.AccessToken == "" {
		return "", fmt.Errorf("empty access token in long-lived exchange")
	}

	return token.AccessToken, nil
}

func (h *HandlerV1) facebookFetchUser(ctx context.Context, userToken string) (models.FacebookUser, error) {
	var user models.FacebookUser
	err := h.facebookGraphGet(ctx, "me", url.Values{
		"fields":       {"id,name"},
		"access_token": {userToken},
	}, &user)
	return user, err
}

// facebookDebugToken inspects a user token via Graph debug_token (authenticated
// with the app token) to learn whether it is still valid and when it expires.
func (h *HandlerV1) facebookDebugToken(ctx context.Context, userToken string) (models.FacebookTokenDebugData, error) {
	var resp models.FacebookTokenDebugResponse

	err := h.facebookGraphGet(
		ctx, "debug_token",
		url.Values{
			"input_token":  {userToken},
			"access_token": {h.baseConf.FacebookAppID + "|" + h.baseConf.FacebookAppSecret},
		},
		&resp,
	)
	if err != nil {
		return models.FacebookTokenDebugData{}, err
	}
	return resp.Data, nil
}

func (h *HandlerV1) facebookListPages(ctx context.Context, userToken string) ([]models.FacebookPage, error) {
	var list models.FacebookPageList

	err := h.facebookGraphGet(
		ctx, "me/accounts", url.Values{
			"fields":       {"id,name,access_token,tasks"},
			"access_token": {userToken},
		},
		&list,
	)
	if err != nil {
		return nil, err
	}

	return list.Data, nil
}

func (h *HandlerV1) facebookListForms(ctx context.Context, pageID, pageToken string) ([]models.FacebookForm, error) {
	var list models.FacebookFormList
	err := h.facebookGraphGet(ctx, pageID+"/leadgen_forms", url.Values{
		"fields":       {"id,name,status,locale"},
		"access_token": {pageToken},
	}, &list)
	if err != nil {
		return nil, err
	}
	return list.Data, nil
}

func (h *HandlerV1) facebookFormQuestions(ctx context.Context, formID, pageToken string) (models.FacebookFormQuestions, error) {
	var form models.FacebookFormQuestions
	err := h.facebookGraphGet(ctx, formID, url.Values{
		"fields":       {"id,name,status,locale,questions"},
		"access_token": {pageToken},
	}, &form)
	return form, err
}

func (h *HandlerV1) facebookFetchLead(ctx context.Context, leadgenID, pageToken string) (models.FacebookLead, error) {
	var lead models.FacebookLead
	err := h.facebookGraphGet(ctx, leadgenID, url.Values{
		"fields":       {"id,created_time,ad_id,ad_name,form_id,platform,is_organic,field_data"},
		"access_token": {pageToken},
	}, &lead)
	return lead, err
}

func (h *HandlerV1) facebookSubscribePage(ctx context.Context, pageID, pageToken string) error {
	var result struct {
		Success bool `json:"success"`
	}

	return h.facebookGraphPost(
		ctx, pageID+"/subscribed_apps",
		url.Values{
			"subscribed_fields": {config.FacebookSubscribedFields},
			"access_token":      {pageToken},
		},
		&result,
	)
}

func (h *HandlerV1) facebookUnsubscribePage(ctx context.Context, pageID, pageToken string) error {
	return h.facebookGraphDelete(ctx, pageID+"/subscribed_apps", url.Values{
		"access_token": {pageToken},
	}, nil)
}
