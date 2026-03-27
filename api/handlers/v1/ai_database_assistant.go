package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/spf13/cast"
)

const (
	timeoutDatabaseAssistant = 120 * time.Second
	schemaCacheTTL           = 5 * time.Minute
	maxAgentIterations       = 4
)

// ============================================================================
// DATABASE FLOW — agentic loop
// ============================================================================

//   - Iteration 1: Claude sees schema → plans what to fetch (no data yet)
//   - Iterations 2-N: Claude sees accumulated results → fetches more or answers
//
// Mutations (create/update/delete) always break out immediately and return
// a PendingAction to the frontend for confirmation. No server-side RAM store.
func (p *ChatProcessor) runDatabaseFlow(ctx context.Context, clarified string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	resourceEnvId, err := p.resolveBuilderResourceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve builder resource ID: %w", err)
	}

	schema, err := p.getProjectSchemaCached(ctx, resourceEnvId)
	if err != nil {
		return nil, fmt.Errorf("schema fetch failed: %w", err)
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("schema marshal failed: %w", err)
	}

	var (
		dataContext string
		lastAction  *models.DatabaseActionRequest
	)

	for i := 0; i < maxAgentIterations; i++ {
		// On the last allowed iteration force a final answer — no more fetches
		if i == maxAgentIterations-1 && dataContext != "" {
			dataContext += "\n\n[SYSTEM: Final iteration. Provide the complete answer in 'reply' now. Do not set needs_more_data=true.]"
		}

		action, err := p.callDatabaseAssistant(ctx, clarified, string(schemaJSON), dataContext, chatHistory)
		if err != nil {
			return nil, fmt.Errorf("claude call failed (iter %d): %w", i, err)
		}
		action.ResourceEnvID = resourceEnvId
		lastAction = action

		// Schema-only or clarification — answer immediately, no DB access needed
		if action.TableSlug == "" || action.Action == "schema" {
			return &models.ParsedClaudeResponse{Description: action.Reply}, nil
		}

		// Mutations break out — frontend handles confirmation flow
		if action.Action == "create" || action.Action == "update" || action.Action == "delete" {
			return p.handleDatabaseMutation(ctx, action)
		}

		// Execute read / count / aggregate
		iterData, execErr := p.executeDBAction(ctx, action)
		if execErr != nil {
			return nil, fmt.Errorf("db execution failed (iter %d, action=%s): %w", i, action.Action, execErr)
		}

		iterJSON, err := json.Marshal(iterData)
		if err != nil {
			return nil, fmt.Errorf("marshal iteration data: %w", err)
		}

		label := fmt.Sprintf("Step %d (%s → %s)", i+1, action.Action, action.TableSlug)
		if action.QueryPlan != "" {
			label = fmt.Sprintf("Step %d — %s", i+1, action.QueryPlan)
		}
		dataContext = appendDataContext(dataContext, label, string(iterJSON))

		// Claude signals it has enough data to answer
		if !action.NeedsMoreData {
			break
		}
	}

	// If the last action already carries a complete reply, use it directly
	// (avoids one extra Claude round-trip)
	if lastAction != nil && !lastAction.NeedsMoreData && strings.TrimSpace(lastAction.Reply) != "" {
		reply := lastAction.Reply + buildDataContextHint(dataContext)
		return &models.ParsedClaudeResponse{Description: reply}, nil
	}

	// Final Claude call: generate polished answer from all accumulated data
	finalAction, err := p.callDatabaseAssistant(ctx, clarified, string(schemaJSON), dataContext, chatHistory)
	if err != nil {
		// Degrade gracefully — return whatever we have
		if lastAction != nil && lastAction.Reply != "" {
			return &models.ParsedClaudeResponse{Description: lastAction.Reply}, nil
		}
		return nil, fmt.Errorf("final claude call failed: %w", err)
	}

	return &models.ParsedClaudeResponse{Description: finalAction.Reply}, nil
}

