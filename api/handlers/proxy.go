package handlers

import (
	"context"
	"encoding/json"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

var (
	waitRedirectMap = helper.NewConcurrentMap()
)

func (h *Handler) Proxy(c *gin.Context) {
	h.handleResponse(c, status_http.OK, "PROXY response")
}

func (h *Handler) CompanyRedirectGetList(data helper.MatchingData, comp services.CompanyServiceI) (*pb.GetListRedirectUrlRes, error) {
	var (
		key = "redirect-" + data.ProjectId + data.EnvId
		res = &pb.GetListRedirectUrlRes{}
		err error
	)

	waitMap := waitRedirectMap.ReadFromMap(key)

	if waitMap.Timeout != nil {
		if waitMap.Timeout.Err() == context.DeadlineExceeded {
			waitRedirectMap.DeleteKey(key)
			waitMap = waitRedirectMap.ReadFromMap(key)
		}
	}

	if waitMap.Value != config.CACHE_WAIT {
		ctx, _ := context.WithTimeout(context.Background(), 280*time.Second)
		waitRedirectMap.AddKey(key, helper.WaitKey{Value: config.CACHE_WAIT, Timeout: ctx})
	}

	if waitMap.Value == config.CACHE_WAIT {
		ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
		defer cancel()
		for {
			redisResource, err := h.redis.Get(context.Background(), key, h.baseConf.UcodeNamespace, config.LOW_NODE_TYPE)
			if err == nil {
				err = json.Unmarshal([]byte(redisResource), &res)
				if err != nil {
					return nil, err
				}
				break
			}

			if ctx.Err() == context.DeadlineExceeded {
				break
			}

			time.Sleep(time.Millisecond * 10)
		}
	}

	if len(res.RedirectUrls) <= 0 {
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

		err = h.redis.SetX(context.Background(), key, string(body), 5*time.Minute, h.baseConf.UcodeNamespace, config.LOW_NODE_TYPE)
		if err != nil {
			h.log.Error("Error while setting redis", logger.Error(err))
		}
	}

	return res, nil
}
