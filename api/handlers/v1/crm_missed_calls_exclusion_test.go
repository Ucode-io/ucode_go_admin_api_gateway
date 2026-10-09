package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func missedCallExcludedTestTarget(companyID string) missedCallTarget {
	return missedCallTarget{
		ProjectID:     "577d03aa-8301-4d40-88ce-196f2f7a0324",
		EnvironmentID: "8eb5d8c2-1ff0-43a4-9008-7fef20bea388",
		CompanyID:     companyID,
	}
}

func TestMissedCallExcludedTargetDoesNotRegisterWithoutStorage(t *testing.T) {
	for _, companyID := range []string{"company-a", "company-b"} {
		t.Run(companyID, func(t *testing.T) {
			target := missedCallExcludedTestTarget(companyID)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/crm/missed-call-notifications?company-id="+url.QueryEscape(companyID), nil)
			c.Set("project_id", target.ProjectID)
			c.Set("environment_id", target.EnvironmentID)

			// No Redis, logger or SDK: exclusion must precede every dependency.
			h := &HandlerV1{}
			h.ListMissedCallNotifications(c)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
			}
			var response struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if string(response.Data) != "[]" {
				t.Fatalf("notifications = %s, want an empty array", response.Data)
			}
		})
	}
}

func TestMissedCallExcludedPersistedTargetDoesNotPoll(t *testing.T) {
	var h *HandlerV1
	for _, companyID := range []string{"company-a", "company-b", ""} {
		if err := h.pollMissedCallTarget(context.Background(), missedCallExcludedTestTarget(companyID)); err != nil {
			t.Fatalf("excluded company %q: %v", companyID, err)
		}
	}
}

func TestMissedCallOtherTargetsKeepLegacyPath(t *testing.T) {
	target := missedCallExcludedTestTarget("company-a")
	for _, test := range []struct {
		name   string
		target missedCallTarget
	}{
		{name: "other project", target: missedCallTarget{ProjectID: "11111111-1111-4111-8111-111111111111", EnvironmentID: target.EnvironmentID, CompanyID: target.CompanyID}},
		{name: "other environment", target: missedCallTarget{ProjectID: target.ProjectID, EnvironmentID: "22222222-2222-4222-8222-222222222222", CompanyID: target.CompanyID}},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := &HandlerV1{log: logger.NewLogger("missed-call-exclusion-test", logger.LevelError)}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/crm/missed-call-notifications?company-id="+url.QueryEscape(test.target.CompanyID), nil)
			c.Set("project_id", test.target.ProjectID)
			c.Set("environment_id", test.target.EnvironmentID)
			h.ListMissedCallNotifications(c)
			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("other target bypassed the legacy storage requirement: status %d", recorder.Code)
			}

			readErr := errors.New("local fixture: legacy registration read")
			hook := &missedCallRedisReadHook{err: readErr}
			client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
			t.Cleanup(func() { _ = client.Close() })
			client.AddHook(hook)
			h.centralRedis = client
			if err := h.pollMissedCallTarget(context.Background(), test.target); !errors.Is(err, readErr) {
				t.Fatalf("legacy poll did not return the intercepted Redis error: %v", err)
			}
			wantKey := missedCallRegisteredPrefix + missedCallTargetKey(test.target)
			if len(hook.args) != 2 || hook.args[0] != "get" || hook.args[1] != wantKey {
				t.Fatalf("legacy poll command = %v, want GET %s", hook.args, wantKey)
			}
		})
	}
}

func TestMissedCallExclusionPreservesRequestValidation(t *testing.T) {
	target := missedCallExcludedTestTarget("company-a")
	for _, test := range []struct {
		name        string
		project     string
		environment string
		query       string
	}{
		{name: "missing project", environment: target.EnvironmentID, query: "company-id=company-a"},
		{name: "missing environment", project: target.ProjectID, query: "company-id=company-a"},
		{name: "missing company", project: target.ProjectID, environment: target.EnvironmentID},
		{name: "blank company", project: target.ProjectID, environment: target.EnvironmentID, query: "company-id=%20%20"},
		{name: "malformed company query", project: target.ProjectID, environment: target.EnvironmentID, query: "company-id=%zz"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/crm/missed-call-notifications?"+test.query, nil)
			if test.project != "" {
				c.Set("project_id", test.project)
			}
			if test.environment != "" {
				c.Set("environment_id", test.environment)
			}
			h := &HandlerV1{log: logger.NewLogger("missed-call-exclusion-test", logger.LevelError)}
			h.ListMissedCallNotifications(c)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

type missedCallRedisReadHook struct {
	args []interface{}
	err  error
}

func (h *missedCallRedisReadHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	h.args = cmd.Args()
	return ctx, h.err
}

func (*missedCallRedisReadHook) AfterProcess(context.Context, redis.Cmder) error { return nil }

func (h *missedCallRedisReadHook) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, h.err
}

func (*missedCallRedisReadHook) AfterProcessPipeline(context.Context, []redis.Cmder) error {
	return nil
}
