package v1

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/metaads"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/structpb"
)

var telegramTemplateToken = regexp.MustCompile(`\{\{[^{}]+\}\}`)

const (
	telegramNotificationsResourceName    = "CRM Telegram notifications"
	telegramNotificationsResourceStatus  = "crm_notifications"
	telegramNotificationsSecretKey       = "telegram_notifications_settings"
	telegramNotificationsCodePrefix      = "telegram:notifications:connect:"
	telegramNotificationsCodeTTL         = 10 * time.Minute
	telegramNotificationsTargetsKey      = "telegram:notifications:targets"
	telegramNotificationsDailyLockPrefix = "telegram:notifications:daily:"
)

type telegramNotificationTarget struct {
	ProjectID     string
	EnvironmentID string
	CompanyID     string
}

type telegramNotificationConnectCode struct {
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	CompanyID     string `json:"company_id"`
}

func defaultTelegramNotificationSettings(companyID, botUsername string) models.TelegramNotificationSettings {
	return models.TelegramNotificationSettings{
		CompanyID:          companyID,
		BotUsername:        strings.TrimPrefix(botUsername, "@"),
		NewLeadEnabled:     true,
		DailyReportEnabled: true,
		ReportTime:         "21:00",
		Timezone:           "Asia/Tashkent",
		Templates: models.TelegramNotificationTemplates{
			NewLead:     "🆕 <b>Yangi lid</b>\n\n👤 Ismi: {{lead.name}}\n📞 Telefon: {{lead.phone}}\n📣 Manba: {{lead.source}}\n👨‍💼 Mas’ul: {{lead.owner_name}}\n\n🔗 {{lead.url}}",
			DailyReport: "📋 <b>{{report.company}} · {{report.date}}</b>\n\n📣 <b>Reklama natijalari</b>\n💸 Xarajat: {{report.ad_spend}}\n👥 Jami lidlar: {{report.leads_total}}\n💰 CPL: {{report.cpl}}\n\n{{report.statuses}}",
		},
		StatusNotifications: []models.TelegramStatusNotification{},
	}
}

func (h *HandlerV1) telegramNotificationsConfigured() bool {
	return strings.TrimSpace(h.baseConf.TelegramNotificationsBotToken) != "" &&
		strings.TrimSpace(h.baseConf.TelegramNotificationsBotUsername) != "" &&
		strings.TrimSpace(h.baseConf.TelegramNotificationsWebhookSecret) != ""
}

// StartTelegramNotifications registers the global bot webhook at startup.
// Group routing starts only after a group sends its one-time connect command.
func (h *HandlerV1) StartTelegramNotifications(ctx context.Context) {
	if !h.telegramNotificationsConfigured() {
		return
	}
	webhookURL := fmt.Sprintf("%s/v1/telegram-notifications/webhook", strings.TrimRight(h.baseConf.TelegramWebhookBaseURL, "/"))
	if err := newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).setWebhook(ctx, webhookURL, h.baseConf.TelegramNotificationsWebhookSecret, []string{"message", "callback_query"}); err != nil {
		h.log.Error("telegram notifications: set webhook failed", logger.Error(err))
	}
	h.startTelegramDailyReportScheduler(ctx)
}

func (h *HandlerV1) telegramNotificationTarget(c *gin.Context) (telegramNotificationTarget, bool) {
	projectID, projectOK := c.Get("project_id")
	environmentID, environmentOK := c.Get("environment_id")
	companyID := strings.TrimSpace(c.Query("company-id"))
	if !projectOK || !environmentOK || companyID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "project, environment and company-id are required")
		return telegramNotificationTarget{}, false
	}
	return telegramNotificationTarget{ProjectID: fmt.Sprint(projectID), EnvironmentID: fmt.Sprint(environmentID), CompanyID: companyID}, true
}

func (h *HandlerV1) GetTelegramNotificationSettings(c *gin.Context) {
	target, ok := h.telegramNotificationTarget(c)
	if !ok {
		return
	}
	settings, _, err := h.getTelegramNotificationSettings(c.Request.Context(), target)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if settings.Automations == nil {
		settings.Automations = telegramLegacyAutomations(settings)
	}
	normalizeTelegramNotificationGroups(&settings)
	h.registerTelegramNotificationTarget(c.Request.Context(), target)
	h.HandleResponse(c, status_http.OK, settings)
}

func (h *HandlerV1) SaveTelegramNotificationSettings(c *gin.Context) {
	target, ok := h.telegramNotificationTarget(c)
	if !ok {
		return
	}
	var settings models.TelegramNotificationSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if settings.CompanyID != "" && settings.CompanyID != target.CompanyID {
		h.HandleResponse(c, status_http.Forbidden, "company id cannot be changed")
		return
	}
	settings.CompanyID = target.CompanyID
	settings.BotUsername = strings.TrimPrefix(h.baseConf.TelegramNotificationsBotUsername, "@")
	normalizeTelegramNotificationGroups(&settings)
	if err := validateTelegramNotificationSettings(&settings); err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	if err := h.saveTelegramNotificationSettings(c.Request.Context(), target, settings); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.registerTelegramNotificationTarget(c.Request.Context(), target)
	h.HandleResponse(c, status_http.OK, settings)
}

