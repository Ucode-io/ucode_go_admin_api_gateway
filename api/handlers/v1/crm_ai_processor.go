package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ucode/ucode_go_api_gateway/api/handlers/ai/openai"
	"ucode/ucode_go_api_gateway/api/models"
)

const maxCRMAssistantIterations = 6

type crmAssistantResult struct {
	reply         string
	clientActions []models.CRMClientAction
	pendingAction *models.PendingAction
}

func (p *ChatProcessor) runCRMAssistantFlow(
	ctx context.Context,
	agent *openai.OpenAIAgent,
	req models.CRMAssistantRequest,
) (*crmAssistantResult, error) {
	// Common lead analytics only use conventional deal system fields. Run them
	// before the much heavier project-schema fetch so routine dashboard questions
	// stay fast even if a metadata service is temporarily slow.
	if result, handled, analyticsErr := p.runCommonCRMAnalytics(ctx, req, nil); handled {
		return result, analyticsErr
	}

	schema, err := p.getProjectSchemaCached(ctx, p.resourceEnvId)
	if err != nil {
		return nil, fmt.Errorf("load CRM schema: %w", err)
	}

	message := strings.TrimSpace(req.Message)
	if message == "" && len(req.Images) > 0 {
		message = "Bu screenshot asosida kartochkadagi maydonlar ko‘rinishini sozla."
	}

	history := crmAssistantHistory(req.History)
	schemaText := formatSchemaForSQL(schema)
	dataContext := ""

	for iteration := 0; iteration < maxCRMAssistantIterations; iteration++ {
		if err := p.Check(); err != nil {
			return nil, err
		}
		if iteration == maxCRMAssistantIterations-1 && dataContext != "" {
			dataContext += "\n\n[SYSTEM: Final step. Return action=answer using the fetched data. Do not request another query.]"
		}

		images := req.Images
		if iteration > 0 {
			images = nil
		}
		plan, err := agent.CRMQuery(ctx, models.CRMAssistantInput{
			Message:     message,
			SchemaText:  schemaText,
			DataContext: dataContext,
			Images:      images,
			History:     history,
			PageContext: req.PageContext,
		})
		if err != nil {
			return nil, err
		}

		switch plan.Action {
		case "answer", "schema":
			return &crmAssistantResult{reply: plan.Reply}, nil

		case "client_action":
			actions := normalizeCRMClientActions(plan.ClientActions, schema, req.PageContext.Table)
			if len(actions) == 0 {
				reply := strings.TrimSpace(plan.Reply)
				if reply == "" {
					reply = "Maydon nomlarini aniqlab bo‘lmadi. Qaysi maydonlarni ko‘rsatish yoki yashirish kerakligini aniqlashtiring."
				}
				return &crmAssistantResult{reply: reply}, nil
			}
			return &crmAssistantResult{reply: plan.Reply, clientActions: actions}, nil

		case "query":
			sqlType, validationErr := ValidateAndClassifySQL(plan.SQL)
			if validationErr != nil {
				dataContext = appendDataContext(
					dataContext,
					fmt.Sprintf("Rejected query %d", iteration+1),
					fmt.Sprintf("The gateway rejected that SQL: %v. Produce a corrected, single safe query using only the supplied schema.", validationErr),
				)
				continue
			}
			if IsMutation(sqlType) {
				pending, pendingErr := p.handleSQLMutation(ctx, &models.DatabaseActionRequest{
					Action:         "query",
					SQL:            plan.SQL,
					SQLParams:      plan.SQLParams,
					Reply:          plan.Reply,
					SuccessMessage: plan.SuccessMessage,
					CancelMessage:  plan.CancelMessage,
					ResourceEnvID:  p.resourceEnvId,
				}, sqlType)
				if pendingErr != nil {
					return nil, pendingErr
				}
				return &crmAssistantResult{
					reply:         pending.Description,
					pendingAction: pending.PendingAction,
				}, nil
			}

			result, queryErr := p.executeSQLQuery(ctx, EnsureSelectLimit(plan.SQL, defaultSelectLimit), plan.SQLParams, p.resourceEnvId)
			if queryErr != nil {
				dataContext = appendDataContext(
					dataContext,
					fmt.Sprintf("Failed query %d", iteration+1),
					fmt.Sprintf("The database rejected that query: %v. Correct the SQL using the exact schema and try again.", queryErr),
				)
				continue
			}
			encoded, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				return nil, fmt.Errorf("encode CRM query result: %w", marshalErr)
			}
			label := plan.QueryPlan
			if strings.TrimSpace(label) == "" {
				label = fmt.Sprintf("CRM query %d", iteration+1)
			}
			dataContext = appendDataContext(dataContext, label, string(encoded))

		default:
			return nil, fmt.Errorf("unsupported CRM assistant action %q", plan.Action)
		}
	}

	return nil, fmt.Errorf("CRM assistant exceeded %d iterations", maxCRMAssistantIterations)
}

