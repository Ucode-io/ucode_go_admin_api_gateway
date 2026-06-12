package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/ai"
	"ucode/ucode_go_api_gateway/api/handlers/ai/anthropic"
	"ucode/ucode_go_api_gateway/api/handlers/ai/gemini"
	"ucode/ucode_go_api_gateway/api/handlers/ai/openai"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/spf13/cast"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	agentRunMaxTokens    = 4096
	agentStepTimeout     = 120 * time.Second
	defaultAgentMaxSteps = 8

	agentStatusSucceeded = "succeeded"
	agentStatusFailed    = "failed"
)

// newChatModel resolves the concrete provider for an agent's model id.
func (h *HandlerV1) newChatModel(model string) ai.ChatModel {
	switch {
	case strings.HasPrefix(model, "gpt"):
		return openai.NewOpenAIChatModel(h.baseConf)
	case strings.HasPrefix(model, "gemini"):
		return gemini.NewGeminiChatModel(h.baseConf, h.geminiKeyPool)
	default: // claude* and any unknown id fall back to Anthropic
		return anthropic.NewAnthropicChatModel(h.baseConf)
	}
}

// agentToolset is the per-agent tool surface plus the permission matrix used to
// authorize every tool call server-side (defense in depth on top of the schema).
type agentToolset struct {
	defs  []ai.ToolDef
	perms map[string]*nb.AgentPermission // table_slug → rule
}

func buildAgentToolset(perms []*nb.AgentPermission) agentToolset {
	permMap := make(map[string]*nb.AgentPermission, len(perms))
	var creatable, readable, listable, updatable, deletable []string

	for _, p := range perms {
		slug := p.GetTableSlug()
		if slug == "" {
			continue
		}
		permMap[slug] = p
		if p.GetCanCreate() {
			creatable = append(creatable, slug)
		}
		if p.GetCanRead() {
			readable = append(readable, slug)
		}
		if p.GetCanList() {
			listable = append(listable, slug)
		}
		if p.GetCanUpdate() {
			updatable = append(updatable, slug)
		}
		if p.GetCanDelete() {
			deletable = append(deletable, slug)
		}
	}

	var defs []ai.ToolDef
	if len(creatable) > 0 {
		defs = append(defs, itemCreateTool(sortStrings(creatable)))
	}
	if len(readable) > 0 {
		defs = append(defs, itemGetTool(sortStrings(readable)))
	}
	if len(listable) > 0 {
		defs = append(defs, itemListTool(sortStrings(listable)))
	}
	if len(updatable) > 0 {
		defs = append(defs, itemUpdateTool(sortStrings(updatable)))
	}
	if len(deletable) > 0 {
		defs = append(defs, itemDeleteTool(sortStrings(deletable)))
	}

	// web_fetch is always available: it lets the agent research up-to-date external
	// data (e.g. exchange rates) that does not live in the project's own tables.
	defs = append(defs, webFetchTool())

	return agentToolset{defs: defs, perms: permMap}
}

// ── tool definitions ──────────────────────────────────────────────────────────

func tableSlugSchema(tables []string) map[string]any {
	return map[string]any{
		"type":        "string",
		"enum":        tables,
		"description": "Slug of the target table.",
	}
}

func itemCreateTool(tables []string) ai.ToolDef {
	return ai.ToolDef{
		Name:        "item_create",
		Description: "Create a new record in a table. Returns the created record.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table_slug": tableSlugSchema(tables),
				"data": map[string]any{
					"type":                 "object",
					"description":          "Field values for the new record.",
					"additionalProperties": true,
				},
			},
			"required": []string{"table_slug", "data"},
		},
	}
}

func itemGetTool(tables []string) ai.ToolDef {
	return ai.ToolDef{
		Name:        "item_get",
		Description: "Fetch a single record by its guid. Returns the record or an error if not found.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table_slug": tableSlugSchema(tables),
				"guid":       map[string]any{"type": "string", "description": "The record's guid."},
			},
			"required": []string{"table_slug", "guid"},
		},
	}
}

func itemListTool(tables []string) ai.ToolDef {
	return ai.ToolDef{
		Name:        "item_list",
		Description: "List records in a table, optionally filtered. Returns matching records and a total count.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table_slug": tableSlugSchema(tables),
				"filters": map[string]any{
					"type":                 "object",
					"description":          "Optional equality filters keyed by field name.",
					"additionalProperties": true,
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of records to return (default 20, max 100).",
				},
			},
			"required": []string{"table_slug"},
		},
	}
}