func (p *ChatProcessor) executeDBAction(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {

	switch action.Action {
	case "read":
		return p.executeDatabaseRead(ctx, action)
	case "count":
		return p.executeDatabaseCount(ctx, action)
	case "aggregate":
		return p.executeDatabaseAggregate(ctx, action)
	default:
		return nil, fmt.Errorf("unknown read-type action: %s", action.Action)
	}
}

// appendDataContext accumulates labelled step results into a single string
// that grows with each agentic iteration and is passed back to Claude.
func appendDataContext(existing, label, jsonData string) string {
	if existing == "" {
		return fmt.Sprintf("=== %s ===\n%s", label, jsonData)
	}
	return fmt.Sprintf("%s\n\n=== %s ===\n%s", existing, label, jsonData)
}

// ============================================================================
// DATABASE EXECUTORS
// ============================================================================

func (p *ChatProcessor) executeDatabaseRead(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {
	if action.TableSlug == "" {
		return nil, fmt.Errorf("table_slug is required")
	}

	dataMap := make(map[string]any, len(action.Filters)+3)
	for k, v := range action.Filters {
		dataMap[k] = v
	}

	dataMap["limit"] = 50
	if action.Limit > 0 {
		dataMap["limit"] = action.Limit
	}
	if action.Offset > 0 {
		dataMap["offset"] = action.Offset
	}
	if action.OrderBy != "" {
		dataMap["order_by"] = action.OrderBy
	}

	structData, err := helperFunc.ConvertMapToStruct(dataMap)
	if err != nil {
		return nil, fmt.Errorf("convert filters: %w", err)
	}

	resp, err := p.service.GoObjectBuilderService().ObjectBuilder().GetList2(
		ctx, &nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: action.ResourceEnvID,
			Data:      structData,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("GetList2 failed: %w", err)
	}

	if resp.GetData() != nil {
		result, err := helperFunc.ConvertStructToMap(resp.GetData())
		if err != nil {
			return nil, fmt.Errorf("convert GetList2 response: %w", err)
		}
		return result, nil
	}

	return map[string]any{"response": []any{}, "count": 0}, nil
}

// executeDatabaseCount returns the server-side count without fetching rows.
// GetList2 with limit=1 still runs the full SELECT COUNT(*) internally.
func (p *ChatProcessor) executeDatabaseCount(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {
	data, err := p.executeDatabaseRead(ctx, &models.DatabaseActionRequest{
		TableSlug:     action.TableSlug,
		Filters:       action.Filters,
		Limit:         1,
		ResourceEnvID: action.ResourceEnvID,
	})
	if err != nil {
		return nil, fmt.Errorf("count read failed: %w", err)
	}

	return map[string]any{
		"count":      extractCountFromData(data),
		"table_slug": action.TableSlug,
		"filters":    action.Filters,
	}, nil
}

// executeDatabaseAggregate delegates SUM/AVG/MIN/MAX to PostgreSQL via GetListAggregation.
// Zero rows are transferred to Go — the DB returns a single aggregate value.
func (p *ChatProcessor) executeDatabaseAggregate(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {
	if action.AggregationField == "" {
		return nil, fmt.Errorf("aggregation_field is required")
	}

	aggFunc := strings.ToUpper(action.Aggregation)
	if aggFunc == "" {
		aggFunc = "SUM"
	}

	colExpr := fmt.Sprintf("%s(%s) as result", aggFunc, action.AggregationField)
	columns := []string{colExpr}
	var groupBy []string
	if action.GroupBy != "" {
		columns = append([]string{action.GroupBy}, columns...)
		groupBy = []string{action.GroupBy}
	}

	queryParams := map[string]any{
		"operation": "SELECT",
		"table":     fmt.Sprintf(`"%s"`, action.TableSlug),
		"columns":   columns,
		"where":     buildWhereClause(action.Filters),
	}
	if len(groupBy) > 0 {
		queryParams["group_by"] = groupBy
	}

	structData, err := helperFunc.ConvertMapToStruct(queryParams)
	if err != nil {
		return nil, fmt.Errorf("build aggregation params: %w", err)
	}

	resp, err := p.service.GoObjectBuilderService().ObjectBuilder().GetListAggregation(ctx, &nb.CommonMessage{
		TableSlug: action.TableSlug,
		ProjectId: action.ResourceEnvID,
		Data:      structData,
	})
	if err != nil {
		return nil, fmt.Errorf("GetListAggregation failed: %w", err)
	}

	var aggResult any = map[string]any{"data": []any{}}
	if resp.GetData() != nil {
		if m, err := helperFunc.ConvertStructToMap(resp.GetData()); err == nil {
			aggResult = m
		}
	}

	return map[string]any{
		"aggregation": aggFunc,
		"field":       action.AggregationField,
		"group_by":    action.GroupBy,
		"table_slug":  action.TableSlug,
		"result":      aggResult,
	}, nil
}

// buildWhereClause converts a filters map into a raw SQL WHERE string.
// Used exclusively by GetListAggregation which takes raw SQL (no parameterization).
// Values originate from Claude (not end-users), but string literals are escaped defensively.
func buildWhereClause(filters map[string]any) string {
	conditions := []string{"deleted_at IS NULL"}

	for k, v := range filters {
		switch val := v.(type) {
		case string:
			safe := strings.ReplaceAll(val, "'", "''")
			conditions = append(conditions, fmt.Sprintf(`"%s" = '%s'`, k, safe))
		case float64:
			conditions = append(conditions, fmt.Sprintf(`"%s" = %v`, k, val))
		case bool:
			conditions = append(conditions, fmt.Sprintf(`"%s" = %v`, k, val))
		case map[string]any:
			for op, opVal := range val {
				switch op {
				case "$gt":
					conditions = append(conditions, fmt.Sprintf(`"%s" > %v`, k, opVal))
				case "$gte":
					conditions = append(conditions, fmt.Sprintf(`"%s" >= %v`, k, opVal))
				case "$lt":
					conditions = append(conditions, fmt.Sprintf(`"%s" < %v`, k, opVal))
				case "$lte":
					conditions = append(conditions, fmt.Sprintf(`"%s" <= %v`, k, opVal))
				}
			}
		}
	}

	return strings.Join(conditions, " AND ")
}

// ============================================================================
// MUTATION HANDLER — stateless, no RAM store
// ============================================================================

func (p *ChatProcessor) handleDatabaseMutation(ctx context.Context, action *models.DatabaseActionRequest) (*models.ParsedClaudeResponse, error) {
	if action.Action == "update" || action.Action == "delete" {
		if _, hasGuid := action.Filters["guid"]; !hasGuid {
			guids, err := p.resolveGuidsFromFilters(ctx, action)
			if err != nil {
				return nil, fmt.Errorf("guid resolution failed: %w", err)
			}
			if len(guids) == 0 {
				return &models.ParsedClaudeResponse{
					Description: "Записи с такими параметрами не найдены. Уточните запрос.",
				}, nil
			}
			// Заменяем filters: теперь там только guid(ы)
			if len(guids) == 1 {
				action.Filters = map[string]any{"guid": guids[0]}
			} else {
				// Несколько записей — кладём список, executeMutation обработает
				action.Filters = map[string]any{"guid": guids[0], "_resolved_guids": guids}
			}
		}
	}
	// ─────────────────────────────────────────────────────────────────────────

	affectedCount := 1
	if action.Action != "create" && len(action.Filters) > 0 {
		if data, err := p.executeDatabaseRead(ctx, &models.DatabaseActionRequest{
			TableSlug:     action.TableSlug,
			Filters:       action.Filters,
			Limit:         1,
			ResourceEnvID: action.ResourceEnvID,
		}); err == nil {
			affectedCount = extractCountFromData(data)
		}
	}

	confirmationPrompt := action.Reply
	if confirmationPrompt == "" {
		switch action.Action {
		case "create":
			confirmationPrompt = fmt.Sprintf("Создать новую запись в `%s`?", action.TableSlug)
		case "update":
			confirmationPrompt = fmt.Sprintf("Обновить **%d** запис(ей) в `%s`?", affectedCount, action.TableSlug)
		case "delete":
			confirmationPrompt = fmt.Sprintf("⚠️ Удалить **%d** запис(ей) из `%s`?", affectedCount, action.TableSlug)
		}
	}

	successMessage := action.SuccessMessage
	if successMessage == "" {
		switch action.Action {
		case "create":
			successMessage = fmt.Sprintf("✅ Запись успешно создана в `%s`.", action.TableSlug)
		case "update":
			successMessage = fmt.Sprintf("✅ Обновлено **%d** запис(ей) в `%s`.", affectedCount, action.TableSlug)
		case "delete":
			successMessage = fmt.Sprintf("✅ Удалено **%d** запис(ей) из `%s`.", affectedCount, action.TableSlug)
		}
	}

	cancelMessage := action.CancelMessage
	if cancelMessage == "" {
		cancelMessage = "Окей, действие отменено. Ничего не изменено."
	}

	pending := &models.PendingAction{
		Action:             action.Action,
		TableSlug:          action.TableSlug,
		Filters:            action.Filters,
		Data:               action.Data,
		AffectedCount:      affectedCount,
		Description:        confirmationPrompt,
		ProjectID:          action.ResourceEnvID,
		ResourceEnvID:      action.ResourceEnvID,
		SuccessMessage:     successMessage,
		CancelMessage:      cancelMessage,
		ConfirmationPrompt: confirmationPrompt,
	}

	return &models.ParsedClaudeResponse{
		Description:   confirmationPrompt,
		PendingAction: pending,
	}, nil
}

func (p *ChatProcessor) resolveGuidsFromFilters(ctx context.Context, action *models.DatabaseActionRequest) ([]string, error) {
	filterMap := make(map[string]any, len(action.Filters)+1)
	for k, v := range action.Filters {
		filterMap[k] = v
	}
	filterMap["limit"] = 100

	structData, err := helperFunc.ConvertMapToStruct(filterMap)
	if err != nil {
		return nil, fmt.Errorf("convert filters for guid resolution: %w", err)
	}

	resp, err := p.service.GoObjectBuilderService().ObjectBuilder().GetList2(
		ctx,
		&nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: action.ResourceEnvID,
			Data:      structData,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("GetList2 for guid resolution: %w", err)
	}

	var guids []string
	if resp.GetData() != nil {
		listData, err := helperFunc.ConvertStructToMap(resp.GetData())
		if err == nil {
			for _, item := range extractItemsFromData(listData) {
				if g, ok := item["guid"].(string); ok && g != "" {
					guids = append(guids, g)
				}
			}
		}
	}
	return guids, nil
}

func executeMutation(ctx context.Context, action *models.PendingAction, service services.ServiceManagerI) (any, error) {
	if action.TableSlug == "" {
		return nil, fmt.Errorf("table_slug is required")
	}

	effectiveProjectID := action.ResourceEnvID
	if effectiveProjectID == "" {
		effectiveProjectID = action.ProjectID
	}

	switch action.Action {

	// ── CREATE ────────────────────────────────────────────────────────────────
	case "create":
		data := action.Data
		if data == nil {
			data = map[string]any{}
		}
		data["company_service_project_id"] = action.ProjectID

		structData, err := helperFunc.ConvertMapToStruct(data)
		if err != nil {
			return nil, fmt.Errorf("convert create data: %w", err)
		}
		resp, err := service.GoObjectBuilderService().Items().Create(ctx, &nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: effectiveProjectID,
			Data:      structData,
		})
		if err != nil {
			return nil, fmt.Errorf("create failed: %w", err)
		}
		if resp.GetData() != nil {
			return helperFunc.ConvertStructToMap(resp.GetData())
		}
		return map[string]any{"status": "created"}, nil

	// ── UPDATE ────────────────────────────────────────────────────────────────
	case "update":
		var guidsToUpdate []string

		if raw, ok := action.Filters["_resolved_guids"]; ok {
			for _, g := range cast.ToStringSlice(raw) {
				if g != "" {
					guidsToUpdate = append(guidsToUpdate, g)
				}
			}
		}

		if len(guidsToUpdate) == 0 {
			if guid, ok := action.Filters["guid"]; ok {
				if g := cast.ToString(guid); g != "" {
					guidsToUpdate = append(guidsToUpdate, g)
				}
			}
		}

		if len(guidsToUpdate) == 0 {
			filterMap := make(map[string]any, len(action.Filters)+1)
			for k, v := range action.Filters {
				if k == "_resolved_guids" {
					continue
				}
				filterMap[k] = v
			}
			filterMap["limit"] = 100

			filterStruct, err := helperFunc.ConvertMapToStruct(filterMap)
			if err != nil {
				return nil, fmt.Errorf("convert lookup filters for update: %w", err)
			}
			listResp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(
				ctx,
				&nb.CommonMessage{
					TableSlug: action.TableSlug,
					ProjectId: effectiveProjectID,
					Data:      filterStruct,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("lookup before update failed: %w", err)
			}
			if listResp.GetData() != nil {
				listData, err := helperFunc.ConvertStructToMap(listResp.GetData())
				if err == nil {
					for _, item := range extractItemsFromData(listData) {
						if g, ok := item["guid"].(string); ok && g != "" {
							guidsToUpdate = append(guidsToUpdate, g)
						}
					}
				}
			}
		}

		if len(guidsToUpdate) == 0 {
			return nil, fmt.Errorf("update failed: no records found with given filters (guid is required by backend)")
		}

		var lastResp any
		for _, guid := range guidsToUpdate {
			updateData := make(map[string]any, len(action.Data)+3)
			for k, v := range action.Data {
				updateData[k] = v
			}
			updateData["guid"] = guid
			updateData["id"] = guid
			updateData["company_service_project_id"] = action.ProjectID

			structData, err := helperFunc.ConvertMapToStruct(updateData)
			if err != nil {
				return nil, fmt.Errorf("convert update data for guid %s: %w", guid, err)
			}
			resp, err := service.GoObjectBuilderService().Items().Update(ctx, &nb.CommonMessage{
				TableSlug: action.TableSlug,
				ProjectId: effectiveProjectID,
				Data:      structData,
			})
			if err != nil {
				return nil, fmt.Errorf("update failed for guid %s: %w", guid, err)
			}
			if resp.GetData() != nil {
				lastResp, _ = helperFunc.ConvertStructToMap(resp.GetData())
			}
		}

		if lastResp != nil {
			return lastResp, nil
		}
		return map[string]any{"status": "updated", "count": len(guidsToUpdate)}, nil

	// ── DELETE ────────────────────────────────────────────────────────────────
	case "delete":
		var idToDelete string

		if guid, ok := action.Filters["guid"]; ok {
			idToDelete = cast.ToString(guid)
		}

		if idToDelete == "" {
			if id, ok := action.Filters["id"]; ok {
				idToDelete = cast.ToString(id)
			}
		}

		if idToDelete == "" {
			filterMap := make(map[string]any, len(action.Filters)+1)
			for k, v := range action.Filters {
				filterMap[k] = v
			}
			filterMap["limit"] = 1

			filterStruct, err := helperFunc.ConvertMapToStruct(filterMap)
			if err != nil {
				return nil, fmt.Errorf("convert lookup filters for delete: %w", err)
			}
			listResp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(
				ctx,
				&nb.CommonMessage{
					TableSlug: action.TableSlug,
					ProjectId: effectiveProjectID,
					Data:      filterStruct,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("lookup before delete failed: %w", err)
			}
			if listResp.GetData() != nil {
				listData, err := helperFunc.ConvertStructToMap(listResp.GetData())
				if err == nil {
					items := extractItemsFromData(listData)
					if len(items) > 0 {
						idToDelete, _ = items[0]["guid"].(string)
					}
				}
			}
		}

		if idToDelete == "" {
			return nil, fmt.Errorf("delete failed: record not found (guid is required by backend)")
		}

		deleteData := map[string]any{
			"id":                         idToDelete,
			"guid":                       idToDelete,
			"company_service_project_id": action.ProjectID,
		}
		structData, err := helperFunc.ConvertMapToStruct(deleteData)
		if err != nil {
			return nil, fmt.Errorf("convert delete data: %w", err)
		}
		if _, err = service.GoObjectBuilderService().Items().Delete(ctx, &nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: effectiveProjectID,
			Data:      structData,
		}); err != nil {
			return nil, fmt.Errorf("delete failed: %w", err)
		}
		return map[string]any{"status": "deleted", "affected_count": action.AffectedCount}, nil

	default:
		return nil, fmt.Errorf("unsupported action: %s", action.Action)
	}
}

// ============================================================================
// SCHEMA — cached per ChatProcessor instance lifecycle
// ============================================================================

func (p *ChatProcessor) getProjectSchemaCached(ctx context.Context, resourceEnvId string) ([]models.TableSchema, error) {
	if p.schemaCache != nil && time.Now().Before(p.schemaCachedAt.Add(schemaCacheTTL)) {
		return p.schemaCache, nil
	}
	schema, err := p.fetchProjectSchema(ctx, resourceEnvId)
	if err != nil {
		return nil, err
	}
	p.schemaCache = schema
	p.schemaCachedAt = time.Now()
	return schema, nil
}

func (p *ChatProcessor) fetchProjectSchema(ctx context.Context, resourceEnvId string) ([]models.TableSchema, error) {
	if p.mcpProjectID == "" {
		return nil, fmt.Errorf("no backend project associated with this chat")
	}

	tablesResp, err := p.service.GoObjectBuilderService().Table().GetAll(ctx, &nb.GetAllTablesRequest{
		ProjectId: resourceEnvId,
		Limit:     100,
	})
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}

	tables := tablesResp.GetTables()

	type schemaResult struct {
		idx    int
		schema models.TableSchema
	}
	ch := make(chan schemaResult, len(tables))

	for i, tbl := range tables {
		go func(idx int, t interface {
			GetId() string
			GetSlug() string
			GetLabel() string
		}) {
			fieldsResp, err := p.service.GoObjectBuilderService().Field().GetAll(ctx, &nb.GetAllFieldsRequest{
				TableId:   t.GetId(),
				ProjectId: resourceEnvId,
				Limit:     200,
			})
			var fields []models.FieldSchema
			if err == nil {
				for _, f := range fieldsResp.GetFields() {
					fields = append(fields, models.FieldSchema{
						Slug:  f.GetSlug(),
						Label: f.GetLabel(),
						Type:  f.GetType(),
					})
				}
			}
			ch <- schemaResult{idx: idx, schema: models.TableSchema{
				Slug:   t.GetSlug(),
				Label:  t.GetLabel(),
				Fields: fields,
			}}
		}(i, tbl)
	}

	collected := make([]models.TableSchema, len(tables))
	for range tables {
		r := <-ch
		collected[r.idx] = r.schema
	}

	schemas := make([]models.TableSchema, 0, len(collected))
	for _, s := range collected {
		schemas = append(schemas, s)
	}
	return schemas, nil
}

// ============================================================================
// RESOURCE RESOLUTION — cached on ChatProcessor
// ============================================================================

func (p *ChatProcessor) resolveBuilderResourceID(ctx context.Context) (string, error) {
	if p.builderResourceID != "" {
		return p.builderResourceID, nil
	}

	mcpProject, err := p.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(ctx, &nb.McpProjectId{
		ResourceEnvId: p.resourceEnvID,
		Id:            p.mcpProjectID,
	})
	if err != nil {
		return "", fmt.Errorf("get MCP project metadata: %w", err)
	}

	backendProjectID := mcpProject.GetUcodeProjectId()
	backendEnvID := mcpProject.GetEnvironmentId()
	if backendProjectID == "" || backendEnvID == "" {
		return "", fmt.Errorf("no backend project linked (project=%q env=%q)", backendProjectID, backendEnvID)
	}

	resp, err := p.service.CompanyService().ServiceResource().GetSingle(ctx, &company_service.GetSingleServiceResourceReq{
		ProjectId:     backendProjectID,
		EnvironmentId: backendEnvID,
		ServiceType:   company_service.ServiceType_BUILDER_SERVICE,
	})
	if err != nil {
		return "", fmt.Errorf("resolve builder resource (%s/%s): %w", backendProjectID, backendEnvID, err)
	}

	p.builderResourceID = resp.ResourceEnvironmentId
	return p.builderResourceID, nil
}

// ============================================================================
// CLAUDE API CALL
// ============================================================================

func (p *ChatProcessor) callDatabaseAssistant(ctx context.Context, clarified, schemaJSON, dataContext string, chatHistory []models.ChatMessage) (*models.DatabaseActionRequest, error) {
	content := helper.ProcessDatabaseAssistantPrompt(clarified, schemaJSON, dataContext)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.InspectorMaxTokens,
			System:    helper.SystemPromptDatabaseAssistant,
			Messages:  messages,
		},
		timeoutDatabaseAssistant,
	)
	if err != nil {
		return nil, fmt.Errorf("anthropic API: %w", err)
	}

	text, err := helper.ExtractPlainText(response)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}

	var action models.DatabaseActionRequest
	if err = json.Unmarshal([]byte(helper.CleanJSONResponse(text)), &action); err != nil {
		return nil, fmt.Errorf("parse action JSON: %w | raw=%.300s", err, text)
	}

	return &action, nil
}

