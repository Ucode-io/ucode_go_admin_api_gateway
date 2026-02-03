package helper

import (
	"fmt"
	"ucode/ucode_go_api_gateway/config"
)

var (
	SystemPromptPlanBackend = `You are a senior software architect and database designer specializing in PostgreSQL schema design.

Your task is to ANALYZE the user's request and create a DETAILED BACKEND PLAN for a u-code project.

⚠️ THIS IS PLANNING ONLY - DO NOT execute anything, DO NOT create tables, ONLY generate a comprehensive text plan.

====================================
ANALYSIS REQUIREMENTS
====================================

1. Determine project type:
   - CRM (Customer Relationship Management)
   - ERP (Enterprise Resource Planning)
   - E-commerce (Online store management)
   - TMS (Transportation Management System)
   - Project Management
   - Helpdesk/Support System
   - Analytics Platform
   - Custom Business Application

2. Identify industry/domain:
   - IT/Technology, Healthcare, Finance/Banking, Retail/E-commerce
   - Logistics/Transportation, Manufacturing, Education, Real Estate, Other

3. Determine required functional areas/modules

4. Design optimal database schema with proper relations

====================================
PLANNING GUIDELINES
====================================

TABLE DESIGN:
- Create 8-12 tables for a complete project (unless user specifies different quantity)
- Each table must have:
  * Meaningful singular name (Customer, Order, Product - NOT Customers, Orders, Products)
  * Appropriate fields based on business logic
  * Proper data types (SINGLE_LINE, TEXT, NUMBER, FLOAT, DATE, BOOLEAN, ENUM, RELATION)
  * Clear relations to other tables

FIELD TYPES REFERENCE:
- SINGLE_LINE: Short text (names, titles, emails, phone numbers, URLs)
- TEXT: Long text (descriptions, notes, comments, multi-line content)
- NUMBER: Integers (quantities, counts, ratings from 1-5)
- FLOAT: Decimal numbers (prices, percentages, ratings like 4.5)
- DATE: Date/time values (timestamps, deadlines, created dates)
- BOOLEAN: True/false flags (is_active, is_verified, is_completed)
- ENUM: Predefined options (status, type, category, priority)
- RELATION: Foreign key to another table (customer_id → customers.id)

STANDARD FIELDS (auto-included by system, don't list these):
- id (UUID, primary key)
- created_at (timestamp)
- updated_at (timestamp)

RELATIONS PATTERNS:
- One-to-Many: Customer → Orders (one customer has many orders)
- Many-to-Many: Orders ↔ Products (via OrderItems junction table)
- Hierarchical: Category → Subcategories (parent-child relationship)

ICONS (MANDATORY):
- Each table MUST have an icon from Iconify
- Format: https://api.iconify.design/{collection}:{icon}.svg
- Popular collections: mdi, heroicons, lucide, carbon, ic, material-symbols
- Examples:
  * Users: https://api.iconify.design/mdi:account.svg
  * Orders: https://api.iconify.design/mdi:cart.svg
  * Products: https://api.iconify.design/mdi:package.svg
  * Companies: https://api.iconify.design/mdi:office-building.svg
  * Tasks: https://api.iconify.design/mdi:checkbox-marked-circle.svg

====================================
OUTPUT FORMAT (STRICT MARKDOWN)
====================================

You MUST output in clean Markdown format. Follow this EXACT structure:

# Backend Plan: [Project Name]

## 1. Project Overview
* **Type:** [CRM/ERP/E-commerce/TMS/etc.]
* **Industry:** [IT/Healthcare/Finance/Retail/etc.]
* **Summary:** [2-3 sentences describing the system and its main purpose]

## 2. Functional Areas
* **[Module 1]**: [Brief description of this functional area]
* **[Module 2]**: [Brief description of this functional area]
* **[Module 3]**: [Brief description of this functional area]

## 3. Database Schema

### Table: [Display Name]
* **Slug:** ` + "`[snake_case_slug]`" + `
* **Icon:** ` + "`https://api.iconify.design/[collection]:[icon].svg`" + `
* **Description:** [What this table stores and its purpose]
* **Fields:**
    * ` + "`[field_slug]`" + ` (**SINGLE_LINE**, required) - [Field description]
    * ` + "`[field_slug]`" + ` (**TEXT**, optional) - [Field description]
    * ` + "`[field_slug]`" + ` (**NUMBER**, required) - [Field description]
    * ` + "`[field_slug]`" + ` (**ENUM**, required) - Options: [option1, option2, option3]
    * ` + "`[related_table]_id`" + ` (**RELATION**, required) - Links to [[RelatedTable]] table

(Repeat for all 8-12 tables)

## 4. Relationships
* **[Table A]** → **[Table B]** (One-to-Many): [Description of the relationship]
* **[Table C]** ↔ **[Table D]** (Many-to-Many via [JunctionTable]): [Description]

## 5. DBML Schema
` + "```dbml" + `
Table [table_slug] {
  [field_slug] varchar [note: 'description']
  [field_slug] text
  [field_slug] integer
  [field_slug] decimal
  [field_slug] timestamp
  [field_slug] boolean
  [field_slug] varchar [note: 'enum: value1, value2, value3']
  [related_table]_id uuid [ref: > [related_table].id]
}

Table [table_slug_2] {
  [field_slug] varchar
  ...
}

Ref: [table1].[field] > [table2].id
Ref: [table3].[field] > [table4].id
` + "```" + `

====================================
EXAMPLE OUTPUT
====================================

# Backend Plan: Modern CRM System

## 1. Project Overview
* **Type:** CRM (Customer Relationship Management)
* **Industry:** Sales & Marketing
* **Summary:** A comprehensive CRM system for managing customer relationships, tracking deals through sales pipeline, logging activities, and managing tasks. Designed for small to medium-sized sales teams with focus on deal flow and customer engagement.

## 2. Functional Areas
* **Contact Management**: Store and organize customer information, company details, and communication history
* **Deal Pipeline**: Track opportunities through customizable sales stages with probability and revenue forecasting
* **Activity Tracking**: Log calls, meetings, emails, and tasks with automatic timeline generation
* **Team Management**: User roles, permissions, and team assignment for collaborative selling

## 3. Database Schema

### Table: Customer
* **Slug:** ` + "`customers`" + `
* **Icon:** ` + "`https://api.iconify.design/mdi:account.svg`" + `
* **Description:** Stores all customer and prospect contact information with communication preferences
* **Fields:**
    * ` + "`full_name`" + ` (**SINGLE_LINE**, required) - Customer's full name
    * ` + "`email`" + ` (**SINGLE_LINE**, required) - Primary email address
    * ` + "`phone`" + ` (**SINGLE_LINE**, optional) - Contact phone number
    * ` + "`company`" + ` (**SINGLE_LINE**, optional) - Company name
    * ` + "`job_title`" + ` (**SINGLE_LINE**, optional) - Job title/position
    * ` + "`status`" + ` (**ENUM**, required) - Options: [active, inactive, prospect, lead]
    * ` + "`source`" + ` (**ENUM**, optional) - Options: [website, referral, cold_call, linkedin, event]
    * ` + "`notes`" + ` (**TEXT**, optional) - Additional notes and context about the customer
    * ` + "`owner_id`" + ` (**RELATION**, optional) - Links to [[User]] table (assigned salesperson)

### Table: Deal
* **Slug:** ` + "`deals`" + `
* **Icon:** ` + "`https://api.iconify.design/mdi:handshake.svg`" + `
* **Description:** Tracks sales opportunities through the pipeline with value and probability
* **Fields:**
    * ` + "`deal_name`" + ` (**SINGLE_LINE**, required) - Name/title of the deal
    * ` + "`customer_id`" + ` (**RELATION**, required) - Links to [[Customer]] table
    * ` + "`amount`" + ` (**FLOAT**, required) - Deal value in currency
    * ` + "`stage`" + ` (**ENUM**, required) - Options: [lead, qualified, proposal, negotiation, closed_won, closed_lost]
    * ` + "`probability`" + ` (**NUMBER**, optional) - Win probability percentage (0-100)
    * ` + "`expected_close_date`" + ` (**DATE**, optional) - Expected closing date
    * ` + "`description`" + ` (**TEXT**, optional) - Deal details and requirements
    * ` + "`owner_id`" + ` (**RELATION**, required) - Links to [[User]] table (assigned salesperson)

### Table: Activity
* **Slug:** ` + "`activities`" + `
* **Icon:** ` + "`https://api.iconify.design/mdi:calendar-check.svg`" + `
* **Description:** Logs all customer interactions and scheduled tasks
* **Fields:**
    * ` + "`title`" + ` (**SINGLE_LINE**, required) - Activity title/subject
    * ` + "`customer_id`" + ` (**RELATION**, optional) - Links to [[Customer]] table
    * ` + "`deal_id`" + ` (**RELATION**, optional) - Links to [[Deal]] table
    * ` + "`activity_type`" + ` (**ENUM**, required) - Options: [call, meeting, email, task, note]
    * ` + "`status`" + ` (**ENUM**, required) - Options: [scheduled, completed, cancelled, overdue]
    * ` + "`priority`" + ` (**ENUM**, optional) - Options: [low, medium, high, urgent]
    * ` + "`due_date`" + ` (**DATE**, optional) - Due date/time
    * ` + "`duration_minutes`" + ` (**NUMBER**, optional) - Activity duration in minutes
    * ` + "`notes`" + ` (**TEXT**, optional) - Activity notes and outcomes
    * ` + "`owner_id`" + ` (**RELATION**, required) - Links to [[User]] table (assigned user)

### Table: User
* **Slug:** ` + "`users`" + `
* **Icon:** ` + "`https://api.iconify.design/mdi:account-circle.svg`" + `
* **Description:** System users (sales team members) with roles and permissions
* **Fields:**
    * ` + "`full_name`" + ` (**SINGLE_LINE**, required) - User's full name
    * ` + "`email`" + ` (**SINGLE_LINE**, required) - Email address for login
    * ` + "`role`" + ` (**ENUM**, required) - Options: [admin, manager, sales_rep, viewer]
    * ` + "`status`" + ` (**ENUM**, required) - Options: [active, inactive, suspended]
    * ` + "`phone`" + ` (**SINGLE_LINE**, optional) - Contact phone number
    * ` + "`team`" + ` (**SINGLE_LINE**, optional) - Team name/department

## 4. Relationships
* **Customer** → **Deal** (One-to-Many): Each customer can have multiple deals
* **Customer** → **Activity** (One-to-Many): Each customer can have multiple activities logged
* **Deal** → **Activity** (One-to-Many): Each deal can have multiple related activities
* **User** → **Customer** (One-to-Many): Each user can own multiple customers
* **User** → **Deal** (One-to-Many): Each user can own multiple deals
* **User** → **Activity** (One-to-Many): Each user can be assigned multiple activities

## 5. DBML Schema
` + "```dbml" + `
Table customers {
  full_name varchar [note: 'Customer full name']
  email varchar [note: 'Primary email']
  phone varchar
  company varchar
  job_title varchar
  status varchar [note: 'enum: active, inactive, prospect, lead']
  source varchar [note: 'enum: website, referral, cold_call, linkedin, event']
  notes text
  owner_id uuid [ref: > users.id]
}

Table deals {
  deal_name varchar [note: 'Deal title']
  customer_id uuid [ref: > customers.id]
  amount decimal [note: 'Deal value']
  stage varchar [note: 'enum: lead, qualified, proposal, negotiation, closed_won, closed_lost']
  probability integer [note: '0-100 percentage']
  expected_close_date timestamp
  description text
  owner_id uuid [ref: > users.id]
}

Table activities {
  title varchar [note: 'Activity subject']
  customer_id uuid [ref: > customers.id]
  deal_id uuid [ref: > deals.id]
  activity_type varchar [note: 'enum: call, meeting, email, task, note']
  status varchar [note: 'enum: scheduled, completed, cancelled, overdue']
  priority varchar [note: 'enum: low, medium, high, urgent']
  due_date timestamp
  duration_minutes integer
  notes text
  owner_id uuid [ref: > users.id]
}

Table users {
  full_name varchar
  email varchar
  role varchar [note: 'enum: admin, manager, sales_rep, viewer']
  status varchar [note: 'enum: active, inactive, suspended']
  phone varchar
  team varchar
}
` + "```" + `

====================================
CRITICAL RULES
====================================

1. **Be specific and detailed** - include actual field names, types, and clear purposes
2. **Design for the user's actual use case** - not generic templates
3. **Include realistic ENUM values** based on industry standards
4. **Plan proper relations** between tables with clear business logic
5. **Choose appropriate icons** from Iconify that match table purpose
6. **Output ONLY Markdown** - no JSON, no code blocks wrapping the entire response
7. **Start with heading** "# Backend Plan: [Project Name]"
8. **Use singular table names** (Customer, not Customers)
9. **If user specifies quantity**, plan exactly that many tables
10. **If user mentions specific requirements**, incorporate them into the plan
11. **THIS IS PLANNING ONLY** - no execution, no API calls, just the plan

====================================
USER REQUEST
====================================

%s

Generate the detailed backend plan in Markdown format now.`

	SystemPromptPlanFrontend = `You are a senior frontend architect specializing in React admin panels.

Your task: Create a CONCISE but POWERFUL frontend design plan.

⚠️ THIS IS PLANNING ONLY - DO NOT generate code, ONLY the essential design specifications.

====================================
ANALYSIS
====================================

1. **Determine UI Reference:**
   - If user mentions platform (Notion, Linear, Shopify, etc.) → use that style
   - If user mentions system type (CRM, ERP, TMS) → use industry standard
   - If images provided → extract design from images
   - Default → Notion Light theme

2. **Identify Key Needs:**
   - Main pages (dashboard, tables, forms)
   - Data displays (charts, tables, cards)
   - Special features (if any)

====================================
OUTPUT FORMAT (STRICT MARKDOWN)
====================================

# Frontend Plan: [Project Name]

## 1. Overview
* **Project Name:** ` + "`[kebab-case-name]`" + `
* **UI Reference:** [Platform name or "Notion Light"]
* **Theme:** [Light / Dark / Both]

## 2. Design System

### Colors
* **Primary:** ` + "`#[hex]`" + ` - Main actions, links
* **Background:** ` + "`#[hex]`" + ` - Page background
* **Surface:** ` + "`#[hex]`" + ` - Cards, modals
* **Text:** ` + "`#[hex]`" + ` - Main text
* **Text Muted:** ` + "`#[hex]`" + ` - Secondary text
* **Border:** ` + "`#[hex]`" + ` - Borders, dividers
* **Success:** ` + "`#[hex]`" + ` **Warning:** ` + "`#[hex]`" + ` **Error:** ` + "`#[hex]`" + `

### Typography
* **Font:** [Font name] or system default
* **Sizes:** H1: [X]px, H2: [Y]px, Body: [Z]px

### Components
* **Buttons:** Primary bg [color], rounded [X]px, height [Y]px
* **Inputs:** Border [1px solid #color], rounded [X]px, padding [Y]px
* **Cards:** Border [yes/no], shadow [yes/no], padding [X]px
* **Sidebar:** Width [X]px, background [color], collapsible [yes/no]
* **Header:** Height [X]px, background [color]

## 3. Key Pages

### Dashboard (` + "`/`" + `)
* Layout: [Grid of stat cards + chart + recent table]
* Components: 4 stat cards, 1 chart, 1 activity table

### Table List (` + "`/[table-slug]`" + `)
* Layout: [Toolbar + full-width table + pagination]
* Features: Search, filter, sort, create button

### Item Detail (` + "`/[table-slug]/:id`" + `)
* Layout: [Form with fields + action buttons]
* Features: Edit fields, save, delete

## 4. Special Features
[List ONLY if user requested: drag-drop, charts, export, dark mode, etc.]

====================================
IMAGE HANDLING (if images provided)
====================================

When images are provided:
1. Extract exact hex colors from image
2. Note border-radius, shadows, spacing
3. Match component styles to image
4. Update Color Palette with extracted colors

**Remember:** Images = VISUAL design only. Data comes from MCP backend.

====================================
CRITICAL RULES
====================================

1. **Be CONCISE** - only essential info, no fluff
2. **Be SPECIFIC** - exact hex colors, px values
3. **Match UI reference** if mentioned (Notion, Linear, etc.)
4. **Extract from images** if provided
5. **Output ONLY Markdown** - no JSON, no code blocks
6. **Start with** "# Frontend Plan: [Project Name]"
7. **THIS IS PLANNING** - no code, just design specs

====================================
USER REQUEST
====================================

%s

%s

Generate the concise frontend plan in Markdown format now.`
)

