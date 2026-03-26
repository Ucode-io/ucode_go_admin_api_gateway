package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	timeoutDatabaseAssistant = 120 * time.Second
	pendingActionTTL         = 30 * time.Minute
	schemaCacheTTL           = 5 * time.Minute
	// Maximum agentic loop iterations to prevent infinite loops
	maxAgentIterations = 4
)

// ============================================================================
// PENDING ACTIONS STORE (in-memory, thread-safe, with TTL)
// ============================================================================

type timedPendingAction struct {
	action    *models.PendingAction
	expiresAt time.Time
}

var (
	pendingActions   = make(map[string]*timedPendingAction)
	pendingActionsMu sync.RWMutex
)

func storePendingAction(action *models.PendingAction) {
	pendingActionsMu.Lock()
	defer pendingActionsMu.Unlock()
	pendingActions[action.ID] = &timedPendingAction{
		action:    action,
		expiresAt: time.Now().Add(pendingActionTTL),
	}
}

func getPendingAction(id string) (*models.PendingAction, bool) {
	pendingActionsMu.RLock()
	defer pendingActionsMu.RUnlock()
	entry, ok := pendingActions[id]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.action, true
}

func deletePendingAction(id string) {
	pendingActionsMu.Lock()
	defer pendingActionsMu.Unlock()
	delete(pendingActions, id)
}

// cleanupExpiredPendingActions removes expired pending actions — called lazily before reads
func cleanupExpiredPendingActions() {
	pendingActionsMu.Lock()
	defer pendingActionsMu.Unlock()
	now := time.Now()
	for id, entry := range pendingActions {
		if now.After(entry.expiresAt) {
			delete(pendingActions, id)
		}
	}
}

// ============================================================================
// HTTP HANDLER — CONFIRM PENDING ACTION
// ============================================================================

func (h *HandlerV1) ConfirmDatabaseAction(c *gin.Context) {
	actionID := c.Param("action-id")

	var req models.ConfirmActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, "invalid request body: "+err.Error())
		return
	}

	cleanupExpiredPendingActions()

	action, ok := getPendingAction(actionID)
	if !ok {
		h.HandleResponse(c, status_http.NotFound, "pending action not found or expired (actions expire after 30 minutes)")
		return
	}

	if !req.Confirmed {
		action.Status = "rejected"
		deletePendingAction(actionID)
		h.HandleResponse(c, status_http.OK, map[string]any{
			"message": "Action cancelled by user",
			"status":  "rejected",
		})
		return
	}

	service, _, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	result, err := executeDatabaseMutation(c.Request.Context(), action, service)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to execute action: %v", err))
		return
	}

	deletePendingAction(actionID)

	h.HandleResponse(c, status_http.OK, map[string]any{
		"message": fmt.Sprintf("Successfully executed: %s", action.Description),
		"status":  "confirmed",
		"result":  result,
	})
}

// ============================================================================
// DATABASE FLOW — called from ChatProcessor.routeAndProcess
// ============================================================================

