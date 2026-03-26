package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

const timeoutDatabaseAssistant = 120 * time.Second

// ============================================================================
// PENDING ACTIONS STORE (in-memory, thread-safe)
// ============================================================================

var (
	pendingActions   = make(map[string]*models.PendingAction)
	pendingActionsMu sync.RWMutex
)

func storePendingAction(action *models.PendingAction) {
	pendingActionsMu.Lock()
	defer pendingActionsMu.Unlock()
	pendingActions[action.ID] = action
}

func getPendingAction(id string) (*models.PendingAction, bool) {
	pendingActionsMu.RLock()
	defer pendingActionsMu.RUnlock()
	action, ok := pendingActions[id]
	return action, ok
}

func deletePendingAction(id string) {
	pendingActionsMu.Lock()
	defer pendingActionsMu.Unlock()
	delete(pendingActions, id)
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

	action, ok := getPendingAction(actionID)
	if !ok {
		h.HandleResponse(c, status_http.NotFound, "pending action not found or expired")
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
	schema, err := p.getProjectSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project schema: %w", err)
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	action, err := p.callDatabaseAssistant(ctx, clarified, string(schemaJSON), "", chatHistory)
	if err != nil {
		return nil, fmt.Errorf("database assistant failed: %w", err)
	}

	log.Printf("[DB_ASSISTANT] action=%s table=%s", action.Action, action.TableSlug)

	// If it's a schema/metadata question or no table is targeted, just return the reply
	if action.TableSlug == "" || action.Action == "schema" {
		return &models.ParsedClaudeResponse{
			Description: action.Reply,
		}, nil
	}

	switch action.Action {
	case "read", "count", "aggregate":
		return p.handleDatabaseRead(ctx, action, clarified, string(schemaJSON), chatHistory)

	case "create", "update", "delete":
		return p.handleDatabaseMutation(ctx, action)

	default:
		return &models.ParsedClaudeResponse{
			Description: action.Reply,
		}, nil
	}
}

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
		return &models.ParsedClaudeResponse{
			Description: action.Reply,
		}, nil
	}

	return &models.ParsedClaudeResponse{
		Description: finalAction.Reply,
	}, nil
}

func (p *ChatProcessor) handleDatabaseMutation(ctx context.Context, action *models.DatabaseActionRequest) (*models.ParsedClaudeResponse, error) {
	affectedCount := 1
	if action.Action != "create" && len(action.Filters) > 0 {
		countAction := &models.DatabaseActionRequest{
			Action:    "read",
			TableSlug: action.TableSlug,
			Filters:   action.Filters,
			Limit:     1000,
		}
		data, err := p.executeDatabaseRead(ctx, countAction)
		if err == nil {
			if dataMap, ok := data.(map[string]any); ok {
				if resp, ok := dataMap["response"]; ok {
					if items, ok := resp.([]any); ok {
						affectedCount = len(items)
					}
				}
				if count, ok := dataMap["count"].(float64); ok {
					affectedCount = int(count)
				}
			}
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

	log.Printf("[DB_ASSISTANT] Created pending action: id=%s action=%s table=%s affected=%d",
		pending.ID, pending.Action, pending.TableSlug, pending.AffectedCount)

	// Construct a response that includes the pending action for the frontend
	description := action.Reply
	if description == "" {
		switch action.Action {
		case "create":
			description = fmt.Sprintf("Create a new record in table '%s'.", action.TableSlug)
		case "update":
			description = fmt.Sprintf("Update %d record(s) in table '%s'.", affectedCount, action.TableSlug)
		case "delete":
			description = fmt.Sprintf("Delete %d record(s) from table '%s'.", affectedCount, action.TableSlug)
		}
	}

	return &models.ParsedClaudeResponse{
		Description:   description,
		PendingAction: pending,
	}, nil
}

// ============================================================================
// SCHEMA INTROSPECTION
// ============================================================================

func (p *ChatProcessor) getProjectSchema(ctx context.Context) ([]models.TableSchema, error) {
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

	var schemas []models.TableSchema

	for _, table := range tablesResp.GetTables() {
		fieldsResp, err := p.service.GoObjectBuilderService().Field().GetAll(
			ctx, &nb.GetAllFieldsRequest{
				TableId:   table.GetId(),
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

		schemas = append(schemas, models.TableSchema{
			Slug:   table.GetSlug(),
			Label:  table.GetLabel(),
			Fields: fields,
		})
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

	// 1. Get MCP project to find the actual linked backend IDs
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

	// 2. Resolve the builder resource ID using backend metadata
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
	log.Printf("[DB_ASSISTANT] Resolved builder resource ID: %s", p.builderResourceID)
	return p.builderResourceID, nil
}

// ============================================================================
// CLAUDE AI CALL
// ============================================================================

func (p *ChatProcessor) callDatabaseAssistant(ctx context.Context, clarified, schemaJSON, dataContext string, chatHistory []models.ChatMessage) (*models.DatabaseActionRequest, error) {
	var (
		content  = helper.ProcessDatabaseAssistantPrompt(clarified, schemaJSON, dataContext)
		messages = buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})
	)

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
		return nil, fmt.Errorf("database assistant api call failed: %w", err)
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

	// Build the data struct with filters, limit, offset
	dataMap := make(map[string]any)

	if len(action.Filters) > 0 {
		for k, v := range action.Filters {
			dataMap[k] = v
		}
	}

	if action.Limit > 0 {
		dataMap["limit"] = action.Limit
	} else {
		dataMap["limit"] = 100
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

	resp, err := p.service.GoObjectBuilderService().ObjectBuilder().GetList(
		ctx, &nb.CommonMessage{
			TableSlug: action.TableSlug,
			ProjectId: builderID,
			Data:      structData,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("items GetList failed: %w", err)
	}

	// Convert response data to a map
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

	// We need the ucode project ID — it's stored in the action
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
		// Merge filters as search criteria
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
