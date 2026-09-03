package chat_prompts

import (
	"fmt"
	"strconv"
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
- Understand requests by meaning, not by exact keywords. Treat informal conversational Uzbek, transliteration, omitted suffixes, ordinary spelling mistakes, and mixed Uzbek/Russian/English as normal input. Use recent conversation turns to resolve pronouns and short follow-ups.
- For every request asking for live CRM facts, records, counts, lists, statuses, field values, or analytics, action="answer" is forbidden until at least one relevant database query result has been supplied. Never invent a number and never claim that records were not found without querying the CRM. If the meaning is genuinely ambiguous, ask one concise clarification question instead.
- Base relative dates such as "today" on page_context.now and page_context.timezone.
- Understand casual Uzbek period wording and common typos: "kechigi/kechagi" means yesterday when discussing incoming leads; "bu/shu hafta", "o'tgan/otgan/o'tkan/otkan/avvalgi/oldingi hafta", "bu/shu oy", and "o'tgan/otgan/o'tkan/otkan/avvalgi/oldingi oy" are date ranges, not lead statuses. Russian and English equivalents have the same meaning.
- An explicit period in the current message always overrides earlier conversation history. Never answer a current week/month question using a previous yesterday/today interval.
- On the deals page, Uzbek "lid", English "lead", and Russian "лид" mean a row in the deals table unless the user explicitly names another table.
- "Lid keldi", "lead came/arrived", and "лид пришёл" mean deals created during that period: count by deals.created_at across every stage and pipeline. Do not use start_date, updated_at, due_date, or a stage filter unless the user explicitly asks for one.
- When the user asks for incoming lead phone numbers, use the same created_at period on deals and resolve the related contact through deals.contacts_id to contacts.guid, then read contacts.phone. Do not reinterpret "kechigi kegan lidlar" as overdue leads.
- Treat the server-resolved relative-date block as authoritative. Choose its local-wall interval for a timestamp without time zone column and its offset interval for a timestamp with time zone column. Do not change its day or timezone.
- Do not expose SQL, timestamps, timezone offsets, or technical query intervals in the final answer unless the user asks. Prefer a direct answer such as "Kecha 12 ta lid keldi."

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
	now, err := time.Parse(time.RFC3339, strings.TrimSpace(nowText))
	if err != nil {
		return ""
	}
	location := now.Location()
	if requestedLocation, locationErr := time.LoadLocation(strings.TrimSpace(timezone)); locationErr == nil {
		location = requestedLocation
	}
	localNow := now.In(location)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	period := ""
	var start, end time.Time
	switch {
	case containsAny(lowerMessage, "o‘tgan hafta", "o'tgan hafta", "o’tgan hafta", "otgan hafta", "o‘tkan hafta", "o'tkan hafta", "o’tkan hafta", "otkan hafta", "utgan hafta", "avvalgi hafta", "oldingi hafta", "last week", "previous week", "прошлой недел", "прошлую недел", "предыдущей недел"):
		period = "last_week"
		start = startOfPromptWeek(today).AddDate(0, 0, -7)
		end = start.AddDate(0, 0, 7)
	case containsAny(lowerMessage, "bu hafta", "shu hafta", "ushbu hafta", "this week", "current week", "этой недел", "эту недел", "текущей недел"):
		period = "this_week"
		start = startOfPromptWeek(today)
		end = start.AddDate(0, 0, 7)
	case containsAny(lowerMessage, "o‘tgan oy", "o'tgan oy", "o’tgan oy", "otgan oy", "o‘tkan oy", "o'tkan oy", "o’tkan oy", "otkan oy", "utgan oy", "avvalgi oy", "oldingi oy", "last month", "previous month", "прошлом месяц", "предыдущем месяц"):
		period = "last_month"
		end = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, location)
		start = end.AddDate(0, -1, 0)
	case containsAny(lowerMessage, "bu oy", "shu oy", "ushbu oy", "this month", "current month", "этом месяц", "текущем месяц"):
		period = "this_month"
		start = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 1, 0)
	case containsAny(lowerMessage, "kecha", "kechagi", "kechigi", "yesterday", "вчера"):
		period = "yesterday"
		start = today.AddDate(0, 0, -1)
		end = today
	case containsAny(lowerMessage, "bugun", "today", "сегодня"):
		period = "today"
		start = today
		end = today.AddDate(0, 0, 1)
	default:
		month, found := findPromptMonth(lowerMessage)
		if !found {
			return ""
		}
		year := findPromptYear(lowerMessage)
		if year == 0 {
			year = localNow.Year()
			if month > localNow.Month() {
				year--
			}
		}
		period = "named_month"
		start = time.Date(year, month, 1, 0, 0, 0, 0, location)
		end = start.AddDate(0, 1, 0)
	}

	return fmt.Sprintf(
		"requested_period=%s\nlocal_date=%s\nlocal_wall_interval=[%s, %s)\noffset_interval=[%s, %s)\nFor timestamp without time zone use local_wall_interval. For timestamp with time zone use offset_interval. Keep the same relative-period wording in the final answer without printing these technical intervals.\n",
		period,
		start.Format("2006-01-02"),
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)
}

func startOfPromptWeek(day time.Time) time.Time {
	daysSinceMonday := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -daysSinceMonday)
}

func findPromptMonth(text string) (time.Month, bool) {
	aliases := []struct {
		month    time.Month
		prefixes []string
	}{
		{time.January, []string{"yanvar", "january", "январ"}},
		{time.February, []string{"fevral", "february", "феврал"}},
		{time.March, []string{"mart", "march", "март"}},
		{time.April, []string{"aprel", "april", "апрел"}},
		{time.May, []string{"may", "май"}},
		{time.June, []string{"iyun", "june", "июн"}},
		{time.July, []string{"iyul", "july", "июл"}},
		{time.August, []string{"avgust", "august", "август"}},
		{time.September, []string{"sentabr", "september", "сентябр"}},
		{time.October, []string{"oktabr", "october", "октябр"}},
		{time.November, []string{"noyabr", "november", "ноябр"}},
		{time.December, []string{"dekabr", "december", "декабр"}},
	}
	words := strings.Fields(strings.NewReplacer(
		".", " ", ",", " ", "?", " ", "!", " ", ":", " ", ";", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ",
	).Replace(strings.ToLower(text)))
	for _, word := range words {
		for _, candidate := range aliases {
			for _, prefix := range candidate.prefixes {
				if strings.HasPrefix(word, prefix) {
					if prefix == "may" && word != "may" && !strings.HasPrefix(word, "mayda") && !strings.HasPrefix(word, "mayning") {
						continue
					}
					return candidate.month, true
				}
			}
		}
	}
	return 0, false
}

func findPromptYear(text string) int {
	for index := 0; index+4 <= len(text); index++ {
		year, err := strconv.Atoi(text[index : index+4])
		if err == nil && year >= 2000 && year <= 2100 {
			return year
		}
	}
	return 0
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
