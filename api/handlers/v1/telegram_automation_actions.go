package v1

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"google.golang.org/protobuf/types/known/structpb"
)

const telegramAutomationActionPrefix = "telegram:notifications:action:"

type telegramAutomationAction struct {
	Target    telegramNotificationTarget `json:"target"`
	ChatID    string                     `json:"chat_id"`
	MessageID int64                      `json:"message_id"`
	RecordID  string                     `json:"record_id"`
	RuleID    string                     `json:"rule_id"`
	TriggerID string                     `json:"trigger_id"`
}

func telegramAutomationFieldLabel(trigger models.TelegramAutomationTrigger, field, value string) string {
	for _, option := range trigger.FieldOptions[field] {
		if option.Value == value || option.Label == value || option.Slug == value {
			return option.Label
		}
	}
	return value
}

func renderTelegramAutomationMessage(trigger models.TelegramAutomationTrigger, item map[string]any) string {
	readable := make(map[string]any, len(item))
	for key, value := range item {
		readable[key] = value
	}
	for field := range trigger.FieldOptions {
		if value := telegramDealValue(item, field); value != "" {
			readable[field] = telegramAutomationFieldLabel(trigger, field, value)
		}
	}
	return renderTelegramNotificationTemplateWithDeal(telegramAutomationTriggerTemplate(trigger), readable)
}

func renderTelegramAutomationCompletedMessage(trigger models.TelegramAutomationTrigger, item map[string]any, status string) string {
	return renderTelegramAutomationMessage(trigger, item) + "\n\n✅ <b>Текущий статус:</b> " + html.EscapeString(status)
}

func (h *HandlerV1) sendTelegramAutomationAction(target telegramNotificationTarget, chatID string, rule models.TelegramAutomation, trigger models.TelegramAutomationTrigger, item map[string]any) {
	message := renderTelegramAutomationMessage(trigger, item)
	recordID := telegramDealValue(item, "guid", "id")
	if trigger.Table != "deals" || trigger.StatusField == "" || len(trigger.StatusButtons) == 0 || recordID == "" || h.centralRedis == nil {
		h.sendTelegramCRMNotification(chatID, message)
		return
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		h.sendTelegramCRMNotification(chatID, message)
		return
	}
	token := hex.EncodeToString(random[:])
	buttons := make([][]map[string]string, 0, len(trigger.StatusButtons))
	for _, button := range trigger.StatusButtons {
		if button.ID == "" || button.Value == "" {
			continue
		}
		buttons = append(buttons, []map[string]string{{"text": button.Label, "callback_data": "crm:" + token + ":" + button.ID}})
	}
	if len(buttons) == 0 {
		h.sendTelegramCRMNotification(chatID, message)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client := newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken)
		sent, err := client.sendHTMLMessageWithMarkup(ctx, chatID, message, map[string]any{"inline_keyboard": buttons})
		if err != nil {
			h.log.Error("telegram automation send failed", logger.Error(err))
			return
		}
		action := telegramAutomationAction{Target: target, ChatID: chatID, MessageID: sent.MessageID, RecordID: recordID, RuleID: rule.ID, TriggerID: trigger.ID}
		body, _ := json.Marshal(action)
		if err := h.centralRedis.Set(ctx, telegramAutomationActionPrefix+token, body, 30*24*time.Hour).Err(); err != nil {
			h.log.Error("telegram automation action storage failed", logger.Error(err))
			_ = client.removeInlineKeyboard(ctx, chatID, sent.MessageID)
		}
	}()
}