func (h *HandlerV1) CreateTelegramNotificationConnectCode(c *gin.Context) {
	if !h.telegramNotificationsConfigured() {
		h.HandleResponse(c, status_http.BadRequest, "telegram notifications bot is not configured")
		return
	}
	target, ok := h.telegramNotificationTarget(c)
	if !ok {
		return
	}
	code, err := telegramRandomToken(12)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	body, err := json.Marshal(telegramNotificationConnectCode{ProjectID: target.ProjectID, EnvironmentID: target.EnvironmentID, CompanyID: target.CompanyID})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if h.centralRedis == nil {
		h.HandleResponse(c, status_http.GRPCError, "central redis is not configured")
		return
	}
	if err = h.centralRedis.Set(c.Request.Context(), telegramNotificationsCodePrefix+code, body, telegramNotificationsCodeTTL).Err(); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"code": code, "bot_username": strings.TrimPrefix(h.baseConf.TelegramNotificationsBotUsername, "@"), "expires_at": time.Now().Add(telegramNotificationsCodeTTL).UTC().Format(time.RFC3339)})
}

func (h *HandlerV1) TelegramNotificationsWebhook(c *gin.Context) {
	if !h.telegramNotificationsConfigured() || subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Telegram-Bot-Api-Secret-Token")), []byte(h.baseConf.TelegramNotificationsWebhookSecret)) != 1 {
		c.Status(http.StatusUnauthorized)
		return
	}
	var update telegramUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.Status(http.StatusOK)
		return
	}
	if update.CallbackQuery != nil {
		h.handleTelegramAutomationCallback(c.Request.Context(), update.CallbackQuery)
		c.Status(http.StatusOK)
		return
	}
	if update.Message == nil {
		c.Status(http.StatusOK)
		return
	}
	message := update.Message
	if message.Chat.Type != "group" && message.Chat.Type != "supergroup" {
		c.Status(http.StatusOK)
		return
	}
	code := telegramNotificationCode(message.Text)
	if code == "" || h.centralRedis == nil {
		c.Status(http.StatusOK)
		return
	}
	body, err := h.centralRedis.Get(c.Request.Context(), telegramNotificationsCodePrefix+code).Bytes()
	if err != nil {
		c.Status(http.StatusOK)
		return
	}
	var connect telegramNotificationConnectCode
	if err = json.Unmarshal(body, &connect); err != nil {
		c.Status(http.StatusOK)
		return
	}
	target := telegramNotificationTarget{ProjectID: connect.ProjectID, EnvironmentID: connect.EnvironmentID, CompanyID: connect.CompanyID}
	settings, _, err := h.getTelegramNotificationSettings(c.Request.Context(), target)
	if err == nil {
		chatID, chatTitle := fmt.Sprint(message.Chat.ID), strings.TrimSpace(message.Chat.Title)
		settings.ChatID, settings.ChatTitle = chatID, chatTitle
		found := false
		for index := range settings.Groups {
			if settings.Groups[index].ChatID == chatID {
				settings.Groups[index].ChatTitle, found = chatTitle, true
			}
		}
		if !found {
			settings.Groups = append(settings.Groups, models.TelegramNotificationGroup{ChatID: chatID, ChatTitle: chatTitle})
		}
		settings.BotUsername = strings.TrimPrefix(h.baseConf.TelegramNotificationsBotUsername, "@")
		err = h.saveTelegramNotificationSettings(c.Request.Context(), target, settings)
	}
	if err == nil {
		_ = h.centralRedis.Del(c.Request.Context(), telegramNotificationsCodePrefix+code).Err()
		h.registerTelegramNotificationTarget(c.Request.Context(), target)
		_, _ = newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(c.Request.Context(), settings.ChatID, "✅ <b>CRM Telegram guruhi ulandi.</b>\nEndi shu kompaniyaning notificationlari shu guruhga yuboriladi.")
	}
	c.Status(http.StatusOK)
}

func (h *HandlerV1) SendTelegramNotificationTest(c *gin.Context) {
	target, ok := h.telegramNotificationTarget(c)
	if !ok {
		return
	}
	var request models.TelegramNotificationTestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	settings, _, err := h.getTelegramNotificationSettings(c.Request.Context(), target)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if settings.ChatID == "" {
		h.HandleResponse(c, status_http.BadRequest, "telegram group is not connected")
		return
	}
	template, err := telegramNotificationTemplate(settings, request.Type, request.RuleID)
	if err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	message := renderTelegramNotificationTemplate(template)
	isDailyReport := request.Type == "daily_report"
	if request.Type == "automation" {
		for _, rule := range settings.Automations {
			if rule.ID == request.RuleID && telegramAutomationHasTrigger(rule, "daily_report") {
				isDailyReport = true
				message, err = h.telegramDailyReportMessageWithTemplate(c.Request.Context(), target, settings, time.Now(), template)
				if err != nil {
					h.HandleResponse(c, status_http.GRPCError, err.Error())
					return
				}
				break
			}
		}
	}
	if !isDailyReport {
		// Use a real deal for the test message so its CRM link can be opened,
		// rather than rendering the old non-clickable preview label.
		if deal, dealErr := h.telegramNotificationTestDeal(c.Request.Context(), target); dealErr == nil {
			message = renderTelegramNotificationTemplateWithDeal(template, deal)
		}
	}
	if _, err = newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(c.Request.Context(), settings.ChatID, message); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"sent": true})
}

