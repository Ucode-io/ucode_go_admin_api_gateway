package crmguard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/config"
)

const canonicalPBXInvocation = "/v2/invoke_function/professional-crm-pbx-integration-call"
const executableBodyLimit = 2 << 20

func executableEnabled(cfg config.CRMNativeConfig) bool { return cfg.Enabled || cfg.InboundEnabled }

func executableAlias(path string) bool {
	path = strings.ToLower(path)
	for _, prefix := range []string{"/v1/invoke_function", "/v2/invoke_function", "/v1/knative", "/api", "/x-api"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	if (strings.HasPrefix(path, "/v1/functions/") || strings.HasPrefix(path, "/v2/functions/")) && (strings.HasSuffix(path, "/run") || strings.HasSuffix(path, "/invoke")) {
		return true
	}
	return path == "/v1/crm-ai/chat" || path == "/v2/ai-builder/messages" || strings.HasPrefix(path, "/v1/ai-chat/")
}

func denyExecutable(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "error", "message": "record unavailable"})
}

// Inspect only positively selected target aliases before any managed-key lookup.
// Missing, conflicting, malformed or oversized context keeps its original path.
func ExecutableBoundary(cfg config.CRMNativeConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !executableEnabled(cfg) || c.Request.Method == http.MethodOptions || !executableAlias(path) || path == canonicalPBXInvocation {
			c.Next()
			return
		}
		target, selected := executableRequestSelection(c.Request, cfg)
		if selected {
			if target {
				denyExecutable(c)
			} else {
				c.Next()
			}
			return
		}
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			original := c.Request.Body
			body, err := io.ReadAll(io.LimitReader(original, executableBodyLimit+1))
			c.Request.Body = struct {
				io.Reader
				io.Closer
			}{Reader: io.MultiReader(bytes.NewReader(body), original), Closer: original}
			if err == nil && len(body) <= executableBodyLimit {
				object, valid := executableJSONObject(body)
				if valid && executableObjectTarget(object, cfg) {
					denyExecutable(c)
					return
				}
			}
		}
		c.Next()
	}
}

func executableRequestSelection(r *http.Request, cfg config.CRMNativeConfig) (bool, bool) {
	var project, environment string
	projectCount, environmentCount := 0, 0
	inspect := func(key string, values []string) {
		key = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
		switch key {
		case "project-id":
			projectCount += len(values)
			if len(values) == 0 {
				projectCount++
			}
			if len(values) == 1 {
				project = values[0]
			}
		case "environment-id":
			environmentCount += len(values)
			if len(values) == 0 {
				environmentCount++
			}
			if len(values) == 1 {
				environment = values[0]
			}
		}
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if queryErr != nil {
		for _, component := range strings.Split(r.URL.RawQuery, "&") {
			key, _, _ := strings.Cut(component, "=")
			key, _ = url.QueryUnescape(key)
			key = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
			if key == "project-id" || key == "environment-id" {
				return false, true
			}
		}
	}
	for key, values := range query {
		inspect(key, values)
	}
	for key, values := range r.Header {
		inspect(key, values)
	}
	selected := projectCount != 0 || environmentCount != 0
	// Explicit selectors govern routing. Partial, duplicate or conflicting
	// selectors preserve the old dispatch rather than infer a tenant from JSON.
	target := projectCount == 1 && environmentCount == 1 && validID(cfg.Project) && validID(cfg.Environment) && project == cfg.Project && environment == cfg.Environment
	return target, selected
}

func executableObjectTarget(object map[string]json.RawMessage, cfg config.CRMNativeConfig) bool {
	if !validID(cfg.Project) || !validID(cfg.Environment) {
		return false
	}
	_, hasProject := object["project_id"]
	_, hasEnvironment := object["environment_id"]
	var project, environment string
	_ = json.Unmarshal(object["project_id"], &project)
	_ = json.Unmarshal(object["environment_id"], &environment)
	if hasProject || hasEnvironment {
		return hasProject && hasEnvironment && project == cfg.Project && environment == cfg.Environment
	}
	data, valid := executableJSONObject(object["data"])
	if !valid {
		return false
	}
	project, environment = "", ""
	_ = json.Unmarshal(data["project_id"], &project)
	_ = json.Unmarshal(data["environment_id"], &environment)
	return project == cfg.Project && environment == cfg.Environment
}

func executableJSONObject(body []byte) (map[string]json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, false
	}
	object := map[string]json.RawMessage{}
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return nil, false
		}
		key = strings.ToLower(key)
		var raw json.RawMessage
		if decoder.Decode(&raw) != nil {
			return nil, false
		}
		if _, duplicate := object[key]; duplicate && (key == "project_id" || key == "environment_id" || key == "data") {
			return nil, false
		}
		object[key] = raw
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
		return nil, false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, false
	}
	return object, true
}
