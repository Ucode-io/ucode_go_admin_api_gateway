package v1

import (
	"context"
	"encoding/json"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

var (
	waitRedirectMap = caching.NewConcurrentMap()
)

func (h *HandlerV1) Proxy(c *gin.Context) {
	h.handleResponse(c, status_http.OK, "PROXY response")
}

func (h *HandlerV1) CompanyRedirectGetList(data helper.MatchingData, comp services.CompanyServiceI) (*pb.GetListRedirectUrlRes, error) {
	var (
		key = "redirect-" + data.ProjectId + data.EnvId
		res = &pb.GetListRedirectUrlRes{}
		err error
	)

	var redirectWaitKey = config.CACHE_WAIT + "-redirect"
	_, redirectOk := h.cache.Get(redirectWaitKey)
	if !redirectOk {
		h.cache.Add(redirectWaitKey, []byte(redirectWaitKey), config.REDIS_KEY_TIMEOUT)
	}

	if redirectOk {
		ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
		defer cancel()

		for {
			redirectBody, ok := h.cache.Get(key)
			if ok {
				err = json.Unmarshal(redirectBody, &res)
				if err != nil {
					return nil, err
				}
			}

			if len(res.RedirectUrls) > 0 {
				return res, nil
			}

			if ctx.Err() == context.DeadlineExceeded {
				break
			}

			time.Sleep(config.REDIS_SLEEP)
		}
	}

	res, err = comp.Redirect().GetList(context.Background(), &pb.GetListRedirectUrlReq{
		ProjectId: data.ProjectId,
		EnvId:     data.EnvId,
		Offset:    0,
		Limit:     100,
	})
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	h.cache.Add(key, body, config.REDIS_TIMEOUT)

	return res, nil
}
