package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"ucode/ucode_go_api_gateway/api/handlers/ai/chat_prompts"
	"ucode/ucode_go_api_gateway/api/models"
)

func (a *OpenAIAgent) CRMQuery(ctx context.Context, in models.CRMAssistantInput) (*models.CRMAssistantPlan, error) {
	content := chat_prompts.BuildCRMAssistantMessage(in)
	messages := buildOpenAIMessages(in.History, buildContentParts(content, in.Images))
	cfg := a.conf.OpenAIAgents.DatabaseAssistant

	raw, usage, err := callTool(ctx, a.conf, cfg, chat_prompts.PromptCRMAssistant, messages, crmAssistantTool())
	a.tracker.RecordUsage(usage, cfg.Model, "CRM assistant")
	a.tracker.Deduct(int64(usage.InputTokens + usage.OutputTokens))
	if err != nil {
		return nil, fmt.Errorf("crm assistant: %w", err)
	}

	var plan models.CRMAssistantPlan
	if err = json.Unmarshal(raw, &plan); err != nil {
		return nil, fmt.Errorf("crm assistant: parse response: %w", err)
	}
	return &plan, nil
}

func crmAssistantTool() chatTool {
	stringArray := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	return chatTool{
		Type: "function",
		Function: functionDef{
			Name:        "respond_crm_assistant",
			Description: "Return the next safe CRM database step, final answer, or card field visibility action.",
			Parameters: map[string]any{
				"type":     "object",
				"required": []string{"action", "needs_more_data", "reply"},
				"properties": map[string]any{
					"action":          map[string]any{"type": "string", "enum": []string{"query", "answer", "client_action", "schema"}},
					"sql":             map[string]any{"type": "string"},
					"sql_params":      map[string]any{"type": "array", "items": map[string]any{}},
					"needs_more_data": map[string]any{"type": "boolean"},
					"query_plan":      map[string]any{"type": "string"},
					"reply":           map[string]any{"type": "string"},
					"success_message": map[string]any{"type": "string"},
					"cancel_message":  map[string]any{"type": "string"},
					"client_actions": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":     "object",
							"required": []string{"type", "table"},
							"properties": map[string]any{
								"type":        map[string]any{"type": "string", "enum": []string{"set_card_field_visibility"}},
								"table":       map[string]any{"type": "string"},
								"show_fields": stringArray,
								"hide_fields": stringArray,
								"field_order": stringArray,
							},
						},
					},
				},
			},
		},
	}
}
