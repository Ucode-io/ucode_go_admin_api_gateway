package v1

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	telegramNotificationsResourceName   = "CRM Telegram notifications"
	telegramNotificationsResourceStatus = "crm_notifications"
	telegramNotificationsSecretKey      = "telegram_notifications_settings"
	telegramNotificationsCodePrefix     = "telegram:notifications:connect:"
	telegramNotificationsCodeTTL        = 10 * time.Minute
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
	if err := newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).setWebhook(ctx, webhookURL, h.baseConf.TelegramNotificationsWebhookSecret, []string{"message"}); err != nil {
		h.log.Error("telegram notifications: set webhook failed", logger.Error(err))
	}
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
	if err := validateTelegramNotificationSettings(&settings); err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	if err := h.saveTelegramNotificationSettings(c.Request.Context(), target, settings); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
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
	if err := c.ShouldBindJSON(&update); err != nil || update.Message == nil {
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
		settings.ChatID = fmt.Sprint(message.Chat.ID)
		settings.ChatTitle = strings.TrimSpace(message.Chat.Title)
		settings.BotUsername = strings.TrimPrefix(h.baseConf.TelegramNotificationsBotUsername, "@")
		err = h.saveTelegramNotificationSettings(c.Request.Context(), target, settings)
	}
	if err == nil {
		_ = h.centralRedis.Del(c.Request.Context(), telegramNotificationsCodePrefix+code).Err()
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
	if _, err = newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(c.Request.Context(), settings.ChatID, renderTelegramNotificationTemplate(template)); err != nil {
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
	if _, err = newTelegramAPIClient(h.baseConf.TelegramNotificationsBotToken).sendHTMLMessage(c.Request.Context(), settings.ChatID, renderTelegramNotificationTemplate(settings.Templates.DailyReport)); err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.HandleResponse(c, status_http.OK, gin.H{"sent": true})
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
	return settings, resource, nil
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
	_, err = h.companyServices.Resource().UpdateProjectResource(ctx, &pb.ProjectResource{Id: resource.GetId(), Name: resource.GetName(), ProjectId: target.ProjectID, EnvironmentId: target.EnvironmentID, Type: pb.ResourceType_TELEGRAM.String(), ResourceType: int32(pb.ResourceType_TELEGRAM), ExternalId: target.CompanyID, Settings: telegramNotificationResourceSettings(h.baseConf.TelegramNotificationsBotUsername), Secret: secret})
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

func telegramNotificationResourceSettings(username string) *pb.Settings {
	return &pb.Settings{Telegram: &pb.TelegramCredentials{BotUsername: strings.TrimPrefix(username, "@"), Status: telegramNotificationsResourceStatus}}
}

func validateTelegramNotificationSettings(settings *models.TelegramNotificationSettings) error {
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
	}
	return "", errors.New("notification template was not found")
}

func renderTelegramNotificationTemplate(template string) string {
	replacer := strings.NewReplacer(
		"{{lead.name}}", "Azizbek Karimov", "{{lead.phone}}", "+998 90 123 45 67", "{{lead.source}}", "Instagram", "{{lead.owner_name}}", "Madina", "{{lead.url}}", "CRMda ochish",
		"{{contact.name}}", "Azizbek Karimov", "{{contact.phone}}", "+998 90 123 45 67", "{{deal.amount}}", "1 200 000 so‘m", "{{deal.owner_name}}", "Madina", "{{deal.url}}", "Dealni CRMda ochish", "{{deal.service}}", "IELTS kursi",
		"{{report.company}}", "PROFESSIONAL CRM", "{{report.date}}", time.Now().Format("02.01.2006"), "{{report.ad_spend}}", "600 000 so‘m", "{{report.leads_total}}", "40", "{{report.cpl}}", "20 000 so‘m", "{{report.statuses}}", "🆕 Yangi — 12\n📞 Bog‘lanildi — 15\n✅ Success deal — 3",
	)
	return replacer.Replace(template)
}
