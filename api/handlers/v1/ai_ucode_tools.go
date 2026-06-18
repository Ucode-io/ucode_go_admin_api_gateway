package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cast"

	"ucode/ucode_go_api_gateway/api/handlers/ai"
	"ucode/ucode_go_api_gateway/config"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
)

const (
	toolGetSchema      = "get_schema"
	toolCreateTable    = "create_table"
	toolCreateField    = "create_field"
	toolCreateRelation = "create_relation"
	toolCreateMenu     = "create_menu"
	toolInsertItems    = "insert_items"
)

var ucodeFieldTypes = []string{
	"SINGLE_LINE", "MULTI_LINE", "NUMBER", "BOOLEAN", "DATE", "DATE_TIME",
	"EMAIL", "PHONE", "PHOTO", "JSON", "PICK_LIST",
}

func ucodeToolDefs() []ai.ToolDef {
	return []ai.ToolDef{
		{
			Name:        toolGetSchema,
			Description: "Inspect the project's data model. With no arguments it returns the list of existing table slugs. With table_slug it returns that table's field slugs. Call this before creating objects to avoid duplicates and to reuse what already exists.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_slug": map[string]any{
						"type":        "string",
						"description": "Optional. When set, returns the fields of this table instead of the table list.",
					},
				},
			},
		},
		{
			Name:        toolCreateTable,
			Description: "Create a new table. It automatically appears in the project menu. Returns the table id; no-op (status=already_exists) if a table with this slug exists.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug":  map[string]any{"type": "string", "description": "snake_case identifier, e.g. \"order_item\"."},
					"label": map[string]any{"type": "string", "description": "Human-readable name, e.g. \"Order Item\"."},
					"menu_id": map[string]any{
						"type":        "string",
						"description": "Optional. Place the table under this menu folder (a menu_id returned by create_menu). Defaults to the main menu.",
					},
				},
				"required": []string{"slug", "label"},
			},
		},
		{
			Name:        toolCreateField,
			Description: "Add a field (column) to an existing table. No-op (status=already_exists) if the field already exists. To link tables use create_relation instead of a field.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_slug": map[string]any{"type": "string", "description": "Slug of the table to add the field to."},
					"slug":       map[string]any{"type": "string", "description": "snake_case field identifier."},
					"label":      map[string]any{"type": "string", "description": "Human-readable field name."},
					"type": map[string]any{
						"type":        "string",
						"enum":        ucodeFieldTypes,
						"description": "Field type. Defaults to SINGLE_LINE when omitted.",
					},
				},
				"required": []string{"table_slug", "slug", "type"},
			},
		},
		{
			Name:        toolCreateRelation,
			Description: "Create a many-to-one relation: each row of table_from references one row of table_to. This adds a foreign-key field ({table_to}_id) on table_from. Both tables must already exist.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_from": map[string]any{"type": "string", "description": "The 'many' side that gets the foreign key (e.g. order_item)."},
					"table_to":   map[string]any{"type": "string", "description": "The 'one' side being referenced (e.g. product)."},
				},
				"required": []string{"table_from", "table_to"},
			},
		},
		{
			Name:        toolCreateMenu,
			Description: "Create a menu folder to group tables under. Returns its menu_id, which you can pass to create_table's menu_id to nest tables inside the folder.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"label": map[string]any{"type": "string", "description": "Folder name shown in the menu."},
					"icon":  map[string]any{"type": "string", "description": "Optional Lucide icon name, e.g. \"package\"."},
				},
				"required": []string{"label"},
			},
		},
		{
			Name:        toolInsertItems,
			Description: "Insert one or more records into a table. Each item is an object of field_slug → value. Returns the created record guids (use them to wire foreign keys in later inserts).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_slug": map[string]any{"type": "string", "description": "Slug of the target table."},
					"items": map[string]any{
						"type":        "array",
						"description": "Records to insert.",
						"items":       map[string]any{"type": "object", "additionalProperties": true},
					},
				},
				"required": []string{"table_slug", "items"},
			},
		},
	}
}