func itemUpdateTool(tables []string) ai.ToolDef {
	return ai.ToolDef{
		Name:        "item_update",
		Description: "Update an existing record identified by guid with the provided fields. Returns the updated record.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table_slug": tableSlugSchema(tables),
				"guid":       map[string]any{"type": "string", "description": "The record's guid."},
				"data": map[string]any{
					"type":                 "object",
					"description":          "Field values to change.",
					"additionalProperties": true,
				},
			},
			"required": []string{"table_slug", "guid", "data"},
		},
	}
}

func itemDeleteTool(tables []string) ai.ToolDef {
	return ai.ToolDef{
		Name:        "item_delete",
		Description: "Delete a record identified by guid.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table_slug": tableSlugSchema(tables),
				"guid":       map[string]any{"type": "string", "description": "The record's guid."},
			},
			"required": []string{"table_slug", "guid"},
		},
	}
}

// ── engine ────────────────────────────────────────────────────────────────────

// runAgent executes one agent invocation: it opens an agent_run audit record,
// drives the native tool-use loop until the model produces a final answer (or the
// step budget is exhausted / an error occurs), then finalizes the run. The
// returned AgentRun carries the terminal status, output, steps and token usage.
func (h *HandlerV1) runAgent(ctx context.Context, service services.ServiceManagerI, resourceEnvId string, agent *nb.Agent, message string, runContext map[string]any) (*nb.AgentRun, error) {

	inputMap := map[string]any{"message": message}
	if len(runContext) > 0 {
		inputMap["context"] = runContext
	}
	inputStruct, _ := helperFunc.ConvertMapToStruct(inputMap)

	run, err := service.GoObjectBuilderService().Agent().CreateAgentRun(ctx, &nb.CreateAgentRunRequest{
		ResourceEnvId: resourceEnvId,
		AgentId:       agent.GetId(),
		ProjectId:     agent.GetProjectId(),
		Input:         inputStruct,
	})
	if err != nil {
		return nil, fmt.Errorf("create agent run: %w", err)
	}

	model := h.newChatModel(agent.GetModel())
	toolset := buildAgentToolset(agent.GetPermissions())
	system := buildAgentSystemPrompt(agent, toolset)

	maxSteps := int(agent.GetMaxSteps())
	if maxSteps <= 0 {
		maxSteps = defaultAgentMaxSteps
	}

	userText := message
	if len(runContext) > 0 {
		if ctxJSON, jErr := json.Marshal(runContext); jErr == nil {
			userText = message + "\n\n## Context provided by the application\n" + string(ctxJSON)
		}
	}
	messages := []ai.ConversationMessage{{Role: "user", Text: userText}}

	var (
		steps       []*nb.AgentRunStep
		totalTokens int32
		finalText   string
		lastText    string
		runErr      error
	)

	for step := 0; step < maxSteps; step++ {
		result, callErr := model.Complete(ctx, ai.CompletionRequest{
			Model:     agent.GetModel(),
			MaxTokens: agentRunMaxTokens,
			System:    system,
			Messages:  messages,
			Tools:     toolset.defs,
			Timeout:   agentStepTimeout,
		})
		if result != nil {
			totalTokens += int32(result.Usage.InputTokens + result.Usage.OutputTokens)
		}
		if callErr != nil {
			runErr = callErr
			break
		}
		if result.Text != "" {
			lastText = result.Text
		}

		messages = append(messages, ai.ConversationMessage{
			Role:      "assistant",
			Text:      result.Text,
			ToolCalls: result.ToolCalls,
		})

		if len(result.ToolCalls) == 0 {
			finalText = result.Text
			break
		}

		toolResults := make([]ai.ToolResult, 0, len(result.ToolCalls))
		for _, call := range result.ToolCalls {
			content, isErr := h.executeAgentTool(ctx, service, resourceEnvId, agent.GetProjectId(), toolset, call)
			toolResults = append(toolResults, ai.ToolResult{
				ToolCallID: call.ID,
				Content:    content,
				IsError:    isErr,
			})
			toolInput, _ := helperFunc.ConvertMapToStruct(call.Input)
			steps = append(steps, &nb.AgentRunStep{
				Index:      int32(len(steps)),
				ToolName:   call.Name,
				ToolInput:  toolInput,
				ToolResult: content,
				IsError:    isErr,
			})
		}

		messages = append(messages, ai.ConversationMessage{Role: "user", ToolResults: toolResults})
	}

	if finalText == "" {
		finalText = lastText
	}

	status, errMsg := agentStatusSucceeded, ""
	if runErr != nil {
		status, errMsg = agentStatusFailed, runErr.Error()
	}

	updated, err := service.GoObjectBuilderService().Agent().UpdateAgentRun(ctx, &nb.UpdateAgentRunRequest{
		ResourceEnvId: resourceEnvId,
		Id:            run.GetId(),
		Status:        status,
		Output:        finalText,
		Steps:         steps,
		TokensUsed:    totalTokens,
		Error:         errMsg,
	})
	if err != nil {
		return nil, fmt.Errorf("finalize agent run: %w", err)
	}

	return updated, nil
}

