package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ucode/ucode_go_api_gateway/config"

	"github.com/stretchr/testify/require"
)

func TestFacebookDebugTokenFallsBackToLegacyApp(t *testing.T) {
	var appTokens []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appToken := r.URL.Query().Get("access_token")
		appTokens = append(appTokens, appToken)
		w.Header().Set("Content-Type", "application/json")
		if appToken == "legacy-app|legacy-secret" {
			_, _ = w.Write([]byte(`{"data":{"app_id":"legacy-app","is_valid":true,"expires_at":1893456000}}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"The App_id in the input_token did not match the Viewing App","type":"OAuthException","code":100}}`))
	}))
	defer server.Close()

	h := HandlerV1{baseConf: config.BaseConfig{
		FacebookAppID:           "ucode-app",
		FacebookAppSecret:       "ucode-secret",
		FacebookLegacyAppID:     "legacy-app",
		FacebookLegacyAppSecret: "legacy-secret",
		FacebookGraphBaseURL:    server.URL,
		FacebookGraphAPIVersion: "v26.0",
	}}

	debug, err := h.facebookDebugToken(context.Background(), "legacy-user-token")
	require.NoError(t, err)
	require.True(t, debug.IsValid)
	require.Equal(t, []string{"ucode-app|ucode-secret", "legacy-app|legacy-secret"}, appTokens)
}

func TestFacebookDebugTokenUsesPrimaryAppFirst(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"app_id":"ucode-app","is_valid":true,"expires_at":1893456000}}`))
	}))
	defer server.Close()

	h := HandlerV1{baseConf: config.BaseConfig{
		FacebookAppID:           "ucode-app",
		FacebookAppSecret:       "ucode-secret",
		FacebookLegacyAppID:     "legacy-app",
		FacebookLegacyAppSecret: "legacy-secret",
		FacebookGraphBaseURL:    server.URL,
		FacebookGraphAPIVersion: "v26.0",
	}}

	debug, err := h.facebookDebugToken(context.Background(), "ucode-user-token")
	require.NoError(t, err)
	require.True(t, debug.IsValid)
	require.Equal(t, 1, requests)
}
