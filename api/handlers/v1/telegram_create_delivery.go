package v1

import (
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

type telegramCreateDelivery struct {
	ChatID  string
	Message string
}

// A Telegram group may be connected to several CRM workspaces. A configured
// automation for that group supersedes legacy lead notifications from the
// other workspaces, so one created deal does not arrive in both formats.
func telegramCreateDeliveries(settingsList []models.TelegramNotificationSettings, deal map[string]any, now time.Time) []telegramCreateDelivery {
	chatOrder := make([]string, 0, len(settingsList))
	seenChat := make(map[string]bool, len(settingsList))
	modernChats := make(map[string]bool, len(settingsList))
	modernMessages := make(map[string][]string, len(settingsList))
	legacyMessages := make(map[string]string, len(settingsList))

	for _, settings := range settingsList {
		chatID := strings.TrimSpace(settings.ChatID)
		if chatID == "" {
			continue
		}
		if !seenChat[chatID] {
			seenChat[chatID] = true
			chatOrder = append(chatOrder, chatID)
		}
		if settings.Automations == nil {
			if settings.NewLeadEnabled && legacyMessages[chatID] == "" {
				legacyMessages[chatID] = renderTelegramNotificationTemplateWithDeal(settings.Templates.NewLead, deal)
			}
			continue
		}
		for _, rule := range settings.Automations {
			ruleChatID := strings.TrimSpace(rule.ChatID)
			if ruleChatID == "" {
				ruleChatID = chatID
			}
			if ruleChatID == "" {
				continue
			}
			if !seenChat[ruleChatID] {
				seenChat[ruleChatID] = true
				chatOrder = append(chatOrder, ruleChatID)
			}
			if len(rule.TriggerConfigs) > 0 {
				modernChats[ruleChatID] = true
				if trigger, ok := telegramAutomationMatchingTrigger(rule, "deals", "create", deal, nil, now); ok {
					modernMessages[ruleChatID] = append(modernMessages[ruleChatID], renderTelegramNotificationTemplateWithDeal(telegramAutomationTriggerTemplate(trigger), deal))
				}
				continue
			}
			if rule.Trigger == "new_lead" && telegramAutomationMatches(rule, "created", deal, now) {
				if legacyMessages[ruleChatID] == "" {
					legacyMessages[ruleChatID] = renderTelegramNotificationTemplateWithDeal(telegramAutomationTemplate(rule, settings, "created"), deal)
				}
				continue
			}
			if telegramAutomationMatches(rule, "created", deal, now) {
				modernMessages[ruleChatID] = append(modernMessages[ruleChatID], renderTelegramNotificationTemplateWithDeal(telegramAutomationTemplate(rule, settings, "created"), deal))
			}
		}
	}

	deliveries := make([]telegramCreateDelivery, 0, len(chatOrder))
	for _, chatID := range chatOrder {
		if modernChats[chatID] {
			seenMessage := make(map[string]bool)
			for _, message := range modernMessages[chatID] {
				if message != "" && !seenMessage[message] {
					deliveries = append(deliveries, telegramCreateDelivery{ChatID: chatID, Message: message})
					seenMessage[message] = true
				}
			}
		} else if message := legacyMessages[chatID]; message != "" {
			deliveries = append(deliveries, telegramCreateDelivery{ChatID: chatID, Message: message})
		}
	}
	return deliveries
}
