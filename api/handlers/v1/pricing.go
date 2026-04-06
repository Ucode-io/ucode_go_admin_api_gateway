package v1

import (
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
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

	// 1. Get limits from company service
	limitsResp, err := h.companyServices.Billing().GetPricingLimits(c.Request.Context(), &company_service.GetPricingLimitsRequest{
		ProjectId: cast.ToString(projectId),
	})
	if err != nil {
		h.log.Error("GetPricingLimits error", logger.Error(err), logger.String("project_id", cast.ToString(projectId)))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	// 2. Get usage from object builder service
	srvc, err := h.services.Get(h.baseConf.UcodeNamespace)
	if err != nil {
		h.log.Error("Get service error", logger.Error(err), logger.String("project_id", cast.ToString(projectId)))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	usageResp, err := srvc.GoObjectBuilderService().ObjectBuilder().GetResourceUsage(c.Request.Context(), &nb.GetResourceUsageRequest{
		ProjectId: cast.ToString(projectId),
	})
	if err != nil {
		h.log.Error("GetResourceUsage error", logger.Error(err), logger.String("project_id", cast.ToString(projectId)))
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
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
			Current: float64(usageResp.AssetSize),
			Unit:    "bytes",
		},
		DatabaseSize: models.PricingUsage{
			Current: float64(usageResp.DatabaseSize),
			Unit:    "bytes",
		},
	}

	// Map limits
	for _, limit := range limitsResp.Limits {
		switch limit.Type {
		case "function":
			response.Functions.Limit = cast.ToFloat64(limit.Value)
		case "microfrontend":
			response.Microfrontend.Limit = cast.ToFloat64(limit.Value)
		case "database":
			response.DatabaseSize.Limit = cast.ToFloat64(limit.Value)
		case "asset_storage":
			response.AssetSize.Limit = cast.ToFloat64(limit.Value)
		}
	}

	h.HandleResponse(c, status_http.OK, response)
}
