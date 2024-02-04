package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HandlerV1 struct {
	baseConf        config.BaseConfig
	projectConfs    map[string]config.Config
	log             logger.LoggerI
	services        services.ServiceNodesI
	storage         storage.StorageI
	companyServices services.CompanyServiceI
	authService     services.AuthServiceManagerI
	apikeyService   services.AuthServiceManagerI
	redis           storage.RedisStorageI
	cache           *caching.ExpiringLRUCache
}

func NewHandlerV1(baseConf config.BaseConfig, projectConfs map[string]config.Config, log logger.LoggerI, svcs services.ServiceNodesI, cmpServ services.CompanyServiceI, authService services.AuthServiceManagerI, redis storage.RedisStorageI, cache *caching.ExpiringLRUCache) HandlerV1 {
	return HandlerV1{
		baseConf:        baseConf,
		projectConfs:    projectConfs,
		log:             log,
		services:        svcs,
		companyServices: cmpServ,
		authService:     authService,
		redis:           redis,
		cache:           cache,
	}
}

func (h *HandlerV1) GetProjectSrvc(c context.Context, projectId string, nodeType string) (services.ServiceManagerI, error) {
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

func (h *HandlerV1) handleResponse(c *gin.Context, status status_http.Status, data interface{}) {
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

func (h *HandlerV1) getOffsetParam(c *gin.Context) (offset int, err error) {
	offsetStr := c.DefaultQuery("offset", h.baseConf.DefaultOffset)
	return strconv.Atoi(offsetStr)
}

func (h *HandlerV1) getLimitParam(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", h.baseConf.DefaultLimit)
	return strconv.Atoi(limitStr)
}

func (h *HandlerV1) getPageParam(c *gin.Context) (page int, err error) {
	pageStr := c.DefaultQuery("page", "1")
	return strconv.Atoi(pageStr)
}

func (h *HandlerV1) versionHistory(c *gin.Context, req *models.CreateVersionHistoryRequest) error {
	var (
		current  = map[string]interface{}{"data": req.Current}
		previous = map[string]interface{}{"data": req.Previous}
		request  = map[string]interface{}{"data": req.Request}
		response = map[string]interface{}{"data": req.Response}
	)

	if req.Current == nil {
		current["data"] = make(map[string]interface{})
	}
	if req.Previous == nil {
		previous["data"] = make(map[string]interface{})
	}
	if req.Request == nil {
		request["data"] = make(map[string]interface{})
	}
	if req.Response == nil {
		response["data"] = make(map[string]interface{})
	}

	_, err := req.Services.GetBuilderServiceByType(req.NodeType).VersionHistory().Create(
		context.Background(),
		&object_builder_service.CreateVersionHistoryRequest{
			Id:                uuid.NewString(),
			ProjectId:         req.ProjectId,
			ActionSource:      req.ActionSource,
			ActionType:        req.ActionType,
			Previus:           fromMapToString(previous),
			Current:           fromMapToString(current),
			UsedEnvrironments: req.UsedEnvironments,
			Date:              time.Now().Format("2006-01-02 15:04:05"),
			UserInfo:          req.UserInfo,
			Request:           fromMapToString(request),
			Response:          fromMapToString(response),
			ApiKey:            req.ApiKey,
		},
	)
	if err != nil {
		fmt.Println("=======================================================")
		log.Println(err)
		fmt.Println("=======================================================")
		return err
	}
	return nil
}

func fromMapToString(req map[string]interface{}) string {
	reqString, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(reqString)
}
