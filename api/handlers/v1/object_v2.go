package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"google.golang.org/grpc/status"
)

var (
	waitSlimResourceMap = caching.NewConcurrentMap()
)

// GetListV2 godoc
// @Security ApiKeyAuth
// @ID get_list_objects_v2
// @Router /v2/object/get-list/{table_slug} [POST]
// @Summary Get all objects version 2
// @Description Get all objects version 2
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param language_setting query string false "language_setting"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListV2(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		resp          *obs.CommonMessage
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	if tokenInfo != nil {
		if tokenInfo.Tables != nil {
			objectRequest.Data["tables"] = tokenInfo.GetTables()
		}
		objectRequest.Data["user_id_from_token"] = tokenInfo.GetUserId()
		objectRequest.Data["role_id_from_token"] = tokenInfo.GetRoleId()
		objectRequest.Data["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
	}
	objectRequest.Data["language_setting"] = c.DefaultQuery("language_setting", "")

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(c.Request.Context())
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	if viewId, ok := objectRequest.Data["builder_service_view_id"].(string); ok {
		if util.IsValidUUID(viewId) {
			switch resource.ResourceType {
			case pb.ResourceType_MONGODB:
				redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), projectId.(string), resource.NodeType)
				if err == nil {
					resp := make(map[string]interface{})
					m := make(map[string]interface{})
					err = json.Unmarshal([]byte(redisResp), &m)
					if err != nil {
						h.log.Error("Error while unmarshal redis", logger.Error(err))
					} else {
						resp["data"] = m
						fmt.Println(":~>>>> response handled from redis")
						h.handleResponse(c, status_http.OK, resp)
						return
					}
				}

				resp, err = service.GroupByColumns(
					context.Background(),
					&obs.CommonMessage{
						TableSlug: c.Param("table_slug"),
						Data:      structData,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)

				if err == nil {
					if resp.IsCached {
						jsonData, _ := resp.GetData().MarshalJSON()
						err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
						if err != nil {
							h.log.Error("Error while setting redis", logger.Error(err))
						}
					}
				}

				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			case pb.ResourceType_POSTGRESQL:
				resp, err = services.PostgresBuilderService().ObjectBuilder().GroupByColumns(
					context.Background(),
					&obs.CommonMessage{
						TableSlug: c.Param("table_slug"),
						Data:      structData,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)

				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			}
		}
	} else {
		switch resource.ResourceType {
		case pb.ResourceType_MONGODB:
			redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), projectId.(string), resource.NodeType)
			if err == nil {
				resp := make(map[string]interface{})
				m := make(map[string]interface{})
				err = json.Unmarshal([]byte(redisResp), &m)
				if err != nil {
					h.log.Error("Error while unmarshal redis", logger.Error(err))
				} else {
					resp["data"] = m
					if _, ok := objectRequest.Data["load_test"].(bool); ok {
						config.CountReq += 1
					}
					h.handleResponse(c, status_http.OK, resp)
					return
				}
			}

			resp, err = service.GetList2(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: c.Param("table_slug"),
					Data:      structData,
					ProjectId: resource.ResourceEnvironmentId,
				},
			)

			if err == nil {
				if resp.IsCached {
					jsonData, _ := resp.GetData().MarshalJSON()
					err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
					if err != nil {
						h.log.Error("Error while setting redis", logger.Error(err))
					}
				}
			}

			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case pb.ResourceType_POSTGRESQL:
			resp, err = services.PostgresBuilderService().ObjectBuilder().GetList2(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: c.Param("table_slug"),
					Data:      structData,
					ProjectId: resource.ResourceEnvironmentId,
				},
			)

			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetListSlimV2 godoc
