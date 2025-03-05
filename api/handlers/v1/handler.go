package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HandlerV1 struct {
	baseConf        config.BaseConfig
	projectConfs    map[string]config.Config
	log             logger.LoggerI
	services        services.ServiceNodesI
	companyServices services.CompanyServiceI
	authService     services.AuthServiceManagerI
	redis           storage.RedisStorageI
	cache           *caching.ExpiringLRUCache
	rateLimiter     *util.ApiKeyRateLimiter
}

func NewHandlerV1(baseConf config.BaseConfig, projectConfs map[string]config.Config, log logger.LoggerI, svcs services.ServiceNodesI, cmpServ services.CompanyServiceI, authService services.AuthServiceManagerI, redis storage.RedisStorageI, cache *caching.ExpiringLRUCache, limiter *util.ApiKeyRateLimiter) HandlerV1 {
	return HandlerV1{
		baseConf:        baseConf,
		projectConfs:    projectConfs,
		log:             log,
		services:        svcs,
		companyServices: cmpServ,
		authService:     authService,
		redis:           redis,
		cache:           cache,
		rateLimiter:     limiter,
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

func (h *HandlerV1) handleError(c *gin.Context, statusHttp status_http.Status, err error) {
	st, _ := status.FromError(err)
	if statusHttp.Status == status_http.BadRequest.Status {
		c.JSON(http.StatusInternalServerError, status_http.Response{
			Status:        statusHttp.Status,
			Description:   st.String(),
			Data:          "Invalid JSON",
			CustomMessage: statusHttp.CustomMessage,
		})
	} else if st.Code() == codes.AlreadyExists {
		c.JSON(http.StatusInternalServerError, status_http.Response{
			Status:        statusHttp.Status,
			Description:   st.String(),
			Data:          "This slug already exists. Please choose a unique one.",
			CustomMessage: statusHttp.CustomMessage,
		})
	} else if st.Code() == codes.FailedPrecondition {
		c.JSON(http.StatusInternalServerError, status_http.Response{
			Status:        statusHttp.Status,
			Description:   st.String(),
			Data:          "Cannot drop or modify the object because dependent objects exist.",
			CustomMessage: statusHttp.CustomMessage,
		})
	} else if st.Err() != nil {
		c.JSON(http.StatusInternalServerError, status_http.Response{
			Status:        statusHttp.Status,
			Description:   st.String(),
			Data:          st.Message(),
			CustomMessage: statusHttp.CustomMessage,
		})
	}
}

func (h *HandlerV1) getOffsetParam(c *gin.Context) (offset int, err error) {
	offsetStr := c.DefaultQuery("offset", h.baseConf.DefaultOffset)
	return strconv.Atoi(offsetStr)
}

func (h *HandlerV1) getLimitParam(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", h.baseConf.DefaultLimit)
	return strconv.Atoi(limitStr)
}

func (h *HandlerV1) getLimitParamWithoutDefault(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", "0")
	return strconv.Atoi(limitStr)
}

func (h *HandlerV1) versionHistory(req *models.CreateVersionHistoryRequest) error {
	var (
		current  = map[string]interface{}{"data": req.Current}
		previous = map[string]interface{}{"data": req.Previous}
		request  = map[string]interface{}{"data": req.Request}
		response = map[string]interface{}{"data": req.Response}
		user     = ""
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

	if util.IsValidUUID(req.UserInfo) {
		info, err := h.authService.User().GetUserByID(
			context.Background(),
			&auth.UserPrimaryKey{
				Id: req.UserInfo,
			},
		)
		if err == nil {
			if info.Login != "" {
				user = info.Login
			} else {
				user = info.Phone
			}
		}
	}

	_, err := req.Services.GetBuilderServiceByType(req.NodeType).VersionHistory().Create(
		context.Background(),
		&obs.CreateVersionHistoryRequest{
			Id:                uuid.NewString(),
			ProjectId:         req.ProjectId,
			ActionSource:      req.ActionSource,
			ActionType:        req.ActionType,
			Previus:           fromMapToString(previous),
			Current:           fromMapToString(current),
			UsedEnvrironments: req.UsedEnvironments,
			Date:              time.Now().Format("2006-01-02T15:04:05.000Z"),
			UserInfo:          user,
			Request:           fromMapToString(request),
			Response:          fromMapToString(response),
			ApiKey:            req.ApiKey,
			Type:              req.Type,
			TableSlug:         req.TableSlug,
		},
	)
	if err != nil {
		h.log.Error("Error while create version history", logger.Any("err", err))
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

func (h *HandlerV1) versionHistoryGo(c *gin.Context, req *models.CreateVersionHistoryRequest) error {
	var (
		current  = map[string]interface{}{"data": req.Current}
		previous = map[string]interface{}{"data": req.Previous}
		request  = map[string]interface{}{"data": req.Request}
		response = map[string]interface{}{"data": req.Response}
		user     = ""
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

	if util.IsValidUUID(req.UserInfo) {
		info, err := h.authService.User().GetUserByID(
			context.Background(),
			&auth.UserPrimaryKey{
				Id: req.UserInfo,
			},
		)
		if err == nil {
			if info.Login != "" {
				user = info.Login
			} else {
				user = info.Phone
			}
		}
	}

	_, err := req.Services.GoObjectBuilderService().VersionHistory().Create(
		c,
		&nb.CreateVersionHistoryRequest{
			Id:                uuid.NewString(),
			ProjectId:         req.ProjectId,
			ActionSource:      req.ActionSource,
			ActionType:        req.ActionType,
			Previus:           fromMapToString(previous),
			Current:           fromMapToString(current),
			UsedEnvrironments: req.UsedEnvironments,
			Date:              time.Now().Format("2006-01-02T15:04:05.000Z"),
			UserInfo:          user,
			Request:           fromMapToString(request),
			Response:          fromMapToString(response),
			ApiKey:            req.ApiKey,
			Type:              req.Type,
			TableSlug:         req.TableSlug,
			VersionId:         req.VersionId,
		},
	)
	if err != nil {
		log.Println("ERROR FROM VERSION CREATE >>>>>", err)
		return err
	}
	return nil
}

func (h *HandlerV1) MakeProxy(c *gin.Context, proxyUrl, path string) (err error) {
	var req = c.Request

	proxy, err := url.Parse(proxyUrl)
	if err != nil {
		h.log.Error("error in parse addr: %v", logger.Error(err))
		c.String(http.StatusInternalServerError, "error")
		return
	}

	req.URL.Scheme = proxy.Scheme
	req.URL.Host = proxy.Host
	req.URL.Path = path
	transport := http.DefaultTransport

	resp, err := transport.RoundTrip(req)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Header(k, v)
		}
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	_, _ = bufio.NewReader(resp.Body).WriteTo(c.Writer)
	return
}
