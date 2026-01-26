package helper

import (
	"regexp"
	"strings"
)

func CleanJSONResponse(input string) string {
	re := regexp.MustCompile("(?s)^.*?(?:```json|```)?\\s*(\\{.*\\})\\s*(?:```)?.*?$")
	match := re.FindStringSubmatch(input)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start != -1 && end != -1 && end > start {
		return input[start : end+1]
	}

	return input
}
