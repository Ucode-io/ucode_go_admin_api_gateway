package v1

import (
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

type telegramScheduledReport struct {
	target   telegramNotificationTarget
	settings models.TelegramNotificationSettings
	template string
	pipeline string
	now      time.Time
	modern   bool
}

// Several workspaces can point at one Telegram group. Prefer the configured
// automation and send at most one report per group when the scheduler runs.
func telegramReportsForChats(candidates []telegramScheduledReport, modernChats map[string]bool) []telegramScheduledReport {
	selected := make(map[string]telegramScheduledReport)
	order := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		chatID := strings.TrimSpace(candidate.settings.ChatID)
		if chatID == "" || modernChats[chatID] && !candidate.modern {
			continue
		}
		previous, exists := selected[chatID]
		if !exists {
			order = append(order, chatID)
		}
		if !exists || candidate.modern && !previous.modern {
			selected[chatID] = candidate
		}
	}
	reports := make([]telegramScheduledReport, 0, len(order))
	for _, chatID := range order {
		reports = append(reports, selected[chatID])
	}
	return reports
}