func (h *HandlerV1) SendTelegramDailyReport(c *gin.Context) {
	target, ok := h.telegramNotificationTarget(c)
	if !ok {
		return
	}
	settings, _, err := h.getTelegramNotificationSettings(c.Request.Context(), target)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if settings.ChatID == "" {
		h.HandleResponse(c, status_http.BadRequest, "telegram group is not connected")
		return
	}
	message, err := h.telegramDailyReportMessage(c.Request.Context(), target, settings, time.Now())
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if _, err = newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(c.Request.Context(), settings.ChatID, message); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"sent": true})
}

// startTelegramDailyReportScheduler checks once a minute because each company
// chooses its own local report time. Targets are kept in central Redis so every
// replica sees the same configuration; a per-company/day lock prevents a
// duplicated report when several replicas are running.
func (h *HandlerV1) startTelegramDailyReportScheduler(ctx context.Context) {
	if h.centralRedis == nil {
		h.log.Warn("telegram notifications: daily scheduler needs central redis")
		return
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.runTelegramDailyReports()
			}
		}
	}()
}

func (h *HandlerV1) runTelegramDailyReports() {
	targets, err := h.centralRedis.SMembers(context.Background(), telegramNotificationsTargetsKey).Result()
	if err != nil {
		h.log.Warn("telegram notifications: read daily report targets failed", logger.Error(err))
		return
	}
	candidates := make([]telegramScheduledReport, 0, len(targets))
	modernChats := make(map[string]bool)
	for _, encodedTarget := range targets {
		target, ok := parseTelegramNotificationTarget(encodedTarget)
		if !ok {
			continue
		}
		settings, _, err := h.getTelegramNotificationSettings(context.Background(), target)
		if err != nil || strings.TrimSpace(settings.ChatID) == "" {
			continue
		}
		location, err := time.LoadLocation(settings.Timezone)
		if err != nil {
			location = time.UTC
		}
		now := time.Now().In(location)
		if settings.Automations != nil {
			for _, rule := range settings.Automations {
				ruleSettings := settings
				if strings.TrimSpace(rule.ChatID) != "" {
					ruleSettings.ChatID = strings.TrimSpace(rule.ChatID)
				}
				if len(rule.TriggerConfigs) > 0 {
					for _, trigger := range rule.TriggerConfigs {
						if trigger.Kind == "daily_report" {
							modernChats[ruleSettings.ChatID] = true
						}
						if trigger.Kind != "daily_report" || now.Format("15:04") != trigger.ReportTime {
							continue
						}
						if _, ok := telegramAutomationMatchingTrigger(rule, "", "daily_report", nil, nil, now); ok {
							candidates = append(candidates, telegramScheduledReport{target, ruleSettings, telegramAutomationTriggerTemplate(trigger), now, true})
						}
					}
					continue
				}
				if !telegramAutomationHasTrigger(rule, "daily_report") || now.Format("15:04") != rule.ReportTime || !telegramAutomationMatches(rule, "daily_report", nil, now) {
					continue
				}
				candidates = append(candidates, telegramScheduledReport{target, ruleSettings, telegramAutomationTemplate(rule, settings, "daily_report"), now, false})
			}
			continue
		}
		if !settings.DailyReportEnabled || now.Format("15:04") != settings.ReportTime {
			continue
		}
		candidates = append(candidates, telegramScheduledReport{target, settings, settings.Templates.DailyReport, now, false})
	}
	for _, candidate := range telegramReportsForChats(candidates, modernChats) {
		h.sendScheduledTelegramReport(candidate)
	}
}

func (h *HandlerV1) sendScheduledTelegramReport(report telegramScheduledReport) {
	lockKey := telegramNotificationsDailyLockPrefix + "chat:" + strings.TrimSpace(report.settings.ChatID) + ":" + report.now.Format("2006-01-02")
	locked, err := h.centralRedis.SetNX(context.Background(), lockKey, "sending", 36*time.Hour).Result()
	if err != nil || !locked {
		return
	}
	message, err := h.telegramDailyReportMessageWithTemplate(context.Background(), report.target, report.settings, report.now, report.template)
	if err != nil {
		_ = h.centralRedis.Del(context.Background(), lockKey).Err()
		h.log.Error("telegram notifications: daily report build failed", logger.Error(err))
		return
	}
	if _, err = newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(context.Background(), report.settings.ChatID, message); err != nil {
		_ = h.centralRedis.Del(context.Background(), lockKey).Err()
		h.log.Error("telegram notifications: daily report send failed", logger.Error(err))
	}
}

func (h *HandlerV1) registerTelegramNotificationTarget(ctx context.Context, target telegramNotificationTarget) {
	if h.centralRedis == nil || target.ProjectID == "" || target.EnvironmentID == "" || target.CompanyID == "" {
		return
	}
	if err := h.centralRedis.SAdd(ctx, telegramNotificationsTargetsKey, encodeTelegramNotificationTarget(target)).Err(); err != nil {
		h.log.Warn("telegram notifications: register daily report target failed", logger.Error(err))
	}
}

