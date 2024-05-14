package v1

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pba "ucode/ucode_go_api_gateway/genproto/auth_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"

	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"encoding/base64"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// CreateObject godoc
// @Security ApiKeyAuth
// @ID create_object
// @Router /v1/object/{table_slug}/ [POST]
// @Summary Create object
// @Description Create object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "CreateObjectRequestBody"
// @Success 201 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateObject(c *gin.Context) {
	var (
		objectRequest               models.CommonMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Created"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	objectRequest.Data["company_service_project_id"] = resource.GetProjectId()
	objectRequest.Data["company_service_environment_id"] = resource.GetEnvironmentId()

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
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	var id string
	uid, _ := uuid.NewRandom()
	id = uid.String()

	guid, ok := objectRequest.Data["guid"]
	if ok {
		if util.IsValidUUID(guid.(string)) {
			id = objectRequest.Data["guid"].(string)
		}
	}

	objectRequest.Data["guid"] = id

	// THIS for loop is written to create child objects (right now it is used in the case of One2One relation)
	//start = time.Now()
	for key, value := range objectRequest.Data {
		if key[0] == '$' {

			interfaceToMap := value.(map[string]interface{})

			id, _ := uuid.NewRandom()
			interfaceToMap["guid"] = id

			mapToStruct, err := helper.ConvertMapToStruct(interfaceToMap)
			if err != nil {
				h.handleResponse(c, status_http.InvalidArgument, err.Error())
				return
			}

			_, err = service.Create(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: key[1:],
					Data:      mapToStruct,
					ProjectId: resource.ResourceEnvironmentId,
				},
			)

			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			objectRequest.Data[key[1:]+"_id"] = id
		}
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)

	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	//start = time.Now()
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "CREATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	//start = time.Now()
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{id},
			TableSlug:    c.Param("table_slug"),
			ObjectData:   objectRequest.Data,
			Method:       "CREATE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	//start = time.Now()
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.Create(
			context.Background(),
			&obs.CommonMessage{
				TableSlug:         c.Param("table_slug"),
				Data:              structData,
				ProjectId:         resource.ResourceEnvironmentId,
				BlockedLoginTable: cast.ToBool(c.DefaultQuery("blocked_login_table", "false")),
				BlockedBuilder:    cast.ToBool(c.DefaultQuery("block_builder", "false"))},
		)
		// this logic for custom error message, object builder service may be return 400, 404, 500
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().Create(
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

	if data, ok := resp.Data.AsMap()["data"].(map[string]interface{}); ok {
		objectRequest.Data = data
		if _, ok = data["guid"].(string); ok {
			id = data["guid"].(string)
		}
	}
	//start = time.Now()
	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{id},
				TableSlug:    c.Param("table_slug"),
				ObjectData:   objectRequest.Data,
				Method:       "CREATE",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetSingle godoc
// @Security ApiKeyAuth
// @ID get_object_by_id
// @Router /v1/object/{table_slug}/{object_id} [GET]
// @Summary Get object by id
// @Description Get object by id
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object_id path string true "object_id"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingle(c *gin.Context) {
	var (
		object     models.CommonMessage
		resp       *obs.CommonMessage
		statusHttp = status_http.GrpcStatusToHTTP["Ok"]
	)

	object.Data = make(map[string]interface{})

	objectID := c.Param("object_id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, status_http.InvalidArgument, "object_id is an invalid uuid")
		return
	}

	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	if tokenInfo != nil {
		object.Data["user_id_from_token"] = tokenInfo.GetUserId()
		object.Data["role_id_from_token"] = tokenInfo.GetRoleId()
		object.Data["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
	}

	object.Data["id"] = objectID

	structData, err := helper.ConvertMapToStruct(object.Data)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.GetSingle(
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().GetSingle(
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
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetSingleSlim godoc
// @Security ApiKeyAuth
// @ID get_object_by_id_slim
// @Router /v1/object-slim/{table_slug}/{object_id} [GET]
// @Summary Get object by id slim
// @Description Get object by id slim
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object_id path string true "object_id"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingleSlim(c *gin.Context) {
	var (
		object     models.CommonMessage
		statusHttp = status_http.GrpcStatusToHTTP["Ok"]
	)

	object.Data = make(map[string]interface{})

	objectID := c.Param("object_id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, status_http.InvalidArgument, "object_id is an invalid uuid")
		return
	}

	object.Data["id"] = objectID

	structData, err := helper.ConvertMapToStruct(object.Data)
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

	userId, _ := c.Get("user_id")

	apiKey := c.GetHeader("X-API-KEY")
	if apiKey != "" {
		canRequest, exists := h.cache.GetValue(apiKey + "slim")
		if !exists {
			apiKeyLimit, err := h.authService.ApiKeyUsage().CheckLimit(
				c.Request.Context(),
				&pba.CheckLimitRequest{ApiKey: apiKey},
			)
			if err != nil || apiKeyLimit.IsLimitReached {
				h.handleResponse(c, status_http.TooManyRequests, err.Error())
				return
			}

			canRequest = !apiKeyLimit.IsLimitReached
			h.cache.AddKey(apiKey+"slim", true, time.Minute)
		}

		if !canRequest {
			h.handleResponse(c, status_http.TooManyRequests, "Monthly limit reached")
			return
		}
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

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
			logReq.Response = m
			go h.versionHistory(c, logReq)
			return
		}
	} else {
		h.log.Error("Error while getting redis", logger.Error(err))
	}

	resp, err := service.GetSingleSlim(
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
		go h.versionHistory(c, logReq)
		h.handleResponse(c, statusHttp, err.Error())
		return
	}

	logReq.Response = resp
	go h.versionHistory(c, logReq)

	if resp.IsCached {
		jsonData, _ := resp.GetData().MarshalJSON()
		err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
		if err != nil {
			h.log.Error("Error while setting redis", logger.Error(err))
		}
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// UpdateObject godoc
// @Security ApiKeyAuth
// @ID update_object
// @Router /v1/object/{table_slug} [PUT]
// @Summary Update object
// @Description Update object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "UpdateObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateObject(c *gin.Context) {
	var (
		objectRequest               models.CommonMessage
		resp, singleObject          *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Ok"]
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
	var id string
	if objectRequest.Data["guid"] != nil {
		id = objectRequest.Data["guid"].(string)
	} else {
		h.handleResponse(c, status_http.BadRequest, "guid is required")
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

	var resource *pb.ServiceResourceModel
	resourceBody, ok := c.Get("resource")
	if ok {
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

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(c.Request.Context())
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	fromOfs := c.Query("from-ofs")

	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "UPDATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}

		switch resource.ResourceType {
		case pb.ResourceType_MONGODB:
			singleObject, err = service.GetSingle(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: c.Param("table_slug"),
					Data: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"id": structpb.NewStringValue(id),
						},
					},
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
		case pb.ResourceType_POSTGRESQL:
			singleObject, err = services.PostgresBuilderService().ObjectBuilder().GetSingle(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: c.Param("table_slug"),
					Data: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"id": structpb.NewStringValue(id),
						},
					},
					ProjectId: resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{id},
			TableSlug:    c.Param("table_slug"),
			ObjectData:   objectRequest.Data,
			Method:       "UPDATE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.Update(
			context.Background(),
			&obs.CommonMessage{
				TableSlug:      c.Param("table_slug"),
				Data:           structData,
				ProjectId:      resource.ResourceEnvironmentId,
				BlockedBuilder: cast.ToBool(c.DefaultQuery("block_builder", "false")),
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().Update(
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

	if c.Param("table_slug") == "record_permission" {
		if objectRequest.Data["role_id"] == nil {
			err := errors.New("role id must be have in update permission")
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		_, err = h.authService.Session().UpdateSessionsByRoleId(
			context.Background(),
			&pba.UpdateSessionByRoleIdRequest{
				RoleId:    objectRequest.Data["role_id"].(string),
				IsChanged: true,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents:           afterActions,
				IDs:                    []string{id},
				TableSlug:              c.Param("table_slug"),
				ObjectData:             objectRequest.Data,
				Method:                 "UPDATE",
				ObjectDataBeforeUpdate: singleObject.Data.AsMap(),
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, status_http.OK, resp)
}

// DeleteObject godoc
// @Security ApiKeyAuth
// @ID delete_object
// @Router /v1/object/{table_slug}/{object_id} [DELETE]
// @Summary Delete object
// @Description Delete object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "DeleteObjectRequestBody"
// @Param object_id path string true "object_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteObject(c *gin.Context) {
	var (
		objectRequest               models.CommonMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["NoContent"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	objectID := c.Param("object_id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, status_http.InvalidArgument, "object id is an invalid uuid")
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

	resource, _ := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	objectRequest.Data["id"] = objectID
	objectRequest.Data["company_service_project_id"] = projectId.(string)
	objectRequest.Data["company_service_environment_id"] = environmentId.(string)

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "DELETE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{objectID},
			TableSlug:    c.Param("table_slug"),
			ObjectData:   objectRequest.Data,
			Method:       "DELETE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.Delete(
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
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().Delete(
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

	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{objectID},
				TableSlug:    c.Param("table_slug"),
				ObjectData:   objectRequest.Data,
				Method:       "DELETE",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetList godoc
// @Security ApiKeyAuth
// @ID get_list_objects
// @Router /v1/object/get-list/{table_slug} [POST]
// @Summary Get all objects
// @Description Get all objects
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param language_setting query string false "language_setting"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetList(c *gin.Context) {
	var (
		objectRequest               models.CommonMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Ok"]
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

	if c.Param("table_slug") == "orders" {
		fmt.Println("\n\n role_id ~~~>>> ", objectRequest.Data["role_id_from_token"])
		fmt.Println("\n\n")
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
	fmt.Println("\n\n>>>>>>>>> test #2")
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
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "GETLIST", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}

	go func() {
		if len(beforeActions) > 0 {
			functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
				CustomEvents: beforeActions,
				TableSlug:    c.Param("table_slug"),
				ObjectData:   objectRequest.Data,
				Method:       "GETLIST",
			},
				c,
				h,
			)
			if err != nil {
				h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
				return
			}
		}
	}()

	if viewId, ok := objectRequest.Data["builder_service_view_id"].(string); ok {
		if util.IsValidUUID(viewId) {
			switch resource.ResourceType {
			case pb.ResourceType_MONGODB:
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

				resp, err = service.GroupByColumns(
					context.Background(),
					&obs.CommonMessage{
						TableSlug: c.Param("table_slug"),
						Data:      structData,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)

				if err == nil {
					// if resp.IsCached {
					// 	jsonData, _ := resp.GetData().MarshalJSON()
					// 	err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
					// 	if err != nil {
					// 		h.log.Error("Error while setting redis", logger.Error(err))
					// 	}
					// }
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
					h.handleResponse(c, status_http.OK, resp)
					return
				}
			}

			resp, err = service.GetList(
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

			fmt.Println("HELELEOOEL")
			fmt.Println(resource.ResourceEnvironmentId)

			resp, err := services.GoObjectBuilderService().ObjectBuilder().GetAll(
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

	go func() {
		if len(afterActions) > 0 {
			functionName, err := DoInvokeFuntion(
				DoInvokeFuntionStruct{
					CustomEvents:           afterActions,
					TableSlug:              c.Param("table_slug"),
					ObjectData:             objectRequest.Data,
					Method:                 "GETLIST",
					ObjectDataBeforeUpdate: resp.Data.AsMap(),
				},
				c, // gin context,
				h, // handler
			)
			if err != nil {
				h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
				return
			}
		}
	}()

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetListSlim godoc
// @Security ApiKeyAuth
// @ID get_list_objects_slim
// @Router /v1/object-slim/get-list/{table_slug} [GET]
// @Summary Get all objects slim
// @Description Get all objects slim
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
func (h *HandlerV1) GetListSlim(c *gin.Context) {
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

	//if _, ok := queryMap["limit"]; !ok {
	//	queryMap["limit"] = 10
	//}
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
	if withRelation, ok := objectRequest.Data["with_relations"]; ok {
		if withRelation.(bool) {
			var (
				client, role string
			)
			if clientTypeId, ok := c.Get("client_type_id"); ok {
				client = clientTypeId.(string)
			}
			if roleId, ok := c.Get("role_id"); ok {
				role = roleId.(string)
			}
			if client != "" && role != "" {
				objectRequest.Data["client_type_id"] = client
				objectRequest.Data["role_id"] = role
			}
		}
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

	userId, _ := c.Get("user_id")

	apiKey := c.GetHeader("X-API-KEY")
	if apiKey != "" {
		canRequest, exists := h.cache.GetValue(apiKey + "slim")
		if !exists {
			apiKeyLimit, err := h.authService.ApiKeyUsage().CheckLimit(
				c.Request.Context(),
				&pba.CheckLimitRequest{ApiKey: apiKey},
			)
			if err != nil || apiKeyLimit.IsLimitReached {
				h.handleResponse(c, status_http.TooManyRequests, err.Error())
				return
			}

			canRequest = !apiKeyLimit.IsLimitReached
			h.cache.AddKey(apiKey+"slim", true, time.Minute)
		}

		if !canRequest {
			h.handleResponse(c, status_http.TooManyRequests, "Monthly limit reached")
			return
		}
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

	// service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(c.Request.Context())
	// if err != nil {
	// 	h.handleResponse(c, status_http.InternalServerError, err)
	// 	return
	// }
	// defer conn.Close()
	service := services.BuilderService().ObjectBuilder()

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
			logReq.Response = m
			go h.versionHistory(c, logReq)
			return
		}
	} else {
		h.log.Error("Error while getting redis while get list ", logger.Error(err))
	}
	resp, err := service.GetListSlim(
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
		go h.versionHistory(c, logReq)
		h.handleResponse(c, statusHttp, err.Error())
		return
	}

	logReq.Response = resp
	go h.versionHistory(c, logReq)

	if resp.IsCached {
		jsonData, _ := resp.GetData().MarshalJSON()
		err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
		if err != nil {
			h.log.Error("Error while setting redis", logger.Error(err))
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetListInExcel godoc
// @Security ApiKeyAuth
// @ID get_list_objects_in_excel
// @Router /v1/object/excel/{table_slug} [POST]
// @Summary Get all objects in excel
// @Description Get all objects in excel
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListInExcel(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := service.GetListInExcel(
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

		statusHttp.CustomMessage = resp.GetCustomMessage()
		h.handleResponse(c, statusHttp, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().ObjectBuilder().GetListInExcel(
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
	}

}

// DeleteManyToMany godoc
// @Security ApiKeyAuth
// @ID delete_many2many
// @Router /v1/many-to-many [DELETE]
// @Summary Delete Many2Many
// @Description Delete Many2Many
// @Tags Object
// @Accept json
// @Produce json
// @Param object body obs.ManyToManyMessage true "DeleteManyToManyBody"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteManyToMany(c *gin.Context) {
	var (
		m2mMessage                  obs.ManyToManyMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["NoContent"]
	)

	err := c.ShouldBindJSON(&m2mMessage)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	m2mMessage.ProjectId = resource.ResourceEnvironmentId
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "DELETE_MANY2MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{m2mMessage.IdFrom},
			TableSlug:    m2mMessage.TableTo,
			ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_to": m2mMessage.TableFrom},
			Method:       "DELETE_MANY2MANY",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.ManyToManyDelete(
			context.Background(),
			&m2mMessage,
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().ManyToManyDelete(
			context.Background(),
			&m2mMessage,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{m2mMessage.IdFrom},
				TableSlug:    c.Param("table_slug"),
				ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_from": m2mMessage.TableTo},
				Method:       "DELETE_MANY2MANY",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, status_http.NoContent, resp)
}

// AppendManyToMany godoc
// @Security ApiKeyAuth
// @ID append_many2many
// @Router /v1/many-to-many [PUT]
// @Summary Update many-to-many
// @Description Update many-to-many
// @Tags Object
// @Accept json
// @Produce json
// @Param object body obs.ManyToManyMessage true "UpdateMany2ManyRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) AppendManyToMany(c *gin.Context) {
	var (
		m2mMessage                  obs.ManyToManyMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Ok"]
	)

	err := c.ShouldBindJSON(&m2mMessage)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	m2mMessage.ProjectId = resource.ResourceEnvironmentId
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "APPEND_MANY2MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{m2mMessage.IdFrom},
			TableSlug:    m2mMessage.TableTo,
			ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_from": m2mMessage.TableFrom},
			Method:       "APPEND_MANY2MANY",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.ManyToManyAppend(
			context.Background(),
			&m2mMessage,
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().ManyToManyAppend(
			context.Background(),
			&m2mMessage,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{m2mMessage.IdFrom},
				TableSlug:    m2mMessage.TableTo,
				ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_from": m2mMessage.TableFrom},
				Method:       "APPEND_MANY2MANY",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// UpsertObject godoc
// @Security ApiKeyAuth
// @ID upsert_object
// @Router /v1/object-upsert/{table_slug} [POST]
// @Summary Upsert object
// @Description Upsert object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.UpsertCommonMessage true "CreateObjectRequestBody"
// @Success 201 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpsertObject(c *gin.Context) {
	var (
		objectRequest               models.UpsertCommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Created"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	objects := objectRequest.Data["objects"].([]interface{})
	editedObjects := make([]interface{}, 0, len(objects))
	var objectIds = make([]string, 0, len(objects))
	for _, object := range objects {
		newObject := object.(map[string]interface{})
		if newObject["guid"] == nil || newObject["guid"] == "" {
			guid, _ := uuid.NewRandom()
			newObject["guid"] = guid.String()
			newObject["is_new"] = true
		}
		objectIds = append(objectIds, newObject["guid"].(string))
		editedObjects = append(editedObjects, newObject)
	}
	objectRequest.Data["objects"] = editedObjects
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "MULTIPLE_UPDATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{},
			TableSlug:    c.Param("table_slug"),
			ObjectData:   objectRequest.Data,
			Method:       "MULTIPLE_UPDATE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	// THIS for loop is written to create child objects (right now it is used in the case of One2One relation)
	for key, value := range objectRequest.Data {
		if key[0] == '$' {

			interfaceToMap := value.(map[string]interface{})

			id, _ := uuid.NewRandom()
			interfaceToMap["guid"] = id

			_, err := helper.ConvertMapToStruct(interfaceToMap)
			if err != nil {
				h.handleResponse(c, status_http.InvalidArgument, err.Error())
				return
			}
			// _, err = services.GetBuilderServiceByType(resource.NodeType)..ObjectBuilde().Create(
			// 	context.Background(),
			// 	&obs.CommonMessage{
			// 		TableSlug: key[1:],
			// 		Data:      mapToStruct,
			// 	},
			// )

			objectRequest.Data[key[1:]+"_id"] = id
		}
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	var resp *obs.CommonMessage
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.Batch(
			context.Background(),
			&obs.BatchRequest{
				TableSlug:     c.Param("table_slug"),
				Data:          structData,
				UpdatedFields: objectRequest.UpdatedFields,
				ProjectId:     resource.ResourceEnvironmentId,
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().Batch(
			context.Background(),
			&obs.BatchRequest{
				TableSlug:     c.Param("table_slug"),
				Data:          structData,
				UpdatedFields: objectRequest.UpdatedFields,
				ProjectId:     resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	if c.Param("table_slug") == "record_permission" {
		roleId := objectRequest.Data["objects"].([]interface{})[0].(map[string]interface{})["role_id"]
		if roleId == nil {
			err := errors.New("role id must be have in upsert permission")
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		_, err = h.authService.Session().UpdateSessionsByRoleId(
			context.Background(),
			&pba.UpdateSessionByRoleIdRequest{
				RoleId:    objectRequest.Data["role_id"].(string),
				IsChanged: true,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          objectIds,
				TableSlug:    c.Param("table_slug"),
				ObjectData:   objectRequest.Data,
				Method:       "MULTIPLE_UPDATE",
			},
			c, // gin context
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, status_http.Created, resp)
}

// MultipleUpdateObject godoc
// @Security ApiKeyAuth
// @ID multiple_update_object
// @Router /v1/object/multiple-update/{table_slug} [PUT]
// @Summary Multiple Update object
// @Description Multiple Update object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "MultipleUpdateObjectRequestBody"
// @Success 201 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) MultipleUpdateObject(c *gin.Context) {
	var (
		objectRequest               models.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Created"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	var resource *pb.ServiceResourceModel
	resourceBody, ok := c.Get("resource")
	if ok {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	objects := objectRequest.Data["objects"].([]interface{})
	editedObjects := make([]map[string]interface{}, 0, len(objects))
	var objectIds = make([]string, 0, len(objects))
	for _, object := range objects {
		newObjects := object.(map[string]interface{})
		_, ok := newObjects["guid"].(string)
		if !ok {

			id, _ := uuid.NewRandom()
			newObjects["guid"] = id.String()
			newObjects["is_new"] = true

		}
		newObjects["company_service_project_id"] = resource.GetProjectId()
		objectIds = append(objectIds, newObjects["guid"].(string))
		editedObjects = append(editedObjects, newObjects)
	}
	structData, err := helper.ConvertMapToStruct(objectRequest.Data)

	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	objectRequest.Data["objects"] = editedObjects

	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "MULTIPLE_UPDATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          objectIds,
			TableSlug:    c.Param("table_slug"),
			ObjectData:   objectRequest.Data,
			Method:       "MULTIPLE_UPDATE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	var resp *obs.CommonMessage
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.MultipleUpdate(
			context.Background(),
			&obs.CommonMessage{
				TableSlug:      c.Param("table_slug"),
				Data:           structData,
				ProjectId:      resource.ResourceEnvironmentId,
				BlockedBuilder: cast.ToBool(c.DefaultQuery("block_builder", "false")),
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = service.MultipleUpdate(
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

	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          objectIds,
				TableSlug:    c.Param("table_slug"),
				ObjectData:   objectRequest.Data,
				Method:       "MULTIPLE_UPDATE",
			},
			c, // gin context
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, status_http.Created, resp)
}

// GetFinancialAnalytics godoc
// @Security ApiKeyAuth
// @ID get_financial_analytics
// @Router /v1/object/get-financial-analytics/{table_slug} [POST]
// @Summary Get financial analytics
// @Description Get financial analytics
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetFinancialAnalyticsRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetFinancialAnalytics(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	//tokenInfo := h.GetAuthInfo
	objectRequest.Data["tables"] = authInfo.GetTables()
	objectRequest.Data["user_id_from_token"] = authInfo.GetUserId()
	objectRequest.Data["role_id_from_token"] = authInfo.GetRoleId()
	objectRequest.Data["client_type_id_from_token"] = authInfo.GetClientTypeId()
	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	resp, err := service.GetFinancialAnalytics(
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
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, status_http.OK, resp)
}

// GetListGroupByObject godoc
// @Security ApiKeyAuth
// @ID get_list_group_by_objects
// @Router /v1/object/get-list-group-by/{table_slug}/{column_table_slug} [POST]
// @Summary Get List Group By Object
// @Description Get List Group By Object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param column_table_slug path string true "column_table_slug"
// @Param limit query string false "limit"
// @Param search query string false "search"
// @Param project query string false "project"
// @Param object body models.CommonMessage true "GetGroupByFieldObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListGroupBy(c *gin.Context) {

	var object models.CommonMessage

	err := c.ShouldBindJSON(&object)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if c.Param("table_slug") == "" {
		h.handleResponse(c, status_http.BadRequest, "table_slug required")
		return
	}

	if c.Param("column_table_slug") == "" {
		h.handleResponse(c, status_http.BadRequest, "table_slug required")
		return
	}

	relationSlug := c.Param("column_table_slug")
	selectedGuid := cast.ToSlice(object.Data["additional_values"])
	relationTableSlug := relationSlug
	relationTableSlug = util.PluralizeWord(relationSlug)
	// if relationSlug[len(relationSlug)-1] != 's' {
	// 	relationTableSlug = relationSlug + "s"
	// }

	object.Data = map[string]interface{}{
		"match": map[string]interface{}{
			"$match": map[string]interface{}{},
		},
		"lookups": []interface{}{
			map[string]interface{}{
				"$lookup": map[string]interface{}{
					"from":         relationTableSlug,
					"localField":   relationSlug + "_id",
					"foreignField": "guid",
					"as":           relationSlug + "_details",
				},
			},
		},
		"query": map[string]interface{}{
			"$group": map[string]interface{}{
				"_id":                "$" + relationSlug + "_id",
				"guid":               map[string]interface{}{"$first": "$" + relationSlug + "_id"},
				relationSlug + "_id": map[string]interface{}{"$first": "$" + relationSlug + "_id"},
			},
		},
		"sort": map[string]interface{}{
			"$sort": map[string]interface{}{
				"_id": 1,
			},
		},
	}

	if c.Query("limit") != "" {
		object.Data["limit"] = cast.ToInt(c.Query("limit"))
	}

	if c.Query("project") != "" {
		object.Data["project"] = map[string]interface{}{
			"$project": map[string]interface{}{
				"_id":              0,
				"guid":             1,
				c.Query("project"): map[string]interface{}{"$first": "$" + relationSlug + "_details." + c.Query("project")},
			},
		}
	}

	if c.Query("project") != "" && c.Query("search") != "" {
		object.Data["second_match"] = map[string]interface{}{
			relationSlug + "_details." + c.Query("project"): map[string]interface{}{
				"$regex":   c.Query("search"),
				"$options": "i",
			},
		}
		object.Data["sort"] = map[string]interface{}{"$sort": map[string]interface{}{c.Query("project"): 1}}
	}

	if len(selectedGuid) > 0 {
		object.Data["match"] = map[string]interface{}{
			"$match": map[string]interface{}{
				relationSlug + "_id": map[string]interface{}{"$nin": selectedGuid},
			},
		}
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	structData, err := helper.ConvertMapToStruct(object.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	tableResp, err := service.GetGroupByField(
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

	var (
		count    = cast.ToInt(tableResp.Data.AsMap()["count"])
		response = cast.ToSlice(tableResp.Data.AsMap()["response"])
	)

	if len(selectedGuid) > 0 {
		object.Data["match"] = map[string]interface{}{
			"$match": map[string]interface{}{
				relationSlug + "_id": map[string]interface{}{"$in": selectedGuid},
			},
		}

		delete(object.Data, "second_match")
		delete(object.Data, "limit")

		response := cast.ToSlice(tableResp.Data.AsMap()["response"])
		response = append(response, selectedGuid...)

		structData, err = helper.ConvertMapToStruct(object.Data)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}

		selectedTableResp, err := service.GetGroupByField(
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

		count += cast.ToInt(selectedTableResp.Data.AsMap()["count"])

		response = append(response, cast.ToSlice(tableResp.Data.AsMap()["response"])...)

		h.handleResponse(c, status_http.OK, struct {
			TableSlug string                 `json:"table_slug"`
			Data      map[string]interface{} `json:"data"`
		}{
			TableSlug: tableResp.TableSlug,
			Data: map[string]interface{}{
				"count":    count,
				"response": response,
			},
		})
		return
	}

	h.handleResponse(c, status_http.OK, struct {
		TableSlug string                 `json:"table_slug"`
		Data      map[string]interface{} `json:"data"`
	}{
		TableSlug: tableResp.TableSlug,
		Data: map[string]interface{}{
			"count":    count,
			"response": response,
		},
	})
}

// GetGroupByFieldObject godoc
// @Security ApiKeyAuth
// @ID get_group_by_field_objects
// @Router /v1/object/get-group-by-field/{table_slug} [POST]
// @Summary Get Group By Field Object
// @Description Get Group By Field Object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetGroupByFieldObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetGroupByField(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		resp          *obs.CommonMessage
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		// start := time.Now()

		redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("group-%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), projectId.(string), resource.NodeType)
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

		resp, err = service.GetGroupByField(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("table_slug"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err == nil {
			jsonData, _ := resp.GetData().MarshalJSON()
			err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("group-%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
			if err != nil {
				h.log.Error("Error while setting redis", logger.Error(err))
			}
		}

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteManyObject godoc
// @Security ApiKeyAuth
// @ID delete_many_object
// @Router /v1/object/{table_slug} [DELETE]
// @Summary Delete many objects
// @Description Delete many objects
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.Ids true "DeleteManyObjectRequestBody"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteManyObject(c *gin.Context) {
	var (
		objectRequest               models.Ids
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["NoContent"]
		data                        = make(map[string]interface{})
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	resource, _ := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	data["company_service_project_id"] = projectId.(string)
	data["company_service_environment_id"] = environmentId.(string)
	data["ids"] = objectRequest.Ids

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	structData, err := helper.ConvertMapToStruct(data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "DELETE_MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          objectRequest.Ids,
			TableSlug:    c.Param("table_slug"),
			ObjectData:   data,
			Method:       "DELETE_MANY",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.DeleteMany(
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
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().DeleteMany(
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

	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          objectRequest.Ids,
				TableSlug:    c.Param("table_slug"),
				ObjectData:   data,
				Method:       "DELETE_MANY",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetListWithOut godoc
// @Security ApiKeyAuth
// @ID get_list_objects_without_relation
// @Router /v1/object/get-list-without-relation/{table_slug} [POST]
// @Summary Get all objects without relation
// @Description Get all objects without relation
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListWithOutRelation(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(4))
	defer cancel()

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(ctx)
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		// start := time.Now()

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

		resp, err = service.GetListWithOutRelations(
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
		resp, err = services.PostgresBuilderService().ObjectBuilder().GetListWithOutRelations(
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

	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}

// GetListAggregate godoc
// @Security ApiKeyAuth
// @ID get_list_aggregate
// @Router /v1/object/get-list-aggregate/{table_slug} [POST]
// @Summary Get List Aggregate
// @Description Get List Aggregate
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Param sort_type query string false "sort_type"
// @Param object body models.CommonMessage true "GetGroupByFieldObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListAggregate(c *gin.Context) {

	var (
		objectRequest models.CommonMessage
		resp          *obs.CommonMessage
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	service, conn, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderConnPool(c.Request.Context())
	if err != nil {
		h.log.Info("Error while getting "+resource.NodeType+" object builder service", logger.Error(err))
		h.log.Info("ConnectionPool", logger.Any("CONNECTION", conn))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer conn.Close()

	var (
		object    models.CommonMessage
		fieldResp = &obs.GetAllFieldsResponse{}
	)

	if len(cast.ToSlice(objectRequest.Data["group_selects"])) <= 0 || len(cast.ToSlice(objectRequest.Data["projects"])) <= 0 {
		var (
			fieldKey     = fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), projectId.(string), environmentId.(string))
			fieldKeyWait = config.CACHE_WAIT + "-field"
		)

		_, fieldOK := h.cache.Get(fieldKeyWait)
		if !fieldOK {
			h.cache.Add(fieldKeyWait, []byte(fieldKeyWait), config.REDIS_KEY_TIMEOUT)
		}

		if fieldOK {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()
			for {
				fieldBody, ok := h.cache.Get(fieldKey)
				if ok {
					err = json.Unmarshal(fieldBody, &fieldResp)
					if err != nil {
						h.handleResponse(c, status_http.BadRequest, "cant get auth info")
						c.Abort()
						return
					}
				}

				if len(fieldResp.Fields) >= 0 {
					break
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		}

		if len(fieldResp.Fields) <= 0 {
			fieldResp, err = services.GetBuilderServiceByType(resource.NodeType).Field().GetAll(context.Background(), &obs.GetAllFieldsRequest{
				TableSlug: c.Param("table_slug"),
				ProjectId: resource.ResourceEnvironmentId,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			go func() {
				body, err := json.Marshal(fieldResp)
				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
				h.cache.Add(fieldKey, body, config.REDIS_TIMEOUT)
			}()
		}
	}

	object.Data = map[string]interface{}{
		"match": map[string]interface{}{"$match": map[string]interface{}{}},
		"sort":  map[string]interface{}{"$sort": map[string]interface{}{"_id": 1}},
	}

	if c.Query("limit") != "" {
		object.Data["limit"] = cast.ToInt(c.Query("limit"))
	}

	if c.Query("offset") != "" {
		object.Data["offset"] = cast.ToInt(c.Query("offset"))
	}

	if _, ok := objectRequest.Data["search"]; ok {
		var (
			searchQuery = map[string]interface{}{}
			search      = cast.ToStringMap(objectRequest.Data["search"])
		)
		for key, value := range search {
			if cast.ToString(value) == "" {
				searchQuery[key] = value
			} else {
				searchQuery[key] = map[string]interface{}{"$regex": value, "$options": "i"}
			}
		}
		object.Data["second_match"] = searchQuery
	}

	if _, ok := objectRequest.Data["match"]; ok {
		var (
			matchQuery = map[string]interface{}{}
			match      = cast.ToStringMap(objectRequest.Data["match"])
		)
		for key, value := range match {
			matchQuery[key] = value
		}
		object.Data["match"] = map[string]interface{}{"$match": matchQuery}
	}

	var groupQuery = map[string]interface{}{"_id": "$guid"}
	if _, ok := objectRequest.Data["groups"]; ok {
		var (
			groupSlice = cast.ToStringSlice(objectRequest.Data["groups"])
			groupLen   = len(groupSlice)
		)
		if groupLen > 0 {
			if groupLen > 1 {
				var manyGroup = map[string]interface{}{}
				for _, value := range groupSlice {
					manyGroup[value] = "$" + value
				}
				groupQuery = map[string]interface{}{"_id": manyGroup}
			} else {
				groupQuery = map[string]interface{}{"_id": "$" + groupSlice[0]}
			}
		}
	}
	groupQuery["guid"] = map[string]interface{}{"$first": "$guid"}

	if _, ok := objectRequest.Data["group_selects"]; ok {
		var groupSelects = cast.ToStringSlice(objectRequest.Data["group_selects"])
		for _, value := range groupSelects {
			groupQuery[value] = map[string]interface{}{"$first": "$" + value}
		}
	} else {
		for _, field := range fieldResp.Fields {
			groupQuery[field.Slug] = map[string]interface{}{"$first": "$" + field.Slug}
		}
	}

	if _, ok := objectRequest.Data["group_query"]; ok {
		var groupQueryRequest = cast.ToStringMap(objectRequest.Data["group_query"])
		for key, value := range groupQueryRequest {
			groupQuery[key] = value
		}
	}
	object.Data["query"] = map[string]interface{}{"$group": groupQuery}

	if _, ok := objectRequest.Data["lookups"]; ok {
		var (
			lookups      = cast.ToSlice(objectRequest.Data["lookups"])
			lookupsQuery = []interface{}{}
		)
		for _, objectLookup := range lookups {
			var (
				lookupMap     = cast.ToStringMap(objectLookup)
				fromSlug      = cast.ToString(lookupMap["from"])
				lastCharacter string
			)

			if len(fromSlug) > 0 {
				lastCharacter = string(fromSlug[len(fromSlug)-1])
			}

			if lastCharacter != "s" {
				if lastCharacter == "y" {
					fromSlug = fromSlug[:len(fromSlug)-1]
					fromSlug += "ies"
				} else {
					fromSlug += "s"
				}
			}

			lookupQuery := map[string]interface{}{
				"from":         fromSlug,
				"foreignField": lookupMap["from_field"],
				"localField":   lookupMap["to_field"],
				"as":           lookupMap["as"],
			}

			if len(cast.ToSlice(lookupMap["pipeline"])) > 0 {
				lookupQuery["pipeline"] = cast.ToSlice(lookupMap["pipeline"])
			}

			lookupsQuery = append(lookupsQuery, map[string]interface{}{"$lookup": lookupQuery})
		}
		object.Data["lookups"] = lookupsQuery
	}

	var projectQuery = map[string]interface{}{"_id": 0, "guid": 1}
	if _, ok := objectRequest.Data["projects"]; ok {
		var projects = cast.ToStringSlice(objectRequest.Data["projects"])
		for _, value := range projects {
			if strings.Contains(value, ".") {
				var key = strings.ReplaceAll(value, ".", "_")
				projectQuery[key] = map[string]interface{}{"$first": "$" + value}
			} else {
				projectQuery[value] = 1
			}
		}
	} else {
		for _, field := range fieldResp.Fields {
			projectQuery[field.Slug] = 1
		}
	}

	if _, ok := objectRequest.Data["project_query"]; ok {
		var projectQueries = cast.ToStringMap(objectRequest.Data["project_query"])
		for key, value := range projectQueries {
			projectQuery[key] = value
		}
	}
	object.Data["project"] = map[string]interface{}{"$project": projectQuery}

	if _, ok := objectRequest.Data["sorts"]; ok {
		var (
			sorts      = cast.ToStringSlice(objectRequest.Data["sorts"])
			sortQuery  = map[string]interface{}{}
			sortedType = 1
		)

		if len(c.Query("sort_type")) > 0 {
			switch c.Query("sort_type") {
			case "desc":
				sortedType = -1
			case "asc":
				sortedType = 1
			}
		}

		for _, value := range sorts {
			sortQuery[value] = sortedType
		}
		object.Data["sort"] = map[string]interface{}{"$sort": sortQuery}
	}

	structData, err := helper.ConvertMapToStruct(object.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var (
		key              = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("aggregate-%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId)))
		aggregateWaitKey = config.CACHE_WAIT + "-aggregate"
	)
	if !cast.ToBool(c.Query("block_cached")) {
		_, aggregateOk := h.cache.Get(aggregateWaitKey)
		if !aggregateOk {
			h.cache.Add(aggregateWaitKey, []byte(aggregateWaitKey), 20*time.Second)
		}

		if aggregateOk {
			ctx, cancel := context.WithTimeout(context.Background(), config.REDIS_WAIT_TIMEOUT)
			defer cancel()

			for {
				aggregateBody, ok := h.cache.Get(key)
				if ok {
					m := make(map[string]interface{})
					err = json.Unmarshal(aggregateBody, &m)
					if err != nil {
						h.handleResponse(c, status_http.GRPCError, err.Error())
						return
					}
					resp := map[string]interface{}{"data": m}
					h.handleResponse(c, status_http.OK, resp)
					return
				}

				if ctx.Err() == context.DeadlineExceeded {
					break
				}

				time.Sleep(config.REDIS_SLEEP)
			}
		}
	}

	resp, err = service.GetGroupByField(
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

	if !cast.ToBool(c.Query("block_cached")) {
		go func() {
			jsonData, _ := json.Marshal(resp.Data)
			h.cache.Add(key, []byte(jsonData), 20*time.Second)
		}()
	}

	h.handleResponse(c, status_http.OK, resp)
}
