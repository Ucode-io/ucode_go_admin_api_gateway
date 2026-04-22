package helper

import "ucode/ucode_go_api_gateway/api/models"

// ============================================================================
// Claude Function Tool Schemas
//
// Each variable here corresponds to one Anthropic tool-use tool.
// When a tool is passed with tool_choice={type:"tool", name:"<name>"}, Claude
// MUST populate the tool's input_schema exactly — no text, no markdown, no JSON
// escaping bugs.  The output is decoded directly into the matching Go struct.
//
// Tool ↔ Go struct mapping:
//   ToolArchitectPlan  → models.ArchitectPlan
//   ToolPlanChanges    → models.SonnetPlanResult
//   ToolEmitProject    → models.GeneratedProject
//   ToolEmitDiagrams   → models.HaikuPlan
//   ToolEmitVisualEdit → VisualEditOutput  (defined in ai_messging.go)
// ============================================================================

var ToolArchitectPlan = models.ClaudeFunctionTool{
	Name:        "plan_architecture",
	Description: "Return the complete project architecture: database tables with fields and mock data, plus a rich UI structure description for the frontend developer.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"project_name", "project_type", "tables", "ui_structure"},
		"properties": map[string]any{
			"project_name": map[string]any{"type": "string", "description": "Human-readable project name"},
			"project_type": map[string]any{
				"type":        "string",
				"enum":        []string{"admin_panel", "landing", "web", "other"},
				"description": "Detected project type",
			},
			"tables": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"slug", "label", "fields", "mock_data"},
					"properties": map[string]any{
						"slug":           map[string]any{"type": "string"},
						"label":          map[string]any{"type": "string"},
						"is_login_table": map[string]any{"type": "boolean"},
						"login_strategy": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"fields": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":     "object",
								"required": []string{"slug", "label", "type"},
								"properties": map[string]any{
									"slug":  map[string]any{"type": "string"},
									"label": map[string]any{"type": "string"},
									"type":  map[string]any{"type": "string"},
								},
							},
						},
						"mock_data": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": true,
							},
						},
					},
				},
			},
			"ui_structure": map[string]any{
				"type":        "string",
				"description": "Rich, detailed description of pages, layout, features and visual design for the frontend developer",
			},
		},
	},
}

var ToolPlanChanges = models.ClaudeFunctionTool{
	Name:        "plan_changes",
	Description: "List every file that needs to be created or modified to fulfil the requested code change. Do not include file contents — only paths and one-sentence descriptions.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"files_to_change", "files_to_create", "summary"},
		"properties": map[string]any{
			"files_to_change": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"path", "description"},
					"properties": map[string]any{
						"path":        map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
				},
			},
			"files_to_create": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"path", "description"},
					"properties": map[string]any{
						"path":        map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
					},
				},
			},
			"summary": map[string]any{"type": "string", "description": "One sentence summary of what will change"},
		},
	},
}

var ToolEmitProject = models.ClaudeFunctionTool{
	Name:        "emit_project",
	Description: "Return the complete set of generated project files. Include every file needed to run the project. File contents must be complete — never truncate.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"project_name", "files", "env"},
		"properties": map[string]any{
			"project_name": map[string]any{"type": "string"},
			"env": map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "string"},
				"description":          "All VITE_* environment variables with their real values",
			},
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"path", "content"},
					"properties": map[string]any{
						"path":    map[string]any{"type": "string", "description": "Relative path e.g. src/App.tsx"},
						"content": map[string]any{"type": "string", "description": "Full file content"},
					},
				},
			},
		},
	},
}

var ToolEmitDiagrams = models.ClaudeFunctionTool{
	Name:        "emit_diagrams",
	Description: "Return the BPMN 2.0 process diagram and the infrastructure dependency diagram for the project.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"bpmn_xml", "infra_diagram"},
		"properties": map[string]any{
			"bpmn_xml": map[string]any{
				"type":        "string",
				"description": "Full BPMN 2.0 XML. Newlines must be literal \\n inside the string value.",
			},
			"infra_diagram": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"from", "to", "label"},
					"properties": map[string]any{
						"from":  map[string]any{"type": "string"},
						"to":    map[string]any{"type": "string"},
						"label": map[string]any{"type": "string"},
					},
				},
			},
		},
	},
}

var ToolEmitVisualEdit = models.ClaudeFunctionTool{
	Name:        "emit_visual_edit",
	Description: "Return the surgically edited files and a one-sentence summary of what was changed. Only include files that actually changed.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"files", "change_summary"},
		"properties": map[string]any{
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"path", "content"},
					"properties": map[string]any{
						"path":    map[string]any{"type": "string"},
						"content": map[string]any{"type": "string", "description": "Complete updated file content"},
					},
				},
			},
			"change_summary": map[string]any{"type": "string", "description": "One sentence describing what was changed and why"},
		},
	},
}

// ForcedTool returns a ToolChoice that forces Claude to call a specific tool by name.
func ForcedTool(toolName string) *models.ToolChoice {
	return &models.ToolChoice{Type: "tool", Name: toolName}
}
