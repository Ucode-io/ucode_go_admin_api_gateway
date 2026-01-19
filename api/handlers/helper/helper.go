package helper

import "bytes"

func CleanJSONResponse(text string) string {

	text = string(bytes.TrimSpace([]byte(text)))
	if bytes.HasPrefix([]byte(text), []byte("```json")) {
		text = string(bytes.TrimPrefix([]byte(text), []byte("```json")))
		text = string(bytes.TrimPrefix([]byte(text), []byte("```")))
	}
	if bytes.HasSuffix([]byte(text), []byte("```")) {
		text = string(bytes.TrimSuffix([]byte(text), []byte("```")))
	}

	return string(bytes.TrimSpace([]byte(text)))
}