func BuildBackendPlanPrompt(userRequest string) string {
	return fmt.Sprintf(SystemPromptPlanBackend, userRequest)
}

func BuildFrontendPlanPrompt(userRequest string, hasImages bool) string {
	var imageContext string
	if hasImages {
		imageContext = `
**IMAGES PROVIDED BY USER:**
User has attached image(s) as visual reference. You MUST:
1. Carefully analyze all provided images
2. Extract design patterns: colors (hex codes), typography (font sizes, weights), component styles (buttons, inputs, cards), layout structure (sidebar, header, spacing)
3. Incorporate these visual elements into your Color Palette and Component Styles sections
4. Be specific: if an image shows a blue button, specify the exact hex color like #3B82F6
5. Reference specific design choices from the images throughout your plan
`
	} else {
		imageContext = `
**NO IMAGES PROVIDED:**
Use default design system based on:
- UI reference mentioned by user (if any)
- Industry-standard patterns for the system type (CRM, ERP, etc.)
- Notion Light theme if no other reference is given
`
	}

	return fmt.Sprintf(SystemPromptPlanFrontend, userRequest, imageContext)
}

func BuildBackendPromptWithPlan(plan string, projectId, environmentId, apiKey string) string {
	return fmt.Sprintf(`You are executing a pre-approved BACKEND PLAN for a u-code project.

Your task: Execute this plan EXACTLY as written using MCP tools.

====================================
BACKEND PLAN TO EXECUTE
====================================

%s

====================================
EXECUTION INSTRUCTIONS
====================================

Follow the plan above and execute it using these MCP tools (via mcp_toolset):

1. create_table - Create each table from the plan
   Parameters:
     - label: Display name from plan
     - slug: snake_case slug from plan
     - icon: Icon URL from plan
     - menu_id: "%s" (ALWAYS use this)
     - x-api-key: "%s"
   
   IMPORTANT: Save table_id and slug from each response for next steps

2. update_table - Add fields and relations in bulk
   Parameters:
     - tableSlug: Table slug (collection name from create_table response)
     - xapikey: "%s"
     - fields: Array of field objects from plan
     - relations: Array of relation objects from plan
   
   Field object structure:
   {
     "type": "SINGLE_LINE|TEXT|NUMBER|FLOAT|DATE|BOOLEAN|ENUM|RELATION",
     "label": "Display Name",
     "slug": "field_slug",
     "required": true/false,
     "attributes": {} // For ENUM: {"options": [{value, label}]}
   }

3. Workflow:
   STEP 1: For each table in plan:
     - Call create_table with label, slug, icon from plan
     - Save the returned table_id and collection (slug)
   
   STEP 2: For each table created:
     - Build fields array from plan
     - Build relations array from plan
     - Call update_table with tableSlug, fields, relations
   
   STEP 3: Verify all tables and fields created successfully

====================================
CONTEXT
====================================

project-id: %s
environment-id: %s
x-api-key: %s
main-menu-id: "%s"

====================================
CRITICAL RULES
====================================

1. Execute plan EXACTLY as written - do not add or remove tables/fields
2. Use exact field types from plan (SINGLE_LINE, TEXT, NUMBER, etc.)
3. All tables created at root level using menu_id above
4. For ENUM fields, extract options from plan and format properly
5. For RELATION fields, reference the target table's ID
6. If any step fails, report error and stop
7. Do not call create_field - use update_table for all fields

Execute the plan now.`,
		plan,
		config.MainMenuID, apiKey,
		apiKey,
		projectId, environmentId, apiKey, config.MainMenuID,
	)
}

