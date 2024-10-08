package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/security"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"google.golang.org/grpc/status"
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

	projectIDs := []string{"0f111e78-3a93-4bec-945a-2a77e0e0a82d", "25d16930-b1a9-4ae5-ab01-b79cc993f06e"}
	if objectRequest.Data["view_type"] != "CALENDAR" {
		if _, ok := objectRequest.Data["limit"]; ok {
			if cast.ToInt(objectRequest.Data["limit"]) > 40 {
				objectRequest.Data["limit"] = 40
			}
		} else {
			found := false
			for _, id := range projectIDs {
				if projectId == id {
					found = true
					break
				}
			}
			if !found {
				objectRequest.Data["limit"] = 10
			}
		}
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	// defer cancel()

	// service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(c.Request.Context())
	// if err != nil {
	// 	h.handleResponse(c, status_http.InternalServerError, err)
	// 	return
	// }
	// defer conn.Close()
	service := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder()

	if viewId, ok := objectRequest.Data["builder_service_view_id"].(string); ok {
		if util.IsValidUUID(viewId) {
			switch resource.ResourceType {
			case pb.ResourceType_MONGODB:
				if resource.ResourceEnvironmentId == "49ae6c46-5397-4975-b238-320617f0190c" { // starex
					h.handleResponse(c, statusHttp, pb.Empty{})
					return
				}
				redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), projectId.(string), resource.NodeType)
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
				resp, err := services.GoObjectBuilderService().ObjectBuilder().GetGroupByField(
					context.Background(),
					&nb.CommonMessage{
						TableSlug: c.Param("table_slug"),
						Data:      structData,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)

				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}

				statusHttp.CustomMessage = resp.GetCustomMessage()
				h.handleResponse(c, statusHttp, resp)
				return
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
			// redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), projectId.(string), resource.NodeType)
			// if err == nil {
			// 	resp := make(map[string]interface{})
			// 	m := make(map[string]interface{})
			// 	err = json.Unmarshal([]byte(redisResp), &m)
			// 	if err != nil {
			// 		h.log.Error("Error while unmarshal redis", logger.Error(err))
			// 	} else {
			// 		resp["data"] = m
			// 		h.handleResponse(c, status_http.OK, resp)
			// 		return
			// 	}
			// }

			resp, err := services.GoObjectBuilderService().ObjectBuilder().GetList2(
				context.Background(),
				&nb.CommonMessage{
					TableSlug: c.Param("table_slug"),
					Data:      structData,
					ProjectId: resource.ResourceEnvironmentId,
				},
			)

			// if err == nil {
			// 	jsonData, _ := resp.GetData().MarshalJSON()
			// 	err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
			// 	if err != nil {
			// 		h.log.Error("Error while setting redis", logger.Error(err))
			// 	}
			// }

			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			resp.ProjectId = cast.ToString(projectId)
			statusHttp.CustomMessage = resp.GetCustomMessage()
			h.handleResponse(c, statusHttp, resp)
			return
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
		hashed        bool
	)

	queryParams := c.Request.URL.Query()
	if ok := queryParams.Has("data"); ok {
		queryData = queryParams.Get("data")
	}

	if ok := queryParams.Has("data"); ok {
		hashData, err := security.Decrypt(queryParams.Get("data"), h.baseConf.SecretKey)
		if err == nil {
			queryData = strings.TrimSpace(hashData)
			hashed = true
		}
	}

	queryMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(queryData), &queryMap); err != nil {
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

	if projectId == "42ab0799-deff-4f8c-bf3f-64bf9665d304" {
		limit = 100
	} else {
		if limit > 40 {
			limit = 40
		}
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

	userId, _ := c.Get("user_id")
	apiKey := c.GetHeader("X-API-KEY")

	var resource *pb.ServiceResourceModel
	resourceBody, ok := c.Get("resource")
	if resourceBody != "" && ok {
		var resourceList *pb.GetResourceByEnvIDResponse
		err = json.Unmarshal([]byte(resourceBody.(string)), &resourceList)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		for _, resourceObject := range resourceList.ServiceResources {
			if resourceObject.Title == pb.ServiceType_name[1] {
				resource = &pb.ServiceResourceModel{
					Id:                    resourceObject.Id,
					ServiceType:           resourceObject.ServiceType,
					ProjectId:             resourceObject.ProjectId,
					Title:                 resourceObject.Title,
					ResourceId:            resourceObject.ResourceId,
					ResourceEnvironmentId: resourceObject.ResourceEnvironmentId,
					EnvironmentId:         resourceObject.EnvironmentId,
					ResourceType:          resourceObject.ResourceType,
					NodeType:              resourceObject.NodeType,
				}
				break
			}
		}
	} else {
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

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "GET",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &structData,
			ApiKey:    apiKey,
			Type:      "API_KEY",
			TableSlug: c.Param("table_slug"),
		}
	)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		service := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder()
		var slimKey = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("slim-%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId)))
		if !cast.ToBool(c.Query("block_cached")) {
			if cast.ToBool(c.Query("is_wait_cached")) {
				var slimWaitKey = config.CACHE_WAIT + "-slim"
				_, slimOK := h.cache.Get(slimWaitKey)
				if !slimOK {
					h.cache.Add(slimWaitKey, []byte(slimWaitKey), 15*time.Second)
				}

				if slimOK {
					ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
					defer cancel()

					for {
						slimBody, ok := h.cache.Get(slimKey)
						if ok {
							m := make(map[string]interface{})
							err = json.Unmarshal(slimBody, &m)
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
						logReq.Response = m
						go h.versionHistory(logReq)
						return
					}
				} else {
					h.log.Error("Error while getting redis while get list ", logger.Error(err))
				}
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
			logReq.Response = err.Error()
			go h.versionHistory(logReq)
			h.handleResponse(c, statusHttp, err.Error())
			return
		}

		logReq.Response = resp
		go h.versionHistory(logReq)

		if !cast.ToBool(c.Query("block_cached")) {
			jsonData, _ := resp.GetData().MarshalJSON()
			if cast.ToBool(c.Query("is_wait_cached")) {
				h.cache.Add(slimKey, jsonData, 15*time.Second)
			} else if resp.IsCached {
				err = h.redis.SetX(context.Background(), slimKey, string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
				if err != nil {
					h.log.Error("Error while setting redis", logger.Error(err))
				}
			}
		}

		statusHttp.CustomMessage = resp.GetCustomMessage()

		if hashed {
			hash, err := security.Encrypt(resp, h.baseConf.SecretKey)
			if err != nil {
				h.handleResponse(c, status_http.InternalServerError, err.Error())
				return
			}

			h.handleResponse(c, statusHttp, hash)
			return
		}
		h.handleResponse(c, statusHttp, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().ObjectBuilder().GetListSlim(context.Background(),
			&nb.CommonMessage{
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
			logReq.Response = err.Error()
			go h.versionHistoryGo(c, logReq)
			h.handleResponse(c, statusHttp, err.Error())
			return
		}

		logReq.Response = resp
		go h.versionHistoryGo(c, logReq)

		// if err == nil {
		// 	if resp.IsCached {
		// 		jsonData, _ := resp.GetData().MarshalJSON()
		// 		err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second)
		// 		if err != nil {
		// 			h.log.Error("Error while setting redis", logger.Error(err))
		// 		}
		// 	}
		// }

		statusHttp.CustomMessage = resp.GetCustomMessage()
		h.handleResponse(c, statusHttp, resp)
	}

}

// UpdateWithParams godoc
// @Security ApiKeyAuth
// @ID update_with_params
// @Router /v2/update-with/{collection} [PUT]
// @Summary UpdateWith Params
// @Description UpdateWith Params
// @Tags Object
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param item body models.CommonMessage true "UpdateWithParamsBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateWithParams(c *gin.Context) {

	var (
		objectRequest models.CommonMessage
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	body, err := services.GoObjectBuilderService().ObjectBuilder().UpdateWithParams(
		context.Background(),
		&nb.CommonMessage{
			TableSlug: c.Param("collection"),
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, body)
}