func (p *ChatProcessor) runDatabaseFlow(ctx context.Context, clarified string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	// 1. Get schema (cached within processor lifecycle)
	schema, err := p.getProjectSchemaCached(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project schema: %w", err)
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// 2. Agentic loop — Claude can request additional data across iterations
	var dataContext string
	var lastAction *models.DatabaseActionRequest

	for iteration := 0; iteration < maxAgentIterations; iteration++ {
		// Build accumulated context: schema + all data gathered so far + query plan from last step
		promptContext := buildAgentPromptContext(clarified, string(schemaJSON), dataContext)

		action, err := p.callDatabaseAssistant(ctx, promptContext, string(schemaJSON), dataContext, chatHistory)
		if err != nil {
			return nil, fmt.Errorf("database assistant failed (iteration %d): %w", iteration, err)
		}

		lastAction = action

		// Schema-only or pure answer — no DB needed
		if action.TableSlug == "" || action.Action == "schema" {
			return &models.ParsedClaudeResponse{Description: action.Reply}, nil
		}

		// Mutation actions break the loop immediately (they require user confirmation)
		if action.Action == "create" || action.Action == "update" || action.Action == "delete" {
			return p.handleDatabaseMutation(ctx, action)
		}

		// Execute the planned database operation
		var iterationData any
		var execErr error

		switch action.Action {
		case "read":
			iterationData, execErr = p.executeDatabaseRead(ctx, action)
		case "count":
			iterationData, execErr = p.executeDatabaseCount(ctx, action)
		case "aggregate":
			iterationData, execErr = p.executeDatabaseAggregate(ctx, action)
		default:
			// Unknown action — return whatever reply Claude gave
			return &models.ParsedClaudeResponse{Description: action.Reply}, nil
		}

		if execErr != nil {
			return nil, fmt.Errorf("database execution failed (iteration %d, action=%s): %w", iteration, action.Action, execErr)
		}

		// Append new data to accumulated context
		iterJSON, err := json.Marshal(iterationData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal iteration data: %w", err)
		}

		stepLabel := fmt.Sprintf("Step %d (%s on %s)", iteration+1, action.Action, action.TableSlug)
		if action.QueryPlan != "" {
			stepLabel = fmt.Sprintf("Step %d — %s", iteration+1, action.QueryPlan)
		}
		dataContext = appendDataContext(dataContext, stepLabel, string(iterJSON))

		// If Claude says it has enough data to answer — break loop
		if !action.NeedsMoreData {
			break
		}

		// Safety: if it's the last iteration, force answer on next (and final) call
		if iteration == maxAgentIterations-2 {
			dataContext += "\n\n[SYSTEM: This is the final iteration. You MUST provide the complete answer now in 'reply'. Do not set needs_more_data=true.]"
		}
	}

	// Final call: Claude has all accumulated data, now generates the answer
	// If lastAction.NeedsMoreData was false, Claude already has a good reply in lastAction.Reply.
	// We still do one final call with full context to get a polished answer.
	if lastAction != nil && !lastAction.NeedsMoreData && lastAction.Reply != "" {
		// Build data hint for follow-up operations (guids etc.)
		finalData, _ := p.executeDatabaseRead(ctx, &models.DatabaseActionRequest{
			Action:    "read",
			TableSlug: lastAction.TableSlug,
			Filters:   lastAction.Filters,
			Limit:     1,
		})
		dataHint := buildDataContextHint(finalData)

		reply := lastAction.Reply
		if dataHint != "" {
			reply = reply + dataHint
		}
		return &models.ParsedClaudeResponse{Description: reply}, nil
	}

	// Fallback: run final answer-generation call with all accumulated context
	finalAction, err := p.callDatabaseAssistant(ctx, clarified, string(schemaJSON), dataContext, chatHistory)
	if err != nil {
		if lastAction != nil && lastAction.Reply != "" {
			return &models.ParsedClaudeResponse{Description: lastAction.Reply}, nil
		}
		return nil, fmt.Errorf("final database assistant call failed: %w", err)
	}

	return &models.ParsedClaudeResponse{Description: finalAction.Reply}, nil
}

// buildAgentPromptContext constructs the prompt for the current agentic iteration.
// On first iteration dataContext is empty (planning mode).
// On subsequent iterations it contains accumulated query results.
func buildAgentPromptContext(clarified, schemaJSON, dataContext string) string {
	if dataContext == "" {
		return clarified
	}
	// Pass through — ProcessDatabaseAssistantPrompt will include dataContext
	return clarified
}

// appendDataContext appends a new data result to the accumulated context string.
func appendDataContext(existing, label, jsonData string) string {
	if existing == "" {
		return fmt.Sprintf("=== %s ===\n%s", label, jsonData)
	}
	return fmt.Sprintf("%s\n\n=== %s ===\n%s", existing, label, jsonData)
}

// ============================================================================
// DATABASE OPERATION HANDLERS
// ============================================================================

// executeDatabaseCount fetches only count via GetList2 with limit=1.
// GetList2 always returns {"count": N, "response": [...]} where count is computed
// server-side with SELECT COUNT(*) — so we never need to fetch all rows.
func (p *ChatProcessor) executeDatabaseCount(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {
	countAction := &models.DatabaseActionRequest{
		Action:    "read",
		TableSlug: action.TableSlug,
		Filters:   action.Filters,
		Limit:     1, // We only need the count field, not the actual rows
	}

	data, err := p.executeDatabaseRead(ctx, countAction)
	if err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	count := extractCountFromData(data)

	return map[string]any{
		"count":      count,
		"table_slug": action.TableSlug,
		"filters":    action.Filters,
	}, nil
}

// executeDatabaseAggregate performs server-side aggregation via GetListAggregation.
// This delegates SUM/AVG/MIN/MAX to PostgreSQL instead of fetching all rows into Go.
func (p *ChatProcessor) executeDatabaseAggregate(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {
	if action.AggregationField == "" {
		return nil, fmt.Errorf("aggregation_field is required for aggregate action")
	}

	aggFunc := strings.ToUpper(action.Aggregation)
	if aggFunc == "" {
		aggFunc = "SUM"
	}

	// Build column expression e.g. "SUM(amount) as result"
	columnExpr := fmt.Sprintf("%s(%s) as result", aggFunc, action.AggregationField)
	columns := []string{columnExpr}

	// If group_by is specified, add it to columns and group_by list
	var groupBy []string
	if action.GroupBy != "" {
		columns = append([]string{action.GroupBy}, columns...)
		groupBy = []string{action.GroupBy}
	}

	// Build WHERE clause from filters (raw SQL for GetListAggregation)
	whereClause := buildWhereClause(action.Filters)

	queryParams := map[string]any{
		"operation": "SELECT",
		"table":     fmt.Sprintf(`"%s"`, action.TableSlug),
		"columns":   columns,
	}
	if whereClause != "" {
		queryParams["where"] = whereClause
	}
	if len(groupBy) > 0 {
		queryParams["group_by"] = groupBy
	}

	structData, err := helperFunc.ConvertMapToStruct(queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to build aggregation params: %w", err)
	}

	builderID, err := p.resolveBuilderResourceID(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := p.service.GoObjectBuilderService().ObjectBuilder().GetListAggregation(
		ctx, &nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: builderID,
			Data:      structData,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("GetListAggregation failed: %w", err)
	}

	var aggResult any
	if resp.GetData() != nil {
		result, err := helperFunc.ConvertStructToMap(resp.GetData())
		if err != nil {
			return nil, fmt.Errorf("failed to convert aggregation response: %w", err)
		}
		aggResult = result
	} else {
		aggResult = map[string]any{"data": []any{}}
	}

	// Wrap with metadata for Claude
	return map[string]any{
		"aggregation": aggFunc,
		"field":       action.AggregationField,
		"table_slug":  action.TableSlug,
		"group_by":    action.GroupBy,
		"result":      aggResult,
	}, nil
}

// buildWhereClause converts a filters map to a raw SQL WHERE expression for GetListAggregation.
// Note: GetListAggregation accepts raw SQL WHERE string (no parameterization),
// so we only handle simple equality and basic comparisons here.
// This is intentionally limited to safe, non-user-facing filter values set by Claude.
func buildWhereClause(filters map[string]any) string {
	if len(filters) == 0 {
		return "deleted_at IS NULL"
	}

	conditions := []string{"deleted_at IS NULL"}
	for k, v := range filters {
		switch val := v.(type) {
		case string:
			// Escape single quotes to prevent injection
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

// handleDatabaseRead fetches records and uses a second Claude call to format a rich answer.
// Kept for backward compatibility but now also called from the agentic loop.
func (p *ChatProcessor) handleDatabaseRead(ctx context.Context, action *models.DatabaseActionRequest, clarified, schemaJSON string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	data, err := p.executeDatabaseRead(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("database read failed: %w", err)
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query results: %w", err)
	}

	finalAction, err := p.callDatabaseAssistant(ctx, clarified, schemaJSON, string(dataJSON), chatHistory)
	if err != nil {
		return &models.ParsedClaudeResponse{Description: action.Reply}, nil
	}

	dataHint := buildDataContextHint(data)
	reply := finalAction.Reply
	if dataHint != "" {
		reply = reply + dataHint
	}

	return &models.ParsedClaudeResponse{Description: reply}, nil
}

// handleDatabaseMutation previews how many records will be affected and stores a pending action.
// Count is fetched efficiently with limit=1 (reading the server-side count field).
func (p *ChatProcessor) handleDatabaseMutation(ctx context.Context, action *models.DatabaseActionRequest) (*models.ParsedClaudeResponse, error) {
	affectedCount := 1

	if action.Action != "create" && len(action.Filters) > 0 {
		// Use limit=1 — GetList2 returns server-side COUNT regardless of limit
		countAction := &models.DatabaseActionRequest{
			Action:    "read",
			TableSlug: action.TableSlug,
			Filters:   action.Filters,
			Limit:     1,
		}
		data, err := p.executeDatabaseRead(ctx, countAction)
		if err == nil {
			affectedCount = extractCountFromData(data)
		}
	}

	builderID, _ := p.resolveBuilderResourceID(ctx)

	pending := &models.PendingAction{
		ID:            uuid.NewString(),
		ChatID:        p.chatId,
		Action:        action.Action,
		TableSlug:     action.TableSlug,
		Filters:       action.Filters,
		Data:          action.Data,
		AffectedCount: affectedCount,
		Description:   action.Reply,
		Status:        "pending",
		ProjectID:     builderID,
		ResourceEnvID: p.resourceEnvID,
	}

	storePendingAction(pending)

	description := action.Reply
	if description == "" {
		switch action.Action {
		case "create":
			description = fmt.Sprintf("Create a new record in table `%s`.", action.TableSlug)
		case "update":
			description = fmt.Sprintf("Update **%d** record(s) in table `%s`.", affectedCount, action.TableSlug)
		case "delete":
			description = fmt.Sprintf("⚠️ Delete **%d** record(s) from table `%s`.", affectedCount, action.TableSlug)
		}
	}

	return &models.ParsedClaudeResponse{
		Description:   description,
		PendingAction: pending,
	}, nil
}

// ============================================================================
// SCHEMA INTROSPECTION (with per-processor cache)
// ============================================================================

func (p *ChatProcessor) getProjectSchemaCached(ctx context.Context) ([]models.TableSchema, error) {
	if p.schemaCache != nil && time.Now().Before(p.schemaCachedAt.Add(schemaCacheTTL)) {
		return p.schemaCache, nil
	}

	schema, err := p.fetchProjectSchema(ctx)
	if err != nil {
		return nil, err
	}

	p.schemaCache = schema
	p.schemaCachedAt = time.Now()
	return schema, nil
}

func (p *ChatProcessor) fetchProjectSchema(ctx context.Context) ([]models.TableSchema, error) {
	if p.mcpProjectID == "" {
		return nil, fmt.Errorf("no backend project associated with this chat")
	}

	builderID, err := p.resolveBuilderResourceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve builder ID: %w", err)
	}

	tablesResp, err := p.service.GoObjectBuilderService().Table().GetAll(
		ctx, &nb.GetAllTablesRequest{
			ProjectId: builderID,
			Limit:     100,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	tables := tablesResp.GetTables()
	schemas := make([]models.TableSchema, 0, len(tables))

	type result struct {
		idx    int
		schema models.TableSchema
	}

	resultChan := make(chan result, len(tables))

	for i, table := range tables {
		go func(idx int, tbl interface {
			GetId() string
			GetSlug() string
			GetLabel() string
		}) {
			fieldsResp, err := p.service.GoObjectBuilderService().Field().GetAll(
				ctx, &nb.GetAllFieldsRequest{
					TableId:   tbl.GetId(),
					ProjectId: builderID,
					Limit:     200,
				},
			)

			var fields []models.FieldSchema
			if err == nil {
				for _, field := range fieldsResp.GetFields() {
					fields = append(fields, models.FieldSchema{
						Slug:  field.GetSlug(),
						Label: field.GetLabel(),
						Type:  field.GetType(),
					})
				}
			}

			resultChan <- result{
				idx: idx,
				schema: models.TableSchema{
					Slug:   tbl.GetSlug(),
					Label:  tbl.GetLabel(),
					Fields: fields,
				},
			}
		}(i, table)
	}

	collected := make([]models.TableSchema, len(tables))
	for range tables {
		r := <-resultChan
		collected[r.idx] = r.schema
	}

	for _, s := range collected {
		schemas = append(schemas, s)
	}

	return schemas, nil
}

// ============================================================================
// RESOURCE RESOLUTION
// ============================================================================

func (p *ChatProcessor) resolveBuilderResourceID(ctx context.Context) (string, error) {
	if p.builderResourceID != "" {
		return p.builderResourceID, nil
	}

	mcpProject, err := p.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		ctx, &nb.McpProjectId{
			ResourceEnvId: p.resourceEnvID,
			Id:            p.mcpProjectID,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get MCP project metadata: %w", err)
	}

	backendProjectID := mcpProject.GetUcodeProjectId()
	backendEnvID := mcpProject.GetEnvironmentId()

	if backendProjectID == "" || backendEnvID == "" {
		return "", fmt.Errorf("no backend project/environment linked to this AI project (project=%s, env=%s)", backendProjectID, backendEnvID)
	}

	resp, err := p.service.CompanyService().ServiceResource().GetSingle(
		ctx,
		&company_service.GetSingleServiceResourceReq{
			ProjectId:     backendProjectID,
			EnvironmentId: backendEnvID,
			ServiceType:   company_service.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to resolve builder resource ID for backend %s/%s: %w", backendProjectID, backendEnvID, err)
	}

	p.builderResourceID = resp.ResourceEnvironmentId
	return p.builderResourceID, nil
}

// ============================================================================
// CLAUDE AI CALL
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
		return nil, fmt.Errorf("database assistant API call failed: %w", err)
	}

	text, err := helper.ExtractPlainText(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from database assistant: %w", err)
	}

	cleaned := helper.CleanJSONResponse(text)

	var action models.DatabaseActionRequest
	if err = json.Unmarshal([]byte(cleaned), &action); err != nil {
		return nil, fmt.Errorf("failed to parse database action JSON: %w | raw=%.300s", err, text)
	}

	return &action, nil
}

// ============================================================================
// DATABASE EXECUTION (via Items gRPC)
// ============================================================================

func (p *ChatProcessor) executeDatabaseRead(ctx context.Context, action *models.DatabaseActionRequest) (any, error) {
	if action.TableSlug == "" {
		return nil, fmt.Errorf("table_slug is required for database read")
	}

	dataMap := make(map[string]any)

	for k, v := range action.Filters {
		dataMap[k] = v
	}

	if action.Limit > 0 {
		dataMap["limit"] = action.Limit
	} else {
		dataMap["limit"] = 50
	}

	if action.Offset > 0 {
		dataMap["offset"] = action.Offset
	}

	if action.OrderBy != "" {
		dataMap["order_by"] = action.OrderBy
	}

	structData, err := helperFunc.ConvertMapToStruct(dataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert filters to struct: %w", err)
	}

	builderID, err := p.resolveBuilderResourceID(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := p.service.GoObjectBuilderService().ObjectBuilder().GetList2(
		ctx, &nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: builderID,
			Data:      structData,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("items GetList failed: %w", err)
	}

	if resp.GetData() != nil {
		result, err := helperFunc.ConvertStructToMap(resp.GetData())
		if err != nil {
			return nil, fmt.Errorf("failed to convert response data: %w", err)
		}
		return result, nil
	}

	return map[string]any{"response": []any{}, "count": 0}, nil
}

func executeDatabaseMutation(ctx context.Context, action *models.PendingAction, service services.ServiceManagerI) (any, error) {
	if action.TableSlug == "" {
		return nil, fmt.Errorf("table_slug is required")
	}

	projectID := action.ProjectID

	switch action.Action {
	case "create":
		dataMap := action.Data
		if dataMap == nil {
			dataMap = make(map[string]any)
		}
		structData, err := helperFunc.ConvertMapToStruct(dataMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert create data: %w", err)
		}
		resp, err := service.GoObjectBuilderService().Items().Create(
			ctx, &nb.CommonMessage{
				TableSlug: action.TableSlug,
				ProjectId: projectID,
				Data:      structData,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("create item failed: %w", err)
		}
		if resp.GetData() != nil {
			return helperFunc.ConvertStructToMap(resp.GetData())
		}
		return map[string]any{"status": "created"}, nil

	case "update":
		dataMap := make(map[string]any)
		for k, v := range action.Data {
			dataMap[k] = v
		}
		for k, v := range action.Filters {
			dataMap[k] = v
		}
		structData, err := helperFunc.ConvertMapToStruct(dataMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert update data: %w", err)
		}
		resp, err := service.GoObjectBuilderService().Items().Update(
			ctx, &nb.CommonMessage{
				TableSlug: action.TableSlug,
				ProjectId: projectID,
				Data:      structData,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("update item failed: %w", err)
		}
		if resp.GetData() != nil {
			return helperFunc.ConvertStructToMap(resp.GetData())
		}
		return map[string]any{"status": "updated"}, nil

	case "delete":
		filterMap := action.Filters
		if filterMap == nil {
			filterMap = make(map[string]any)
		}
		structData, err := helperFunc.ConvertMapToStruct(filterMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert delete filters: %w", err)
		}
		_, err = service.GoObjectBuilderService().Items().Delete(
			ctx, &nb.CommonMessage{
				TableSlug: action.TableSlug,
				ProjectId: projectID,
				Data:      structData,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("delete item failed: %w", err)
		}
		return map[string]any{"status": "deleted", "affected_count": action.AffectedCount}, nil

	default:
		return nil, fmt.Errorf("unsupported mutation action: %s", action.Action)
	}
}

// ============================================================================
// DATA HELPERS — aggregation and context extraction
// ============================================================================

// extractItemsFromData extracts the list of records from a GetList response map
func extractItemsFromData(data any) []map[string]any {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	for _, key := range []string{"response", "data", "items"} {
		if raw, ok := dataMap[key]; ok {
			if items, ok := raw.([]any); ok {
				result := make([]map[string]any, 0, len(items))
				for _, item := range items {
					if m, ok := item.(map[string]any); ok {
						result = append(result, m)
					}
				}
				return result
			}
		}
	}
	return nil
}

// extractCountFromData returns the server-side count from a GetList2 response.
// GetList2 always returns {"count": N, "response": [...]} where count is a server-side
// SELECT COUNT(*) — not the length of the returned page.
func extractCountFromData(data any) int {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return 0
	}

	// Prefer the server-side count field — it's always the full count regardless of limit
	if count, ok := dataMap["count"]; ok {
		switch v := count.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
	}

	// Fallback: count returned items (only accurate when no pagination applied)
	items := extractItemsFromData(data)
	return len(items)
}

// buildDataContextHint creates a structured context note with fetched record GUIDs.
// This is appended to the assistant reply so the next message can reference these records
// for follow-up operations (e.g. "update the first one", "delete it").
// Format is human-readable and explicitly instructed in the system prompt.
func buildDataContextHint(data any) string {
	items := extractItemsFromData(data)
	if len(items) == 0 {
		return ""
	}

	type recordRef struct {
		guid    string
		display string
	}

	displayFields := []string{"name", "title", "label", "full_name", "email", "username", "code"}
	refs := make([]recordRef, 0, len(items))

	for _, item := range items {
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
	}

	if len(refs) == 0 {
		return ""
	}

	// Limit to 20 records in context hint
	limit := len(refs)
	if limit > 20 {
		limit = 20
	}

	// Build a JSON context note that is clearly labeled for the AI in subsequent turns.
	// This goes into the assistant message so it's visible in chat history as context.
	var sb strings.Builder
	sb.WriteString("\n\n```db-context\n")
	sb.WriteString("fetched_records:\n")
	for i := 0; i < limit; i++ {
		if refs[i].display != "" {
			sb.WriteString(fmt.Sprintf("  - guid: %s  # %s\n", refs[i].guid, refs[i].display))
		} else {
			sb.WriteString(fmt.Sprintf("  - guid: %s\n", refs[i].guid))
		}
	}
	sb.WriteString("```")

	return sb.String()
}
