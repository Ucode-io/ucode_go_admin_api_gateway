package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
)

// ========================== CRUD Flow ==========================

// runCrudFlow orchestrates the entire CRUD flow:
// 1. Fetches DB schema from the builder service
// 2. Calls Claude Haiku (CRUD Agent) to parse user intent into structured JSON
// 3. For INSERT/SELECT: executes immediately
// 4. For UPDATE/DELETE: saves pending operation and asks for confirmation
func (m *messagingStc) runCrudFlow(ctx context.Context, userRequest string) (*models.ParsedClaudeResponse, error) {
	// Step 1: Get DB schema
	schemaJSON, err := m.getDBSchema(ctx)
	if err != nil {
		log.Println("ERROR GETTING DB SCHEMA:", err)
		return &models.ParsedClaudeResponse{
			Description: "Не удалось получить схему базы данных. Попробуйте позже.",
		}, nil
	}

	if schemaJSON == "[]" || schemaJSON == "" {
		return &models.ParsedClaudeResponse{
			Description: "В проекте нет таблиц для работы с данными.",
		}, nil
	}

	// Step 2: Call CRUD Agent (Haiku) to parse user intent
	crudOp, err := m.callCrudAgent(userRequest, schemaJSON)
	if err != nil {
		log.Println("ERROR IN CRUD AGENT:", err)
		return &models.ParsedClaudeResponse{
			Description: "Не удалось распознать операцию. Попробуйте переформулировать запрос.",
		}, nil
	}

	if crudOp.Operation == "unknown" {
		return &models.ParsedClaudeResponse{
			Description: crudOp.PreviewMessage,
		}, nil
	}

	// Step 3: Execute or save pending
	switch crudOp.Operation {
	case "insert":
		return m.executeCrudNow(ctx, crudOp)

	case "select":
		return m.executeCrudNow(ctx, crudOp)

	case "update", "delete":
		// Save pending and ask for confirmation
		return m.savePendingCrud(ctx, crudOp)

	default:
		return &models.ParsedClaudeResponse{
			Description: fmt.Sprintf("Неизвестная операция: %s", crudOp.Operation),
		}, nil
	}
}

// getDBSchema fetches the database schema via gRPC
func (m *messagingStc) getDBSchema(ctx context.Context) (string, error) {
	resp, err := m.service.GoObjectBuilderService().AiChat().GetProjectTablesSchema(
		ctx,
		&pbo.GetProjectTablesSchemaRequest{
			ResourceEnvId: m.resourceEnvId,
		},
	)
	if err != nil {
		return "", fmt.Errorf("GetProjectTablesSchema failed: %w", err)
	}

	// Convert to models.DBTableSchema for JSON serialization
	tables := make([]models.DBTableSchema, 0, len(resp.GetTables()))
	for _, t := range resp.GetTables() {
		cols := make([]models.DBColumn, 0, len(t.GetColumns()))
		for _, c := range t.GetColumns() {
			cols = append(cols, models.DBColumn{
				ColumnName: c.GetColumnName(),
				DataType:   c.GetDataType(),
				IsNullable: c.GetIsNullable(),
			})
		}
		tables = append(tables, models.DBTableSchema{
			TableName: t.GetTableName(),
			Columns:   cols,
		})
	}

	schemaBytes, err := json.Marshal(tables)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	return string(schemaBytes), nil
}

// callCrudAgent calls Claude Haiku with the CRUD system prompt
func (m *messagingStc) callCrudAgent(userRequest, schemaJSON string) (*models.CrudOperation, error) {
	content := helper.ProcessCrudAgentPrompt(userRequest, schemaJSON)
	messages := []models.ChatMessage{
		{
			Role:    "user",
			Content: []models.ContentBlock{{Type: "text", Text: content}},
		},
	}

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeHaikuModel,
			MaxTokens: 1024,
			System:    helper.SystemPromptCrudAgent,
			Messages:  messages,
		},
		60*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("CRUD agent API call failed: %w", err)
	}

	// Extract JSON from response
	jsonStr, err := helper.ExtractPlainText(rawResp)
	if err != nil {
		return nil, fmt.Errorf("failed to extract CRUD agent response: %w", err)
	}

	// Clean up potential markdown wrapping
	jsonStr = helper.ExtractJSON(jsonStr)

	var op models.CrudOperation
	if err := json.Unmarshal([]byte(jsonStr), &op); err != nil {
		return nil, fmt.Errorf("failed to parse CRUD agent JSON: %w (raw: %s)", err, jsonStr)
	}

	return &op, nil
}