// @Security ApiKeyAuth
// @ID get_list_objects_slim_v2
// @Router /v2/object-slim/get-list/{table_slug} [GET]
// @Summary Get all objects slim v2
// @Description Get all objects slim v2
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param data query string false "data"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListSlimV2(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		queryData     string
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
	)
	queryParams := c.Request.URL.Query()
	if ok := queryParams.Has("data"); ok {
		queryData = queryParams.Get("data")
	}

	queryMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryData), &queryMap)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	queryMap["limit"] = limit
	queryMap["offset"] = offset

	objectRequest.Data = queryMap
	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	objectRequest.Data["tables"] = tokenInfo.GetTables()
	objectRequest.Data["user_id_from_token"] = tokenInfo.GetUserId()
	objectRequest.Data["role_id_from_token"] = tokenInfo.GetRoleId()
	objectRequest.Data["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	var (
		resourceKey = fmt.Sprintf("%s-%s", projectId.(string), environmentId.(string))
		resource    = &pb.ServiceResourceModel{}
	)

	waitResourceMap := waitSlimResourceMap.ReadFromMap(resourceKey)
	if waitResourceMap.Timeout != nil {
		if waitResourceMap.Timeout.Err() == context.DeadlineExceeded {
			waitSlimResourceMap.DeleteKey(resourceKey)
			waitResourceMap = waitSlimResourceMap.ReadFromMap(resourceKey)
		}
	}

	if waitResourceMap.Value != config.CACHE_WAIT {
		ctx, _ := context.WithTimeout(context.Background(), config.REDIS_TIMEOUT)
		waitSlimResourceMap.AddKey(resourceKey, caching.WaitKey{Value: config.CACHE_WAIT, Timeout: ctx})
	}

	if waitResourceMap.Value == config.CACHE_WAIT {
		ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
		defer cancel()

		for {
			waitResourceMap := waitSlimResourceMap.ReadFromMap(resourceKey)
			if len(waitResourceMap.Body) > 0 {
				err = json.Unmarshal(waitResourceMap.Body, &resource)
				if err != nil {
					h.log.Error("Error while unmarshal resource redis", logger.Error(err))
					return
				}
			}

			if resource.ResourceEnvironmentId != "" {
				break
			}

			if ctx.Err() == context.DeadlineExceeded {
				break
			}

			time.Sleep(config.REDIS_SLEEP)
		}
	}

	if resource.ResourceEnvironmentId == "" {
		resource, err = h.companyServices.ServiceResource().GetSingle(
			c.Request.Context(),
			&pb.GetSingleServiceResourceReq{
				ProjectId:     projectId.(string),
				EnvironmentId: environmentId.(string),
				ServiceType:   pb.ServiceType_BUILDER_SERVICE,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		body, err := json.Marshal(resource)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		waitSlimResourceMap.WriteBody(resourceKey, body)
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(c.Request.Context())
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	var slimKey = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId)))
	if cast.ToBool(c.Query("is_wait_cached")) {
		waitSlimMap := waitSlimResourceMap.ReadFromMap(slimKey)
		if waitSlimMap.Timeout != nil {
			if waitSlimMap.Timeout.Err() == context.DeadlineExceeded {
				waitSlimResourceMap.DeleteKey(slimKey)
				waitSlimMap = waitSlimResourceMap.ReadFromMap(slimKey)
			}
		}

		if waitSlimMap.Value != config.CACHE_WAIT {
			ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)
			waitSlimResourceMap.AddKey(slimKey, caching.WaitKey{Value: config.CACHE_WAIT, Timeout: ctx})
		}

		if waitSlimMap.Value == config.CACHE_WAIT {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				waitSlimMap := waitSlimResourceMap.ReadFromMap(slimKey)

				if len(waitSlimMap.Body) > 0 {
					m := make(map[string]interface{})
					err = json.Unmarshal(waitSlimMap.Body, &m)
					if err != nil {
						h.handleResponse(c, status_http.GRPCError, err.Error())
						return
					}

					h.handleResponse(c, status_http.OK, map[string]interface{}{"data": m})
					return
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		}
	} else {
		redisResp, err := h.redis.Get(context.Background(), slimKey, projectId.(string), resource.NodeType)
		if err == nil {
			resp := make(map[string]interface{})
			m := make(map[string]interface{})
			err = json.Unmarshal([]byte(redisResp), &m)
			if err != nil {
				h.log.Error("Error while unmarshal redis", logger.Error(err))
			} else {
				resp["data"] = m
				h.handleResponse(c, status_http.OK, resp)
				return
			}
		} else {
			h.log.Error("Error while getting redis while get list ", logger.Error(err))
		}
	}

	resp, err := service.GetListSlimV2(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		statusHttp = status_http.GrpcStatusToHTTP["Internal"]
		stat, ok := status.FromError(err)
		if ok {
			statusHttp = status_http.GrpcStatusToHTTP[stat.Code().String()]
			statusHttp.CustomMessage = stat.Message()
		}
		h.handleResponse(c, statusHttp, err.Error())
		return
	}

	if err == nil {
		if cast.ToBool(c.Query("is_wait_cached")) {
			jsonData, _ := resp.GetData().MarshalJSON()
			waitSlimResourceMap.WriteBody(slimKey, jsonData)
		} else if resp.IsCached {
			jsonData, _ := resp.GetData().MarshalJSON()
			err = h.redis.SetX(context.Background(), slimKey, string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
			if err != nil {
				h.log.Error("Error while setting redis", logger.Error(err))
			}
		}
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}
