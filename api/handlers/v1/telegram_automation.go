package v1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

func (h *HandlerV1) notifyDealAutomations(ctx context.Context, projectID, environmentID string, deal map[string]any, event string, stageChanged bool) {
	h.notifyItemAutomations(ctx, projectID, environmentID, "deals", deal, event, nil, stageChanged)
}

func (h *HandlerV1) notifyItemAutomations(ctx context.Context, projectID, environmentID, table string, item map[string]any, event string, changedFields map[string]any, stageChanged bool) {
	if !h.telegramNotificationsConfigured() {
		return
	}
	targets, err := h.telegramNotificationTargets(ctx, projectID, environmentID)
	if err != nil {
		return
	}
	for _, target := range targets {
		readableItem := h.telegramAutomationReadableItem(ctx, target, item)
		settings, _, err := h.getTelegramNotificationSettings(ctx, target)
		if err != nil || settings.Automations == nil {
			continue
		}
		for _, rule := range settings.Automations {
			chatID := strings.TrimSpace(rule.ChatID)
			if chatID == "" {
				chatID = strings.TrimSpace(settings.ChatID)
			}
			if chatID == "" {
				continue
			}
			if len(rule.TriggerConfigs) > 0 {
				trigger, ok := telegramAutomationMatchingTrigger(rule, table, event, item, changedFields, time.Now())
				if ok {
					h.sendTelegramAutomationAction(target, chatID, rule, trigger, readableItem)
				}
				continue
			}
			var matches bool
			if event == "updated" {
				matches = telegramAutomationMatchesUpdated(rule, item, time.Now(), stageChanged)
			} else {
				matches = telegramAutomationMatches(rule, event, item, time.Now())
			}
			if matches {
				h.sendTelegramCRMNotification(chatID, renderTelegramNotificationTemplateWithDeal(telegramAutomationTemplate(rule, settings, event), readableItem))
			}
		}
	}
}

func (h *HandlerV1) telegramAutomationReadableItem(ctx context.Context, target telegramNotificationTarget, item map[string]any) map[string]any {
	managerID := telegramDealValue(item, "users_id", "sotuv_manajeri")
	if managerID == "" {
		return item
	}
	service, environmentID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return item
	}
	user, found, err := h.lookupItem(ctx, service, environmentID, "users", managerID)
	if err != nil || !found {
		return item
	}
	name := telegramDealValue(user, "full_name", "name", "login", "email")
	if name == "" {
		name = strings.TrimSpace(telegramDealValue(user, "first_name") + " " + telegramDealValue(user, "last_name"))
	}
	if name == "" {
		return item
	}
	readable := make(map[string]any, len(item))
	for key, value := range item {
		readable[key] = value
	}
	readable["users_id"] = name
	readable["sotuv_manajeri"] = name
	return readable
}

func telegramAutomationMatchingTrigger(rule models.TelegramAutomation, table, event string, item, changedFields map[string]any, now time.Time) (models.TelegramAutomationTrigger, bool) {
	if !rule.Enabled || rule.Action != "telegram_notification" {
		return models.TelegramAutomationTrigger{}, false
	}
	for _, trigger := range rule.TriggerConfigs {
		if trigger.Kind == "daily_report" {
			if event != "daily_report" || trigger.ReportTime != "" && now.Format("15:04") != trigger.ReportTime {
				continue
			}
		} else if trigger.Table != table || trigger.Kind != event && !(trigger.Kind == "field" && event == "update") {
			continue
		}
		if trigger.WatchField != "" {
			if _, changed := changedFields[trigger.WatchField]; !changed {
				continue
			}
		}
		if telegramAutomationConditionsMatch(trigger.Conditions, trigger.ConditionMode, item, now) {
			return trigger, true
		}
	}
	return models.TelegramAutomationTrigger{}, false
}