// executeCrudNow executes a CRUD operation immediately (for INSERT and SELECT)
func (m *messagingStc) executeCrudNow(ctx context.Context, op *models.CrudOperation) (*models.ParsedClaudeResponse, error) {
	dataJSON, _ := json.Marshal(op.Data)
	whereJSON, _ := json.Marshal(op.Where)

	resp, err := m.service.GoObjectBuilderService().AiChat().ExecuteCrudOperation(
		ctx,
		&pbo.ExecuteCrudOperationRequest{
			ResourceEnvId: m.resourceEnvId,
			Operation:     op.Operation,
			Table:         op.Table,
			DataJson:      string(dataJSON),
			WhereJson:     string(whereJSON),
		},
	)
	if err != nil {
		log.Println("ERROR EXECUTING CRUD:", err)
		return &models.ParsedClaudeResponse{
			Description: fmt.Sprintf("Ошибка выполнения операции: %v", err),
		}, nil
	}

	switch op.Operation {
	case "select":
		if resp.GetResultJson() == "null" || resp.GetResultJson() == "[]" {
			return &models.ParsedClaudeResponse{
				Description: "Записи не найдены.",
			}, nil
		}

		// Format results as a readable table
		var rows []map[string]any
		if err := json.Unmarshal([]byte(resp.GetResultJson()), &rows); err == nil {
			return &models.ParsedClaudeResponse{
				Description: formatSelectResults(rows),
			}, nil
		}

		return &models.ParsedClaudeResponse{
			Description: resp.GetResultJson(),
		}, nil

	case "insert":
		return &models.ParsedClaudeResponse{
			Description: fmt.Sprintf("✅ Запись добавлена в таблицу `%s`. Затронуто строк: %d", op.Table, resp.GetRowsAffected()),
		}, nil

	default:
		return &models.ParsedClaudeResponse{
			Description: fmt.Sprintf("✅ Операция выполнена. Затронуто строк: %d", resp.GetRowsAffected()),
		}, nil
	}
}

// savePendingCrud saves the pending CRUD operation to the chat's description field
// and returns a confirmation prompt to the user
func (m *messagingStc) savePendingCrud(ctx context.Context, op *models.CrudOperation) (*models.ParsedClaudeResponse, error) {
	pendingJSON, err := json.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize pending operation: %w", err)
	}

	// Save pending operation in chat's description field with a prefix
	pendingDesc := "PENDING_CRUD:" + string(pendingJSON)

	_, err = m.service.GoObjectBuilderService().AiChat().UpdateChat(
		ctx,
		&pbo.UpdateChatRequest{
			ResourceEnvId: m.resourceEnvId,
			Id:            m.chatId,
			Description:   pendingDesc,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save pending CRUD: %w", err)
	}

	return &models.ParsedClaudeResponse{
		Description: op.PreviewMessage,
	}, nil
}

// executePendingCrud retrieves and executes a pending CRUD operation from the chat's description
func (m *messagingStc) executePendingCrud(ctx context.Context) (*models.ParsedClaudeResponse, error) {
	// Get the chat to read the pending operation
	chat, err := m.service.GoObjectBuilderService().AiChat().GetChat(
		ctx,
		&pbo.ChatPrimaryKey{
			ResourceEnvId: m.resourceEnvId,
			Id:            m.chatId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat for pending CRUD: %w", err)
	}

	desc := chat.GetDescription()
	const prefix = "PENDING_CRUD:"
	if len(desc) <= len(prefix) || desc[:len(prefix)] != prefix {
		return &models.ParsedClaudeResponse{
			Description: "Нет ожидающей операции для подтверждения.",
		}, nil
	}

	pendingJSON := desc[len(prefix):]
	var op models.CrudOperation
	if err := json.Unmarshal([]byte(pendingJSON), &op); err != nil {
		return &models.ParsedClaudeResponse{
			Description: "Не удалось прочитать ожидающую операцию. Попробуйте заново.",
		}, nil
	}

	// Clear the pending operation
	_, _ = m.service.GoObjectBuilderService().AiChat().UpdateChat(
		ctx,
		&pbo.UpdateChatRequest{
			ResourceEnvId: m.resourceEnvId,
			Id:            m.chatId,
			Description:   "",
		},
	)

	// Execute
	return m.executeCrudNow(ctx, &op)
}

// formatSelectResults formats SELECT results as a readable markdown table
func formatSelectResults(rows []map[string]any) string {
	if len(rows) == 0 {
		return "Записи не найдены."
	}

	// Collect column headers from the first row
	headers := make([]string, 0)
	for key := range rows[0] {
		headers = append(headers, key)
	}

	// Build markdown table
	result := "| " 
	for _, h := range headers {
		result += h + " | "
	}
	result += "\n|"
	for range headers {
		result += " --- |"
	}
	result += "\n"

	for _, row := range rows {
		result += "| "
		for _, h := range headers {
			val := row[h]
			result += fmt.Sprintf("%v | ", val)
		}
		result += "\n"
	}

	result += fmt.Sprintf("\nВсего записей: %d", len(rows))
	return result
}