func (h *HandlerV1) handleTelegramAutomationCallback(ctx context.Context, callback *telegramCallbackQuery) {
	client := newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken)
	answer := func(message string) { _ = client.answerCallbackQuery(ctx, callback.ID, message) }
	parts := strings.Split(callback.Data, ":")
	if h.centralRedis == nil || len(parts) != 3 || parts[0] != "crm" || callback.Message == nil {
		answer("Недоступное действие")
		return
	}
	key := telegramAutomationActionPrefix + parts[1]
	body, err := h.centralRedis.Get(ctx, key).Bytes()
	if err != nil {
		answer("Действие уже выполнено или устарело")
		return
	}
	var action telegramAutomationAction
	if json.Unmarshal(body, &action) != nil || action.ChatID != fmt.Sprint(callback.Message.Chat.ID) || action.MessageID != callback.Message.MessageID {
		answer("Недоступное действие")
		return
	}
	lockKey := key + ":lock"
	locked, err := h.centralRedis.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil || !locked {
		answer("Обновление уже выполняется")
		return
	}
	defer h.centralRedis.Del(ctx, lockKey)
	settings, _, err := h.getTelegramNotificationSettings(ctx, action.Target)
	if err != nil {
		answer("Не удалось загрузить правило")
		return
	}
	var trigger models.TelegramAutomationTrigger
	var button models.TelegramStatusButton
	valid := false
	for _, rule := range settings.Automations {
		if rule.ID != action.RuleID || !rule.Enabled || rule.ChatID != action.ChatID {
			continue
		}
		for _, candidate := range rule.TriggerConfigs {
			if candidate.ID != action.TriggerID || candidate.Table != "deals" || candidate.StatusField == "" {
				continue
			}
			for _, choice := range candidate.StatusButtons {
				if choice.ID == parts[2] && choice.Value != "" {
					trigger, button, valid = candidate, choice, true
					break
				}
			}
		}
	}
	if !valid {
		answer("Кнопка больше не активна")
		return
	}
	updatedItem, updated, err := h.updateTelegramAutomationDealStatus(ctx, action, trigger.StatusField, button.Value)
	if !updated {
		h.log.Error("telegram automation status update failed", logger.Error(err))
		answer("Не удалось обновить статус")
		return
	}
	_ = h.centralRedis.Del(ctx, key).Err()
	if err != nil {
		h.log.Error("telegram automation deal reload failed", logger.Error(err))
		_ = client.removeInlineKeyboard(ctx, action.ChatID, action.MessageID)
		answer("Статус обновлён, но сообщение не удалось обновить")
		return
	}
	message := renderTelegramAutomationCompletedMessage(trigger, updatedItem, button.Label)
	if err := client.editHTMLMessage(ctx, action.ChatID, action.MessageID, message); err != nil {
		h.log.Error("telegram automation message edit failed", logger.Error(err))
		_ = client.removeInlineKeyboard(ctx, action.ChatID, action.MessageID)
		answer("Статус обновлён, но сообщение не удалось обновить")
		return
	}
	answer("Статус обновлён: " + button.Label)
}

func (h *HandlerV1) updateTelegramAutomationDealStatus(ctx context.Context, action telegramAutomationAction, field, value string) (map[string]any, bool, error) {
	if field == "" || value == "" || action.RecordID == "" {
		return nil, false, errors.New("missing status field, value or record")
	}
	resource, err := h.companyServices.ServiceResource().GetSingle(ctx, &pb.GetSingleServiceResourceReq{ProjectId: action.Target.ProjectID, EnvironmentId: action.Target.EnvironmentID, ServiceType: pb.ServiceType_BUILDER_SERVICE})
	if err != nil {
		return nil, false, err
	}
	services, err := h.GetProjectSrvc(ctx, action.Target.ProjectID, resource.NodeType)
	if err != nil {
		return nil, false, err
	}
	item, found, err := h.lookupItem(ctx, services, resource.ResourceEnvironmentId, "deals", action.RecordID)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, errors.New("deal was not found")
	}
	if telegramDealValue(item, field) == value {
		return item, true, nil
	}
	data, err := structpb.NewStruct(map[string]any{"guid": action.RecordID, "id": action.RecordID, "company_service_project_id": action.Target.ProjectID, field: value})
	if err != nil {
		return nil, false, err
	}
	_, err = services.GoObjectBuilderService().Items().Update(ctx, &nb.CommonMessage{TableSlug: "deals", ProjectId: resource.ResourceEnvironmentId, Data: data})
	if err != nil {
		return nil, false, err
	}
	item, found, err = h.lookupItem(ctx, services, resource.ResourceEnvironmentId, "deals", action.RecordID)
	if err != nil {
		return nil, true, err
	}
	if !found {
		return nil, true, errors.New("updated deal was not found")
	}
	return item, true, nil
}
