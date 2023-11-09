package handlers

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
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
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
func (h *Handler) GetListV2(c *gin.Context) {
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}
	if viewId, ok := objectRequest.Data["builder_service_view_id"].(string); ok {
		if util.IsValidUUID(viewId) {
			switch resource.ResourceType {
			case pb.ResourceType_MONGODB:
				redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))))
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

				resp, err = services.BuilderService().ObjectBuilder().GroupByColumns(
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
						err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second)
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
			redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))))
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

			resp, err = services.BuilderService().ObjectBuilder().GetList2(
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
					err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second)
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
func (h *Handler) GetListSlimV2(c *gin.Context) {
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))))
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
	resp, err := services.BuilderService().ObjectBuilder().GetListSlimV2(
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
		if resp.IsCached {
			jsonData, _ := resp.GetData().MarshalJSON()
			err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second)
			if err != nil {
				h.log.Error("Error while setting redis", logger.Error(err))
			}
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}
