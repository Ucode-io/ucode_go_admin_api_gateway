package anthropic

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
)

func buildContentBlocks(text string, imageURLs []string) []models.ContentBlock {
	blocks := make([]models.ContentBlock, 0, len(imageURLs)+1)
	for _, url := range imageURLs {
		if strings.TrimSpace(url) != "" {
			blocks = append(blocks, models.ContentBlock{
				Type:   "image",
				Source: &models.ImageSource{Type: "url", URL: url},
			})
		}
	}
	blocks = append(blocks, models.ContentBlock{Type: "text", Text: text})
	return blocks
}

func buildAgentMessages(history []models.ChatMessage, blocks []models.ContentBlock) []models.ChatMessage {
	messages := make([]models.ChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, models.ChatMessage{Role: "user", Content: blocks})
	return messages
}

func buildHistoryText(history []models.ChatMessage) string {
	if len(history) == 0 {
		return ""
	}
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	var sb strings.Builder
	for _, msg := range history[start:] {
		var text string
		for _, block := range msg.Content {
			if block.Type == "text" {
				text += block.Text
			}
		}
		if text == "" {
			continue
		}
		if msg.Role == "assistant" {
			sb.WriteString("Assistant: ")
		} else {
			sb.WriteString("User: ")
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String()
}

func parseHaikuRoutingResult(rawJSON string) (*models.HaikuRoutingResult, error) {
	fullText, err := extractPlainText(rawJSON)
	if err != nil {
		return nil, err
	}

	cleaned := extractJSONFromText(fullText)

	var result models.HaikuRoutingResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		log.Printf("[PARSE] haiku routing: unmarshal failed (%v), falling back to plain reply", err)
		return &models.HaikuRoutingResult{
			NextStep: false,
			Intent:   "chat",
			Reply:    fullText,
		}, nil
	}
	return &result, nil
}

func extractPlainText(rawJSON string) (string, error) {
	var resp models.ClaudeResponse
	if err := json.Unmarshal([]byte(rawJSON), &resp); err != nil {
		return "", fmt.Errorf("failed to parse Claude response envelope: %w", err)
	}
	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

func extractJSONFromText(text string) string {
	// Try ```json ... ``` first
	if idx := strings.Index(text, "```json"); idx != -1 {
		rest := text[idx+7:]
		if end := strings.Index(rest, "```"); end != -1 {
			return strings.TrimSpace(rest[:end])
		}
	}
	// Try ``` ... ``` with a JSON container
	if idx := strings.Index(text, "```"); idx != -1 {
		rest := text[idx+3:]
		if end := strings.Index(rest, "```"); end != -1 {
			candidate := strings.TrimSpace(rest[:end])
			if strings.HasPrefix(candidate, "{") || strings.HasPrefix(candidate, "[") {
				return candidate
			}
		}
	}
	startObj := strings.Index(text, "{")
	endObj := strings.LastIndex(text, "}")
	startArr := strings.Index(text, "[")
	endArr := strings.LastIndex(text, "]")

	isObj := startObj != -1 && endObj != -1 && endObj > startObj
	isArr := startArr != -1 && endArr != -1 && endArr > startArr

	if isObj && (!isArr || startObj < startArr) {
		return strings.TrimSpace(text[startObj : endObj+1])
	} else if isArr {
		return strings.TrimSpace(text[startArr : endArr+1])
	}
	return strings.TrimSpace(text)
}

func wrapMaxTokens(err error, usage models.LLMUsage, stage string) error {
	if errors.Is(err, ErrMaxTokens) {
		log.Printf("[AI] max_tokens: %s (in=%d out=%d)", stage, usage.InputTokens, usage.OutputTokens)
		return fmt.Errorf(
			"generation stopped: the project is too large to generate in one pass (used %d output tokens). "+
				"Please describe a smaller scope or break the request into parts",
			usage.OutputTokens,
		)
	}
	return err
}