func encodeTelegramNotificationTarget(target telegramNotificationTarget) string {
	return target.ProjectID + "|" + target.EnvironmentID + "|" + target.CompanyID
}

func parseTelegramNotificationTarget(value string) (telegramNotificationTarget, bool) {
	parts := strings.Split(value, "|")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return telegramNotificationTarget{}, false
	}
	return telegramNotificationTarget{ProjectID: parts[0], EnvironmentID: parts[1], CompanyID: parts[2]}, true
}

func (h *HandlerV1) telegramDailyReportMessage(ctx context.Context, target telegramNotificationTarget, settings models.TelegramNotificationSettings, now time.Time) (string, error) {
	return h.telegramDailyReportMessageWithTemplate(ctx, target, settings, now, settings.Templates.DailyReport)
}

func (h *HandlerV1) telegramDailyReportMessageWithTemplate(ctx context.Context, target telegramNotificationTarget, settings models.TelegramNotificationSettings, now time.Time, template string) (string, error) {
	location, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		location = time.UTC
	}
	day := now.In(location)
	metaReport, err := metaads.NewHandler(h.baseConf, h.centralRedis, h.log).DashboardForDay(ctx, day)
	if err != nil {
		return "", fmt.Errorf("Meta Ads report: %w", err)
	}
	statuses, err := h.telegramDealStatusesForDay(ctx, target, day)
	if err != nil {
		h.log.Warn("telegram notifications: CRM status report unavailable", logger.Error(err))
		statuses = "CRM statuslari vaqtincha olinmadi."
	}
	currency := strings.TrimSpace(metaReport.Account.Currency)
	if currency == "" {
		currency = "so‘m"
	}
	cpl := "—"
	if metaReport.KPIs.CPL != nil {
		cpl = telegramReportMoney(*metaReport.KPIs.CPL, currency)
	}
	values := map[string]string{
		"{{report.company}}":     "CRM",
		"{{report.date}}":        day.Format("02.01.2006"),
		"{{report.ad_spend}}":    telegramReportMoney(metaReport.KPIs.Spend, currency),
		"{{report.leads_total}}": fmt.Sprint(metaReport.KPIs.Leads),
		"{{report.cpl}}":         cpl,
		"{{report.statuses}}":    statuses,
	}
	return renderTelegramTemplateValues(template, values), nil
}

func (h *HandlerV1) telegramDealStatusesForDay(ctx context.Context, target telegramNotificationTarget, day time.Time) (string, error) {
	service, environmentID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return "", err
	}
	response, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
		TableSlug: "deals",
		Data:      mustStruct(map[string]any{"limit": 10000, "offset": 0}),
		ProjectId: environmentID,
	})
	if err != nil {
		return "", err
	}
	counts := map[string]int{}
	for _, row := range telegramResponseRows(response.GetData()) {
		// target.CompanyID identifies the CRM workspace whose Telegram group is
		// configured. A deal's companies_id, on the other hand, is the customer
		// company related to that deal. They are unrelated identifiers, so the
		// report must include every deal in this workspace.
		createdAt, ok := telegramDealCreatedAt(row, day.Location())
		if !ok || createdAt.Format("2006-01-02") != day.Format("2006-01-02") {
			continue
		}
		status := telegramDealStage(row)
		if status == "" {
			status = "Status belgilanmagan"
		}
		counts[status]++
	}
	if len(counts) == 0 {
		return "Bugun kelgan lidlar topilmadi.", nil
	}
	keys := make([]string, 0, len(counts))
	for status := range counts {
		keys = append(keys, status)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, status := range keys {
		lines = append(lines, "• "+status+" — "+fmt.Sprint(counts[status]))
	}
	return strings.Join(lines, "\n"), nil
}

func telegramResponseRows(data *structpb.Struct) []map[string]any {
	if data == nil {
		return nil
	}
	rawRows, _ := data.AsMap()["response"].([]any)
	rows := make([]map[string]any, 0, len(rawRows))
	for _, raw := range rawRows {
		if row, ok := raw.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func telegramDealCreatedAt(deal map[string]any, location *time.Location) (time.Time, bool) {
	for _, key := range []string{"created_at", "created_time", "start_date"} {
		value := telegramDealValue(deal, key)
		if value == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
				return parsed.In(location), true
			}
		}
	}
	return time.Time{}, false
}

func telegramReportMoney(value float64, currency string) string {
	return fmt.Sprintf("%s %.2f", currency, value)
}

func renderTelegramTemplateValues(template string, values map[string]string) string {
	for token, value := range values {
		template = strings.ReplaceAll(template, token, html.EscapeString(value))
	}
	return template
}