func telegramAutomationConditionsMatch(conditions []models.TelegramAutomationCondition, mode string, item map[string]any, now time.Time) bool {
	if len(conditions) == 0 {
		return true
	}
	if mode == "or" {
		for _, condition := range conditions {
			if telegramAutomationConditionMatches(condition, item, now) {
				return true
			}
		}
		return false
	}
	for _, condition := range conditions {
		if !telegramAutomationConditionMatches(condition, item, now) {
			return false
		}
	}
	return true
}

func telegramAutomationTriggerTemplate(trigger models.TelegramAutomationTrigger) string {
	if strings.TrimSpace(trigger.Template) != "" {
		return trigger.Template
	}
	title := strings.Title(strings.ReplaceAll(trigger.Kind, "_", " "))
	if trigger.Kind == "daily_report" {
		title = "Kunlik hisobot"
	}
	lines := make([]string, 0, len(trigger.MessageFields))
	for _, field := range trigger.MessageFields {
		label := strings.Title(strings.ReplaceAll(strings.TrimPrefix(field, "report."), "_", " "))
		token := "{{item." + field + "}}"
		if trigger.Kind == "daily_report" {
			token = "{{" + field + "}}"
		}
		lines = append(lines, label+": "+token)
	}
	return "<b>" + title + "</b>\n\n" + strings.Join(lines, "\n")
}

func telegramLegacyAutomations(settings models.TelegramNotificationSettings) []models.TelegramAutomation {
	rules := []models.TelegramAutomation{
		{ID: "legacy-new-lead", Name: "Yangi lid", Enabled: settings.NewLeadEnabled, Trigger: "new_lead", DealEvent: "updated", ReportTime: settings.ReportTime, ConditionMode: "and", Conditions: []models.TelegramAutomationCondition{}, Action: "telegram_notification", Template: settings.Templates.NewLead},
		{ID: "legacy-daily-report", Name: "Kunlik hisobot", Enabled: settings.DailyReportEnabled, Trigger: "daily_report", DealEvent: "updated", ReportTime: settings.ReportTime, ConditionMode: "and", Conditions: []models.TelegramAutomationCondition{}, Action: "telegram_notification", Template: settings.Templates.DailyReport},
	}
	for _, status := range settings.StatusNotifications {
		conditions := []models.TelegramAutomationCondition{}
		if status.PipelineID != "" {
			conditions = append(conditions, models.TelegramAutomationCondition{ID: status.ID + "-pipeline", Field: "pipeline", Operator: "eq", Value: status.PipelineID})
		}
		if status.StageID != "" {
			conditions = append(conditions, models.TelegramAutomationCondition{ID: status.ID + "-stage", Field: "stage", Operator: "eq", Value: status.StageID})
		}
		rules = append(rules, models.TelegramAutomation{
			ID: status.ID, Name: status.Name, Enabled: status.Enabled && len(conditions) == 2, Trigger: "deal", DealEvent: "updated", ReportTime: settings.ReportTime, ConditionMode: "and", Action: "telegram_notification", Template: status.Template,
			Conditions: conditions,
		})
	}
	return rules
}

func telegramAutomationMatches(rule models.TelegramAutomation, event string, deal map[string]any, now time.Time) bool {
	if !rule.Enabled || rule.Action != "telegram_notification" {
		return false
	}
	if !telegramAutomationHasEvent(rule, event) {
		return false
	}
	if len(rule.Conditions) == 0 {
		return true
	}
	if rule.ConditionMode == "or" {
		for _, condition := range rule.Conditions {
			if telegramAutomationConditionMatches(condition, deal, now) {
				return true
			}
		}
		return false
	}
	for _, condition := range rule.Conditions {
		if !telegramAutomationConditionMatches(condition, deal, now) {
			return false
		}
	}
	return true
}

func telegramAutomationTriggers(rule models.TelegramAutomation) []string {
	if len(rule.Triggers) > 0 {
		return rule.Triggers
	}
	if rule.Trigger != "" {
		return []string{rule.Trigger}
	}
	return nil
}

func telegramAutomationHasTrigger(rule models.TelegramAutomation, trigger string) bool {
	for _, current := range telegramAutomationTriggers(rule) {
		if current == trigger {
			return true
		}
	}
	return false
}

