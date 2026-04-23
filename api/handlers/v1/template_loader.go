package v1

import (
	_ "embed"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"ucode/ucode_go_api_gateway/api/models"
)

//go:embed project_template.json
var adminPanelTemplateRaw []byte

var (
	templateStore     map[string][]models.ProjectFile
	templateStoreOnce sync.Once
)

type templateJSON struct {
	Files []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"files"`
}

func init() {
	templateStoreOnce.Do(func() {
		templateStore = make(map[string][]models.ProjectFile)

		if len(adminPanelTemplateRaw) > 0 {
			var tmpl templateJSON
			if err := json.Unmarshal(adminPanelTemplateRaw, &tmpl); err != nil {
				log.Printf("[TEMPLATE] Error parsing embedded admin_panel template: %v", err)
				return
			}

			files := make([]models.ProjectFile, 0, len(tmpl.Files))
			for _, f := range tmpl.Files {
				files = append(files, models.ProjectFile{
					Path:    f.Path,
					Content: f.Content,
				})
			}
			templateStore["admin_panel"] = files
			log.Printf("[TEMPLATE] Embedded admin_panel template loaded: %d files", len(files))
		}
	})
}

func GetTemplate(projectType string) []models.ProjectFile {
	return templateStore[projectType]
}

// scaffoldPaths are template files that must exist in the output but should NOT
// be sent to the AI as context (they waste tokens and the AI shouldn't re-emit them).
var scaffoldPaths = map[string]bool{
	"package.json":       true,
	"tailwind.config.js": true,
	"tsconfig.json":      true,
	"vite.config.ts":     true,
	"src/App.tsx":        true,
	"src/index.css":      true,
	"src/main.tsx":       true,
}

// GetTemplateContext returns only files the AI should read and import from
// (hooks, utils, types, config). These are sent in the coder prompt.
func GetTemplateContext(projectType string) []models.ProjectFile {
	all := templateStore[projectType]
	out := make([]models.ProjectFile, 0, len(all))
	for _, f := range all {
		if !scaffoldPaths[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

// GetTemplateScaffold returns base scaffold files that are silently merged into
// the AI output without being sent as prompt context.
func GetTemplateScaffold(projectType string) []models.ProjectFile {
	all := templateStore[projectType]
	out := make([]models.ProjectFile, 0, len(all))
	for _, f := range all {
		if scaffoldPaths[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

func MergeTemplateWithAIFiles(templateFiles, aiFiles []models.ProjectFile) []models.ProjectFile {
	aiPathSet := make(map[string]bool, len(aiFiles))
	for _, f := range aiFiles {
		aiPathSet[f.Path] = true
	}

	merged := make([]models.ProjectFile, 0, len(templateFiles)+len(aiFiles))
	for _, tf := range templateFiles {
		if aiPathSet[tf.Path] {
			continue
		}
		if strings.Contains(tf.Content, "[AI:") {
			continue
		}
		merged = append(merged, tf)
	}
	merged = append(merged, aiFiles...)

	return merged
}