func crmAssistantHistory(messages []models.CRMAssistantMessage) []models.ChatMessage {
	const maxHistory = 12
	if len(messages) > maxHistory {
		messages = messages[len(messages)-maxHistory:]
	}

	history := make([]models.ChatMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		content := strings.TrimSpace(message.Content)
		if (role != "user" && role != "assistant") || content == "" {
			continue
		}
		history = append(history, models.ChatMessage{
			Role: role,
			Content: []models.ContentBlock{{
				Type: "text",
				Text: content,
			}},
		})
	}
	return history
}

func normalizeCRMClientActions(
	actions []models.CRMClientAction,
	schema []models.TableSchema,
	fallbackTable string,
) []models.CRMClientAction {
	tables := make(map[string]map[string]string, len(schema))
	for _, table := range schema {
		aliases := make(map[string]string, len(table.Fields)*2)
		for _, field := range table.Fields {
			aliases[normalizeCRMFieldAlias(field.Slug)] = field.Slug
			if field.Label != "" {
				aliases[normalizeCRMFieldAlias(field.Label)] = field.Slug
			}
			for _, alias := range commonCRMFieldAliases(field.Slug) {
				aliases[normalizeCRMFieldAlias(alias)] = field.Slug
			}
		}
		for alias, field := range builtInCRMFieldAliases(table.Slug) {
			key := normalizeCRMFieldAlias(alias)
			if _, exists := aliases[key]; !exists {
				aliases[key] = field
			}
		}
		tables[table.Slug] = aliases
	}

	result := make([]models.CRMClientAction, 0, len(actions))
	for _, action := range actions {
		if action.Type != "set_card_field_visibility" {
			continue
		}
		table := strings.TrimSpace(action.Table)
		if table == "" {
			table = strings.TrimSpace(fallbackTable)
		}
		aliases, ok := tables[table]
		if !ok {
			continue
		}

		show := resolveCRMFields(action.ShowFields, aliases, nil)
		showSet := make(map[string]struct{}, len(show))
		for _, field := range show {
			showSet[field] = struct{}{}
		}
		hide := resolveCRMFields(action.HideFields, aliases, showSet)
		order := resolveCRMFields(action.FieldOrder, aliases, nil)
		if len(show) == 0 && len(hide) == 0 && len(order) == 0 {
			continue
		}

		result = append(result, models.CRMClientAction{
			Type:       action.Type,
			Table:      table,
			ShowFields: show,
			HideFields: hide,
			FieldOrder: order,
		})
	}
	return result
}

func builtInCRMFieldAliases(table string) map[string]string {
	if table != "deals" {
		return nil
	}
	return map[string]string{
		"contacts_id":  "contacts_id",
		"phone":        "contacts_id",
		"phone number": "contacts_id",
		"mobile":       "contacts_id",
		"telefon":      "contacts_id",
		"телефон":      "contacts_id",
		"contact":      "contacts_id",
		"kontakt":      "contacts_id",
	}
}

func commonCRMFieldAliases(slug string) []string {
	switch slug {
	case "amount":
		return []string{"budget", "byudjet", "summa", "бюджет", "сумма"}
	case "contacts_id":
		return []string{"phone", "phone number", "mobile", "telefon", "телефон", "contact", "kontakt"}
	default:
		return nil
	}
}

func resolveCRMFields(fields []string, aliases map[string]string, excluded map[string]struct{}) []string {
	seen := make(map[string]struct{}, len(fields))
	result := make([]string, 0, len(fields))
	for _, requested := range fields {
		field, ok := aliases[normalizeCRMFieldAlias(requested)]
		if !ok {
			continue
		}
		if _, skip := excluded[field]; skip {
			continue
		}
		if _, duplicate := seen[field]; duplicate {
			continue
		}
		seen[field] = struct{}{}
		result = append(result, field)
	}
	return result
}

func normalizeCRMFieldAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", ".", "")
	return replacer.Replace(value)
}
