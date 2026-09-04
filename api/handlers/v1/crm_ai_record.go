package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
)

func newCRMRecordPendingAction(
	request string,
	action *models.CRMRecordAction,
	schema []models.TableSchema,
	resourceEnvID string,
) (*models.PendingAction, error) {
	if action == nil {
		return nil, fmt.Errorf("record_action is required")
	}
	normalized, err := normalizeCRMRecordAction(*action, schema)
	if err != nil {
		return nil, err
	}
	description, success := crmRecordActionMessages(request, normalized)
	return &models.PendingAction{
		Action:             "client_record",
		TableSlug:          normalized.Table,
		Data:               map[string]any{"record_action": normalized},
		ResourceEnvID:      resourceEnvID,
		ProjectID:          resourceEnvID,
		Description:        description,
		ConfirmationPrompt: description,
		SuccessMessage:     success,
		CancelMessage:      crmMutationCancelMessage(request),
	}, nil
}

func normalizeCRMRecordAction(action models.CRMRecordAction, schema []models.TableSchema) (models.CRMRecordAction, error) {
	action.Operation = strings.ToLower(strings.TrimSpace(action.Operation))
	action.Table = strings.ToLower(strings.TrimSpace(action.Table))
	action.RecordGUID = strings.TrimSpace(action.RecordGUID)
	if action.Operation != "create" && action.Operation != "update" && action.Operation != "delete" {
		return action, fmt.Errorf("unsupported record operation %q", action.Operation)
	}
	if action.Table != "deals" && action.Table != "contacts" && action.Table != "companies" && action.Table != "tasks" {
		return action, fmt.Errorf("unsupported CRM table %q", action.Table)
	}
	if action.Operation != "create" && (action.RecordGUID == "" || len(action.RecordGUID) > 128) {
		return action, fmt.Errorf("record_guid is required for update/delete")
	}
	if action.Operation == "delete" {
		action.Data = nil
		return action, nil
	}
	if len(action.Data) == 0 {
		return action, fmt.Errorf("record data is required")
	}
	if len(action.Data) > 100 {
		return action, fmt.Errorf("too many record fields")
	}

	allowedFields := map[string]models.FieldSchema{}
	if len(schema) > 0 {
		table, ok := findCRMSchemaTable(schema, action.Table)
		if !ok {
			return action, fmt.Errorf("table %q is not in the project schema", action.Table)
		}
		for _, field := range table.Fields {
			allowedFields[field.Slug] = field
		}
	}
	protected := map[string]bool{
		"guid": true, "id": true, "project_id": true, "environment_id": true,
		"created_at": true, "updated_at": true, "deleted_at": true,
	}
	cleaned := make(map[string]any, len(action.Data))
	for rawKey, value := range action.Data {
		key := strings.TrimSpace(rawKey)
		if !isCRMPreferenceSlug(key) || protected[strings.ToLower(key)] {
			return action, fmt.Errorf("field %q cannot be changed", rawKey)
		}
		if len(allowedFields) > 0 {
			field, ok := allowedFields[key]
			if !ok {
				return action, fmt.Errorf("field %q is not in table %q", key, action.Table)
			}
			value = normalizeCRMRecordFieldValue(field, value)
		}
		cleaned[key] = value
	}
	action.Data = cleaned
	return action, nil
}

func normalizeCRMRecordFieldValue(field models.FieldSchema, value any) any {
	fieldType := strings.ToLower(strings.TrimSpace(field.Type))
	// Ucode's MULTISELECT columns are PostgreSQL arrays. The item API expects
	// an array even when the user supplies a single option (for example Source).
	if strings.Contains(fieldType, "multi") {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				return []string{}
			}
			return []string{typed}
		case nil:
			return []string{}
		}
	}
	return value
}

func crmRecordActionMessages(request string, action models.CRMRecordAction) (string, string) {
	operation := action.Operation
	if operation == "create" {
		operation = "create " + action.Table
	} else {
		operation += " " + action.Table + " / " + action.RecordGUID
	}
	switch detectCRMRequestLanguage(strings.ToLower(request)) {
	case "ru":
		return fmt.Sprintf("Подтвердите изменение CRM: %s.", operation), "Запись CRM обновлена."
	case "en":
		return fmt.Sprintf("Confirm the CRM change: %s.", operation), "CRM record updated."
	default:
		return fmt.Sprintf("CRM o‘zgarishini tasdiqlang: %s.", operation), "CRM yozuvi yangilandi."
	}
}

func decodeStoredCRMRecordAction(data map[string]any) (*models.CRMRecordAction, error) {
	raw, ok := data["record_action"]
	if !ok {
		return nil, fmt.Errorf("stored record action is missing")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var action models.CRMRecordAction
	if err = json.Unmarshal(encoded, &action); err != nil {
		return nil, err
	}
	normalized, err := normalizeCRMRecordAction(action, nil)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}
