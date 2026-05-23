package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func repairStringifiedFields(input map[string]interface{}) {
	passthroughKeys := map[string]bool{
		"content": true, "ui_structure": true, "bpmn_xml": true,
		"summary": true, "change_summary": true,
	}
	for k, v := range input {
		s, ok := v.(string)
		if !ok || passthroughKeys[k] {
			continue
		}
		s = strings.TrimSpace(s)
		s = stripCodeFence(s)
		if !isJSONContainer(s) {
			continue
		}
		if parsed := tryParseJSON(s); parsed != nil {
			input[k] = parsed
			continue
		}
		if parsed := tryParseJSON(sanitizeJSONContent(s)); parsed != nil {
			input[k] = parsed
			continue
		}
		if parsed := tryParseJSON(repairJSONStrings(sanitizeJSONContent(s))); parsed != nil {
			input[k] = parsed
			continue
		}
		if k == "files" {
			if extracted, ok := extractFilesFromString(s); ok {
				input[k] = extracted
				continue
			}
		}
		preview := s
		if len(preview) > 200 {
			preview = preview[:200]
		}
		log.Printf("[TOOL DECODE] all repair attempts failed for field %q: %s", k, preview)
	}
}

func stripCodeFence(s string) string {
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	} else {
		return s
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}

func isJSONContainer(s string) bool {
	return (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) ||
		(strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}"))
}

func tryParseJSON(s string) interface{} {
	var v interface{}
	if json.Unmarshal([]byte(s), &v) == nil {
		return v
	}
	return nil
}

func doHTTP(req *http.Request, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}
	return string(respBytes), nil
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

func isSourceFilePath(p string) bool {
	if strings.HasPrefix(p, "/") {
		return false
	}
	lower := strings.ToLower(p)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico", ".mp4", ".mp3", ".woff", ".woff2", ".ttf", ".eot", ".otf", ".wav", ".ogg", ".avif"} {
		if strings.HasSuffix(lower, ext) {
			return false
		}
	}
	return true
}

func extractFilesFromString(s string) ([]map[string]interface{}, bool) {
	fileStartRe := regexp.MustCompile(`\{\s*"path"\s*:\s*"([^"\s\\]{1,200})"`)
	allMatches := fileStartRe.FindAllStringSubmatchIndex(s, -1)
	if len(allMatches) == 0 {
		return nil, false
	}

	matches := allMatches[:0]
	for _, m := range allMatches {
		if isSourceFilePath(s[m[2]:m[3]]) {
			matches = append(matches, m)
		}
	}
	if len(matches) == 0 {
		return nil, false
	}

	contentKeyRe := regexp.MustCompile(`"content"\s*:\s*"`)

	var files []map[string]interface{}
	for i, m := range matches {
		chunkStart := m[0]
		var chunkEnd int
		if i+1 < len(matches) {
			chunkEnd = matches[i+1][0]
		} else {
			chunkEnd = len(s)
		}
		chunk := s[chunkStart:chunkEnd]

		pathStart := m[2] - chunkStart
		pathEnd := m[3] - chunkStart
		if pathStart < 0 || pathEnd > len(chunk) {
			continue
		}
		path := chunk[pathStart:pathEnd]

		if !strings.ContainsAny(path, "./") {
			continue
		}

		cLoc := contentKeyRe.FindStringIndex(chunk)
		if cLoc == nil {
			continue
		}
		rawContent := chunk[cLoc[1]:]

		lastQ := strings.LastIndex(rawContent, `"`)
		if lastQ < 0 {
			continue
		}
		rawContent = rawContent[:lastQ]

		content := unescapeJSONString(rawContent)
		files = append(files, map[string]interface{}{
			"path":    path,
			"content": content,
		})
	}

	if len(files) == 0 {
		return nil, false
	}
	return files, true
}

func unescapeJSONString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case '/':
			b.WriteByte('/')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'u':
			if i+4 < len(s) {
				var r rune
				for _, c := range s[i+1 : i+5] {
					r <<= 4
					switch {
					case c >= '0' && c <= '9':
						r |= rune(c - '0')
					case c >= 'a' && c <= 'f':
						r |= rune(c-'a') + 10
					case c >= 'A' && c <= 'F':
						r |= rune(c-'A') + 10
					default:
						r = -1
					}
					if r < 0 {
						break
					}
				}
				if r >= 0 {
					b.WriteRune(r)
					i += 4
					continue
				}
			}
			b.WriteString(`\u`)
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}