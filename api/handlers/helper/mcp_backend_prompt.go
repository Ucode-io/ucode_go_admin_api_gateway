package helper

import (
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

var SystemPromptBackend = `You are connected to an MCP server named "ucode" and have access to tools via the mcp_toolset.
When an external action is required (get_dbml, create_menu, create_table, update_table, dbml_to_ucode, create_table_item), CALL the tools using the MCP tool calling mechanism.
DO NOT call the create_field tool for automated field creation. Instead, ALWAYS use update_table to add or modify fields and relations in bulk (send a fields array and relations array).
Only call create_field when a human explicitly requests a single manual field creation and after receiving explicit confirmation.
Do not invent results — call the appropriate tool with exact parameters. Use the tool names as documented.`

func BuildBackendPrompt(request models.BackendPromptRequest) (content, message string, err error) {
	switch request.Method {
	case "project":
		content = fmt.Sprintf(`You are creating a complete u-code project from scratch. Analyze the user's request and determine:
- Project Type (CRM, ERP, E-commerce, Admin Panel, etc.)
- Management Systems needed (ProjectManagement, Payroll, Inventory, Sales, etc.)
- Industry (IT, Healthcare, Finance, Retail, etc.)

Then follow these steps EXACTLY:

STEP 1: Retrieve current DBML schema (if any exists)
- Use get_dbml tool with:
  project-id = %s
  environment-id = %s
  x-api-key = %s

STEP 2: Analyze user request and generate comprehensive DBML schema
User Request: "%s"

Based on the user's request, determine:
1. What type of project/system they need (CRM, ERP, Admin Panel, etc.)
2. What functional areas/modules are required
3. What industry/domain this is for
4. What tables, relationships, and data structures are needed

Generate a complete DBML schema that:
- Matches the user's requirements EXACTLY
- Uses PostgreSQL syntax
- **EXCLUDES all existing tables** from current schema (if retrieved in STEP 1)
- Includes industry-specific functional areas based on your analysis
- DO NOT include Users or Roles tables (they already exist in u-code)
- Use proper Ref definitions: Ref fk_name: table1.column1 < table2.column2
- For status/type fields, use realistic Enum definitions: Enum "status" { "Active" "Inactive" "Pending" }
- Use camelCase or snake_case consistently
- No indexes, no quotes, no cascade options, no default values
- Use descriptive field names, NO comments
- Follow relational design principles
- If user mentions specific number of menus/tables (e.g., "10 menus"), create exactly that many

STEP 3: Create tables directly using create_table tool
- **Quantity Rule**: If the user did NOT specify a number of tables/menus, strictly create approximately **10 tables** suitable for the project logic.
- **Icon Rule**: You **MUST** provide an 'icon' parameter for EVERY table. Use a valid full URL from Iconify (e.g., "https://api.iconify.design/mdi:account.svg"). Pick an icon relevant to the table's function.
- **Menu ID Rule**: You **MUST** provide 'menu_id' for every table. Use the root parent_id = "%s".
- Call create_table with: label, slug (snake_case), icon, menu_id, x-api-key.
- IMPORTANT: Save the response from each create_table call - you need table_id and slug (collection) for next steps
- Example: create_table({label: "Customers", slug: "customers", icon: "https://api.iconify.design/mdi:account.svg", menu_id: "%s", x_api_key: "%s"})
- Do NOT create folders, all tables are created at root level using the provided menu_id.

STEP 4: Create fields for each table (IMPORTANT: use update_table, not create_field)
- IMPORTANT INSTRUCTION FOR FIELD CREATION:
  - Do NOT call the create_field tool under any circumstances for automated field creation.
  - Instead, always aggregate fields for a table and call update_table with a fields array and relations array.
  - If the plan would create several individual fields, combine them into a single update_table call per table (bulk create).
  - If update_table cannot perform a specific field creation (server returns error), STOP and report the error back (do not retry individual create_field).
  - Use create_field only if a human explicitly requests a single manual field creation and only after explicit confirmation.
- Then use update_table to add all fields and relations for each table.

- For each table created, add ALL necessary fields based on DBML schema
- Use update_table with: tableSlug (collection slug), xapikey, fields array, relations array
- Field types: SINGLE_LINE, TEXT, NUMBER, DATE, BOOLEAN, ENUM, etc.
- Add standard fields: id (auto), name/title (required), created_at, updated_at
- Add all fields from your DBML schema
- Example: update_table({tableSlug: "customers", xapikey: "%s", fields: [...], relations: [...]})

STEP 5: Create test data for each table (CRITICAL: smart relation handling)
- After ALL tables and fields are created, populate each table with approximately **5 test rows**
- Use create_table_item tool with: table_slug, data (object with field values), x_api_key
- Example: create_table_item({table_slug: "customers", data: {name: "John Doe", email: "john@example.com", status: "Active"}, x_api_key: "%s"})

CRITICAL RULES FOR TEST DATA CREATION:
1. **Dependency Order**: Create test data in the correct order based on relationships:
   - First create records in parent/independent tables (tables with NO foreign keys)
   - Then create records in child tables (tables WITH foreign keys to other tables)
   - If Table A references Table B, create test data in Table B FIRST, then Table A
   
2. **Save and Reuse IDs**: When creating test data:
   - SAVE the 'id' field from each created record response
   - Use these saved IDs when creating related records in child tables
   - Example: Create a customer record, save its id, then use that customer_id when creating orders
   
3. **Realistic Foreign Keys**: 
   - For foreign key fields, use ONLY the IDs from previously created parent records
   - DO NOT use random or null values for required foreign keys
   - Distribute references across multiple parent records (e.g., 5 orders distributed among 3 customers)
   
4. **Field Value Generation**:
   - SINGLE_LINE/TEXT: Generate realistic values based on field name and table context
   - NUMBER: Use realistic numbers (prices: 10-10000, quantities: 1-100, percentages: 0-100)
   - DATE: Use recent realistic dates (2025-2026 range)
   - BOOLEAN: Mix true/false values
   - ENUM: Use ONLY values defined in the enum, rotate through them
   - Status fields: Use enum values like "Active", "Pending", "Completed"
   - Email: Generate realistic emails (name@company.com)
   - Phone: Generate realistic phone numbers
   
5. **Required vs Optional**:
   - Always provide values for required fields
   - You may skip optional fields or provide realistic values
   
6. **Data Variety**: Create diverse test data
   - Use different names, values, statuses across the 5 rows
   - Make data realistic for the project type (CRM, ERP, etc.)
   
7. **Error Prevention**:
   - NEVER create orphan records (child records without valid parent references)
   - ALWAYS check if a field is a foreign key before assigning values
   - If create_table_item fails, report the error and STOP
   
8. **Creation Strategy**:
   - Analyze your DBML schema to determine table dependencies
   - Create a dependency graph: independent tables → dependent tables
   - Execute create_table_item calls in dependency order
   - For each table, create all 5 rows before moving to the next table
   
Example Execution Order:
If you have: Customers → Orders → OrderItems
1. Create 5 customers, save their IDs [c1, c2, c3, c4, c5]
2. Create 5 orders using customer IDs {customer_id: c1}, {customer_id: c2}, etc., save order IDs [o1, o2, o3, o4, o5]
3. Create 5 order items using order IDs {order_id: o1}, {order_id: o2}, etc.

STEP 6: Organize tables into menus
- Since no folders are created, all tables use parent_id = "%s"
- Provide view_fields JSON: { "table_slug": "primary_field_slug" }
- Example: { "customers": "name", "orders": "order_number" }

STEP 7: Execute DBML (optional - if you prefer bulk creation)
- If you want to use dbml_to_ucode for bulk creation, provide:
  - dbml: (full DBML string)
  - view_fields: (JSON object)
  - menus: (JSON object where keys are table names, values are table slugs)
  - x-api-key: %s

CRITICAL RULES:
1. All tables are created at root level: parent_id = "%s"
2. Save table_id and collection (slug) from create_table responses
3. Use collection (slug) as the "collection" parameter in create_field (only for manual single-field requests)
4. If any tool call fails, STOP and report the error - do not retry
5. Create a COMPLETE, working project - not just partial structure
6. Analyze user request carefully - if they say "10 menus", create exactly 10 tables
7. Remember: Empty folders are not used, all tables are at root level
8. **NEW**: After creating tables and fields, ALWAYS create test data (5 rows per table) following dependency order

Context:
project-id = %s
environment-id = %s
x-api-key = %s
main-menu-parent-id = %s

User Request: "%s"

Now analyze the user's request, determine project type/systems/industry, create the complete project structure directly as tables at root level, and populate each table with realistic test data.`,
			request.ProjectId, request.EnvironmentId, request.APIKey, request.UserPrompt,

			config.MainMenuID, config.MainMenuID, request.APIKey,

			request.APIKey,

			request.APIKey,

			config.MainMenuID,

			request.APIKey,

			config.MainMenuID,

			request.ProjectId, request.EnvironmentId, request.APIKey, config.MainMenuID,
			request.UserPrompt,
		)

		message = "Your project has been successfully created with test data."

	case "table":
		content = request.UserPrompt
		content += fmt.Sprintf(`

Context:
project-id = %s
environment-id = %s
x-api-key = %s

Rules:
- Use available MCP tools to actually perform the action (do not just reply with text).
- When creating a table:
  - label must match the requested name
  - slug should be a safe snake_case variant of the name
- After tool execution, return the API result (or clear error).`,
			request.ProjectId, request.EnvironmentId, request.APIKey)

		message = "The table has been successfully updated."

	default:
		message = fmt.Sprintf("method not implemented: %s", message)
		err = errors.New(message)
	}

	return content, message, err
}
