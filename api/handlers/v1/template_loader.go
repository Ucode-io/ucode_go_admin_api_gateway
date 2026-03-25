package v1

import (
	_ "embed"
	"encoding/json"
	"log"
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

// MergeTemplateWithAIFiles combines template base files with AI-generated files.
// AI files take precedence — if AI generates a file with the same path as a
// template file, the AI version wins. Template files not touched by AI are preserved.
func MergeTemplateWithAIFiles(templateFiles, aiFiles []models.ProjectFile) []models.ProjectFile {
	aiPathSet := make(map[string]bool, len(aiFiles))
	for _, f := range aiFiles {
		aiPathSet[f.Path] = true
	}

	merged := make([]models.ProjectFile, 0, len(templateFiles)+len(aiFiles))
	for _, tf := range templateFiles {
		if !aiPathSet[tf.Path] {
			merged = append(merged, tf)
		}
	}
	merged = append(merged, aiFiles...)

	return merged
}