func (h *HandlerV1) getTelegramNotificationSettings(ctx context.Context, target telegramNotificationTarget) (models.TelegramNotificationSettings, *pb.ProjectResource, error) {
	settings := defaultTelegramNotificationSettings(target.CompanyID, h.baseConf.TelegramNotificationsBotUsername)
	resource, err := h.telegramNotificationResource(ctx, target)
	if err != nil || resource == nil {
		return settings, resource, err
	}
	if raw := resource.GetSecret().GetFields()[telegramNotificationsSecretKey]; raw != nil {
		if err = json.Unmarshal([]byte(raw.GetStringValue()), &settings); err != nil {
			return settings, resource, fmt.Errorf("decode telegram notification settings: %w", err)
		}
	}
	settings.CompanyID = target.CompanyID
	settings.BotUsername = strings.TrimPrefix(h.baseConf.TelegramNotificationsBotUsername, "@")
	normalizeTelegramNotificationGroups(&settings)
	return settings, resource, nil
}

func normalizeTelegramNotificationGroups(settings *models.TelegramNotificationSettings) {
	seen := make(map[string]bool)
	groups := make([]models.TelegramNotificationGroup, 0, len(settings.Groups)+1)
	for _, group := range settings.Groups {
		group.ChatID, group.ChatTitle = strings.TrimSpace(group.ChatID), strings.TrimSpace(group.ChatTitle)
		if group.ChatID == "" || seen[group.ChatID] {
			continue
		}
		seen[group.ChatID] = true
		groups = append(groups, group)
	}
	if chatID := strings.TrimSpace(settings.ChatID); chatID != "" && !seen[chatID] {
		groups = append(groups, models.TelegramNotificationGroup{ChatID: chatID, ChatTitle: strings.TrimSpace(settings.ChatTitle)})
	}
	settings.Groups = groups
	for index := range settings.Automations {
		if strings.TrimSpace(settings.Automations[index].ChatID) == "" {
			settings.Automations[index].ChatID = strings.TrimSpace(settings.ChatID)
		}
	}
}

func (h *HandlerV1) saveTelegramNotificationSettings(ctx context.Context, target telegramNotificationTarget, settings models.TelegramNotificationSettings) error {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	secret, err := structpb.NewStruct(map[string]any{telegramNotificationsSecretKey: string(encoded)})
	if err != nil {
		return err
	}
	resource, err := h.telegramNotificationResource(ctx, target)
	if err != nil {
		return err
	}
	if resource == nil {
		_, err = h.companyServices.Resource().AddResourceToProject(ctx, &pb.AddResourceToProjectRequest{Name: telegramNotificationsResourceName, ProjectId: target.ProjectID, EnvironmentId: target.EnvironmentID, Type: pb.ResourceType_TELEGRAM, ExternalId: target.CompanyID, Settings: telegramNotificationResourceSettings(h.baseConf.TelegramNotificationsBotUsername), Secret: secret})
		return err
	}
	// The company service treats the secret as immutable on UpdateProjectResource.
	// Telegram settings (including the status rules) live in that secret, so use
	// its upsert endpoint for existing resources as well. It updates the same
	// project/environment/external-id resource without changing the connection.
	_, err = h.companyServices.Resource().UpsertProjectResource(ctx, &pb.AddResourceToProjectRequest{
		Name:          resource.GetName(),
		ProjectId:     target.ProjectID,
		EnvironmentId: target.EnvironmentID,
		Type:          pb.ResourceType_TELEGRAM,
		ExternalId:    target.CompanyID,
		Settings:      telegramNotificationResourceSettings(h.baseConf.TelegramNotificationsBotUsername),
		Secret:        secret,
	})
	return err
}

func (h *HandlerV1) telegramNotificationResource(ctx context.Context, target telegramNotificationTarget) (*pb.ProjectResource, error) {
	list, err := h.companyServices.Resource().GetProjectResourceList(ctx, &pb.GetProjectResourceListRequest{ProjectId: target.ProjectID, EnvironmentId: target.EnvironmentID, Type: pb.ResourceType_TELEGRAM})
	if err != nil {
		return nil, err
	}
	for _, resource := range list.GetResources() {
		if resource.GetExternalId() == target.CompanyID && resource.GetSettings().GetTelegram().GetStatus() == telegramNotificationsResourceStatus {
			return resource, nil
		}
	}
	return nil, nil
}

// telegramNotificationTargets returns the CRM workspaces in a project and
// environment that have Telegram settings. Settings are keyed by the workspace
// company, while deals contain the *customer* company in companies_id; using
// the latter to find a group silently drops messages for ordinary deals.
func (h *HandlerV1) telegramNotificationTargets(ctx context.Context, projectID, environmentID string) ([]telegramNotificationTarget, error) {
	list, err := h.companyServices.Resource().GetProjectResourceList(ctx, &pb.GetProjectResourceListRequest{
		ProjectId:     projectID,
		EnvironmentId: environmentID,
		Type:          pb.ResourceType_TELEGRAM,
	})
	if err != nil {
		return nil, err
	}
	targets := make([]telegramNotificationTarget, 0, len(list.GetResources()))
	for _, resource := range list.GetResources() {
		if resource.GetSettings().GetTelegram().GetStatus() != telegramNotificationsResourceStatus || strings.TrimSpace(resource.GetExternalId()) == "" {
			continue
		}
		targets = append(targets, telegramNotificationTarget{
			ProjectID: projectID, EnvironmentID: environmentID, CompanyID: resource.GetExternalId(),
		})
	}
	return targets, nil
}

