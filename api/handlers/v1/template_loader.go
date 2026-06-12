package v1

import (
	_ "embed"
	"encoding/json"
	"log"

	"ucode/ucode_go_api_gateway/api/models"
)

//go:embed project_template.json
var projectTemplateRaw []byte

var projectTemplate []models.ProjectFile

func init() {
	var tmpl struct {
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.Unmarshal(projectTemplateRaw, &tmpl); err != nil {
		log.Printf("[TEMPLATE] failed to parse project_template.json: %v", err)
		return
	}
	projectTemplate = make([]models.ProjectFile, 0, len(tmpl.Files))
	for _, f := range tmpl.Files {
		projectTemplate = append(projectTemplate, models.ProjectFile{
			Path:    f.Path,
			Content: f.Content,
		})
	}
	log.Printf("[TEMPLATE] loaded %d files", len(projectTemplate))
}

func GetTemplateScaffold() []models.ProjectFile {
	return projectTemplate
}

// forceTemplatePaths lists pre-built files that must ALWAYS come from the
// template, even when the model regenerated them. The auth runtime is the
// motivating case: despite the PRE-BUILT contract (manifest + prompt both say
// "do NOT re-emit them"), the model sometimes still emits a stubbed LoginPage
// ("simulate login", no real /v2/login call) that silently breaks sign-in on
// login-gated panels. Skip-if-present injection then keeps the broken stub.
// Overwriting guarantees the working runtime ships regardless of model output.
var forceTemplatePaths = map[string]bool{
	"src/lib/auth.ts":                        true,
	"src/lib/permissions.ts":                 true,
	"src/components/auth/LoginPage.tsx":      true,
	"src/components/auth/ProtectedRoute.tsx": true,
}

// mergeTemplateScaffold merges template scaffold files into the generated set:
// files in forceTemplatePaths always win (overwriting model output); every other
// scaffold file is added only when the model did not already produce it.
func mergeTemplateScaffold(files, scaffold []models.ProjectFile) []models.ProjectFile {
	if len(scaffold) == 0 {
		return files
	}

	indexByPath := make(map[string]int, len(files))
	for i, f := range files {
		indexByPath[f.Path] = i
	}

	for _, sf := range scaffold {
		if idx, exists := indexByPath[sf.Path]; exists {
			if forceTemplatePaths[sf.Path] {
				files[idx] = sf
			}
			continue
		}
		files = append(files, sf)
		indexByPath[sf.Path] = len(files) - 1
	}

	return files
}

func GetTemplateContext() []models.ProjectFile {
	skip := map[string]bool{
		"package.json":       true,
		"tailwind.config.js": true,
		"tsconfig.json":      true,
		"vite.config.ts":     true,
		"src/App.tsx":        true,
		"src/index.css":      true,
		"src/main.tsx":       true,
	}
	out := make([]models.ProjectFile, 0, len(projectTemplate))
	for _, f := range projectTemplate {
		if !skip[f.Path] {
			out = append(out, f)
		}
	}
	return out
}