func (s *ucodeChatSession) executeTool(ctx context.Context, call ai.ToolCall) (string, bool) {
	switch call.Name {
	case toolGetSchema:
		return s.toolGetSchema(ctx, call)
	case toolCreateTable:
		return s.toolCreateTable(ctx, call)
	case toolCreateField:
		return s.toolCreateField(ctx, call)
	case toolCreateRelation:
		return s.toolCreateRelation(ctx, call)
	case toolCreateMenu:
		return s.toolCreateMenu(ctx, call)
	case toolInsertItems:
		return s.toolInsertItems(ctx, call)
	default:
		return fmt.Sprintf("error: unknown tool %q", call.Name), true
	}
}

// ── schema cache ────────────────────────────────────────────────────────────────

func (s *ucodeChatSession) ensureTablesLoaded(ctx context.Context) error {
	if s.tablesLoaded {
		return nil
	}
	resp, err := s.service.GoObjectBuilderService().Table().GetAll(ctx, &nb.GetAllTablesRequest{
		ProjectId: s.resourceEnvId,
		Limit:     1000,
	})
	if err != nil {
		return err
	}
	for _, t := range resp.GetTables() {
		s.tableIDs[t.GetSlug()] = t.GetId()
	}
	s.tablesLoaded = true
	return nil
}

func (s *ucodeChatSession) ensureFieldsLoaded(ctx context.Context, tableSlug string) error {
	if _, ok := s.fieldSets[tableSlug]; ok {
		return nil
	}
	resp, err := s.service.GoObjectBuilderService().Field().GetAll(ctx, &nb.GetAllFieldsRequest{
		TableSlug: tableSlug,
		TableId:   s.tableIDs[tableSlug],
		ProjectId: s.resourceEnvId,
		Limit:     500,
	})
	if err != nil {
		return err
	}
	set := make(map[string]bool, len(resp.GetFields()))
	for _, f := range resp.GetFields() {
		set[f.GetSlug()] = true
	}
	s.fieldSets[tableSlug] = set
	return nil
}

// ── tool implementations ────────────────────────────────────────────────────────

func (s *ucodeChatSession) toolGetSchema(ctx context.Context, call ai.ToolCall) (string, bool) {
	tableSlug := normalizeSlug(cast.ToString(call.Input["table_slug"]))
	if tableSlug != "" {
		s.emit.Emit(SSEEvent{
			Type:    EvProgress,
			Icon:    IconScanSearch,
			Message: "Читаю поля таблицы",
			Value:   tableSlug,
			Data:    UcodeStepData{Action: StepActionSchema, Status: StepStatusDone, Table: tableSlug},
		})
	} else {
		s.emit.Emit(SSEEvent{
			Type:    EvProgress,
			Icon:    IconScanSearch,
			Message: "Читаю схему проекта",
			Data:    UcodeStepData{Action: StepActionSchema, Status: StepStatusDone},
		})
	}

	if err := s.ensureTablesLoaded(ctx); err != nil {
		return "error: " + err.Error(), true
	}

	if tableSlug != "" {
		if _, ok := s.tableIDs[tableSlug]; !ok {
			return marshalToolResult(map[string]any{"exists": false, "table_slug": tableSlug}), false
		}
		if err := s.ensureFieldsLoaded(ctx, tableSlug); err != nil {
			return "error: " + err.Error(), true
		}
		fields := make([]string, 0, len(s.fieldSets[tableSlug]))
		for slug := range s.fieldSets[tableSlug] {
			fields = append(fields, slug)
		}
		return marshalToolResult(map[string]any{
			"exists":     true,
			"table_slug": tableSlug,
			"fields":     sortStrings(fields),
		}), false
	}

	slugs := make([]string, 0, len(s.tableIDs))
	for slug := range s.tableIDs {
		slugs = append(slugs, slug)
	}
	return marshalToolResult(map[string]any{"tables": sortStrings(slugs), "count": len(slugs)}), false
}

