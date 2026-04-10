package v1

import (
	"fmt"
	"math"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

func parseStorageLimitToMB(val string) float64 {
	if len(val) > 2 {
		return cast.ToFloat64(val[:len(val)-2]) * 1024
	}
	return 0
}

func (h *HandlerV1) GetAllPricingUsage(c *gin.Context) {
	projectId, ok := c.Get("project_id")
	if !ok {
		h.HandleResponse(c, status_http.BadRequest, "project_id is required")
		return
	}
	projectIDStr := cast.ToString(projectId)

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	var (
		limitsResp     *company_service.GetPricingLimitsResponse
		usageResp      *nb.GetResourceUsageResponse
		usersCountResp *auth_service.GetProjectUsersCountResponse
		apiKeysResp    *auth_service.GetProjectApiKeysCountResponse
		tokenMetrics   *company_service.GetAiTokenUsageMetricsResponse
		apiMetrics     *company_service.GetApiCallMonitoringMetricsResponse
	)

	g, gCtx := errgroup.WithContext(c.Request.Context())

	// 2.1. Лимиты прайсинга
	g.Go(func() error {
		var err error
		limitsResp, err = h.companyServices.Billing().GetPricingLimits(gCtx, &company_service.GetPricingLimitsRequest{ProjectId: projectIDStr})
		return err
	})

	g.Go(func() error {
		var err error
		usageResp, err = service.GoObjectBuilderService().ObjectBuilder().GetResourceUsage(gCtx, &nb.GetResourceUsageRequest{ProjectId: resourceEnvId})
		return err
	})

	// 2.3. Количество пользователей
	g.Go(func() error {
		var err error
		usersCountResp, err = h.authService.User().GetProjectUsersCount(gCtx, &auth_service.GetProjectUsersCountRequest{ProjectId: projectIDStr})
		return err
	})

	g.Go(func() error {
		var err error
		apiKeysResp, err = h.authService.ApiKey().GetProjectApiKeysCount(gCtx, &auth_service.GetProjectApiKeysCountRequest{ProjectId: projectIDStr})
		return err
	})

	g.Go(func() error {
		var err error
		tokenMetrics, err = h.companyServices.Billing().GetAiTokenUsageMetrics(gCtx, &company_service.GetAiTokenUsageMetricsRequest{ProjectId: projectIDStr})
		return err
	})

	g.Go(func() error {
		var err error
		apiMetrics, err = h.companyServices.Billing().GetApiCallMonitoringMetrics(gCtx, &company_service.GetApiCallMonitoringMetricsRequest{ProjectId: projectIDStr})
		return err
	})

	if err := g.Wait(); err != nil {
		h.log.Error("GetAllPricingUsage failed", logger.Error(err), logger.String("project_id", projectIDStr))
		h.HandleResponse(c, status_http.InternalServerError, "Failed to fetch pricing usage data")
		return
	}

	response := models.AllPricingUsage{
		Functions:       models.PricingUsage{Current: float64(usageResp.FunctionsCount), Unit: "count"},
		Microfrontend:   models.PricingUsage{Current: float64(usageResp.MicrofrontendsCount), Unit: "count"},
		AssetSize:       models.PricingUsage{Current: float64(usageResp.AssetSize) / (1024 * 1024), Unit: "MB"},
		DatabaseSize:    models.PricingUsage{Current: float64(usageResp.DatabaseSize) / (1024 * 1024), Unit: "MB"},
		Users:           models.PricingUsage{Current: float64(usersCountResp.Count), Unit: "count"},
		Items:           models.PricingUsage{Current: float64(usageResp.ItemsCount), Limit: 100_000, Unit: "count"},
		Tables:          models.PricingUsage{Current: float64(usageResp.TablesCount), Limit: 100, Unit: "count"},
		ApiKeys:         models.PricingUsage{Current: float64(apiKeysResp.Count), Limit: 10, Unit: "count"},
		TodayTokens:     models.PricingUsage{Current: float64(tokenMetrics.TodayInputTokens + tokenMetrics.TodayOutputTokens), Limit: 100_000, Unit: "tokens"},
		MonthlyTokens:   models.PricingUsage{Current: float64(tokenMetrics.MonthlyInputTokens + tokenMetrics.MonthlyOutputTokens), Limit: 1_000_000, Unit: "tokens"},
		MonthlyApiCalls: models.PricingUsage{Current: float64(apiMetrics.TotalMonthlyCalls), Limit: 1_000_000, Unit: "count"},
	}

	for _, limit := range limitsResp.Limits {
		switch limit.Name {
		case "Functions":
			response.Functions.Limit = cast.ToFloat64(limit.Value)
		case "Microfrontend":
			response.Microfrontend.Limit = cast.ToFloat64(limit.Value)
		case "Database Size":
			response.DatabaseSize.Limit = parseStorageLimitToMB(limit.Value)
		case "Asset Size info":
			response.AssetSize.Limit = parseStorageLimitToMB(limit.Value)
		case "Total Users & Roles":
			response.Users.Limit = cast.ToFloat64(limit.Value)
		case "Items":
			response.Items.Limit = cast.ToFloat64(limit.Value)
		case "Tables":
			response.Tables.Limit = cast.ToFloat64(limit.Value)
		case "API Keys":
			response.ApiKeys.Limit = cast.ToFloat64(limit.Value)
		}
	}

	h.HandleResponse(c, status_http.OK, response)
}

// GetTokenUsage godoc
// @Summary Get AI token usage
// @Description Get AI input/output token usage for today and this month
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} status_http.Response{data=models.TokenUsageResponse} "Token usage"
// @Failure 401
// @Router /v1/pricing/token-usage [get]
func (h *HandlerV1) GetTokenUsage(c *gin.Context) {
	projectId := cast.ToString(c.MustGet("project_id"))

	resp, err := h.companyServices.Billing().GetAiTokenUsageMetrics(
		c.Request.Context(),
		&company_service.GetAiTokenUsageMetricsRequest{ProjectId: projectId},
	)
	if err != nil {
		h.log.Error("[GetTokenUsage] GetAiTokenUsageMetrics error", logger.Error(err))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, models.TokenUsageResponse{
		Today: models.TokenUsage{
			InputTokens:  resp.TodayInputTokens,
			OutputTokens: resp.TodayOutputTokens,
		},
		Monthly: models.TokenUsage{
			InputTokens:  resp.MonthlyInputTokens,
			OutputTokens: resp.MonthlyOutputTokens,
		},
	})
}

