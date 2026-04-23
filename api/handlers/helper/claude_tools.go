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
	Description: "Return the complete project architecture: database tables with fields and mock data, a rich UI structure description, and a complete design system for the frontend developer.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"project_name", "project_type", "tables", "ui_structure", "design"},
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
				"description": "Rich, detailed description of pages, layout, features and visual structure for the frontend developer",
			},
			"design": map[string]any{
				"type":        "object",
				"description": "Complete design system tokens. The code generator uses these exact values — fill every field.",
				"required": []string{
					"primary_color", "primary_hsl", "background_color", "background_hsl",
					"surface_color", "sidebar_background", "sidebar_style",
					"text_color", "text_muted_color", "border_color",
					"accent_color", "accent_hsl", "font_family", "body_font",
					"border_radius", "design_inspiration",
				},
				"properties": map[string]any{
					"primary_color":      map[string]any{"type": "string", "description": "Hex color, e.g. #6366f1"},
					"primary_hsl":        map[string]any{"type": "string", "description": "HSL without hsl(), e.g. 239 84% 67%"},
					"background_color":   map[string]any{"type": "string", "description": "Page background hex"},
					"background_hsl":     map[string]any{"type": "string", "description": "Page background HSL"},
					"surface_color":      map[string]any{"type": "string", "description": "Card/panel surface hex"},
					"surface_hsl":        map[string]any{"type": "string", "description": "Card/panel surface HSL"},
					"sidebar_background": map[string]any{"type": "string", "description": "Sidebar bg hex"},
					"sidebar_background_hsl": map[string]any{"type": "string"},
					"sidebar_foreground": map[string]any{"type": "string", "description": "Sidebar text hex"},
					"sidebar_style":      map[string]any{"type": "string", "enum": []string{"light", "medium", "dark", "colored"}, "description": "Sidebar visual weight"},
					"text_color":         map[string]any{"type": "string", "description": "Primary text hex"},
					"text_muted_color":   map[string]any{"type": "string", "description": "Secondary/muted text hex"},
					"border_color":       map[string]any{"type": "string", "description": "Border hex"},
					"accent_color":       map[string]any{"type": "string", "description": "Accent/highlight hex"},
					"accent_hsl":         map[string]any{"type": "string", "description": "Accent HSL"},
					"font_family":        map[string]any{"type": "string", "description": "Heading font name, e.g. Syne or Inter"},
					"body_font":          map[string]any{"type": "string", "description": "Body font name, e.g. DM Sans or Inter"},
					"border_radius":      map[string]any{"type": "string", "description": "Base border radius, e.g. 8px"},
					"design_inspiration": map[string]any{"type": "string", "description": "Archetype name or reference, e.g. Obsidian Cinematic or TMS Domain"},
				},
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