func telegramNotificationResourceSettings(username string) *pb.Settings {
	return &pb.Settings{Telegram: &pb.TelegramCredentials{BotUsername: strings.TrimPrefix(username, "@"), Status: telegramNotificationsResourceStatus}}
}

func validateTelegramNotificationSettings(settings *models.TelegramNotificationSettings) error {
	if settings.Automations != nil {
		if err := validateTelegramAutomations(settings.Automations); err != nil {
			return err
		}
	}
	if len(settings.StatusNotifications) > 100 {
		return errors.New("too many status notifications")
	}
	if len(settings.Templates.NewLead) > 4096 || len(settings.Templates.DailyReport) > 4096 {
		return errors.New("notification template is too long")
	}
	for _, rule := range settings.StatusNotifications {
		if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Name) == "" || len(rule.Template) > 4096 {
			return errors.New("status notification is invalid")
		}
	}
	return nil
}

func telegramNotificationCode(text string) string {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "/connect") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func telegramNotificationTemplate(settings models.TelegramNotificationSettings, kind, ruleID string) (string, error) {
	switch kind {
	case "new_lead":
		return settings.Templates.NewLead, nil
	case "daily_report":
		return settings.Templates.DailyReport, nil
	case "status_notification":
		for _, rule := range settings.StatusNotifications {
			if rule.ID == ruleID {
				return rule.Template, nil
			}
		}
	case "automation":
		for _, rule := range settings.Automations {
			if rule.ID == ruleID {
				if telegramAutomationHasTrigger(rule, "daily_report") {
					return telegramAutomationTemplate(rule, settings, "daily_report"), nil
				}
				return telegramAutomationTemplate(rule, settings, "created"), nil
			}
		}
	}
	return "", errors.New("notification template was not found")
}

func renderTelegramNotificationTemplate(template string) string {
	replacer := strings.NewReplacer(
		"{{lead.name}}", "Azizbek Karimov", "{{lead.phone}}", "+998 90 123 45 67", "{{lead.source}}", "Instagram", "{{lead.owner_name}}", "Madina", "{{lead.url}}", `<a href="https://crm.ucode.co/deals">CRMda ochish</a>`,
		"{{contact.name}}", "Azizbek Karimov", "{{contact.phone}}", "+998 90 123 45 67", "{{deal.amount}}", "1 200 000 so‘m", "{{deal.owner_name}}", "Madina", "{{deal.url}}", `<a href="https://crm.ucode.co/deals">CRMda ochish</a>`, "{{deal.service}}", "IELTS kursi",
		"{{report.company}}", "PROFESSIONAL CRM", "{{report.date}}", time.Now().Format("02.01.2006"), "{{report.ad_spend}}", "600 000 so‘m", "{{report.leads_total}}", "40", "{{report.cpl}}", "20 000 so‘m", "{{report.statuses}}", "🆕 Yangi — 12\n📞 Bog‘lanildi — 15\n✅ Success deal — 3",
	)
	return replacer.Replace(template)
}

func (h *HandlerV1) telegramNotificationTestDeal(ctx context.Context, target telegramNotificationTarget) (map[string]any, error) {
	service, environmentID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return nil, err
	}
	response, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
		TableSlug: "deals",
		Data:      mustStruct(map[string]any{"limit": 1, "offset": 0}),
		ProjectId: environmentID,
	})
	if err != nil {
		return nil, err
	}
	for _, deal := range telegramResponseRows(response.GetData()) {
		if telegramDealValue(deal, "guid", "id") != "" {
			return deal, nil
		}
	}
	return nil, errors.New("no deal available for Telegram notification test")
}

// NotifyDealCreated is called only after the generic item handler has created
// a deal successfully. The delivery group belongs to the CRM workspace, not to
// the customer company attached to an individual deal.
func (h *HandlerV1) NotifyDealCreated(ctx context.Context, projectID, environmentID string, deal map[string]any) {
	if !h.telegramNotificationsConfigured() {
		return
	}
	targets, err := h.telegramNotificationTargets(ctx, projectID, environmentID)
	if err != nil {
		return
	}
	settingsList := make([]models.TelegramNotificationSettings, 0, len(targets))
	for _, target := range targets {
		settings, _, err := h.getTelegramNotificationSettings(ctx, target)
		if err != nil || strings.TrimSpace(settings.ChatID) == "" {
			continue
		}
		settingsList = append(settingsList, settings)
	}
	for _, delivery := range telegramCreateDeliveries(settingsList, deal, time.Now()) {
		h.sendTelegramCRMNotification(delivery.ChatID, delivery.Message)
	}
}

func (h *HandlerV1) NotifyDealUpdated(ctx context.Context, projectID, environmentID string, deal map[string]any, stageChanged bool) {
	h.notifyDealAutomations(ctx, projectID, environmentID, deal, "updated", stageChanged)
}

func (h *HandlerV1) NotifyDealDeleted(ctx context.Context, projectID, environmentID string, deal map[string]any) {
	h.notifyDealAutomations(ctx, projectID, environmentID, deal, "deleted", false)
}

