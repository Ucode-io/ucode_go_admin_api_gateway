package callquality

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/ai"
	"ucode/ucode_go_api_gateway/config"

	"github.com/gin-gonic/gin"
)

type fixedModel struct{}

func (fixedModel) Complete(context.Context, ai.CompletionRequest) (*ai.CompletionResult, error) {
	return &ai.CompletionResult{ToolCalls: []ai.ToolCall{{
		Name: evaluationToolName,
		Input: map[string]any{
			"summary":         "Operator greeted the client",
			"review_required": false,
			"criteria": []any{map[string]any{
				"criterion_id": "opening", "score": 10, "status": statusCompleted,
				"explanation": "Greeting is present", "recommendation": "",
				"evidence": []any{map[string]any{"speaker": "operator", "quote": "Assalomu alaykum", "from": 0}},
			}},
			"violations": []any{},
		},
	}}}, nil
}

func TestEvaluateHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{evaluator: NewEvaluator(fixedModel{}, config.AgentConfig{Model: "test", MaxTokens: 1000, Timeout: time.Second})}
	body, err := json.Marshal(map[string]any{
		"call_id": "call-1", "language": "uz", "transcript_source": "stereo",
		"transcript":       []any{map[string]any{"speaker": "operator", "text": "Assalomu alaykum", "from": 0}},
		"criteria_version": "v1",
		"criteria":         []any{map[string]any{"id": "opening", "name": "Opening", "instructions": "Check greeting", "max_score": 10, "allowed_scores": []any{0, 10}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/crm-ai/call-quality/evaluate", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")

	handler.Evaluate(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			TotalScore float64 `json:"total_score"`
			MaxScore   float64 `json:"max_score"`
		} `json:"data"`
	}
	if err = json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.TotalScore != 10 || response.Data.MaxScore != 10 {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
