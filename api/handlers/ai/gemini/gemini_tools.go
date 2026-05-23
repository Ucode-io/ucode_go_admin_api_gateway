package gemini

// Tool definitions mirror claude_tools.go but use Gemini's funcDeclaration format.
// Names and JSON Schema parameter maps are identical to the Anthropic tools so both
// providers produce output that unmarshals into the same Go structs.

var toolArchitectPlan = funcDeclaration{
	Name:        "plan_architecture",
	Description: "Return the complete project architecture: database tables with fields and mock data, all relations between tables, a rich UI structure description, and a complete design system for the frontend developer.",
	Parameters: map[string]any{
		"type":     "object",
		"required": []string{"project_name", "project_type", "tables", "relations", "ui_structure", "design", "image_keywords", "client_types"},
		"properties": map[string]any{
			"project_name": map[string]any{"type": "string", "description": "Human-readable project name"},
			"client_types": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "User role names extracted from the 'user-types' question answer. Each entry creates a separate client_type + role record. admin_panel: always at least [\"Administrator\"]. landing/web: empty [].",
			},
			"image_keywords": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "2–4 Unsplash search terms that visually represent this project's real-world domain. Be specific and physical: ['freight truck highway','warehouse forklift','shipping containers'] for logistics; ['espresso barista','cafe interior'] for coffee; ['doctor patient','clinic'] for healthcare. NEVER use generic terms like 'business','technology','office','app'.",
			},
			"project_type": map[string]any{
				"type":        "string",
				"enum":        []string{"admin_panel", "landing", "web"},
				"description": "Detected project type. ALWAYS choose one of these three. Use 'web' for multi-page websites, 'landing' for strict single-page promotional sites.",
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
			"relations": map[string]any{
				"type":        "array",
				"description": "Every foreign-key relationship between tables. Only Many2One is supported. The FK column {table_to}_id is auto-created on table_from.",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"table_from", "table_to", "type"},
					"properties": map[string]any{
						"table_from": map[string]any{"type": "string"},
						"table_to":   map[string]any{"type": "string"},
						"type": map[string]any{
							"type": "string",
							"enum": []string{"Many2One"},
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
					"surface_color", "surface_hsl", "sidebar_background", "sidebar_background_hsl", "sidebar_style",
					"text_color", "text_muted_color", "border_color",
					"accent_color", "accent_hsl", "font_family", "body_font",
					"border_radius", "design_inspiration",
				},
				"properties": map[string]any{
					"primary_color":          map[string]any{"type": "string"},
					"primary_hsl":            map[string]any{"type": "string"},
					"background_color":       map[string]any{"type": "string"},
					"background_hsl":         map[string]any{"type": "string"},
					"surface_color":          map[string]any{"type": "string"},
					"surface_hsl":            map[string]any{"type": "string"},
					"sidebar_background":     map[string]any{"type": "string"},
					"sidebar_background_hsl": map[string]any{"type": "string"},
					"sidebar_foreground":     map[string]any{"type": "string"},
					"sidebar_style":          map[string]any{"type": "string", "enum": []string{"light", "medium", "dark", "colored"}},
					"text_color":             map[string]any{"type": "string"},
					"text_muted_color":       map[string]any{"type": "string"},
					"border_color":           map[string]any{"type": "string"},
					"accent_color":           map[string]any{"type": "string"},
					"accent_hsl":             map[string]any{"type": "string"},
					"font_family":            map[string]any{"type": "string"},
					"body_font":              map[string]any{"type": "string"},
					"border_radius":          map[string]any{"type": "string"},
					"design_inspiration":     map[string]any{"type": "string"},
				},
			},
		},
	},
}

var toolPlanChanges = funcDeclaration{
	Name:        "plan_changes",
	Description: "List every file that needs to be created or modified to fulfil the requested code change. Do not include file contents — only paths and one-sentence descriptions.",
	Parameters: map[string]any{
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
			"summary": map[string]any{"type": "string"},
		},
	},
}

var toolEmitProject = funcDeclaration{
	Name:        "emit_project",
	Description: "Return the complete set of generated project files. Include every file needed to run the project. File contents must be complete — never truncate.",
	Parameters: map[string]any{
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
				"type":        "array",
				"description": "CRITICAL: Must be a JSON array value — never a JSON-encoded string. Each element is an object with path and content.",
				"minItems":    1,
				"items": map[string]any{
					"type":     "object",
					"required": []string{"path", "content"},
					"properties": map[string]any{
						"path":    map[string]any{"type": "string"},
						"content": map[string]any{"type": "string"},
					},
				},
			},
		},
	},
}

var toolEmitVisualEdit = funcDeclaration{
	Name:        "emit_visual_edit",
	Description: "Return the surgically edited files and a one-sentence summary of what was changed. Only include files that actually changed.",
	Parameters: map[string]any{
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
						"content": map[string]any{"type": "string"},
					},
				},
			},
			"change_summary": map[string]any{"type": "string"},
		},
	},
}

var toolEmitManifest = funcDeclaration{
	Name:        "emit_manifest",
	Description: "Return the complete file manifest for a React admin panel, grouped by dependency level. Group 0 = foundation (generated first). Groups 1..N = features (generated in parallel after foundation).",
	Parameters: map[string]any{
		"type":     "object",
		"required": []string{"groups"},
		"properties": map[string]any{
			"groups": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"id", "name", "files"},
					"properties": map[string]any{
						"id":   map[string]any{"type": "integer"},
						"name": map[string]any{"type": "string"},
						"files": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":     "object",
								"required": []string{"path", "exports"},
								"properties": map[string]any{
									"path": map[string]any{"type": "string"},
									"exports": map[string]any{
										"type":  "array",
										"items": map[string]any{"type": "string"},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

var toolRepairFile = funcDeclaration{
	Name:        "repair_file",
	Description: "Return the corrected content of a single TypeScript/TSX file. Fix all import errors listed in the prompt. Output the complete file — never truncate.",
	Parameters: map[string]any{
		"type":     "object",
		"required": []string{"content"},
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "Full corrected file content",
			},
		},
	},
}