func (h *HandlerV1) NotifyItemCreated(ctx context.Context, projectID, environmentID, table string, item map[string]any) {
	if table == "deals" {
		h.NotifyDealCreated(ctx, projectID, environmentID, item)
		return
	}
	h.notifyItemAutomations(ctx, projectID, environmentID, table, item, "create", nil, false)
}

func (h *HandlerV1) NotifyItemUpdated(ctx context.Context, projectID, environmentID, table string, item, changedFields map[string]any) {
	stageChanged := table == "deals" && telegramStageFieldChanged(changedFields)
	h.notifyItemAutomations(ctx, projectID, environmentID, table, item, "update", changedFields, stageChanged)
	if table == "deals" {
		h.notifyDealAutomations(ctx, projectID, environmentID, item, "updated", stageChanged)
	}
}

func (h *HandlerV1) NotifyItemDeleted(ctx context.Context, projectID, environmentID, table string, item map[string]any) {
	h.notifyItemAutomations(ctx, projectID, environmentID, table, item, "delete", nil, false)
	if table == "deals" {
		h.notifyDealAutomations(ctx, projectID, environmentID, item, "deleted", false)
	}
}

func telegramStageFieldChanged(changedFields map[string]any) bool {
	for _, key := range []string{"stage", "stage_id", "stageId", "status", "status_id"} {
		if _, ok := changedFields[key]; ok {
			return true
		}
	}
	return false
}

// NotifyDealStatusChanged is called by the item handler only when its request
// contains the stage field. That is more reliable than comparing a legacy
// builder response (whose before-data shape differs by storage engine).
func (h *HandlerV1) NotifyDealStatusChanged(ctx context.Context, projectID, environmentID string, before, after map[string]any) {
	if !h.telegramNotificationsConfigured() {
		return
	}
	targets, err := h.telegramNotificationTargets(ctx, projectID, environmentID)
	if err != nil {
		return
	}
	for _, target := range targets {
		settings, _, err := h.getTelegramNotificationSettings(ctx, target)
		if err != nil || strings.TrimSpace(settings.ChatID) == "" {
			continue
		}
		if settings.Automations != nil {
			continue
		}
		for _, rule := range settings.StatusNotifications {
			if !rule.Enabled || !telegramStatusRuleMatches(rule, after) {
				continue
			}
			h.sendTelegramCRMNotification(settings.ChatID, renderTelegramNotificationTemplateWithDeal(rule.Template, after))
		}
	}
}

func (h *HandlerV1) sendTelegramCRMNotification(chatID, message string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(ctx, chatID, message); err != nil {
			h.log.Error("telegram CRM notification send failed", logger.Error(err))
		}
	}()
}

func telegramNotificationCompanyID(data map[string]any) string {
	return telegramDealValue(data, "companies_id", "company_id", "company")
}

func telegramDealPipeline(data map[string]any) string {
	return telegramDealValue(data, "pipeline", "pipeline_id")
}

func telegramDealStage(data map[string]any) string {
	if stage := telegramDealValue(data, "stage", "stage_id", "status"); stage != "" {
		return stage
	}
	// Each CRM pipeline may store its current stage under a dedicated field
	// such as pipeline_udevs or pipeline_uhrms rather than a shared stage key.
	for key, value := range data {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if strings.HasPrefix(normalizedKey, "pipeline_") || strings.HasPrefix(normalizedKey, "stage_") {
			if stage := telegramValueString(value); stage != "" {
				return stage
			}
		}
	}
	return ""
}

func telegramStatusRuleMatches(rule models.TelegramStatusNotification, deal map[string]any) bool {
	return strings.TrimSpace(rule.PipelineID) != "" && strings.TrimSpace(rule.StageID) != "" &&
		telegramDealPipelineMatches(rule.PipelineID, deal) &&
		telegramDealStageMatches(rule.StageID, deal)
}

func telegramDealPipelineMatches(expected string, deal map[string]any) bool {
	if telegramStatusValueMatches(expected, telegramDealPipeline(deal)) {
		return true
	}
	for key := range deal {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if strings.HasPrefix(normalizedKey, "pipeline_") &&
			telegramStatusValueMatches(expected, strings.TrimPrefix(normalizedKey, "pipeline_")) {
			return true
		}
	}
	return false
}

func telegramDealStageMatches(expected string, deal map[string]any) bool {
	if telegramStatusValueMatches(expected, telegramDealValue(deal, "stage", "stage_id", "status")) {
		return true
	}
	for key, value := range deal {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if (strings.HasPrefix(normalizedKey, "pipeline_") || strings.HasPrefix(normalizedKey, "stage_")) &&
			telegramStatusValueMatches(expected, telegramValueString(value)) {
			return true
		}
	}
	return false
}

// Some legacy CRM stages have an older display spelling ("Выграно") while
// the settings resource stores the corrected value ("Выиграно"). Normalize
// that known alias before comparing a saved notification rule to a deal event.
func telegramStatusValueMatches(expected, actual string) bool {
	normalize := func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		return strings.ReplaceAll(value, "выиграно", "выграно")
	}
	return normalize(expected) == normalize(actual)
}

