package v1

import (
	"log"
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

	// 3. Aggregate results
	response := models.AllPricingUsage{
		Functions: models.PricingUsage{
			Current: float64(usageResp.FunctionsCount) / 1024,
			Unit:    "count",
		},
		Microfrontend: models.PricingUsage{
			Current: float64(usageResp.MicrofrontendsCount) / 1024,
			Unit:    "count",
		},
		AssetSize: models.PricingUsage{
			Current: float64(usageResp.AssetSize) / 1024,
			Unit:    "bytes",
		},
		DatabaseSize: models.PricingUsage{
			Current: float64(usageResp.DatabaseSize) / 1024,
			Unit:    "bytes",
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
			response.DatabaseSize.Limit = cast.ToFloat64(limit.Value[:len(limit.Value)-2]) * 1024
		case "Asset Size info":
			response.AssetSize.Limit = cast.ToFloat64(limit.Value[:len(limit.Value)-2]) * 1024
		}
	}

	h.HandleResponse(c, status_http.OK, response)
}
