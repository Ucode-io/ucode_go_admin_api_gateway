package v1

import (
	"context"
	"github.com/google/uuid"
)

func telegramContactPhone(item map[string]any) string {
	if phone := telegramDealValue(item, "phone", "phone_number", "contact_phone", "telephone", "telefon", "mobile", "mobile_phone", "phones"); phone != "" {
		return phone
	}
	for _, key := range []string{"contacts_id_data", "contact_id_data", "contact", "contacts_id", "contact_id"} {
		if contact, ok := item[key].(map[string]any); ok {
			if phone := telegramDealValue(contact, "phone", "phones", "mobile", "phone_number"); phone != "" {
				return phone
			}
		}
	}
	return ""
}

func telegramContactID(item map[string]any) string {
	for _, key := range []string{"contacts_id", "contact_id", "contacts_id_data", "contact_id_data", "contact"} {
		value := item[key]
		if rows, ok := value.([]any); ok && len(rows) == 1 {
			value = rows[0]
		}
		if row, ok := value.(map[string]any); ok {
			value = telegramDealValue(row, "guid", "id")
		}
		if id, ok := value.(string); ok {
			if _, err := uuid.Parse(id); err == nil {
				return id
			}
		}
	}
	return ""
}

func (h *HandlerV1) telegramItemWithPhone(ctx context.Context, target telegramNotificationTarget, item map[string]any) map[string]any {
	phone := telegramContactPhone(item)
	if phone == "" {
		if id := telegramContactID(item); id != "" {
			service, environmentID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
			if err == nil {
				if contact, found, err := h.lookupItem(ctx, service, environmentID, "contacts", id); err == nil && found {
					phone = telegramContactPhone(contact)
				}
			}
		}
	}
	if phone == "" {
		return item
	}
	readable := make(map[string]any, len(item)+1)
	for key, value := range item {
		readable[key] = value
	}
	readable["phone"] = phone
	return readable
}
