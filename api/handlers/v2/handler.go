package v2

import (
	"context"
	"strconv"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/gin-gonic/gin"
)

type HandlerV2 struct {
	baseConf        config.BaseConfig
	projectConfs    map[string]config.Config
	log             logger.LoggerI
	services        services.ServiceNodesI
	storage         storage.StorageI
	companyServices services.CompanyServiceI
	authService     services.AuthServiceManagerI
	apikeyService   services.AuthServiceManagerI
	redis           storage.RedisStorageI
}

func NewHandlerV2(baseConf config.BaseConfig, projectConfs map[string]config.Config, log logger.LoggerI, svcs services.ServiceNodesI, cmpServ services.CompanyServiceI, authService services.AuthServiceManagerI, redis storage.RedisStorageI) HandlerV2 {
	return HandlerV2{
		baseConf:        baseConf,
		projectConfs:    projectConfs,
		log:             log,
		services:        svcs,
		companyServices: cmpServ,
		authService:     authService,
		redis:           redis,
	}
}

func (h *HandlerV2) GetProjectSrvc(c context.Context, projectId string, nodeType string) (services.ServiceManagerI, error) {
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

func (h *HandlerV2) handleResponse(c *gin.Context, status status_http.Status, data interface{}) {
	switch code := status.Code; {
	// case code < 300:
	// 	h.log.Info(
	// 		"response",
	// 		logger.Int("code", status.Code),
	// 		logger.String("status", status.Status),
	// 		logger.Any("description", status.Description),
	// 		// logger.Any("data", data),
	// 	)
	case code < 400:
		// h.log.Warn(
		// 	"response",
		// 	logger.Int("code", status.Code),
		// 	logger.String("status", status.Status),
		// 	logger.Any("description", status.Description),
		// 	logger.Any("data", data),
		// 	logger.Any("custom_message", status.CustomMessage),
		// )
	default:
		// fmt.Println(customMessage)
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

func (h *HandlerV2) getOffsetParam(c *gin.Context) (offset int, err error) {
	offsetStr := c.DefaultQuery("offset", h.baseConf.DefaultOffset)
	return strconv.Atoi(offsetStr)
}

func (h *HandlerV2) getLimitParam(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", h.baseConf.DefaultLimit)
	return strconv.Atoi(limitStr)
}

func (h *HandlerV2) getPageParam(c *gin.Context) (page int, err error) {
	pageStr := c.DefaultQuery("page", "1")
	return strconv.Atoi(pageStr)
}
