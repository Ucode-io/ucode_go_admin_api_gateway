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
	jsonBlockRegex    = regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```")
	genericBlockRegex = regexp.MustCompile("(?s)```\\s*\\n?(.*?)\\n?```")
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

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(text[start : end+1])
	}

	return strings.TrimSpace(text)
}

func CleanJSONResponse(input string) string {
	return extractJSON(input)
}

func sanitizeJSONContent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 || c == '\n' || c == '\r' || c == '\t' || c >= 0x80 {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func repairJSONStrings(input string) string {
	var out strings.Builder
	out.Grow(len(input) + 512)

	inString := false
	escaped := false

	for i := 0; i < len(input); i++ {
		c := input[i]

		if escaped {
			out.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && inString {
			if i+1 < len(input) {
				next := input[i+1]
				validEscape := next == '"' || next == '\\' || next == '/' ||
					next == 'b' || next == 'f' || next == 'n' ||
					next == 'r' || next == 't' || next == 'u'
				if !validEscape {
					out.WriteString(`\\`)
					continue
				}
			}
			out.WriteByte(c)
			escaped = true
			continue
		}

		if c == '"' && !escaped {
			inString = !inString
			out.WriteByte(c)
			continue
		}

		if inString {
			switch c {
			case '\n':
				out.WriteString(`\n`)
				continue
			case '\r':
				out.WriteString(`\r`)
				continue
			case '\t':
				out.WriteString(`\t`)
				continue
			}
		}

		out.WriteByte(c)
	}
	return out.String()
}

// ParseClaudeResponse parses a raw Claude API response into a structured result.
// Runs up to 3 JSON repair passes if the initial parse fails.
func ParseClaudeResponse(rawJSON string) (*models.ParsedClaudeResponse, error) {
	fullText, resp, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	if resp.StopReason == "max_tokens" {
		// The JSON is guaranteed to be truncated — all repair attempts will fail.
		// Return ErrMaxTokens so callers can surface a useful message to the user
		// instead of silently attempting to parse a broken response.
		log.Printf("[PARSE] response cut off by max_tokens (model=%s input=%d output=%d) — aborting parse",
			resp.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)
		return nil, fmt.Errorf("%w (model=%s input_tokens=%d output_tokens=%d)",
			ErrMaxTokens, resp.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)
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

		// Pass 1: try to parse as-is
		parseErr := json.Unmarshal([]byte(jsonBlock), &project)

		if parseErr != nil {
			// Pass 2: strip invalid control characters, then retry
			sanitized := sanitizeJSONContent(jsonBlock)
			parseErr = json.Unmarshal([]byte(sanitized), &project)

			if parseErr != nil {
				// Pass 3: repair escape sequences, then retry
				repaired := repairJSONStrings(sanitized)
				parseErr = json.Unmarshal([]byte(repaired), &project)

				if parseErr != nil {
					log.Printf("[PARSE] all 3 repair passes failed: %v", parseErr)
					return nil, fmt.Errorf("project JSON parse failed after 3 repair passes: %w", parseErr)
				}
				log.Printf("[PARSE] pass-3 (escape repair) succeeded")
			}
		}

		result.Project = &project
	}

	result.Description = strings.TrimSpace(description)
	return result, nil
}

func ParseHaikuRoutingResult(rawJSON string) (*models.HaikuRoutingResult, error) {
	fullText, _, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	cleaned := extractJSON(fullText)

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

func ParseSonnetPlanResult(rawJSON string) (*models.SonnetPlanResult, error) {
	fullText, resp, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	if resp.StopReason == "max_tokens" {
		log.Printf("[PARSE] planner response cut off by max_tokens — aborting")
		return nil, fmt.Errorf("%w (planner)", ErrMaxTokens)
	}

	cleaned := extractJSON(fullText)

	var result models.SonnetPlanResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse planner JSON: %w", err)
	}

	return &result, nil
}

func ParsePlanResult(rawJSON string) (*models.HaikuPlan, error) {
	fullText, resp, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, err
	}

	if resp.StopReason == "max_tokens" {
		log.Printf("[PARSE] plan generator response cut off by max_tokens — aborting")
		return nil, fmt.Errorf("%w (plan generator)", ErrMaxTokens)
	}

	cleaned := extractJSON(fullText)

	var result models.HaikuPlan
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

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
		return extractJSON(jsonPart), descPart
	}

	if strings.HasPrefix(strings.TrimSpace(text), "{") {
		return extractJSON(text), ""
	}

	return "", text
}

func ParseVisualEditResponse(rawJSON string) ([]models.ProjectFile, string, error) {
	fullText, _, err := extractTextFromClaudeResponse(rawJSON)
	if err != nil {
		return nil, "", err
	}

	cleaned := extractJSON(fullText)

	var result struct {
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
		ChangeSummary string `json:"change_summary"`
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse visual edit JSON: %w", err)
	}

	files := make([]models.ProjectFile, 0, len(result.Files))
	for _, f := range result.Files {
		files = append(files, models.ProjectFile{
			Path:    f.Path,
			Content: f.Content,
		})
	}

	return files, result.ChangeSummary, nil
}