func (s *ucodeChatSession) toolCreateTable(ctx context.Context, call ai.ToolCall) (string, bool) {
	slug := normalizeSlug(cast.ToString(call.Input["slug"]))
	if slug == "" {
		return "error: slug is required", true
	}
	label := strings.TrimSpace(cast.ToString(call.Input["label"]))
	if label == "" {
		label = slugToLabel(slug)
	}

	if err := s.ensureTablesLoaded(ctx); err != nil {
		return "error: " + err.Error(), true
	}
	if id, exists := s.tableIDs[slug]; exists {
		s.emit.Emit(SSEEvent{
			Type:    EvTableDone,
			Icon:    IconDatabase,
			Message: "Таблица уже существует",
			Value:   label,
			Data:    UcodeStepData{Action: StepActionTable, Status: StepStatusSkipped, Table: slug, Label: label, Reason: "already_exists"},
		})
		return marshalToolResult(map[string]any{"status": "already_exists", "table_slug": slug, "table_id": id}), false
	}

	menuID := config.MainMenuID
	if mid := strings.TrimSpace(cast.ToString(call.Input["menu_id"])); mid != "" {
		menuID = mid
	}

	attrs, err := helper.ConvertMapToStruct(map[string]any{"label": "", "label_en": label})
	if err != nil {
		return "error: " + err.Error(), true
	}

	s.emit.Emit(SSEEvent{
		Type:    EvTableStart,
		Icon:    IconDatabase,
		Message: "Создаю таблицу",
		Value:   label,
		Data:    UcodeStepData{Action: StepActionTable, Status: StepStatusStarted, Table: slug, Label: label},
	})

	resp, err := s.service.GoObjectBuilderService().Table().Create(ctx, &nb.CreateTableRequest{
		Label:          label,
		Slug:           slug,
		ProjectId:      s.resourceEnvId,
		EnvId:          s.envId,
		MenuId:         menuID,
		ViewId:         uuid.NewString(),
		LayoutId:       uuid.NewString(),
		ShowInMenu:     true,
		Attributes:     attrs,
		UcodeProjectId: s.projectId,
	})
	if err != nil {
		return "error: " + err.Error(), true
	}

	s.tableIDs[slug] = resp.GetId()
	// A freshly created table has only system fields; mark its user-field set as
	// loaded-and-empty so later create_field calls skip a redundant fetch.
	s.fieldSets[slug] = map[string]bool{}
	s.stats.Tables++

	s.emit.Emit(SSEEvent{
		Type:    EvTableDone,
		Icon:    IconDatabase,
		Message: "Таблица создана",
		Value:   label,
		Data:    UcodeStepData{Action: StepActionTable, Status: StepStatusDone, Table: slug, Label: label},
	})

	return marshalToolResult(map[string]any{"status": "created", "table_slug": slug, "table_id": resp.GetId()}), false
}

