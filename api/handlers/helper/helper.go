package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
)

func extractTextFromClaudeResponse(rawJSON string) (string, *models.ClaudeResponse, error) {
	var (
		resp  models.ClaudeResponse
		parts []string
	)

	if err := json.Unmarshal([]byte(rawJSON), &resp); err != nil {
		return "", nil, fmt.Errorf("failed to parse Claude response envelope: %w", err)
	}

	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}

	text := strings.TrimSpace(strings.Join(parts, "\n"))
	return text, &resp, nil
}

func ExtractPlainText(rawJSON string) (string, error) {
	text, _, err := extractTextFromClaudeResponse(rawJSON)
	return text, err
}

var (
	jsonBlockRegex       = regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```")
	genericBlockRegex    = regexp.MustCompile("(?s)```\\s*\\n?(.*?)\\n?```")
	jsonAndDescRegex     = regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```(.*)")
	jsonOnlyAndDescRegex = regexp.MustCompile("(?s)```\\s*\\n?(\\{.*?\\})\\n?```(.*)")
)

func extractJSON(text string) string {
	if m := jsonBlockRegex.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	if m := genericBlockRegex.FindStringSubmatch(text); len(m) > 1 {
		candidate := strings.TrimSpace(m[1])
		if strings.HasPrefix(candidate, "{") {
			return candidate
		}
	}

	var (
		start = strings.Index(text, "{")
		end   = strings.LastIndex(text, "}")
	)

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}

	return strings.TrimSpace(text)
}

func CleanJSONResponse(input string) string {
	return extractJSON(input)
}

func ParseClaudeResponse(rawJSON string) (*models.ParsedClaudeResponse, error) {
	fullText, resp, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	log.Printf("PARSE_CLAUDE_RESPONSE: stop_reason=%s input_tokens=%d output_tokens=%d text_length=%d",
		resp.StopReason, resp.Usage.InputTokens, resp.Usage.OutputTokens, len(fullText))

	if resp.StopReason == "max_tokens" {
		log.Printf("PARSE_CLAUDE_RESPONSE WARNING: response was cut off by max_tokens! Consider increasing MaxTokens.")
	}

	result := &models.ParsedClaudeResponse{
		Model:      resp.Model,
		MessageID:  resp.ID,
		StopReason: resp.StopReason,
		Usage:      resp.Usage,
	}

	jsonBlock, description := extractJSONAndDescription(fullText)

	if jsonBlock != "" {
		var project models.GeneratedProject
		if err := json.Unmarshal([]byte(jsonBlock), &project); err != nil {
			log.Printf("PARSE_CLAUDE_RESPONSE: failed to unmarshal project JSON: %v | json_preview=%.200s", err, jsonBlock)
		} else {
			result.Project = &project
		}
	}

	result.Description = strings.TrimSpace(description)
	return result, nil
}

func ParseHaikuRoutingResult(rawJSON string) (*models.HaikuRoutingResult, error) {
	fullText, resp, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	log.Printf("PARSE_HAIKU: stop_reason=%s input_tokens=%d output_tokens=%d",
		resp.StopReason, resp.Usage.InputTokens, resp.Usage.OutputTokens)
	log.Printf("PARSE_HAIKU RAW TEXT: %.500s", fullText)

	cleaned := extractJSON(fullText)
	log.Printf("PARSE_HAIKU CLEANED JSON: %.500s", cleaned)

	var result models.HaikuRoutingResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		log.Printf("PARSE_HAIKU: unmarshal failed (%v), falling back to chat reply", err)
		return &models.HaikuRoutingResult{
			NextStep: false,
			Intent:   "chat",
			Reply:    fullText,
		}, nil
	}

	return &result, nil
}

func ParseSonnetPlanResult(rawJSON string) (*models.SonnetPlanResult, error) {
	fullText, resp, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	log.Printf("PARSE_SONNET_PLAN: stop_reason=%s input_tokens=%d output_tokens=%d text_length=%d",
		resp.StopReason, resp.Usage.InputTokens, resp.Usage.OutputTokens, len(fullText))

	if resp.StopReason == "max_tokens" {
		log.Printf("PARSE_SONNET_PLAN WARNING: response cut off by max_tokens!")
	}

	log.Printf("PARSE_SONNET_PLAN RAW TEXT: %.1000s", fullText)

	cleaned := extractJSON(fullText)
	log.Printf("PARSE_SONNET_PLAN CLEANED JSON: %.1000s", cleaned)

	var result models.SonnetPlanResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse sonnet plan json: %w | raw_text_preview=%.300s", err, fullText)
	}

	log.Printf("PARSE_SONNET_PLAN OK: files_to_change=%d files_to_create=%d", len(result.FilesToChange), len(result.FilesToCreate))
	return &result, nil
}

func extractJSONAndDescription(text string) (jsonBlock, description string) {
	re := regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```(.*)")
	if matches := re.FindStringSubmatch(text); len(matches) > 2 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}

	re2 := regexp.MustCompile("(?s)```\\s*\\n?(\\{.*?\\})\\n?```(.*)")
	if matches := re2.FindStringSubmatch(text); len(matches) > 2 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}

	if idx := strings.Index(text, "\n---\n"); idx != -1 {
		jsonPart := strings.TrimSpace(text[:idx])
		descPart := strings.TrimSpace(text[idx+5:])
		jsonPart = extractJSON(jsonPart)
		return jsonPart, descPart
	}

	if strings.HasPrefix(strings.TrimSpace(text), "{") {
		return extractJSON(text), ""
	}

	return "", text
}