func telegramDealValue(data map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		if normalized := telegramValueString(value); normalized != "" {
			return normalized
		}
	}
	// Custom fields are created per CRM and their keys are not always
	// consistently cased or separated. For example, a form can return
	// `phone-number` while another returns `phone_number`. Resolve these to the
	// same notification field without requiring every company to edit a template.
	for _, key := range keys {
		wanted := telegramFieldKey(key)
		for actualKey, value := range data {
			if telegramFieldKey(actualKey) != wanted {
				continue
			}
			if normalized := telegramValueString(value); normalized != "" {
				return normalized
			}
		}
	}
	return ""
}

func telegramFieldKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.NewReplacer("_", "", "-", "", " ", "", ".", "").Replace(key)
}

// Builder fields may be returned as scalar strings, array-backed choices, or
// relation-like objects. Read the user-facing value consistently in all three
// formats so a rule selected in the settings UI matches the saved deal.
func telegramValueString(value any) string {
	switch item := value.(type) {
	case string:
		return strings.TrimSpace(item)
	case []string:
		for _, child := range item {
			if normalized := telegramValueString(child); normalized != "" {
				return normalized
			}
		}
	case []any:
		for _, child := range item {
			if normalized := telegramValueString(child); normalized != "" {
				return normalized
			}
		}
	case map[string]any:
		for _, key := range []string{"label", "name", "title", "value", "guid", "id"} {
			if normalized := telegramValueString(item[key]); normalized != "" {
				return normalized
			}
		}
	default:
		if normalized := strings.TrimSpace(fmt.Sprint(item)); normalized != "" && normalized != "<nil>" {
			return normalized
		}
	}
	return ""
}

func renderTelegramNotificationTemplateWithDeal(template string, deal map[string]any) string {
	values := map[string]string{
		"{{lead.name}}":       telegramDealValue(deal, "name", "full_name", "contact_name"),
		"{{lead.phone}}":      telegramDealValue(deal, "phone", "phone_number", "contact_phone", "telephone", "telefon", "mobile", "mobile_phone"),
		"{{lead.source}}":     telegramDealValue(deal, "source", "lead_source", "lead_channel", "manba"),
		"{{lead.owner_name}}": telegramDealValue(deal, "owner_name", "responsible", "responsible_name", "assignee_name", "manager_name", "assigned_to"),
		"{{lead.url}}":        telegramDealURL(deal),
		"{{contact.name}}":    telegramDealValue(deal, "contact_name", "name", "full_name"),
		"{{contact.phone}}":   telegramDealValue(deal, "contact_phone", "phone", "phone_number", "telephone", "telefon", "mobile", "mobile_phone"),
		"{{deal.amount}}":     telegramDealValue(deal, "amount", "sum", "price", "budget", "total", "deal_amount"),
		"{{deal.owner_name}}": telegramDealValue(deal, "owner_name", "responsible", "responsible_name", "assignee_name", "manager_name", "assigned_to"),
		"{{deal.service}}":    telegramDealValue(deal, "service", "product", "service_name", "product_name", "xizmat", "xizmat_nomi", "course", "direction", "deal_type"),
		"{{deal.url}}":        telegramDealURL(deal),
	}
	for key, value := range deal {
		values["{{deal."+key+"}}"] = telegramDealValue(map[string]any{key: value}, key)
		values["{{item."+key+"}}"] = telegramDealValue(map[string]any{key: value}, key)
	}

	// Fields such as service, amount and responsible person are optional in a
	// CRM. Drop the whole field line when its value is absent instead of sending
	// a misleading `—` placeholder to Telegram. This keeps a new lead/status
	// notification compact while still showing every value that exists.
	lines := make([]string, 0, strings.Count(template, "\n")+1)
	for _, line := range strings.Split(template, "\n") {
		missingValue := false
		for _, token := range telegramTemplateToken.FindAllString(line, -1) {
			if strings.TrimSpace(values[token]) == "" {
				missingValue = true
				break
			}
		}
		if missingValue {
			continue
		}
		for token, value := range values {
			if value != "" {
				if token == "{{lead.url}}" || token == "{{deal.url}}" {
					line = strings.ReplaceAll(line, token, `<a href="`+html.EscapeString(value)+`">CRMda ochish</a>`)
					continue
				}
				if token == "{{item.name}}" || token == "{{item.full_name}}" || token == "{{deal.name}}" || token == "{{lead.name}}" || token == "{{contact.name}}" {
					line = strings.ReplaceAll(line, token, "<b>"+html.EscapeString(value)+"</b>")
					continue
				}
				line = strings.ReplaceAll(line, token, html.EscapeString(value))
			}
		}
		lines = append(lines, line)
	}
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return result
}

// telegramDealURL returns a stable CRM deep link even when the builder event
// does not include a prebuilt URL. The frontend consumes `deal` and `pipeline`
// and opens the matching deal drawer after loading that pipeline.
func telegramDealURL(deal map[string]any) string {
	if url := telegramDealValue(deal, "url", "deal_url"); url != "" {
		return url
	}
	dealID := telegramDealValue(deal, "guid", "id")
	if dealID == "" {
		return ""
	}
	query := "deal=" + url.QueryEscape(dealID)
	if pipeline := telegramDealPipeline(deal); pipeline != "" {
		query += "&pipeline=" + url.QueryEscape(pipeline)
	}
	return "https://crm.ucode.co/deals?" + query
}
