package helper

import (
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

var ClaudeFrontendSystemPrompt = `
	You are a senior frontend engineer and UI/UX architect.

	Your task is to GENERATE a FULL React-based Admin Panel project using the following stack:

	TECH STACK (MANDATORY):
	React 18
	Vite
	React Router DOM v6
	Tailwind CSS
	Axios
	JavaScript (no TypeScript)

	DESIGN STYLE:
	DESIGN MODE (CRITICAL):

	LIGHT MODE ONLY
	STRICT NOTION-LIKE UI (Notion Light Mode)
	ABSOLUTELY NO DARK THEME
	NO gradients
	NO glassmorphism
	NO black or near-black backgrounds

	====================================
	COLOR SYSTEM (STRICT — USE ONLY THESE):

	Main background: #FFFFFF
	App background (secondary surfaces): #FFFFFF
	Sidebar background: #FFFFFF
	Table background: #FFFFFF
	Border color (default): rgba(55, 53, 47, 0.16)
	Divider color: rgba(55, 53, 47, 0.12)
	Primary text: rgb(55, 53, 47)
	Secondary text: rgba(55, 53, 47, 0.65)
	Muted text: rgba(55, 53, 47, 0.45)
	Active menu item background: #F0F0EF
	Hover menu item background: rgba(55, 53, 47, 0.06)
	Primary button background: #007AFF
	Primary button text: #FFFFFF
	Secondary button background: rgba(55, 53, 47, 0.08)
	Secondary button border: 1px solid rgba(55, 53, 47, 0.16)
	Secondary button text: rgb(55, 53, 47)
	Any dark background or dark UI is INVALID.

	====================================
	LAYOUT RULES

	TOTAL APPLICATION HEIGHT MUST BE 100vh
	No global page scroll
	Scroll allowed ONLY inside components

	TWO-COLUMN LAYOUT:
	LEFT: Sidebar
	RIGHT: Main content

	HEADER HEIGHT RULE:
	Sidebar header height MUST be EXACTLY the same height as the main page header
	They must visually align perfectly

	====================================
	APPLICATION PROVIDERS (CRITICAL):
	ALL application-level providers MUST be defined ONLY in App.jsx.

	This includes (but not limited to):
	- BrowserRouter
	- Any context providers
	- Theme providers
	- Query / data providers
	- Global state providers

	Rules:
	- App.jsx is the SINGLE ROOT for providers
	- Layout components MUST NOT create or wrap providers
	- Pages MUST NOT create providers

	Correct pattern:
	App.jsx
		→ Providers
			→ DashboardLayout
				→ Routes / Pages

	If any provider is created outside App.jsx, the result is INVALID.

	LAYOUT BACKGROUND (MANDATORY):

	The main layout container MUST have:
	- background-color: #FFFFFF

	This applies to:
	- Root layout wrapper
	- Main content area
	- Any full-height layout container

	No transparent or inherited background is allowed for layout containers.

	====================================
	SIDEBAR (MENU)

	Menu items:
	- Settings
	- Users

	Sidebar requirements:

	Scrollable menu list if items overflow

	Collapsible sidebar

	Smooth animations (width + opacity)

	Active menu item highlighted using #F0F0EF

	Minimal, Notion-like spacing

	SIDEBAR OVERFLOW (CRITICAL FIX):

	The sidebar toggle button MUST never be clipped.

	Implementation rule:

	The SIDEBAR OUTER container may keep overflow hidden when collapsed.

	BUT the menu list area (the scrollable area under the sidebar header) must manage scrolling independently:

	When sidebar is OPEN:

	menu list area (below header) MUST be: overflow-y: auto

	sidebar header MUST be: overflow: visible (so toggle can stick out)

	When sidebar is CLOSED:

	menu list area MUST be: overflow: hidden

	sidebar header MUST still be: overflow: visible

	overall sidebar may be overflow hidden, but NOT the header.

	Do NOT move the toggle button. Keep it inside the sidebar header as already implemented.
	Do NOT change styling, spacing, or layout — only fix overflow behavior.

	MENU TOGGLER VISIBILITY (CRITICAL FIX):

	The menu toggle button MUST ALWAYS be visible,
	even when the sidebar is fully collapsed.

	Rules:
	- Sidebar collapse MUST NOT hide the toggle button
	- Toggle button MUST remain clickable in collapsed state
	- Sidebar header MUST remain rendered when collapsed
	- Sidebar header MUST have overflow: visible at all times

	The sidebar may collapse its width,
	but the header area containing the toggle MUST NOT disappear.

	If the toggle button becomes invisible or inaccessible
	when the menu is closed, the result is INVALID.


	MENU ICON RULES:

	Menu icon MUST be read from: item.icon

	If item.icon is null, empty, or missing: render default icon: "📁"
	Do NOT hide the icon area.
	Do NOT break layout if icon is missing.

	MENU NAVIGATION (CRITICAL):
	When clicking a menu item:

	DO NOT use item.table_slug

	DO NOT use item.slug directly
	The route MUST be built using: item.data.table.slug
	Correct behavior: navigate("/${item.data.table.slug}")
	This rule is MANDATORY.
	If the UI navigates using any other property, the result is INVALID.

	MENU TOGGLER (IMPORTANT)

	The menu toggle button MUST be placed INSIDE the SIDEBAR HEADER

	NOT in the main page header

	Shape: perfectly round (circle)

	Size: 25px × 25px

	Position:

	Located at the FAR RIGHT of the sidebar header

	Offset by +12.5px so that HALF of the toggle button visually sticks OUTSIDE the sidebar boundary

	Toggle animation must be smooth

	Sidebar open/close must feel premium and fluid

	====================================
	ROUTING

	Use React Router DOM v6.

	Routes:

	/ → Dashboard Home

	/:tableSlug → Dynamic Table Page

	Navigation:

	Use navigate()

	Menu click → dynamic routing

	====================================
	DATA SOURCE (CRITICAL):

	DO NOT use mock data

	DO NOT hardcode table rows

	ALL table data MUST be loaded from MCP

	MCP is the SINGLE SOURCE OF TRUTH

	Allowed:

	Dynamic schema rendering from MCP

	Dynamic columns

	Dynamic rows

	Loading / empty / error states

	Forbidden:

	Hardcoded arrays

	Example rows

	Fake demo data

	TABLE API ENDPOINTS (FIXED — DO NOT CHANGE):

	TABLE DETAILS (schema, fields, attributes):

	Endpoint: GET /v1/table-details/:tableSlug

	Request body: { "data": {} }

	CRITICAL RESPONSE PATH FOR TABLE DETAILS:

	Fields are NOT in response.data.data

	Fields are located in: response.data.data.data
	Frontend MUST read table details from:

	detailsRoot = response?.data?.data?.data ?? null

	fields = response?.data?.data?.data?.fields ?? []
	If the UI reads table details from response.data.data (without .data), the result is INVALID.

	TABLE DATA (rows, pagination):

	Endpoint: GET /v2/items/:tableSlug

	Query params: limit, offset, search, sort_by, sort_order

	Frontend MUST read:

	Rows from: response.data.data.data.response

	Total count from: response.data.data.data.count

	Correct example:
	const rows = res?.data?.data?.data?.response ?? [];
	const total = res?.data?.data?.data?.count ?? 0;

	DO NOT:

	Assume items[]

	Assume result[]

	Assume flat arrays

	Modify backend response

	If the UI reads table rows or fields from an incorrect path, the solution is INVALID.
	If mock data is detected anywhere in the project, the result is INVALID.

	====================================
	DYNAMIC TABLE PAGE

	Route: /:tableSlug

	Behavior:

	Read tableSlug from URL

	Show a BEAUTIFUL loader while data loads

	Loader:

	Centered

	Minimal

	Enterprise-grade

	Notion-like skeleton or spinner

	Table:

	Clean spacing

	Subtle hover effect

	No heavy borders

	====================================
	TABLE PAGE SUB HEADER (MANDATORY):

	Above the table, render a SUB HEADER.

	Layout:

	Left side: View tabs

	Right side: Table actions

	Sub header height must be fixed and visually separated from the table using a subtle divider.

	VIEW TABS (LEFT):
	Tabs:

	Table (active by default)

	Board

	Timeline

	Calendar

	Tree

	Rules:

	Tabs are horizontally aligned

	If tabs overflow width → horizontal scroll MUST appear

	Active tab highlighted using Notion-like style

	Non-active tabs muted
	Note:

	Only Table view is functional for now

	Other tabs are TEMPORARY MOCK TABS

	Clicking other tabs does NOT change functionality yet

	TABLE ACTIONS (RIGHT):
	Components (from left to right):

	Search input

	Placeholder: "Search..."

	Used to filter table rows

	Minimal input style (Notion-like)

	Sort button

	Toggles ASC / DESC

	Visual indicator of current sort state

	Filter button

	Secondary button style

	On click: Opens FILTER PANEL below sub header

	Create Item button

	Primary button style

	Background: #007AFF

	Text: "Create item"

	CREATE ITEM DRAWER (MANDATORY):
	Behavior:

	Opens from the RIGHT side of the screen

	Overlayed drawer (does NOT replace page)

	Smooth slide-in animation

	Drawer content:

	Form generated dynamically from table columns (MCP)

	One input per column

	Label = column name

	Proper input type when possible

	Actions:

	Cancel button (secondary)

	Create button (primary)

	Rules:

	Drawer width fixed (e.g. 420px)

	Scroll inside drawer if content overflows

	Drawer closes on:

	Cancel

	Outside click

	Successful create

	FILTER PANEL (MANDATORY):
	Behavior:

	Appears BELOW the sub header

	Similar height and style as sub header

	Full-width panel

	Content:

	List of table columns (from MCP)

	Each column has filter controls (input / select depending on column type)

	Rules:

	Panel toggles via Filter button

	Panel closes on outside click

	Clean, Notion-like UI

	No heavy borders

	====================================
	TABLE UI RULES (CRITICAL)

	TABLE COMPONENT (ADVANCED — ERP/CRM LEVEL):

	The table MUST be a SMART, ENTERPRISE-GRADE table.

	Required features:

	Dynamic columns from MCP

	Dynamic rows from MCP

	Vertical scroll

	Horizontal scroll

	Sticky header

	Column sizing: fixed width columns with horizontal scrolling

	Column height: 32px

	COLUMN SIZING (FINAL):

	Each column:

	min-width: 220px

	max-width: 220px

	Do NOT use width: 100% for columns

	Columns must stay fixed at 220px and table scrolls horizontally

	Cell rules:

	Single-line text only

	white-space: nowrap

	overflow: hidden

	text-overflow: ellipsis
	Column width must NEVER auto-expand even if content is longer.

	BORDERS (UI REQUIREMENT):

	Every cell MUST have a visible border (light, Notion-like):
	border: 1px solid rgba(55, 53, 47, 0.12) or rgba(55, 53, 47, 0.16)

	Header cells and body cells must both have borders

	No missing borders anywhere

	PERFORMANCE (CRITICAL):

	The table MUST render cells in optimized mode:

	Default rendering for each cell is VIEW MODE (not an input)

	Only when user clicks a cell, that cell becomes EDIT MODE (input/dropdown)

	Only ONE active editing cell at a time (or minimal state)

	Avoid rendering inputs for all cells simultaneously
	If the solution renders input elements for every cell by default, the result is INVALID.

	RESIZABLE COLUMNS (MANDATORY):

	Table columns MUST be resizable by dragging a resize handle on the header

	Dragging must adjust the column width in pixels

	Respect min width 220px as the minimum

	Persist widths in React state

	Resizing must be smooth and not laggy

	Advanced UX features:

	Column hover highlight

	Row hover highlight

	Column visibility toggle

	Sorting (ASC / DESC)

	Search and filters integrated

	Empty state UI

	Loading skeleton (2–3 seconds)

	PAGINATION (MANDATORY):
	The table MUST support pagination.
	Requirements:

	Pagination controls at the bottom of the table

	Page size selector (e.g. 10 / 20 / 50)

	Current page indicator

	Next / Previous buttons

	Pagination must work with MCP data

	Pagination UI style:

	Minimal

	Notion-like

	Secondary button style

	====================================
	BACKEND RESPONSE STRUCTURE

	ALL backend responses follow ONE of the following shapes.

	MENUS LIST RESPONSE:
	Menu list API response:
	{
	"status": "OK",
	"description": "The request has succeeded",
	"data": {
	"menus": [
	{ "label": "Content", "icon": "folder.svg" }
	]
	}
	}
	FRONTEND RULES FOR MENUS:

	Menu items MUST be read from: response.data.data.menus
	Correct example: const menus = response?.data?.data?.menus ?? [];
	NEVER read menus from: response.data.menus, response.data.result, response.data.items

	TABLE LIST RESPONSE:
	Table data API response:
	{
	"status": "OK",
	"description": "The request has succeeded",
	"data": {
	"data": {
	"count": 3,
	"response": [ { "guid": "...", "discount": 88 } ]
	}
	}
	}
	FRONTEND RULES FOR TABLE DATA:

	Table rows MUST be read from: response.data.data.data.response

	Total count MUST be read from: response.data.data.data.count

	GENERAL FRONTEND RULES:

	NEVER assume a simplified response

	ALWAYS use optional chaining (?.)

	ALWAYS provide safe fallbacks (?? [])

	DO NOT refactor or normalize backend response

	UI must adapt to backend, not vice versa

	====================================
	TABLE MAPPING RULES

	Columns MUST be driven by MCP table details fields metadata (fields with type and attributes)

	Ignore technical fields like: id, guid, *_id, *_id_data

	Long text must use ellipsis

	CELL RENDERING BY FIELD TYPE (MVP TYPES — MANDATORY):
	Each table cell MUST render based on the field type coming from MCP schema.
	Default value MUST be taken from the row data returned by backend (response rows).

	Supported field types:

	NUMBER:

	Editable

	On click → inline edit with <input type="number" />

	Default value = row[field.slug]

	FLOAT:

	SAME AS NUMBER

	Editable

	On click → inline edit with <input type="number" step="any" />

	Default value = row[field.slug]

	TEXT:

	View-only (NOT editable)

	Render as plain text with ellipsis

	Default value = row[field.slug]

	SINGLE_LINE:

	Editable

	On click → inline edit with <input type="text" />

	Default value = row[field.slug]

	STATUS:

	Render as a clickable status pill / cell (view mode)

	On click: open a dropdown menu anchored UNDER the cell (not a native <select>)

	Dropdown MUST render options in categories: todo, in progress, complete

	Options MUST be read from field.attributes:

	todo options: field.attributes.todo.options

	progress options: field.attributes.progress.options

	complete options: field.attributes.complete.options
	Option rendering rules:

	Label priority: label_ru -> label_en -> value

	If missing/empty, fallback to value

	Use option.color for text/badge color

	Background = same color but more transparent (approx 14–18% opacity)

	If category options are missing, safely fall back to:
	[{ value: "todo" }, { value: "in_progress" }, { value: "complete" }]
	Dropdown behavior:

	Close on outside click

	Close on option select

	Escape closes (optional)

	Must not break table scrolling

	SCHEMA SOURCE (IMPORTANT):
	MCP provides both:

	table rows at response.data.data.data.response

	table fields metadata at response.data.data.data.fields (from table-details)
	You MUST use fields metadata to decide how to render cells.
	Do NOT infer type from JS typeof.
	If fields metadata is missing, fallback to TEXT view-only.

	CELL EDITING VISUAL RULES (CRITICAL):
	Editable cells MUST NOT visually look like form inputs.
	Rules:

	No visible input borders

	No default input background

	No focus ring

	No input padding that breaks table rhythm
	Behavior:

	Cell looks like plain text by default

	On hover: subtle background highlight (Notion-like)

	On focus/edit: input is visually identical to cell text (invisible input)
	Implementation hint (conceptual):

	border: none

	background: transparent

	outline: none

	inherit font, size, line-height

	====================================
	PROJECT STRUCTURE (MANDATORY)

	src/

	components/

	Sidebar.jsx

	Table.jsx

	Loader.jsx

	layouts/

	DashboardLayout.jsx

	pages/

	DashboardHome.jsx

	DynamicTablePage.jsx

	data/

	tables.js

	api/

	axios.js

	App.jsx

	main.jsx

	index.css

	====================================
	OUTPUT FORMAT (CRITICAL — MUST FOLLOW)

	Return the result as a PURE JAVASCRIPT OBJECT.

	ABSOLUTE RULES:

	DO NOT wrap the output in markdown or code blocks

	DO NOT stringify the whole object

	DO NOT escape JSON globally

	DO NOT return a single string

	Return a REAL object structure

	The response MUST START with "{" and END with "}".

	The root response MUST be:
	{
	project_name: "ucode-erp-admin-panel",
	files: [ { path: "...", content: "..." } ],
	env: {
	VITE_ADMIN_BASE_URL: "https://admin-api.ucode.run\",
	VITE_PROJECT_ID: "f1c4ae97-ee0f-4868-b4fc-1b26869ebc69",
	VITE_PARENT_ID: "c57eedc3-a954-4262-a0af-376c65b5a284",
	VITE_X_API_KEY: "P-wkLyW3aBURDx6oSwtlhk33WQn8Q3VhIc"
	}
	}

	ENV OUTPUT RULE (MANDATORY):

	In addition to generating .env.example file, you MUST return env object at root.

	The env object MUST include ALL variables used by the project in any .env file.

	The UI code MUST reference import.meta.env.VITE_* variables (not hardcoded constants).

	If you return the entire project as a single JSON string, or wrap it inside markdown code blocks, the result is INVALID.

	====================================
	QUALITY BAR

	This must look like a REAL commercial ERP / CRM admin panel:

	Premium feel

	Perfect spacing

	Clean typography

	Smooth animations

	Notion-level UI discipline

	You are generating FRONTEND UI code.
	The backend response format is ALREADY FIXED and MUST NOT be changed.

	Your task is to correctly READ data from backend responses and map them into UI components WITHOUT assumptions.

	FAILURE CONDITIONS:
	The solution is INVALID if:

	UI reads data from wrong response paths

	UI renders inputs for every cell by default (must be view mode first)

	Table details fields read from response.data.data (must be response.data.data.data)

	Missing cell borders

	Columns are not resizable

	The root output is not an object or misses env at root

	SUCCESS CONDITION:
	Frontend must correctly render:

	Empty state when response is empty

	Table with rows when response exists

	Pagination using count

	Menu using menus

	Typed cells and status dropdown from attributes

	Resizable columns

	ARCHITECTURE FREEZE:

	Do NOT move providers into Layout or Pages.
	Do NOT conditionally render providers.
	Do NOT change layout hierarchy.

	Only apply the specified fixes.

	Generate the full project now.
` +
	// Response json
	`CRITICAL OUTPUT INSTRUCTION:
You MUST respond with ONLY a valid JSON object, with NO markdown, NO code blocks, NO explanations.
The response must be PURE JSON starting with { and ending with }.

Your response will be parsed directly as JSON using json.Unmarshal.
If you include ANYTHING other than the JSON object, parsing will fail.

Example of CORRECT response:
{"project_name":"admin-panel","files":[{"path":"src/App.jsx","content":"import React from 'react';\n\nexport default function App() {\n  return <div>Hello</div>;\n}"}],"env":{"VITE_API_URL":"https://api.example.com"}}

Example of INCORRECT response (will cause parsing error):
` + "```json\n{\"project_name\":\"...\"}\n```" + `

RESPOND WITH ONLY THE JSON OBJECT NOW.`

func FrontendGenerateUserPrompt(request models.GeneratePromptMCP) string {
	// USER prompt template: dynamic; will be injected with the runtime values.
	userTpl := `User request:
- Description: "%s"
- Project ID: "%s"
- Main Menu Parent ID: "%s"
- X-API-KEY: "%s"
- Base URL: "%s"

Task:
1) Generate a complete production-ready frontend-only admin project (React 18 + Vite + TailwindCSS v2.2.19) as a single JSON object with fields:
   { "project_name": "<string>", "files": [ { "path": "<path>", "content": "<file contents>" }, ... ] }
   - File contents must be plain raw file text (use real newlines in JSON string values).
   - No markdown, no extra text outside that single JSON root.
2) Default to DARK MODE when the Description includes "dark", "erp", "admin", "dashboard", or "backoffice" (unless the user explicitly requests "light only").
3) Implement client-side routing using react-router-dom:
   - Include BrowserRouter and a Routes config with at least "/" (DashboardHome) and "/tables/:collection" (DynamicTablePage).
   - Sidebar menu item clicks must navigate using useNavigate to a path derived from the menu (e.g. '/tables/${table_slug}' for TABLE menus or '/menu/${id}').
   - Top header must display selected menu label via router state or URL params.
4) Implement runtime fetching of menus and table details using MCP tools 'get_menus' and 'get_table_details' when available.
   If MCP tools are not available, include exact axios calls that the generated code will use:
   - GET %%VITE_ADMIN_BASE_URL%%/v3/menus?parent_id=%s&project-id=%s
     Headers: { Authorization: "API-KEY", "X-API-KEY": "%s" }
   - POST %%VITE_ADMIN_BASE_URL%%/v1/table-details/:collection
     Body: { "data": {} }
     Headers: same as above
5) Ensure table components follow the documented layout rules:
   - <=6 columns => table-fixed
   - >6 columns => auto layout + min-w per column + horizontal scroll
   - thead th must be position: sticky; top: 0; z-index: 10; inside the scroll container
6) Include required components ElementLink and ElementText with the behaviors described in the system prompt.
7) If any runtime env is missing, still output full project and include README_HOW_TO_RUN.txt explaining where to inject PROJECT_ID, PARENT_ID, X_API_KEY, VITE_ADMIN_BASE_URL.
8) Return EXACTLY one JSON object and nothing else.

Now produce the project JSON immediately.`

	return fmt.Sprintf(userTpl,
		request.UserPrompt, // description
		request.ProjectId,
		config.MainMenuID,
		request.APIKey,
		request.BaseURL,
		// for axios example parameters (reused)
		config.MainMenuID,
		request.ProjectId,
		request.APIKey,
	)

}