// executeAgentTool runs a single tool call against the project's items, enforcing
// the agent's permission rule for the targeted table. It returns a JSON string
// result (or an error message) and whether the call failed.
func (h *HandlerV1) executeAgentTool(ctx context.Context, service services.ServiceManagerI, resourceEnvId, projectId string, toolset agentToolset, call ai.ToolCall) (string, bool) {

	// web_fetch is not a table operation, so it bypasses the permission matrix.
	if call.Name == "web_fetch" {
		return executeWebFetch(ctx, call)
	}

	tableSlug := cast.ToString(call.Input["table_slug"])
	if tableSlug == "" {
		return "error: table_slug is required", true
	}
	perm := toolset.perms[tableSlug]
	if perm == nil {
		return fmt.Sprintf("error: agent has no permission for table %q", tableSlug), true
	}

	switch call.Name {
	case "item_create":
		if !perm.GetCanCreate() {
			return fmt.Sprintf("error: create not allowed on table %q", tableSlug), true
		}
		data := asStringMap(call.Input["data"])
		if len(data) == 0 {
			return "error: data is required and must be a non-empty object", true
		}
		data["company_service_project_id"] = projectId
		structData, err := helperFunc.ConvertMapToStruct(data)
		if err != nil {
			return "error: " + err.Error(), true
		}
		resp, err := service.GoObjectBuilderService().Items().Create(ctx, &nb.CommonMessage{
			TableSlug: tableSlug,
			ProjectId: resourceEnvId,
			Data:      structData,
		})
		if err != nil {
			return "error: " + err.Error(), true
		}
		return marshalRecord(resp.GetData(), map[string]any{"status": "created"}), false

	case "item_get":
		if !perm.GetCanRead() {
			return fmt.Sprintf("error: read not allowed on table %q", tableSlug), true
		}
		guid := cast.ToString(call.Input["guid"])
		if guid == "" {
			return "error: guid is required", true
		}
		item, found, err := h.lookupItem(ctx, service, resourceEnvId, tableSlug, guid)
		if err != nil {
			return "error: " + err.Error(), true
		}
		if !found {
			return fmt.Sprintf("error: no record with guid %q in table %q", guid, tableSlug), true
		}
		return marshalToolResult(item), false

	case "item_list":
		if !perm.GetCanList() {
			return fmt.Sprintf("error: list not allowed on table %q", tableSlug), true
		}
		filters := asStringMap(call.Input["filters"])
		limit := cast.ToInt(call.Input["limit"])
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		filters["limit"] = limit
		filterStruct, err := helperFunc.ConvertMapToStruct(filters)
		if err != nil {
			return "error: " + err.Error(), true
		}
		resp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
			TableSlug: tableSlug,
			ProjectId: resourceEnvId,
			Data:      filterStruct,
		})
		if err != nil {
			return "error: " + err.Error(), true
		}
		listData, _ := helperFunc.ConvertStructToMap(resp.GetData())
		return marshalToolResult(map[string]any{
			"items": extractItemsFromData(listData),
			"count": extractCountFromData(listData),
		}), false

	case "item_update":
		if !perm.GetCanUpdate() {
			return fmt.Sprintf("error: update not allowed on table %q", tableSlug), true
		}
		guid := cast.ToString(call.Input["guid"])
		if guid == "" {
			return "error: guid is required", true
		}
		data := asStringMap(call.Input["data"])
		if len(data) == 0 {
			return "error: data is required and must be a non-empty object", true
		}
		data["guid"] = guid
		data["id"] = guid
		data["company_service_project_id"] = projectId
		structData, err := helperFunc.ConvertMapToStruct(data)
		if err != nil {
			return "error: " + err.Error(), true
		}
		resp, err := service.GoObjectBuilderService().Items().Update(ctx, &nb.CommonMessage{
			TableSlug: tableSlug,
			ProjectId: resourceEnvId,
			Data:      structData,
		})
		if err != nil {
			return "error: " + err.Error(), true
		}
		return marshalRecord(resp.GetData(), map[string]any{"status": "updated", "guid": guid}), false

	case "item_delete":
		if !perm.GetCanDelete() {
			return fmt.Sprintf("error: delete not allowed on table %q", tableSlug), true
		}
		guid := cast.ToString(call.Input["guid"])
		if guid == "" {
			return "error: guid is required", true
		}
		structData, err := helperFunc.ConvertMapToStruct(map[string]any{
			"id":                         guid,
			"guid":                       guid,
			"company_service_project_id": projectId,
		})
		if err != nil {
			return "error: " + err.Error(), true
		}
		if _, err = service.GoObjectBuilderService().Items().Delete(ctx, &nb.CommonMessage{
			TableSlug: tableSlug,
			ProjectId: resourceEnvId,
			Data:      structData,
		}); err != nil {
			return "error: " + err.Error(), true
		}
		return marshalToolResult(map[string]any{"status": "deleted", "guid": guid}), false

	default:
		return fmt.Sprintf("error: unknown tool %q", call.Name), true
	}
}

