package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/handlers/ai"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

const baseURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

// callGeminiText sends a free-text request (no tools) and returns the text content
// of the first candidate. Used by RouteRequest, InspectCode, DatabaseQuery.
func callGeminiText(conf config.BaseConfig, agentCfg config.AgentConfig, system string, contents []geminiContent) (string, models.LLMUsage, error) {
	req := geminiRequest{
		SystemInstruction: systemContent(system),
		Contents:          contents,
		GenerationConfig:  generationConfig{MaxOutputTokens: agentCfg.MaxTokens},
	}

	resp, err := doRequest(conf.GeminiAPIKey, agentCfg.Model, agentCfg.Timeout, req)
	if err != nil {
		return "", models.LLMUsage{}, err
	}

	usage := usageFromResponse(resp)

	if len(resp.Candidates) == 0 {
		return "", usage, fmt.Errorf("gemini: no candidates in response")
	}

	var parts []string
	for _, p := range resp.Candidates[0].Content.Parts {
		if p.Text != "" {
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, "\n"), usage, nil
}

// callGeminiTool sends a function-calling request with mode=ANY forcing the model
// to call the specified tool. Returns the raw JSON bytes of the function args.
func callGeminiTool(conf config.BaseConfig, agentCfg config.AgentConfig, system string, contents []geminiContent, tool funcDeclaration) ([]byte, models.LLMUsage, error) {
	req := geminiRequest{
		SystemInstruction: systemContent(system),
		Contents:          contents,
		Tools:             []geminiTool{{FunctionDeclarations: []funcDeclaration{tool}}},
		ToolConfig: &geminiToolCfg{
			FunctionCallingConfig: funcCallingConfig{
				Mode:                 "ANY",
				AllowedFunctionNames: []string{tool.Name},
			},
		},
		GenerationConfig: generationConfig{MaxOutputTokens: agentCfg.MaxTokens},
	}

	resp, err := doRequest(conf.GeminiAPIKey, agentCfg.Model, agentCfg.Timeout, req)
	if err != nil {
		return nil, models.LLMUsage{}, err
	}

	usage := usageFromResponse(resp)

	if len(resp.Candidates) == 0 {
		return nil, usage, fmt.Errorf("gemini: no candidates in response")
	}

	cand := resp.Candidates[0]
	if cand.FinishReason == "MAX_TOKENS" {
		return nil, usage, fmt.Errorf("%w (model=%s input=%d output=%d)",
			ErrMaxTokens, agentCfg.Model, usage.InputTokens, usage.OutputTokens)
	}

	for _, part := range cand.Content.Parts {
		if part.FunctionCall == nil || part.FunctionCall.Name != tool.Name {
			continue
		}
		args := part.FunctionCall.Args
		ai.RepairStringifiedFields(args)
		inputJSON, err := json.Marshal(args)
		if err != nil {
			return nil, usage, fmt.Errorf("gemini: re-marshal tool args: %w", err)
		}
		return inputJSON, usage, nil
	}

	return nil, usage, fmt.Errorf("gemini: no function_call %q in response (finish_reason=%q)",
		tool.Name, cand.FinishReason)
}

func doRequest(apiKey, model string, timeout time.Duration, body geminiRequest) (geminiResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return geminiResponse{}, fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf(baseURL, model, apiKey)
	client := &http.Client{Timeout: timeout}

	const maxRetries = 4
	backoffs := []time.Duration{10 * time.Second, 30 * time.Second, 60 * time.Second}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := backoffs[attempt-1]
			log.Printf("[GEMINI] 429 rate limit — waiting %v before retry %d/%d (model=%s)", wait, attempt, maxRetries-1, model)
			time.Sleep(wait)
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return geminiResponse{}, fmt.Errorf("gemini: create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		httpResp, err := client.Do(req)
		if err != nil {
			return geminiResponse{}, fmt.Errorf("gemini: request failed: %w", err)
		}

		respBytes, readErr := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if readErr != nil {
			return geminiResponse{}, fmt.Errorf("gemini: read response: %w", readErr)
		}

		if httpResp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetries-1 {
				continue
			}
			return geminiResponse{}, fmt.Errorf("gemini: rate limit exceeded after %d retries: %s", maxRetries, string(respBytes))
		}

		if httpResp.StatusCode != http.StatusOK {
			return geminiResponse{}, fmt.Errorf("gemini: unexpected status %d: %s", httpResp.StatusCode, string(respBytes))
		}

		var resp geminiResponse
		if err = json.Unmarshal(respBytes, &resp); err != nil {
			return geminiResponse{}, fmt.Errorf("gemini: parse response: %w", err)
		}
		return resp, nil
	}

	return geminiResponse{}, fmt.Errorf("gemini: all retries exhausted (model=%s)", model)
}