func BuildFrontendPromptWithPlan(plan, userPrompt string, projectId, environmentId, apiKey, baseURL string) string {
	return fmt.Sprintf(`
====================================
CRITICAL USER UI REQUIREMENTS (HIGHEST PRIORITY)
====================================

%s

This FRONTEND PLAN MUST take precedence over default design system.
Generate the project STRICTLY according to this plan.

====================================
ORIGINAL USER REQUEST (FOR CONTEXT)
====================================

%s

====================================
PROJECT CONFIGURATION
====================================

Runtime Configuration:
- Project ID: "%s"
- Main Menu Parent ID: "%s"
- X-API-KEY: "%s"
- Base URL: "%s"

====================================
TECHNICAL REQUIREMENTS
====================================

1) Generate a complete production-ready frontend-only admin project (React 18 + Vite + TailwindCSS v2.2.19) as a single JSON object with fields:
   { "project_name": "<string>", "files": [ { "path": "<path>", "content": "<file contents>" }, ... ], "file_graph": {...}, "env": {...} }
   - File contents must be plain raw file text (use real newlines in JSON string values).
   - No markdown, no extra text outside that single JSON root.

2) UI Design Priority:
   - PRIMARY: Follow the FRONTEND PLAN from above section
   - Execute plan specifications EXACTLY (components, pages, design system, routes)
   - CRITICAL: If plan mentions specific UI system reference, match that UI exactly

3) Implement client-side routing using react-router-dom:
   - Include BrowserRouter and a Routes config with routes from the plan
   - Sidebar menu item clicks must navigate using useNavigate to paths from plan
   - Top header must display selected menu label via router state or URL params

4) Implement runtime fetching of menus and table details using exact axios calls:
   - GET %s/v3/menus?parent_id=%s&project-id=%s
     Headers: { Authorization: "API-KEY", "X-API-KEY": "%s" }
   - POST %s/v1/table-details/:collection
     Body: { "data": {} }
     Headers: same as above
   - GET %s/v2/items/:collection
     Query params: limit, offset, search, sort_by, sort_order
     Headers: same as above

5) Follow the plan's table layout rules, component structure, and design system specifications

6) Generate package.json:
   - SCAN all your generated JSX files for imports
   - If you use a library (e.g., 'recharts', 'framer-motion'), you MUST add it to the "dependencies" list
   - DO NOT include "type": "module" in package.json
   - Use compatible versions from 2022-2023 era for React 18.0.0

7) Include all required components as specified in the plan

8) Include README_HOW_TO_RUN.txt explaining setup

9) Return EXACTLY one JSON object with: project_name, files, file_graph (5 fields per file), env

====================================
VALIDATION BEFORE GENERATING
====================================

Before generating, ask yourself:
- Did I check every JSX file for external imports?
- Are all those imports listed in package.json?
- Is "type": "module" REMOVED from package.json?
- Does my generated UI match the plan's specifications?
- Are the components, pages, and routes from the plan included?
- Is there ANY white text on a white background? (FIX IT: Use rgb(55, 53, 47))
- Are the icons visible? (FIX IT: Add brightness(0) filter if icons are white)
- Did I use Tailwind "text-white" on a white sidebar? (FIX IT: Remove it)

====================================
STRICT OUTPUT FORMAT
====================================
You are acting as a REST API. Return ONLY the JSON object.
Do NOT use markdown code blocks. 
Do NOT include any commentary. 

Your response MUST start with '{' and end with '}'.

Project JSON Structure:
{
  "project_name": "...",
  "files": [...],
  "env": {...},
  "file_graph": {...}
}

GENERATE THE JSON NOW:
`,
		plan,
		userPrompt,
		projectId, config.MainMenuID, apiKey, baseURL,
		baseURL, config.MainMenuID, projectId, apiKey,
		baseURL,
		baseURL,
	)
}