func (s *ucodeChatSession) toolCreateField(ctx context.Context, call ai.ToolCall) (string, bool) {
	tableSlug := normalizeSlug(cast.ToString(call.Input["table_slug"]))
	slug := normalizeSlug(cast.ToString(call.Input["slug"]))
	if tableSlug == "" || slug == "" {
		return "error: table_slug and slug are required", true
	}
	label := strings.TrimSpace(cast.ToString(call.Input["label"]))
	if label == "" {
		label = slugToLabel(slug)
	}

	if err := s.ensureTablesLoaded(ctx); err != nil {
		return "error: " + err.Error(), true
	}
	tableID, ok := s.tableIDs[tableSlug]
	if !ok {
		return fmt.Sprintf("error: table %q does not exist — create it first", tableSlug), true
	}
	if err := s.ensureFieldsLoaded(ctx, tableSlug); err != nil {
		return "error: " + err.Error(), true
	}
	if s.fieldSets[tableSlug][slug] {
		s.emit.Emit(SSEEvent{
			Type:    EvProgress,
			Icon:    IconColumns,
			Message: fmt.Sprintf("Поле уже есть в %s", tableSlug),
			Value:   label,
			Data:    UcodeStepData{Action: StepActionField, Status: StepStatusSkipped, Table: tableSlug, Field: slug, Label: label, Reason: "already_exists"},
		})
		return marshalToolResult(map[string]any{"status": "already_exists", "table_slug": tableSlug, "field_slug": slug}), false
	}

	fieldType := mapFieldType(cast.ToString(call.Input["type"]))
	attrs, err := helper.ConvertMapToStruct(map[string]any{"label": "", "label_en": label})
	if err != nil {
		return "error: " + err.Error(), true
	}

	if _, err = s.service.GoObjectBuilderService().Field().Create(ctx, &nb.CreateFieldRequest{
		Id:         uuid.NewString(),
		TableId:    tableID,
		Label:      label,
		Slug:       slug,
		Type:       fieldType,
		Attributes: attrs,
		ProjectId:  s.resourceEnvId,
		Index:      "string",
		IsVisible:  true,
	}); err != nil {
		return "error: " + err.Error(), true
	}

	s.fieldSets[tableSlug][slug] = true
	s.stats.Fields++

	s.emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    IconColumns,
		Message: fmt.Sprintf("Добавлено поле в %s", tableSlug),
		Value:   label,
		Data:    UcodeStepData{Action: StepActionField, Status: StepStatusDone, Table: tableSlug, Field: slug, FieldType: fieldType, Label: label},
	})

	return marshalToolResult(map[string]any{"status": "created", "table_slug": tableSlug, "field_slug": slug, "type": fieldType}), false
}

func (s *ucodeChatSession) toolCreateRelation(ctx context.Context, call ai.ToolCall) (string, bool) {
	tableFrom := normalizeSlug(cast.ToString(call.Input["table_from"]))
	tableTo := normalizeSlug(cast.ToString(call.Input["table_to"]))
	if tableFrom == "" || tableTo == "" {
		return "error: table_from and table_to are required", true
	}

	if err := s.ensureTablesLoaded(ctx); err != nil {
		return "error: " + err.Error(), true
	}
	if _, ok := s.tableIDs[tableFrom]; !ok {
		return fmt.Sprintf("error: table %q does not exist", tableFrom), true
	}
	if _, ok := s.tableIDs[tableTo]; !ok {
		return fmt.Sprintf("error: table %q does not exist", tableTo), true
	}

	// Relation.Create requires a non-empty ViewFields; ObtainRandomOne yields a
	// real field id from table_from to satisfy it (proven in createBackendFromPlan).
	viewField, err := s.service.GoObjectBuilderService().Field().ObtainRandomOne(ctx, &nb.ObtainRandomRequest{
		TableSlug: tableFrom,
		ProjectId: s.resourceEnvId,
		EnvId:     s.envId,
	})
	if err != nil {
		return "error: " + err.Error(), true
	}

	relFieldSlug := tableTo + "_id"
	attrs, err := helper.ConvertMapToStruct(map[string]any{
		"label_en":    slugToLabel(tableTo),
		"label_to_en": slugToLabel(tableFrom),
	})
	if err != nil {
		return "error: " + err.Error(), true
	}

	if _, err = s.service.GoObjectBuilderService().Relation().Create(ctx, &nb.CreateRelationRequest{
		Id:                uuid.NewString(),
		TableFrom:         tableFrom,
		TableTo:           tableTo,
		Type:              "Many2One",
		RelationTableSlug: tableTo,
		RelationFieldSlug: relFieldSlug,
		RelationFieldId:   uuid.NewString(),
		RelationToFieldId: uuid.NewString(),
		ProjectId:         s.resourceEnvId,
		EnvId:             s.envId,
		ViewFields:        []string{viewField.GetId()},
		Attributes:        attrs,
	}); err != nil {
		return "error: " + err.Error(), true
	}

	// The relation adds the FK field on table_from; reflect it in the cache.
	if set, ok := s.fieldSets[tableFrom]; ok {
		set[relFieldSlug] = true
	}
	s.stats.Relations++

	s.emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    IconLink,
		Message: "Создана связь",
		Value:   fmt.Sprintf("%s → %s", tableFrom, tableTo),
		Data:    UcodeStepData{Action: StepActionRelation, Status: StepStatusDone, TableFrom: tableFrom, TableTo: tableTo, ForeignKey: relFieldSlug},
	})

	return marshalToolResult(map[string]any{
		"status":      "created",
		"table_from":  tableFrom,
		"table_to":    tableTo,
		"foreign_key": relFieldSlug,
	}), false
}

