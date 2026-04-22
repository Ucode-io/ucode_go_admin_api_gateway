package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

var ErrMaxTokens = errors.New("generation stopped: output exceeded the token limit")

func CallAnthropicAPI(baseConf config.BaseConfig, body models.AnthropicRequest, timeout time.Duration) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseConf.AnthropicBaseAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", baseConf.AnthropicAPIKey)
	req.Header.Set("anthropic-version", baseConf.AnthropicVersion)
	req.Header.Set("anthropic-beta", baseConf.AnthropicBeta)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	return string(respBytes), nil
}

// CallAnthropicWithTool sends a tool-use request to the Anthropic API and decodes
// the first tool_use block's input field directly into *T using JSON round-trip.
//
// Returns (result, usage, stopReason, error).
// Returns ErrMaxTokens when stop_reason=="max_tokens" — callers must not retry with repair.
//
// Use this for all structured-generation calls: architect, planner, coder, diagrams, visual edit.
func CallAnthropicWithTool[T any](baseConf config.BaseConfig, body models.AnthropicToolRequest, timeout time.Duration) (*T, models.ClaudeUsage, string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, models.ClaudeUsage{}, "", fmt.Errorf("failed to marshal tool request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseConf.AnthropicBaseAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, models.ClaudeUsage{}, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", baseConf.AnthropicAPIKey)
	req.Header.Set("anthropic-version", baseConf.AnthropicVersion)
	req.Header.Set("anthropic-beta", baseConf.AnthropicBeta)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, models.ClaudeUsage{}, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, models.ClaudeUsage{}, "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, models.ClaudeUsage{}, "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	var toolResp models.ToolUseResponse
	if err = json.Unmarshal(respBytes, &toolResp); err != nil {
		return nil, models.ClaudeUsage{}, "", fmt.Errorf("failed to parse tool response envelope: %w", err)
	}

	// A truncated response means the tool input is incomplete — cannot be decoded.
	if toolResp.StopReason == "max_tokens" {
		return nil, toolResp.Usage, toolResp.StopReason,
			fmt.Errorf("%w (model=%s input_tokens=%d output_tokens=%d)",
				ErrMaxTokens, toolResp.Model, toolResp.Usage.InputTokens, toolResp.Usage.OutputTokens)
	}

	// Find the first tool_use block and decode its input into T.
	for _, block := range toolResp.Content {
		if block.Type != "tool_use" {
			continue
		}
		// Re-marshal the map → JSON → unmarshal into T.
		// This is the safe path: the map was already validated by json.Unmarshal above.
		inputJSON, marshalErr := json.Marshal(block.Input)
		if marshalErr != nil {
			return nil, toolResp.Usage, toolResp.StopReason,
				fmt.Errorf("failed to re-marshal tool input: %w", marshalErr)
		}
		var result T
		if err = json.Unmarshal(inputJSON, &result); err != nil {
			return nil, toolResp.Usage, toolResp.StopReason,
				fmt.Errorf("failed to decode tool %q input into %T: %w", block.Name, result, err)
		}
		return &result, toolResp.Usage, toolResp.StopReason, nil
	}

	return nil, toolResp.Usage, toolResp.StopReason,
		fmt.Errorf("no tool_use block in response (stop_reason=%q)", toolResp.StopReason)
}
