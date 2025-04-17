package handlers

import (
	"context"
	"strconv"
	v1 "ucode/ucode_go_api_gateway/api/handlers/v1"
	v2 "ucode/ucode_go_api_gateway/api/handlers/v2"
	"ucode/ucode_go_api_gateway/api/status_http"
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
	storage         storage.StorageI
	companyServices services.CompanyServiceI
	authService     services.AuthServiceManagerI
	apikeyService   services.AuthServiceManagerI
	redis           storage.RedisStorageI
	V1              v1.HandlerV1
	V2              v2.HandlerV2
	cache           *caching.ExpiringLRUCache
	rateLimiter     *util.ApiKeyRateLimiter
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
		cache:           cache,
	}
}

func (h *Handler) GetCompanyService(c *gin.Context) services.CompanyServiceI {
	return h.companyServices
}

func (h *Handler) GetAuthService(c *gin.Context) services.AuthServiceManagerI {
	return h.authService
}

func (h *Handler) GetProjectConfig(c *gin.Context, projectId string) config.Config {
	return h.projectConfs[projectId]
}

func (h *Handler) GetProjectSrvc(c context.Context, projectId string, nodeType string) (services.ServiceManagerI, error) {
	if nodeType == config.ENTER_PRICE_TYPE {
		srvc, err := h.services.Get(projectId)
		if err != nil {
			return nil, err
		}

		return srvc, nil
	} else {
		srvc, err := h.services.Get(h.baseConf.UcodeNamespace)
		if err != nil {
			return nil, err
		}

		return srvc, nil
	}
}

func (h *Handler) handleResponse(c *gin.Context, status status_http.Status, data any) {
	switch code := status.Code; {
	case code < 400:
	default:
		h.log.Error(
			"response",
			logger.Int("code", status.Code),
			logger.String("status", status.Status),
			logger.Any("description", status.Description),
			logger.Any("data", data),
			logger.Any("custom_message", status.CustomMessage),
		)
	}

	c.JSON(status.Code, status_http.Response{
		Status:        status.Status,
		Description:   status.Description,
		Data:          data,
		CustomMessage: status.CustomMessage,
	})
}

func (h *Handler) getOffsetParam(c *gin.Context) (offset int, err error) {
	offsetStr := c.DefaultQuery("offset", h.baseConf.DefaultOffset)
	return strconv.Atoi(offsetStr)
}

func (h *Handler) getLimitParam(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", h.baseConf.DefaultLimit)
	return strconv.Atoi(limitStr)
}

func (h *Handler) getPageParam(c *gin.Context) (page int, err error) {
	pageStr := c.DefaultQuery("page", "1")
	return strconv.Atoi(pageStr)
}