func telegramAutomationHasEvent(rule models.TelegramAutomation, event string) bool {
	if event == "created" && telegramAutomationHasTrigger(rule, "new_lead") {
		return true
	}
	if event == "daily_report" && telegramAutomationHasTrigger(rule, "daily_report") {
		return true
	}
	return telegramAutomationHasTrigger(rule, "deal") && event == rule.DealEvent
}

func telegramAutomationTemplate(rule models.TelegramAutomation, settings models.TelegramNotificationSettings, event string) string {
	if strings.TrimSpace(rule.Template) != "" {
		return rule.Template
	}
	if event == "daily_report" {
		return settings.Templates.DailyReport
	}
	if event != "created" || telegramAutomationHasTrigger(rule, "deal") && !telegramAutomationHasTrigger(rule, "new_lead") {
		return "🔔 <b>Deal hodisasi</b>\n\n👤 Mijoz: {{contact.name}}\n💵 Summa: {{deal.amount}}\n🔗 {{deal.url}}"
	}
	return settings.Templates.NewLead
}

func telegramAutomationMatchesUpdated(rule models.TelegramAutomation, deal map[string]any, now time.Time, stageChanged bool) bool {
	if stageChanged {
		return telegramAutomationMatches(rule, "updated", deal, now)
	}
	filtered := make([]models.TelegramAutomationCondition, 0, len(rule.Conditions))
	for _, condition := range rule.Conditions {
		if condition.Field != "stage" {
			filtered = append(filtered, condition)
		}
	}
	if len(filtered) == len(rule.Conditions) {
		return telegramAutomationMatches(rule, "updated", deal, now)
	}
	if rule.ConditionMode == "and" || len(filtered) == 0 {
		return false
	}
	rule.Conditions = filtered
	return telegramAutomationMatches(rule, "updated", deal, now)
}

func telegramAutomationConditionMatches(condition models.TelegramAutomationCondition, deal map[string]any, now time.Time) bool {
	var actual string
	switch condition.Field {
	case "pipeline":
		actual = telegramDealPipeline(deal)
	case "stage":
		actual = telegramDealStage(deal)
	case "weekday":
		actual = now.Weekday().String()
	default:
		actual = telegramDealValue(deal, condition.Field)
	}
	if condition.Field == "pipeline" {
		if condition.Operator == "eq" {
			return telegramDealPipelineMatches(condition.Value, deal)
		}
		if condition.Operator == "neq" {
			return !telegramDealPipelineMatches(condition.Value, deal)
		}
	}
	if condition.Field == "stage" {
		if condition.Operator == "eq" {
			return telegramDealStageMatches(condition.Value, deal)
		}
		if condition.Operator == "neq" {
			return !telegramDealStageMatches(condition.Value, deal)
		}
	}
	actual = strings.TrimSpace(actual)
	expected := strings.TrimSpace(condition.Value)
	if condition.Field == "weekday" {
		expected = telegramAutomationWeekday(expected)
	}
	switch condition.Operator {
	case "not_empty":
		return actual != ""
	case "eq":
		return strings.EqualFold(actual, expected)
	case "neq":
		return !strings.EqualFold(actual, expected)
	case "contains":
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
	case "gt", "lt":
		left, leftErr := strconv.ParseFloat(strings.ReplaceAll(actual, " ", ""), 64)
		right, rightErr := strconv.ParseFloat(strings.ReplaceAll(expected, " ", ""), 64)
		if leftErr != nil || rightErr != nil {
			return false
		}
		if condition.Operator == "gt" {
			return left > right
		}
		return left < right
	default:
		return false
	}
}

func telegramAutomationWeekday(value string) string {
	aliases := map[string]string{
		"yakshanba": "Sunday", "dushanba": "Monday", "seshanba": "Tuesday", "chorshanba": "Wednesday",
		"payshanba": "Thursday", "juma": "Friday", "shanba": "Saturday",
	}
	if translated, ok := aliases[strings.ToLower(value)]; ok {
		return translated
	}
	return value
}

