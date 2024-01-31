package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/pkg/easy_to_travel"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func (h *HandlerV1) EasyToTravelFunctionRun(c *gin.Context, requestData models.HttpRequest, faasSettings map[string]interface{}) (map[string]interface{}, error) {

	var (
		isCache   bool
		faasPaths = cast.ToStringSlice(faasSettings["paths"])
		params    = c.Request.URL.Query()

		invokeFunction        models.InvokeFunctionRequest
		appId                 = cast.ToString(faasSettings["app_id"])
		nodeType              = cast.ToString(faasSettings["node_type"])
		projectId             = cast.ToString(faasSettings["project_id"])
		resourceEnvironmentId = cast.ToString(faasSettings["resource_environment_id"])
	)
	if faasSettings != nil {
		isCache = cast.ToBool(faasSettings["is_cache"])
	}

	if helper.Contains(faasPaths, c.Param("function-id")) {
		for _, param := range cast.ToStringSlice(faasSettings["delete_params"]) {
			params.Del(param)
		}
	}

	var key = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("ett-%s-%s-%s", c.Request.Header.Get("Prev_path"), params.Encode(), resourceEnvironmentId)))
	_, exists := h.cache.Get(config.CACHE_WAIT)
	if isCache {
		if exists {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				functionBody, ok := h.cache.Get(key)
				if ok {
					resp := make(map[string]interface{})
					err := json.Unmarshal(functionBody, &resp)
					if err != nil {
						h.log.Error("Error while json unmarshal", logger.Any("err", err))
						return nil, err
					}

					if _, ok := resp["code"]; ok {
						return resp, nil
					}

					if helper.Contains(faasPaths, c.Param("function-id")) && faasSettings["function_name"] != nil {
						var filters = map[string]interface{}{}
						for key, val := range c.Request.URL.Query() {
							if len(val) > 0 {
								filters[key] = val[0]
							}
						}
						resp["filters"] = filters

						data, err := easy_to_travel.AgentApiGetProduct(resp)
						if err != nil {
							h.log.Error("Error while EasyToTravelAgentApiGetProduct function:", logger.Any("err", err))
							result, _ := helper.InterfaceToMap(data)
							return result, nil
						}

						resp, err = helper.InterfaceToMap(data)
						if err != nil {
							h.log.Error("Error while InterfaceToMap:", logger.Any("err", err))
							return nil, err
						}
					}

					return resp, nil
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		} else {
			h.cache.Add(config.CACHE_WAIT, []byte(config.CACHE_WAIT), 20*time.Second)
		}
	}

	srvs, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId,
		nodeType,
	)
	if err != nil {
		h.log.Error("Error while GetProjectSrvc", logger.Any("err", err))
		return nil, err
	}

	var (
		function = &fc.Function{}
		resp     models.InvokeFunctionResponse
	)
	if faasSettings["function_name"] != nil {
		if customFunction, ok := faasSettings["function_name"].(func(services.ServiceManagerI, []byte) string); ok {
			fmt.Println("~~~~~~~~~~~~~~>>> function id:", c.Param("function-id"))
			requestBody, err := json.Marshal(models.FunctionRunV2{
				Auth:        models.AuthData{},
				RequestData: requestData,
				Data: map[string]interface{}{
					"object_ids":              invokeFunction.ObjectIDs,
					"attributes":              invokeFunction.Attributes,
					"app_id":                  appId,
					"node_type":               nodeType,
					"resource_environment_id": resourceEnvironmentId,
				},
			})
			if err != nil {
				h.log.Error("Error while json marshal fn:", logger.Any("err", err))
				return nil, err
			}

			respByte := customFunction(srvs, requestBody)
			if err != nil {
				h.log.Error("Error while easytotravel fn:", logger.Any("err", err))
				return nil, err
			}

			err = json.Unmarshal([]byte(respByte), &resp)
			if err != nil {
				fmt.Println("\n\ncustomFunction::", c.Param("function-id"), string(respByte))
				return nil, err
			}
		}
	} else {
		if util.IsValidUUID(c.Param("function-id")) {
			function, err = srvs.FunctionService().FunctionService().GetSingle(
				context.Background(),
				&fc.FunctionPrimaryKey{
					Id:        c.Param("function-id"),
					ProjectId: resourceEnvironmentId,
				},
			)
			if err != nil {
				h.log.Error("Error while function service GetSingle:", logger.Any("err", err))
				return nil, err
			}
		} else {
			function.Path = c.Param("function-id")
		}

		resp, err = util.DoRequest("https://ofs.u-code.io/function/"+function.Path, "POST", models.FunctionRunV2{
			Auth:        models.AuthData{},
			RequestData: requestData,
			Data: map[string]interface{}{
				"object_ids": invokeFunction.ObjectIDs,
				"attributes": invokeFunction.Attributes,
				"app_id":     appId,
			},
		})
		if err != nil {
			h.log.Error("Error while function DoRequest:", logger.Any("err", err))
			return nil, err
		}
	}

	fmt.Println("resp:", resp)

	if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}

		h.log.Error("Error while function DoRequest errStr:", logger.Any("err", resp.ServerError))
		return nil, errors.New(errStr)
	}

	if isOwnData, ok := resp.Attributes["is_own_data"].(bool); ok {
		if isOwnData {
			if err == nil && isCache {
				jsonData, _ := json.Marshal(resp.Data)
				h.cache.Add(key, []byte(jsonData), 20*time.Second)
			}

			if _, ok := resp.Data["code"]; ok {
				return resp.Data, nil
			}

			if helper.ContainsLike(faasPaths, c.Param("function-id")) && faasSettings["function_name"] != nil {
				var filters = map[string]interface{}{}
				for key, val := range c.Request.URL.Query() {
					if len(val) > 0 {
						filters[key] = val[0]
					}
				}
				resp.Data["filters"] = filters

				data, err := easy_to_travel.AgentApiGetProduct(resp.Data)
				if err != nil {
					fmt.Println("Error while EasyToTravelAgentApiGetProduct function:", err.Error())
					result, _ := helper.InterfaceToMap(data)
					return result, nil
				}

				resp.Data, err = helper.InterfaceToMap(data)
				if err != nil {
					h.log.Error("Error while InterfaceToMap function:", logger.Any("err", err))
					return nil, err
				}
			}

			return resp.Data, nil
		}
	}

	return resp.Data, nil
}
