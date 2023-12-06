package v1

import (
	"context"
	"encoding/json"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

var (
	waitRedirectMap = helper.NewConcurrentMap()
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

	waitMap := waitRedirectMap.ReadFromMap(key)

	if waitMap.Timeout != nil {
		if waitMap.Timeout.Err() == context.DeadlineExceeded {
			waitRedirectMap.DeleteKey(key)
			waitMap = waitRedirectMap.ReadFromMap(key)
		}
	}

	if waitMap.Value != config.CACHE_WAIT {
		ctx, _ := context.WithTimeout(context.Background(), config.REDIS_TIMEOUT)
		waitRedirectMap.AddKey(key, helper.WaitKey{Value: config.CACHE_WAIT, Timeout: ctx})
	}

	if waitMap.Value == config.CACHE_WAIT {
		ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
		defer cancel()
		for {
			waitMap := waitRedirectMap.ReadFromMap(key)
			if len(waitMap.Body) > 0 {
				err = json.Unmarshal(waitMap.Body, &res)
				if err != nil {
					return nil, err
				}
			}

			if len(res.RedirectUrls) > 0 {
				break
			}

			if ctx.Err() == context.DeadlineExceeded {
				break
			}

			time.Sleep(config.REDIS_SLEEP)
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

		waitRedirectMap.WriteBody(key, body)
	}

	return res, nil
}
