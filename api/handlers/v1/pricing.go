package v1

import (
	"fmt"
	"log"
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
)

// GetAllPricingUsage godoc
// @Summary Get all pricing usage
// @Description Get all pricing usage for a project, including limits and current usage
// @Tags Billing
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} status_http.Response{data=models.AllPricingUsage} "Pricing usage data"
// @Failure 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
// @Router /v1/pricing/all [get]
func (h *HandlerV1) GetAllPricingUsage(c *gin.Context) {
	projectId, ok := c.Get("project_id")
	if !ok {
		h.HandleResponse(c, status_http.BadRequest, "project_id is required")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	limitsResp, err := h.companyServices.Billing().GetPricingLimits(
		c.Request.Context(), &company_service.GetPricingLimitsRequest{
			ProjectId: cast.ToString(projectId),
		},
	)
	if err != nil {
		h.log.Error("GetPricingLimits error", logger.Error(err), logger.String("project_id", cast.ToString(projectId)))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	usageResp, err := service.GoObjectBuilderService().ObjectBuilder().GetResourceUsage(
		c.Request.Context(), &nb.GetResourceUsageRequest{
			ProjectId: resourceEnvId,
		},
	)
	if err != nil {
		h.log.Error("GetResourceUsage error", logger.Error(err), logger.String("project_id", cast.ToString(projectId)))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	usersCountResp, err := h.authService.User().GetProjectUsersCount(
		c.Request.Context(), &auth_service.GetProjectUsersCountRequest{
			ProjectId: projectId.(string),
		},
	)
	if err != nil {
		h.log.Error("GetProjectUsersCount error", logger.Error(err), logger.String("project_id", resourceEnvId))
	}

	var userCount float64
	if usersCountResp != nil {
		userCount = float64(usersCountResp.Count)
	}

	// 3. Aggregate results
	response := models.AllPricingUsage{
		Functions: models.PricingUsage{
			Current: float64(usageResp.FunctionsCount),
			Unit:    "count",
		},
		Microfrontend: models.PricingUsage{
			Current: float64(usageResp.MicrofrontendsCount),
			Unit:    "count",
		},
		AssetSize: models.PricingUsage{
			Current: float64(usageResp.AssetSize) / (1024 * 1024),
			Unit:    "MB",
		},
		DatabaseSize: models.PricingUsage{
			Current: float64(usageResp.DatabaseSize) / (1024 * 1024),
			Unit:    "MB",
		},
		Users: models.PricingUsage{
			Current: userCount,
			Unit:    "count",
		},
		Items: models.PricingUsage{
			Current: float64(usageResp.ItemsCount),
			Limit:   100000, // Static limit as requested
			Unit:    "count",
		},
	}

	// Map limits
	for _, limit := range limitsResp.Limits {

		log.Println("AAA:", limit.Name, limit.Value)

		switch limit.Name {
		case "Functions":
			response.Functions.Limit = cast.ToFloat64(limit.Value)
		case "Microfrontend":
			response.Microfrontend.Limit = cast.ToFloat64(limit.Value)
		case "Database Size":
			if len(limit.Value) > 2 {
				response.DatabaseSize.Limit = cast.ToFloat64(limit.Value[:len(limit.Value)-2]) * 1024
			}
		case "Asset Size info":
			if len(limit.Value) > 2 {
				response.AssetSize.Limit = cast.ToFloat64(limit.Value[:len(limit.Value)-2]) * 1024
			}
		case "Total Users & Roles":
			response.Users.Limit = cast.ToFloat64(limit.Value)
		case "Items":
			response.Items.Limit = cast.ToFloat64(limit.Value)
		}
	}

	h.HandleResponse(c, status_http.OK, response)
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

	now := time.Now().UTC()
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

	// Smooth RPS calculation: blending previous and current minute counts
	rps := (((float64(prevRpm) * (60 - sec)) + (float64(rpm) * sec)) / 60) / 60

	// Get historical DB metrics
	metricsResp, err := h.companyServices.Billing().GetMonitoringMetrics(
		c.Request.Context(),
		&company_service.GetMonitoringMetricsRequest{ProjectId: projectId},
	)

	var monthly, lastDay int64
	if err == nil && metricsResp != nil {
		monthly = metricsResp.TotalMonthlyCalls
		lastDay = metricsResp.TotalLastDayCalls
	}

	res := models.ApiMetricsResponse{
		Rps:          rps,
		Rpm:          rpm,
		Rph:          rph,
		TodayCalls:   today,
		MonthlyCalls: monthly,
		LastDayCalls: lastDay,
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

	metricsResp, err := h.companyServices.Billing().GetMonitoringMetrics(
		c.Request.Context(),
		&company_service.GetMonitoringMetricsRequest{ProjectId: projectId},
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
