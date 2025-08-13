package handlers

import (
	v1 "ucode/ucode_go_api_gateway/api/handlers/v1"
	v2 "ucode/ucode_go_api_gateway/api/handlers/v2"
	v3 "ucode/ucode_go_api_gateway/api/handlers/v3"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	baseConf        config.BaseConfig
	projectConfs    map[string]config.Config
	log             logger.LoggerI
	services        services.ServiceNodesI
	companyServices services.CompanyServiceI
	authService     services.AuthServiceManagerI
	redis           storage.RedisStorageI
	V1              v1.HandlerV1
	V2              v2.HandlerV2
	V3              v3.HandlerV3
	cache           *caching.ExpiringLRUCache
}

func NewHandler(baseConf config.BaseConfig, projectConfs map[string]config.Config, log logger.LoggerI, svcs services.ServiceNodesI, cmpServ services.CompanyServiceI, authService services.AuthServiceManagerI, redis storage.RedisStorageI, cache *caching.ExpiringLRUCache, limiter *util.ApiKeyRateLimiter) Handler {
	return Handler{
		baseConf:        baseConf,
		projectConfs:    projectConfs,
		log:             log,
		services:        svcs,
		companyServices: cmpServ,
		authService:     authService,
		redis:           redis,
		V1:              v1.NewHandlerV1(baseConf, projectConfs, log, svcs, cmpServ, authService, redis, cache, limiter),
		V2:              v2.NewHandlerV2(baseConf, projectConfs, log, svcs, cmpServ, authService, redis, cache, limiter),
		V3: v3.NewHandlerV3(&v3.HandlerV3Config{
			BaseConf:        baseConf,
			ProjectConfs:    projectConfs,
			Log:             log,
			Services:        svcs,
			CompanyServices: cmpServ,
			AuthService:     authService,
			Redis:           redis,
			RateLimiter:     limiter,
			Cache:           cache,
		}),
		cache: cache,
	}
}

func (h *Handler) GetCompanyService(c *gin.Context) services.CompanyServiceI {
	return h.companyServices
}
