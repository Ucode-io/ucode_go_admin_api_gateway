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