// GetApiMetrics godoc
// @Summary API metrics
// @Description Real-time API rate metrics and historical call counts
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} status_http.Response{data=models.ApiMetricsResponse} "Metrics Data"
// @Failure 401
// @Router /v1/pricing/api-metrics [get]
func (h *HandlerV1) GetApiMetrics(c *gin.Context) {
	projectId := cast.ToString(c.MustGet("project_id"))

	now := time.Now()
	prevNow := now.Add(-1 * time.Minute)

	minKey := fmt.Sprintf(config.KeyRateMin, projectId, now.Format("2006-01-02-15-04"))
	prevMinKey := fmt.Sprintf(config.KeyRateMin, projectId, prevNow.Format("2006-01-02-15-04"))
	hourKey := fmt.Sprintf(config.KeyRateHour, projectId, now.Format("2006-01-02-15"))
	dayKey := fmt.Sprintf(config.KeyRateDay, projectId, now.Format("2006-01-02"))

	var getRedisInt = func(key string) int64 {
		valStr, _ := h.redis.Get(c.Request.Context(), key, "", "")
		return cast.ToInt64(valStr)
	}

	rpm := getRedisInt(minKey)
	rph := getRedisInt(hourKey)
	today := getRedisInt(dayKey)
	prevRpm := getRedisInt(prevMinKey)

	sec := float64(now.Second())

	prevMinuteWeight := (60.0 - sec) / 60.0
	rolling60sCalls := float64(rpm) + (float64(prevRpm) * prevMinuteWeight)

	rawRps := rolling60sCalls / 60.0

	rps := math.Round(rawRps*100) / 100

	// Получаем исторические данные
	metricsResp, err := h.companyServices.Billing().GetApiCallMonitoringMetrics(
		c.Request.Context(),
		&company_service.GetApiCallMonitoringMetricsRequest{ProjectId: projectId},
	)

	var monthly int64
	if err == nil && metricsResp != nil {
		monthly = metricsResp.TotalMonthlyCalls
	} else if err != nil {
		h.log.Error("[GetApiMetrics] GetMonitoringMetrics error", logger.Error(err))
	}

	res := models.ApiMetricsResponse{
		Rps:          rps,
		Rpm:          rpm,
		Rph:          rph,
		TodayCalls:   today,
		MonthlyCalls: monthly,
	}

	h.HandleResponse(c, status_http.OK, res)
}

// GetApiChart godoc
// @Summary API Chart
// @Description Historical daily chart array
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} status_http.Response{data=models.ApiChartResponse} "Chart"
// @Failure 401
// @Router /v1/pricing/api-chart [get]
func (h *HandlerV1) GetApiChart(c *gin.Context) {
	projectId := cast.ToString(c.MustGet("project_id"))

	metricsResp, err := h.companyServices.Billing().GetApiCallMonitoringMetrics(
		c.Request.Context(),
		&company_service.GetApiCallMonitoringMetricsRequest{ProjectId: projectId},
	)

	if err != nil {
		h.log.Error("GetMonitoringMetrics chart error", logger.Error(err))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	chart := make([]models.DailyChartPoint, 0)
	if metricsResp != nil && metricsResp.DailyUsageChart != nil {
		for _, dbPoint := range metricsResp.DailyUsageChart {
			chart = append(chart, models.DailyChartPoint{
				Date:  dbPoint.Date,
				Count: dbPoint.Count,
			})
		}
	}

	h.HandleResponse(c, status_http.OK, models.ApiChartResponse{Chart: chart})
}

// GetPerformanceMetrics godoc
// @Summary Get performance metrics
// @Description Get average response duration (ms) and error rate for today's requests
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} status_http.Response{data=models.PerformanceMetricsResponse} "Performance metrics"
// @Failure 401
// @Router /v1/pricing/performance [get]
func (h *HandlerV1) GetPerformanceMetrics(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	today := time.Now().Format("2006-01-02")
	resp, err := service.GoObjectBuilderService().VersionHistory().GetPerformanceMetrics(
		c.Request.Context(),
		&nb.GetPerformanceMetricsRequest{
			ProjectId: resourceEnvId,
			FromDate:  today,
			ToDate:    today,
		},
	)
	if err != nil {
		h.log.Error("[GetPerformanceMetrics] GetPerformanceMetrics error", logger.Error(err))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, models.PerformanceMetricsResponse{
		AverageResponseTime: models.PricingUsage{
			Current: float64(resp.AverageDuration),
			Unit:    "ms",
		},
		ErrorRate: models.PricingUsage{
			Current: float64(resp.ErrorRate),
			Unit:    "%",
		},
	})
}