func validateTelegramAutomations(rules []models.TelegramAutomation) error {
	if len(rules) > 100 {
		return fmt.Errorf("too many automations")
	}
	ids := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Name) == "" || len(rule.Template) > 4096 {
			return fmt.Errorf("automation name or template is invalid")
		}
		if _, exists := ids[rule.ID]; exists {
			return fmt.Errorf("duplicate automation id")
		}
		ids[rule.ID] = struct{}{}
		if rule.Action != "telegram_notification" || rule.ConditionMode != "and" && rule.ConditionMode != "or" {
			return fmt.Errorf("automation action or condition mode is invalid")
		}
		if len(rule.TriggerConfigs) > 0 {
			if err := validateTelegramAutomationTriggers(rule.TriggerConfigs); err != nil {
				return err
			}
			continue
		}
		triggers := telegramAutomationTriggers(rule)
		if len(triggers) == 0 || len(triggers) > 3 {
			return fmt.Errorf("automation triggers are invalid")
		}
		seen := make(map[string]bool, len(triggers))
		for _, trigger := range triggers {
			if seen[trigger] {
				return fmt.Errorf("duplicate automation trigger")
			}
			seen[trigger] = true
			switch trigger {
			case "new_lead", "daily_report":
			case "deal":
				if rule.DealEvent != "created" && rule.DealEvent != "updated" && rule.DealEvent != "deleted" {
					return fmt.Errorf("deal event is invalid")
				}
			default:
				return fmt.Errorf("automation trigger is invalid")
			}
		}
		if telegramAutomationHasTrigger(rule, "daily_report") {
			if _, err := time.Parse("15:04", rule.ReportTime); err != nil {
				return fmt.Errorf("report time is invalid")
			}
		}
		if len(rule.Conditions) > 20 {
			return fmt.Errorf("too many automation conditions")
		}
		for _, condition := range rule.Conditions {
			if condition.Field == "" || condition.Operator != "not_empty" && strings.TrimSpace(condition.Value) == "" {
				return fmt.Errorf("automation condition is incomplete")
			}
			switch condition.Operator {
			case "eq", "neq", "contains", "gt", "lt", "not_empty":
			default:
				return fmt.Errorf("automation condition operator is invalid")
			}
			if telegramAutomationHasTrigger(rule, "daily_report") && len(triggers) == 1 && condition.Field != "weekday" {
				return fmt.Errorf("daily report condition field is invalid")
			}
		}
	}
	return nil
}

func validateTelegramAutomationTriggers(triggers []models.TelegramAutomationTrigger) error {
	if len(triggers) > 20 {
		return fmt.Errorf("too many automation triggers")
	}
	for _, trigger := range triggers {
		if strings.TrimSpace(trigger.ID) == "" || len(trigger.Template) > 4096 || len(trigger.MessageFields) == 0 {
			return fmt.Errorf("automation trigger is incomplete")
		}
		switch trigger.Kind {
		case "create", "update", "delete", "field":
			if trigger.Table != "deals" && trigger.Table != "contacts" && trigger.Table != "tasks" {
				return fmt.Errorf("automation table is invalid")
			}
		case "daily_report":
			if _, err := time.Parse("15:04", trigger.ReportTime); err != nil {
				return fmt.Errorf("report time is invalid")
			}
		default:
			return fmt.Errorf("automation trigger kind is invalid")
		}
		if trigger.ConditionMode != "and" && trigger.ConditionMode != "or" {
			return fmt.Errorf("automation trigger condition mode is invalid")
		}
		for _, condition := range trigger.Conditions {
			if condition.Field == "" || condition.Operator != "not_empty" && strings.TrimSpace(condition.Value) == "" {
				return fmt.Errorf("automation condition is incomplete")
			}
			switch condition.Operator {
			case "eq", "neq", "contains", "gt", "lt", "not_empty":
			default:
				return fmt.Errorf("automation condition operator is invalid")
			}
		}
	}
	return nil
}
