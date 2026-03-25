package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
)

// extractTextFromClaudeResponse — достаёт весь текст из ContentBlock[] и возвращает ClaudeResponse
func extractTextFromClaudeResponse(rawJSON string) (string, *models.ClaudeResponse, error) {
	var resp models.ClaudeResponse
	if err := json.Unmarshal([]byte(rawJSON), &resp); err != nil {
		return "", nil, fmt.Errorf("failed to parse Claude response envelope: %w", err)
	}

	var parts []string
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

// extractJSON — многоуровневая стратегия извлечения JSON из текста модели
// Порядок попыток:
//  1. ```json ... ``` блок
//  2. ``` ... ``` блок
//  3. Первый { до последнего } (самый агрессивный fallback)
func extractJSON(text string) string {
	// Стратегия 1: ```json ... ```
	re1 := regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```")
	if m := re1.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	// Стратегия 2: ``` ... ```
	re2 := regexp.MustCompile("(?s)```\\s*\\n?(.*?)\\n?```")
	if m := re2.FindStringSubmatch(text); len(m) > 1 {
		candidate := strings.TrimSpace(m[1])
		if strings.HasPrefix(candidate, "{") {
			return candidate
		}
	}

	// Стратегия 3: найти первый { и последний }
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}

	return text
}

// CleanJSONResponse — публичная обёртка над extractJSON (для обратной совместимости)
func CleanJSONResponse(input string) string {
	return extractJSON(input)
}

// ParseClaudeResponse — парсит полный ответ Sonnet с проектом (JSON + description)
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

// ParseSonnetPlanResult — парсит JSON ответ от Sonnet планировщика
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

// extractJSONAndDescription — разделяет JSON блок и текстовое описание после него
func extractJSONAndDescription(text string) (jsonBlock, description string) {
	// Пробуем ```json ... ``` с описанием после
	re := regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```(.*)")
	if matches := re.FindStringSubmatch(text); len(matches) > 2 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}

	// Пробуем ``` ... ``` с описанием после
	re2 := regexp.MustCompile("(?s)```\\s*\\n?(\\{.*?\\})\\n?```(.*)")
	if matches := re2.FindStringSubmatch(text); len(matches) > 2 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}

	// Пробуем разделитель ---
	if idx := strings.Index(text, "\n---\n"); idx != -1 {
		jsonPart := strings.TrimSpace(text[:idx])
		descPart := strings.TrimSpace(text[idx+5:])
		jsonPart = extractJSON(jsonPart)
		return jsonPart, descPart
	}

	// Fallback: весь текст это JSON
	if strings.HasPrefix(strings.TrimSpace(text), "{") {
		return extractJSON(text), ""
	}

	return "", text
}