// lookupItem fetches a single record by guid via the list endpoint.
func (h *HandlerV1) lookupItem(ctx context.Context, service services.ServiceManagerI, resourceEnvId, tableSlug, guid string) (map[string]any, bool, error) {

	filterStruct, err := helperFunc.ConvertMapToStruct(map[string]any{"guid": guid, "limit": 1})
	if err != nil {
		return nil, false, err
	}
	resp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
		TableSlug: tableSlug,
		ProjectId: resourceEnvId,
		Data:      filterStruct,
	})
	if err != nil {
		return nil, false, err
	}
	listData, _ := helperFunc.ConvertStructToMap(resp.GetData())
	items := extractItemsFromData(listData)
	if len(items) == 0 {
		return nil, false, nil
	}
	return items[0], true, nil
}

// ── prompt + small helpers ────────────────────────────────────────────────────

func buildAgentSystemPrompt(agent *nb.Agent, toolset agentToolset) string {
	var b strings.Builder
	b.WriteString(agent.GetInstruction())

	if len(toolset.perms) > 0 {
		b.WriteString("\n\n## Data access\n")
		b.WriteString("You can read and modify application data ONLY through the provided tools, ")
		b.WriteString("and ONLY on the tables listed below with the listed operations. ")
		b.WriteString("Never assume access to a table or operation that is not listed.\n\n")

		slugs := make([]string, 0, len(toolset.perms))
		for slug := range toolset.perms {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
		for _, slug := range slugs {
			b.WriteString(fmt.Sprintf("- %s: %s\n", slug, strings.Join(allowedOps(toolset.perms[slug]), ", ")))
		}

		b.WriteString("\nWhen you have fully addressed the request, reply with a final natural-language message and stop calling tools.")
	}

	b.WriteString("\n\n## External data\n")
	b.WriteString("Use the web_fetch tool to fetch public web URLs (e.g. JSON APIs or pages) when you need up-to-date external information such as exchange rates, prices, or reference data that is not stored in the application. For anything that lives in the application's own database, use the data tools above instead.")

	return b.String()
}

func allowedOps(p *nb.AgentPermission) []string {
	var ops []string
	if p.GetCanCreate() {
		ops = append(ops, "create")
	}
	if p.GetCanRead() {
		ops = append(ops, "read")
	}
	if p.GetCanList() {
		ops = append(ops, "list")
	}
	if p.GetCanUpdate() {
		ops = append(ops, "update")
	}
	if p.GetCanDelete() {
		ops = append(ops, "delete")
	}
	if len(ops) == 0 {
		ops = append(ops, "none")
	}
	return ops
}

// asStringMap coerces a tool-argument value into a string-keyed map.
func asStringMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func marshalToolResult(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// marshalRecord returns the record's data as JSON, or a fallback map when the
// service returned no body.
func marshalRecord(data *structpb.Struct, fallback map[string]any) string {
	if m, err := helperFunc.ConvertStructToMap(data); err == nil && len(m) > 0 {
		return marshalToolResult(m)
	}
	return marshalToolResult(fallback)
}

func sortStrings(in []string) []string {
	sort.Strings(in)
	return in
}
