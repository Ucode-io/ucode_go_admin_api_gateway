package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

var ErrMaxTokens = errors.New("generation stopped: output exceeded the token limit")

type systemBlock struct {
	Type         string     `json:"type"` // always "text"
	Text         string     `json:"text"`
	CacheControl *cacheCtrl `json:"cache_control,omitempty"`
}

type cacheCtrl struct {
	Type string `json:"type"` // "ephemeral"
}

type wireToolRequest struct {
	Model      string                      `json:"model"`
	MaxTokens  int                         `json:"max_tokens"`
	System     []systemBlock               `json:"system,omitempty"`
	Messages   []models.ChatMessage        `json:"messages"`
	Tools      []models.ClaudeFunctionTool `json:"tools"`
	ToolChoice *models.ToolChoice          `json:"tool_choice,omitempty"`
}

// inputKeys returns the keys of a map for diagnostic logging.
func inputKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

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

	var wire = wireToolRequest{
		Model:      body.Model,
		MaxTokens:  body.MaxTokens,
		Messages:   body.Messages,
		Tools:      body.Tools,
		ToolChoice: body.ToolChoice,
	}
	if body.System != "" {
		wire.System = []systemBlock{{Type: "text", Text: body.System, CacheControl: &cacheCtrl{Type: "ephemeral"}}}
	}

	jsonBody, err := json.Marshal(wire)
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

	req.Header.Set("anthropic-beta", config.AnthropicCachingBeta)

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

	if toolResp.StopReason == "max_tokens" {
		return nil, toolResp.Usage, toolResp.StopReason,
			fmt.Errorf("%w (model=%s input_tokens=%d output_tokens=%d)",
				ErrMaxTokens, toolResp.Model, toolResp.Usage.InputTokens, toolResp.Usage.OutputTokens)
	}

	for _, block := range toolResp.Content {
		if block.Type != "tool_use" {
			continue
		}

		for k, v := range block.Input {
			if s, ok := v.(string); ok {
				// Prevent parsing fields that are strictly meant to be raw text/code,
				// even if they coincidentally look like valid JSON objects/arrays.
				if k == "content" || k == "ui_structure" || k == "bpmn_xml" || k == "summary" || k == "change_summary" {
					continue
				}
				s = strings.TrimSpace(s)
				if strings.HasPrefix(s, "```json") {
					s = strings.TrimPrefix(s, "```json")
					s = strings.TrimSuffix(strings.TrimSpace(s), "```")
					s = strings.TrimSpace(s)
				} else if strings.HasPrefix(s, "```") {
					s = strings.TrimPrefix(s, "```")
					s = strings.TrimSuffix(strings.TrimSpace(s), "```")
					s = strings.TrimSpace(s)
				}

				if (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) || (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
					var parsed interface{}
					if err := json.Unmarshal([]byte(s), &parsed); err == nil {
						block.Input[k] = parsed
					} else {
						sanitized := sanitizeJSONContent(s)
						if err2 := json.Unmarshal([]byte(sanitized), &parsed); err2 == nil {
							block.Input[k] = parsed
						} else {
							repaired := repairJSONStrings(sanitized)
							if err3 := json.Unmarshal([]byte(repaired), &parsed); err3 == nil {
								block.Input[k] = parsed
							} else {
								preview := s
								if len(preview) > 200 {
									preview = preview[:200]
								}
								log.Printf("[TOOL DECODE] Failed to parse stringified JSON field %q. Repair also failed. Error: %v\nPreview: %s", k, err3, preview)
							}
						}
					}
				}
			}
		}

		inputJSON, marshalErr := json.Marshal(block.Input)
		if marshalErr != nil {
			return nil, toolResp.Usage, toolResp.StopReason,
				fmt.Errorf("failed to re-marshal tool input: %w", marshalErr)
		}
		var result T
		if err = json.Unmarshal(inputJSON, &result); err != nil {
			log.Printf("[TOOL DECODE] failed to decode tool %q input into %T: %v\nraw input keys: %v",
				block.Name, result, err, inputKeys(block.Input))
			return nil, toolResp.Usage, toolResp.StopReason,
				fmt.Errorf("failed to decode tool %q input into %T: %w", block.Name, result, err)
		}
		return &result, toolResp.Usage, toolResp.StopReason, nil
	}

	return nil, toolResp.Usage, toolResp.StopReason,
		fmt.Errorf("no tool_use block in response (stop_reason=%q)", toolResp.StopReason)
}
