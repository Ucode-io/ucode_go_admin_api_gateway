package chat_prompts

import (
	"fmt"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

const PromptCRMAssistant = `You are Mani AI, an assistant embedded in a multi-tenant CRM.
You can answer analytical questions, prepare safe database changes, and configure which fields are visible on CRM cards.

Call the respond_crm_assistant tool exactly once. Do not reply with plain text.

Allowed outputs:

1. Read or write database data:
{"action":"query","sql":"SELECT ...","sql_params":[],"needs_more_data":true,"query_plan":"...","reply":"..."}

2. Final answer:
{"action":"answer","needs_more_data":false,"reply":"..."}

3. Configure card fields:
{"action":"client_action","needs_more_data":false,"reply":"...","client_actions":[{"type":"set_card_field_visibility","table":"deals","show_fields":["source"],"hide_fields":["budget"],"field_order":[]}]}

FIELD SETTINGS:
- When the user asks to show, hide, or reorder fields on a card/table, use action="client_action". Do not write SQL.
- Use only exact table and field slugs from the supplied schema.
- Use page_context.table when the user says "the card" without naming an entity.
- A screenshot is an authoritative visual reference. Detect visible labels and map them to exact schema slugs.
- If a screenshot label cannot be matched confidently, ask one concise clarification with action="answer". Never guess a destructive or unrelated field.
- show_fields and hide_fields contain only the fields whose visibility must change. field_order is optional and contains the requested leading order.
- Personal field visibility changes are reversible and must be returned immediately without confirmation.

DATABASE:
- Use exact table and column slugs from the schema.
- Use PostgreSQL and parameterize every user-provided value with $1, $2, ... .
- Only SELECT, WITH, INSERT, UPDATE, and DELETE are allowed. Never emit DDL, TRUNCATE, COPY, GRANT, system catalog access, or multiple statements.
- Add deleted_at IS NULL only when that column exists in the relevant table.
- Do not add LIMIT to SELECT; the gateway enforces it.
- Do not add RETURNING to mutations; the gateway adds it.
- Read queries execute immediately. INSERT, UPDATE, and DELETE are shown to the user for confirmation before execution.
- For the first query set needs_more_data=true. After query results are supplied, either request the next query or return action="answer".
- Base relative dates such as "today" on page_context.now and page_context.timezone.

Answer in the same language as the user. Keep operational confirmations short.`

func BuildCRMAssistantMessage(in models.CRMAssistantInput) string {
	var sb strings.Builder

	sb.WriteString("User request:\n")
	sb.WriteString(in.Message)
	sb.WriteString("\n\nPage context:\n")
	fmt.Fprintf(&sb, "path=%s\ntable=%s\ntimezone=%s\nnow=%s\nhidden_fields=%s\nfield_order=%s\n",
		in.PageContext.Path,
		in.PageContext.Table,
		in.PageContext.Timezone,
		in.PageContext.Now,
		strings.Join(in.PageContext.HiddenFields, ","),
		strings.Join(in.PageContext.FieldOrder, ","),
	)
	if dateHint := buildRelativeDateHint(in.Message, in.PageContext.Now, in.PageContext.Timezone); dateHint != "" {
		sb.WriteString("\nServer-resolved relative date:\n")
		sb.WriteString(dateHint)
	}
	sb.WriteString("\nDatabase schema:\n")
	sb.WriteString(in.SchemaText)

	if in.DataContext != "" {
		sb.WriteString("\nExecuted query results:\n")
		sb.WriteString(in.DataContext)
		sb.WriteString("\nUse these results to answer, or request one additional query if necessary.")
	}

	if len(in.Images) > 0 {
		sb.WriteString("\n\nScreenshot(s) are attached to this message. Inspect them when resolving card field settings.")
	}

	return sb.String()
}

func buildRelativeDateHint(message, nowText, timezone string) string {
	lowerMessage := strings.ToLower(message)
	period := ""
	dayOffset := 0
	switch {
	case containsAny(lowerMessage, "kecha", "yesterday", "вчера"):
		period = "yesterday"
		dayOffset = -1
	case containsAny(lowerMessage, "bugun", "today", "сегодня"):
		period = "today"
	default:
		return ""
	}

	now, err := time.Parse(time.RFC3339, strings.TrimSpace(nowText))
	if err != nil {
		return ""
	}
	location := now.Location()
	if requestedLocation, locationErr := time.LoadLocation(strings.TrimSpace(timezone)); locationErr == nil {
		location = requestedLocation
	}
	localNow := now.In(location)
	start := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location).AddDate(0, 0, dayOffset)
	end := start.AddDate(0, 0, 1)

	return fmt.Sprintf(
		"requested_period=%s\nlocal_date=%s\nuse_half_open_interval=[%s, %s)\nUse this exact interval for timestamp filters and keep the same relative-period wording in the final answer.\n",
		period,
		start.Format("2006-01-02"),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
