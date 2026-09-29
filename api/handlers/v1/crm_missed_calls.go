package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/status_http"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	missedCallTargetsKey             = "crm:missed-calls:targets"
	missedCallLockKey                = "crm:missed-calls:poll-lock"
	missedCallRegisteredPrefix       = "crm:missed-calls:registered:"
	missedCallNotificationsPrefix    = "crm:missed-calls:notifications:"
	missedCallNotificationDataPrefix = "crm:missed-calls:notification-data:"
	missedCallRetention              = 5 * 365 * 24 * time.Hour
)

type missedCallTarget struct {
	ProjectID     string
	EnvironmentID string
	CompanyID     string
}

type missedCallNotification struct {
	ID        string `json:"id"`
	CallID    string `json:"call_id"`
	DealID    string `json:"deal_id"`
	Phone     string `json:"phone"`
	StartedAt string `json:"started_at"`
}

func missedCallTargetKey(target missedCallTarget) string {
	return target.ProjectID + "|" + target.EnvironmentID + "|" + target.CompanyID
}

func missedCallDealID(projectID, callID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("crm-missed-call:"+projectID+":"+callID)).String()
}

func missedCallFromRow(row map[string]any) (string, string, time.Time, bool) {
	if telegramDealValue(row, "direction") != "incoming" || telegramDealValue(row, "status") != "no_answer" {
		return "", "", time.Time{}, false
	}
	callID := telegramDealValue(row, "call_uuid", "idempotency_key")
	phone := strings.TrimSpace(telegramDealValue(row, "client_phone"))
	startedAt, err := time.Parse(time.RFC3339Nano, telegramDealValue(row, "started_at", "created_at"))
	if callID == "" || phone == "" || err != nil {
		return "", "", time.Time{}, false
	}
	return callID, phone, startedAt, true
}

// ListMissedCallNotifications also registers this CRM workspace for the server
// poller, so missed calls continue to create deals after the browser closes.
func (h *HandlerV1) ListMissedCallNotifications(c *gin.Context) {
	project, projectOK := c.Get("project_id")
	environment, environmentOK := c.Get("environment_id")
	companyID := strings.TrimSpace(c.Query("company-id"))
	if !projectOK || !environmentOK || companyID == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "project, environment and company-id are required")
		return
	}
	if h.centralRedis == nil {
		h.HandleResponse(c, status_http.InternalServerError, "notification storage is unavailable")
		return
	}
	target := missedCallTarget{ProjectID: fmt.Sprint(project), EnvironmentID: fmt.Sprint(environment), CompanyID: companyID}
	targetKey := missedCallTargetKey(target)
	ctx := c.Request.Context()
	if err := h.centralRedis.SetNX(ctx, missedCallRegisteredPrefix+targetKey, time.Now().UTC().Add(-2*time.Minute).Format(time.RFC3339Nano), missedCallRetention).Err(); err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	if err := h.centralRedis.SAdd(ctx, missedCallTargetsKey, targetKey).Err(); err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	_ = h.centralRedis.Expire(ctx, missedCallTargetsKey, missedCallRetention).Err()
	ids, err := h.centralRedis.ZRevRange(ctx, missedCallNotificationsPrefix+targetKey, 0, 49).Result()
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	notifications := make([]missedCallNotification, 0, len(ids))
	if len(ids) > 0 {
		values, readErr := h.centralRedis.HMGet(ctx, missedCallNotificationDataPrefix+targetKey, ids...).Result()
		if readErr != nil {
			h.HandleResponse(c, status_http.InternalServerError, readErr.Error())
			return
		}
		for _, value := range values {
			raw, ok := value.(string)
			if !ok {
				continue
			}
			var notification missedCallNotification
			if json.Unmarshal([]byte(raw), &notification) == nil {
				notifications = append(notifications, notification)
			}
		}
	}
	h.HandleResponse(c, status_http.OK, notifications)
}

func (h *HandlerV1) StartMissedCallPoller(ctx context.Context) {
	if h.centralRedis == nil {
		h.log.Warn("missed calls: central redis unavailable")
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.pollMissedCalls(ctx)
			}
		}
	}()
}