func (s *ucodeChatSession) toolCreateMenu(ctx context.Context, call ai.ToolCall) (string, bool) {
	label := strings.TrimSpace(cast.ToString(call.Input["label"]))
	if label == "" {
		return "error: label is required", true
	}
	icon := strings.TrimSpace(cast.ToString(call.Input["icon"]))

	attrs, err := helper.ConvertMapToStruct(map[string]any{"label_en": label})
	if err != nil {
		return "error: " + err.Error(), true
	}

	menuID := uuid.NewString()

	menu, err := s.service.GoObjectBuilderService().Menu().Create(ctx, &nb.CreateMenuRequest{
		Id:         menuID,
		Label:      label,
		Icon:       icon,
		Type:       "FOLDER",
		ProjectId:  s.resourceEnvId,
		EnvId:      s.envId,
		ParentId:   config.MainMenuID,
		IsVisible:  true,
		Attributes: attrs,
	})
	if err != nil {
		return "error: " + err.Error(), true
	}
	if id := menu.GetId(); id != "" {
		menuID = id
	}
	s.stats.Menus++

	s.emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    IconFolder,
		Message: "Создан раздел меню",
		Value:   label,
		Data:    UcodeStepData{Action: StepActionMenu, Status: StepStatusDone, MenuID: menuID, Label: label},
	})

	return marshalToolResult(map[string]any{"status": "created", "menu_id": menuID, "label": label}), false
}

func (s *ucodeChatSession) toolInsertItems(ctx context.Context, call ai.ToolCall) (string, bool) {
	tableSlug := normalizeSlug(cast.ToString(call.Input["table_slug"]))
	if tableSlug == "" {
		return "error: table_slug is required", true
	}
	rawItems, ok := call.Input["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return "error: items is required and must be a non-empty array of objects", true
	}

	if err := s.ensureTablesLoaded(ctx); err != nil {
		return "error: " + err.Error(), true
	}
	if _, ok := s.tableIDs[tableSlug]; !ok {
		return fmt.Sprintf("error: table %q does not exist", tableSlug), true
	}

	var (
		created  int
		guids    []string
		failures []string
	)
	for i, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			failures = append(failures, fmt.Sprintf("item[%d]: not an object", i))
			continue
		}
		structData, err := helper.ConvertMapToStruct(item)
		if err != nil {
			failures = append(failures, fmt.Sprintf("item[%d]: %v", i, err))
			continue
		}
		resp, err := s.service.GoObjectBuilderService().Items().Create(ctx, &nb.CommonMessage{
			TableSlug: tableSlug,
			ProjectId: s.resourceEnvId,
			Data:      structData,
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("item[%d]: %v", i, err))
			continue
		}
		created++
		if m, convErr := helper.ConvertStructToMap(resp.GetData()); convErr == nil {
			if g, _ := m["guid"].(string); g != "" {
				guids = append(guids, g)
			}
		}
	}

	s.stats.Items += created

	s.emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    IconPlusCircle,
		Message: fmt.Sprintf("Добавлены записи в %s", tableSlug),
		Value:   fmt.Sprintf("%d из %d", created, len(rawItems)),
		Data:    UcodeStepData{Action: StepActionItems, Status: StepStatusDone, Table: tableSlug, Created: created, Failed: len(failures), Total: len(rawItems)},
	})

	result := map[string]any{"status": "completed", "table_slug": tableSlug, "created": created, "guids": guids}
	if len(failures) > 0 {
		result["errors"] = failures
	}

	return marshalToolResult(result), created == 0 && len(failures) > 0
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastUnderscore = false
		case r == ' ' || r == '-' || r == '_':
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