// ============================================================================
// DATA HELPERS
// ============================================================================

// extractItemsFromData pulls the records slice out of a GetList2 response map.
// Tries "response", "data", "items" keys in that order.
func extractItemsFromData(data any) []map[string]any {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"response", "data", "items"} {
		if raw, ok := m[key]; ok {
			if items, ok := raw.([]any); ok {
				out := make([]map[string]any, 0, len(items))
				for _, item := range items {
					if row, ok := item.(map[string]any); ok {
						out = append(out, row)
					}
				}
				return out
			}
		}
	}
	return nil
}

// extractCountFromData reads the server-side "count" field from a GetList2 response.
// This is always the full dataset size (SELECT COUNT(*)), not the page length.
func extractCountFromData(data any) int {
	m, ok := data.(map[string]any)
	if !ok {
		return 0
	}
	if count, ok := m["count"]; ok {
		switch v := count.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
	}
	return len(extractItemsFromData(data))
}

// buildDataContextHint parses GUIDs from accumulated dataContext steps and formats them
// as a fenced ```db-context``` block. Claude reads this in subsequent turns to resolve
// references like "update the first one" or "delete it" without re-fetching.
func buildDataContextHint(dataContext string) string {
	if dataContext == "" {
		return ""
	}

	type recordRef struct {
		guid    string
		display string
	}

	displayFields := []string{"name", "title", "label", "full_name", "email", "username", "code"}
	var refs []recordRef

	// Each step is separated by "\n\n=== "; parse the JSON blob of each section
	sections := strings.Split(dataContext, "\n\n=== ")
	for _, section := range sections {
		newline := strings.Index(section, "\n")
		if newline == -1 {
			continue
		}
		blob := strings.TrimSpace(section[newline+1:])
		if blob == "" {
			continue
		}

		var parsed map[string]any
		if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
			continue
		}

		for _, item := range extractItemsFromData(parsed) {
			guid, _ := item["guid"].(string)
			if guid == "" {
				continue
			}
			var display string
			for _, df := range displayFields {
				if val, ok := item[df]; ok && val != nil {
					display = fmt.Sprintf("%v", val)
					break
				}
			}
			refs = append(refs, recordRef{guid: guid, display: display})
			if len(refs) >= 20 {
				goto done
			}
		}
	}

done:
	if len(refs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n```db-context\nfetched_records:\n")
	for _, r := range refs {
		if r.display != "" {
			sb.WriteString(fmt.Sprintf("  - guid: %s  # %s\n", r.guid, r.display))
		} else {
			sb.WriteString(fmt.Sprintf("  - guid: %s\n", r.guid))
		}
	}
	sb.WriteString("```")
	return sb.String()
}