func (h *HandlerV1) pollMissedCalls(ctx context.Context) {
	acquired, err := h.centralRedis.SetNX(ctx, missedCallLockKey, time.Now().UnixNano(), 29*time.Second).Result()
	if err != nil || !acquired {
		return
	}
	keys, err := h.centralRedis.SMembers(ctx, missedCallTargetsKey).Result()
	if err != nil {
		h.log.Warn("missed calls: read targets failed", logger.Error(err))
		return
	}
	for _, key := range keys {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		target := missedCallTarget{ProjectID: parts[0], EnvironmentID: parts[1], CompanyID: parts[2]}
		if err := h.pollMissedCallTarget(ctx, target); err != nil {
			h.log.Warn("missed calls: target poll failed", logger.String("project_id", target.ProjectID), logger.Error(err))
		}
	}
}

func (h *HandlerV1) pollMissedCallTarget(ctx context.Context, target missedCallTarget) error {
	key := missedCallTargetKey(target)
	registered, err := h.centralRedis.Get(ctx, missedCallRegisteredPrefix+key).Result()
	if err != nil {
		return err
	}
	since, err := time.Parse(time.RFC3339Nano, registered)
	if err != nil {
		return err
	}
	svc, resourceEnvID, err := h.resolveProjectBuilder(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return err
	}
	response, err := svc.GoObjectBuilderService().ObjectBuilder().GetList2(ctx, &nb.CommonMessage{
		TableSlug:        "pbx_calls",
		Data:             mustStruct(map[string]any{"limit": 500, "offset": 0, "order_by": "created_at", "order": "DESC"}),
		ProjectId:        resourceEnvID,
		CompanyProjectId: target.ProjectID,
	})
	if err != nil {
		return err
	}
	for _, row := range telegramResponseRows(response.GetData()) {
		callID, phone, startedAt, ok := missedCallFromRow(row)
		if !ok || startedAt.Before(since) {
			continue
		}
		if err := h.createMissedCallDeal(ctx, svc, resourceEnvID, target, callID, phone, startedAt, telegramDealValue(row, "users_id")); err != nil {
			h.log.Warn("missed calls: create deal failed", logger.String("call_id", callID), logger.Error(err))
		}
	}
	return nil
}

func (h *HandlerV1) createMissedCallDeal(ctx context.Context, svc services.ServiceManagerI, resourceEnvID string, target missedCallTarget, callID, phone string, startedAt time.Time, operatorID string) error {
	dealID := missedCallDealID(target.ProjectID, callID)
	key := missedCallTargetKey(target)
	saved, err := h.centralRedis.HExists(ctx, missedCallNotificationDataPrefix+key, dealID).Result()
	if err != nil {
		return err
	}
	if saved {
		return nil
	}
	mapping := defaultCRMMapping()
	payload := map[string]any{
		"guid":                     dealID,
		"name":                     "Пропущенный звонок: " + phone,
		"phone":                    phone,
		"description":              "Пропущенный входящий звонок. PBX call ID: " + callID,
		mapping.PipelineField:      []string{mapping.PipelineValue},
		mapping.StageField:         []string{mapping.StageValue},
		mapping.PipelineStageField: mapping.StageValue,
		mapping.StartDateField:     startedAt.UTC().Format(time.RFC3339),
	}
	if _, err := uuid.Parse(operatorID); err == nil {
		payload["users_id"] = operatorID
	}
	if err := h.crmCreateItem(ctx, svc, resourceEnvID, mapping.DealsTable, payload); err != nil && !isAlreadyExists(err) {
		return err
	}
	notification := missedCallNotification{ID: dealID, CallID: callID, DealID: dealID, Phone: phone, StartedAt: startedAt.UTC().Format(time.RFC3339)}
	encoded, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	pipe := h.centralRedis.TxPipeline()
	pipe.HSet(ctx, missedCallNotificationDataPrefix+key, dealID, string(encoded))
	pipe.ZAdd(ctx, missedCallNotificationsPrefix+key, &redis.Z{Score: float64(startedAt.Unix()), Member: dealID})
	pipe.Expire(ctx, missedCallNotificationDataPrefix+key, missedCallRetention)
	pipe.Expire(ctx, missedCallNotificationsPrefix+key, missedCallRetention)
	_, err = pipe.Exec(ctx)
	return err
}
