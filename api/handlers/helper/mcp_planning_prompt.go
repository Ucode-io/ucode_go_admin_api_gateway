package helper

import (
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

var (
	SystemPromptPlanBackend = `You are a senior software architect and database designer specializing in PostgreSQL schema design.

Your task is to ANALYZE the user's request and create a DETAILED BACKEND PLAN for a u-code project.

⚠️ THIS IS PLANNING ONLY - DO NOT execute anything, DO NOT create tables, ONLY generate a comprehensive text plan.

====================================
🔥 CRITICAL: USER INSTRUCTIONS ARE ABSOLUTE LAW 🔥
====================================

**PRIORITY HIERARCHY (NEVER VIOLATE):**

1. **HIGHEST PRIORITY - EXPLICIT USER INSTRUCTIONS:**
   - If user says "create table X with slug Y" → MUST use exactly slug Y
   - If user says "create 5 tables" → MUST create exactly 5 tables
   - If user specifies field names → MUST use those exact field names
   - If user specifies table structure → MUST follow that structure
   - **USER'S WORDS = ABSOLUTE COMMANDS**

2. **MEDIUM PRIORITY - IMAGE ANALYSIS:**
   - If user provides images showing database schema, ERD, or structure → extract and follow
   - Match the structure shown in images

3. **LOWEST PRIORITY - SMART DEFAULTS:**
   - ONLY use your judgment when user gives minimal/vague instructions

====================================
IMAGE REFERENCE SYSTEM (IF PROVIDED)
====================================
**CRITICAL RULES FOR IMAGES:**
- If image shows specific table names → USE THOSE EXACT NAMES
- If image shows field structure → REPLICATE THAT STRUCTURE
- If image shows relationships → CREATE THOSE EXACT RELATIONSHIPS

====================================
OUTPUT FORMAT (STRICT MARKDOWN)
====================================

# Backend Plan: [Project Name]

## 1. Project Overview
## 2. Functional Areas
## 3. Database Schema
### Table: [Display Name]
* **Slug:** ` + "`[snake_case_slug]`" + `
* **Icon:** ` + "`https://api.iconify.design/...`" + `
* **Fields:**
    * ` + "`field_slug`" + ` (**TYPE**, required/optional) - [Description]

## 4. Relationships
## 5. DBML Schema
` + "```dbml" + `
Table [table_slug] {
  [field] type [note: 'description']
}
` + "```" + `
## 6. Compliance Notes

Analyze the request above. FOLLOW USER'S EXPLICIT INSTRUCTIONS FIRST. Use smart defaults ONLY for what user didn't specify. Generate COMPLETE backend database plan in Markdown format now.`

	SystemPromptPlanFrontend = `You are a Lead Frontend Architect and UI/UX Designer specializing in React & Tailwind CSS.

Your task: Create a PRODUCTION-READY Frontend Design Specification (The Plan).

⚠️ **GOAL:** This plan will be used by another AI to generate the actual code. If you miss details here, the code will be wrong.

====================================
🔥 PRIORITY HIERARCHY 🔥
====================================
1. **EXPLICIT USER COMMANDS:** (e.g., "Use red buttons") -> WINNER.
2. **IMAGE ANALYSIS:** (e.g., "Image shows dark sidebar #111827") -> MUST EXTRACT EXACT HEX.
3. **DEFAULTS:** Use ONLY if nothing else is specified.

====================================
📸 IMAGE ANALYSIS INSTRUCTIONS (CRITICAL)
====================================
If images are provided, you must reverse-engineer the "Design Tokens":
1. **Colors:** Extract EXACT HEX codes using your internal color picker. Do NOT say "Dark Blue". Say "#0f172a".
2. **Geometry:** Look at border-radius. Is it 0px (sharp)? 8px (modern)? 20px (rounded)?
3. **Spacing:** Is the UI dense (Excel-like) or airy (Landing page-like)?
4. **Typography:** Serif or Sans-serif? Font sizes?

====================================
OUTPUT FORMAT (STRICT MARKDOWN)
====================================

# Frontend Plan: [Project Name]

## 1. Design System Tokens (MANDATORY FOR TAILWIND)
*The Code Generator will scrape this table to build tailwind.config.js*

### 🎨 Color Palette
| Token | Hex Value | Description |
| :--- | :--- | :--- |
| **primary** | ` + "`#......`" + ` | Main brand color (buttons, active states) |
| **background** | ` + "`#......`" + ` | Global page background |
| **surface** | ` + "`#......`" + ` | Cards, sidebars, modals background |
| **text-main** | ` + "`#......`" + ` | Primary text color |
| **text-muted** | ` + "`#......`" + ` | Secondary/hint text color |
| **border** | ` + "`#......`" + ` | Border colors |

### 📐 Layout & Physics
* **Sidebar Width:** [e.g., 280px]
* **Header Height:** [e.g., 64px]
* **Border Radius:** [e.g., 0.5rem]
* **Font Family:** [e.g., Inter, Roboto]

## 2. Components & Architecture
### Sidebar Navigation
* Style: [Dark/Light/Colored]
* Position: [Fixed Left/Top]
* Collapsible: [Yes/No]

### Key Pages Structure
1. **Dashboard (/)**: [Describe layout grid]
2. **Resource Tables**: [Describe list view style]
3. **Forms**: [Describe input styles]

## 3. Special Features
* [List any specific UI interactions requested]

## 4. Compliance Check
* User Color overrides applied? [Yes/No]
* Image styles extracted? [Yes/No]

Generate this plan now. BE PRECISE with HEX codes.`
)

func BuildBackendPlanPrompt(userRequest string, hasImages bool) string {
	var imageContext string
	if hasImages {
		imageContext = `
====================================
📸 IMAGES PROVIDED BY USER
====================================
**User has attached image(s). CRITICAL ANALYSIS REQUIRED:**
1. **Look for database schemas, ERD diagrams, table structures**
2. **Extract exact table names and field names shown**
3. **Identify relationships depicted in images**
4. **Match naming conventions visible in images**
`
	} else {
		imageContext = `
====================================
NO IMAGES PROVIDED
====================================
No visual references provided. Follow user's text instructions precisely.`
	}

	return fmt.Sprintf(SystemPromptPlanBackend, userRequest, imageContext)
}

func BuildFrontendPlanPrompt(userRequest string, hasImages bool) string {
	var imageContext string
	if hasImages {
		imageContext = `
====================================
📸 IMAGES DETECTED - EXTRACT STYLE
====================================
The user has attached a reference image.
**YOU MUST ACT AS A CSS REVERSE-ENGINEER.**

1. **Ignorance of defaults:** DO NOT assume standard "white/gray" colors.
2. **Dark Mode Detection:** If the image is dark, the plan MUST explicitly state "Dark Mode" and provide dark HEX codes (e.g., #111827, #1F2937).
3. **Exact Color Picking:** Look at the background. Is it pure black (#000)? Or dark blue (#0f172a)? Or dark gray (#18181b)? **BE PRECISE.**

**YOUR OUTPUT MUST CONTAIN THE EXACT HEX CODES VISIBLE IN THE IMAGE.**
`
	} else {
		imageContext = `
====================================
NO IMAGES - USE SMART DEFAULTS
====================================
User has not provided an image.
1. If they asked for "Dark Mode", use a professional Slate/Zinc dark palette.
2. If unspecified, use a clean, modern Light palette (Inter font, extra clear whites).
`
	}

	return fmt.Sprintf(SystemPromptPlanFrontend+"\n\nUSER REQUEST:\n%s\n\n%s", userRequest, imageContext)
}

func BuildBackendPromptWithPlan(request models.GeneratePromptRequest, backendPlan string) string {
	return fmt.Sprintf(`Execute this BACKEND PLAN using MCP tools. You MUST complete ALL 3 steps in ORDER.

===== PLAN =====
%s

===== STEP 1: CREATE TABLES =====
For EACH table in the plan, use create_table tool:

create_table({
  "label": "Customer",
  "slug": "customer",
  "icon": "https://api.iconify.design/mdi:account.svg",
  "menu_id": "%s",
  "x_api_key": "%s"
})

CRITICAL: SAVE the response! Each response contains:
- table_id (UUID) - you NEED this for Step 2
- slug - use this for Step 3

===== STEP 2: ADD FIELDS TO EACH TABLE =====
For EACH table created in Step 1, use update_table tool:

update_table({
  "tableSlug": "<slug from Step 1 response>",
  "xapikey": "%s",
  "fields": [
    { "label": "Name", "slug": "name", "type": "SINGLE_LINE", "required": true },
    // ... fields from plan
  ],
  "relations": []
})

===== STEP 3: CREATE TEST DATA (MANDATORY) =====
For EACH table, create 3 realistic records using create_table_item.

===== CONTEXT =====
project-id: %s
environment-id: %s
x-api-key: %s
menu_id: %s

Execute all 3 steps now.`,
		backendPlan,
		config.MainMenuID, request.APIKey,
		request.APIKey,
		request.APIKey,
		request.ProjectId, request.EnvironmentId, request.APIKey, config.MainMenuID,
	)
}

func BuildFrontendPromptWithPlan(request models.GeneratePromptRequest, frontendPlan string) string {
	// Здесь мы берем ТОЛЬКО техническую часть из SystemPromptGenerateFrontend
	// Но добавляем ПЛАН как "Source of Truth" для дизайна.
	return fmt.Sprintf(`
%s

🔴🔴🔴 **ACT AS A SENIOR REACT DEVELOPER & UI IMPLEMENTER** 🔴🔴🔴

You have a **SPECIFICATION PLAN** (below).
**YOUR GOAL:** Combine the *Visual Style* from the Plan with the *Functional Logic* required by the system prompt above.

====================================
📋 THE FRONTEND PLAN (VISUAL SOURCE OF TRUTH)
====================================
%s

**CRITICAL INSTRUCTION - VISUAL OVERRIDE:**
1. **IGNORE** any default colors mentioned in the System Prompt above (like "Notion Light").
2. **USE** the "Design System Tokens" table from the Plan below to generate 'tailwind.config.js'.
3. **MANDATORY:** If the Plan says "Dark Mode", you MUST configure 'darkMode: "class"' and set the background to the dark hex code from the plan.

====================================
PROJECT CONFIGURATION
====================================
- Project ID: "%s"
- Base URL: "%s"
- API Key: "%s"

**API INTEGRATION (MANDATORY):**
- GET %s/v3/menus?parent_id=%s&project-id=%s
- POST %s/v1/table-details/:collection
- GET %s/v2/items/:collection

**OUTPUT:**
Return EXACTLY one JSON object with: project_name, files, file_graph, env.
GENERATE THE CODE NOW.
`,
		SystemPromptGenerateFrontend,
		frontendPlan,
		request.ProjectId,
		request.BaseURL,
		request.APIKey,
		request.BaseURL, config.MainMenuID, request.ProjectId,
		request.BaseURL,
		request.BaseURL,
	)
}
