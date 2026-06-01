package anthropic

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

var ToolArchitectPlan = claudeFunctionTool{
	Name:        "plan_architecture",
	Description: "Return the complete project architecture: database tables with fields and mock data, all relations between tables, a rich UI structure description, and a complete design system for the frontend developer.",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"project_name", "project_type", "tables", "relations", "ui_structure", "design", "image_keywords", "client_types"},
		"properties": map[string]any{
			"project_name": map[string]any{"type": "string", "description": "Human-readable project name"},
			"client_types": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "User role names extracted from the 'user-types' question answer. Each entry creates a separate client_type + role record. admin_panel and webapp: always at least [\"Administrator\"] (webapp products have end-user roles too). landing/web: empty [].",
			},
			"image_keywords": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "2–4 Unsplash search terms that visually represent this project's real-world domain. Be specific and physical: ['freight truck highway','warehouse forklift','shipping containers'] for logistics; ['espresso barista','cafe interior'] for coffee; ['doctor patient','clinic'] for healthcare. NEVER use generic terms like 'business','technology','office','app'.",
			},
			"project_type": map[string]any{
				"type":        "string",
				"enum":        []string{"admin_panel", "landing", "web", "webapp"},
				"description": "Detected project type. ALWAYS choose one of these four. KEYWORD OVERRIDE (highest priority unless it is explicitly an internal staff/admin tool): if the user calls it an 'app', 'web app', 'webapp', 'mobile app', 'application', or 'mobile application' → choose 'webapp'. There is NO mobile/native type — a mobile app is built as a responsive 'webapp'. Otherwise: 'landing' = strict single-page promotional site. 'web' = multi-page marketing/content website. 'admin_panel' = internal back-office tool for staff to manage data. 'webapp' = a product SaaS workspace used by end-users (Linear/Notion/Trello/Slack/Asana-like) — focused, keyboard-driven app shell with workspace nav, command palette, boards/lists/inbox, NOT a marketing site and NOT an internal admin dashboard.",
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
				"description": "Every foreign-key relationship between tables. Only Many2One is supported. The FK column {table_to}_id is auto-created on table_from (e.g. orders→customers creates column 'customers_id' on orders).",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"table_from", "table_to", "type"},
					"properties": map[string]any{
						"table_from": map[string]any{"type": "string", "description": "Source table slug — the 'many' side (e.g. 'orders' when many orders belong to one customer)"},
						"table_to":   map[string]any{"type": "string", "description": "Target table slug — the 'one' side (e.g. 'customers')"},
						"type": map[string]any{
							"type":        "string",
							"enum":        []string{"Many2One"},
							"description": "Always Many2One. Creates FK column {table_to}_id on table_from.",
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
					"primary_color":          map[string]any{"type": "string", "description": "Hex color, e.g. #6366f1"},
					"primary_hsl":            map[string]any{"type": "string", "description": "HSL without hsl(), e.g. 239 84% 67%"},
					"background_color":       map[string]any{"type": "string", "description": "Page background hex"},
					"background_hsl":         map[string]any{"type": "string", "description": "Page background HSL"},
					"surface_color":          map[string]any{"type": "string", "description": "Card/panel surface hex"},
					"surface_hsl":            map[string]any{"type": "string", "description": "Card/panel surface HSL"},
					"sidebar_background":     map[string]any{"type": "string", "description": "Sidebar bg hex"},
					"sidebar_background_hsl": map[string]any{"type": "string"},
					"sidebar_foreground":     map[string]any{"type": "string", "description": "Sidebar text hex"},
					"sidebar_style":          map[string]any{"type": "string", "enum": []string{"light", "medium", "dark", "colored"}, "description": "Sidebar visual weight"},
					"text_color":             map[string]any{"type": "string", "description": "Primary text hex"},
					"text_muted_color":       map[string]any{"type": "string", "description": "Secondary/muted text hex"},
					"border_color":           map[string]any{"type": "string", "description": "Border hex"},
					"accent_color":           map[string]any{"type": "string", "description": "Accent/highlight hex"},
					"accent_hsl":             map[string]any{"type": "string", "description": "Accent HSL"},
					"font_family":            map[string]any{"type": "string", "description": "Heading font name, e.g. Syne or Inter"},
					"body_font":              map[string]any{"type": "string", "description": "Body font name, e.g. DM Sans or Inter"},
					"border_radius":          map[string]any{"type": "string", "description": "Base border radius, e.g. 8px"},
					"design_inspiration":     map[string]any{"type": "string", "description": "Archetype name or reference, e.g. Obsidian Cinematic or TMS Domain"},
				},
			},
		},
	},
}

var ToolPlanChanges = claudeFunctionTool{
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

var ToolEmitProject = claudeFunctionTool{
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
				"type":        "array",
				"description": "CRITICAL: Must be a JSON array value — never a JSON-encoded string. Each element is an object with path and content.",
				"minItems":    1,
				"items": map[string]any{
					"type":     "object",
					"required": []string{"path", "content"},
					"properties": map[string]any{
						"path":    map[string]any{"type": "string", "description": "Relative file path e.g. src/App.tsx"},
						"content": map[string]any{"type": "string", "description": "Complete file content as a plain string — no extra JSON encoding"},
					},
				},
			},
		},
	},
}

var ToolEmitDiagrams = claudeFunctionTool{
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

var ToolEmitVisualEdit = claudeFunctionTool{
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

var ToolEmitManifest = claudeFunctionTool{
	Name:        "emit_manifest",
	Description: "Return the complete file manifest for a React admin panel, grouped by dependency level. Group 0 = foundation (generated first). Groups 1..N = features (generated in parallel after foundation).",
	InputSchema: map[string]any{
		"type":     "object",
		"required": []string{"groups"},
		"properties": map[string]any{
			"groups": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"id", "name", "files"},
					"properties": map[string]any{
						"id":   map[string]any{"type": "integer", "description": "0 for foundation, 1..N for feature groups"},
						"name": map[string]any{"type": "string", "description": "Group name, e.g. 'foundation', 'users', 'orders'"},
						"files": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":     "object",
								"required": []string{"path", "exports"},
								"properties": map[string]any{
									"path": map[string]any{"type": "string", "description": "Relative file path, e.g. src/pages/UsersPage.tsx"},
									"exports": map[string]any{
										"type":        "array",
										"items":       map[string]any{"type": "string"},
										"description": "All exported names from this file that other files might import",
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

var ToolRepairFile = claudeFunctionTool{
	Name:        "repair_file",
	Description: "Return the corrected content of a single TypeScript/TSX file. Fix all import errors listed in the prompt. Output the complete file — never truncate.",
	InputSchema: map[string]any{
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

func ForcedTool(toolName string) *toolChoice {
	return &toolChoice{Type: "tool", Name: toolName}
}
