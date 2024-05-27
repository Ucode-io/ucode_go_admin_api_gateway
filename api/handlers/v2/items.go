package v2

import (
	"context"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
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

// CreateItem godoc
// @Security ApiKeyAuth
// @ID create_item
// @Router /v2/items/{collection} [POST]
// @Summary Create item
// @Description Create item
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param object body models.CommonMessage true "CreateItemsRequestBody"
// @Success 201 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateItem(c *gin.Context) {
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

	userId, _ := c.Get("user_id")

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

	objectRequest.Data["company_service_project_id"] = resource.GetProjectId()
	objectRequest.Data["company_service_environment_id"] = resource.GetEnvironmentId()

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

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)

	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("collection"), "", "CREATE", c, h)
		if err != nil {
			fmt.Println("I am here")
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			//return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{id},
			TableSlug:    c.Param("collection"),
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

	logReq := &models.CreateVersionHistoryRequest{
		Services:     services,
		NodeType:     resource.NodeType,
		ProjectId:    resource.ResourceEnvironmentId,
		ActionSource: c.Request.URL.String(),
		ActionType:   "CREATE ITEM",
		UsedEnvironments: map[string]bool{
			cast.ToString(environmentId): true,
		},
		UserInfo:  cast.ToString(userId),
		Request:   &structData,
		TableSlug: c.Param("collection"),
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().Create(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		// this logic for custom error message, object builder service may be return 400, 404, 500
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
	case pb.ResourceType_POSTGRESQL:
		body, err := services.GoObjectBuilderService().Items().Create(
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

		err = helper.MarshalToStruct(body, &resp)
		if err != nil {
			return
		}
	}

	if data, ok := resp.Data.AsMap()["data"].(map[string]interface{}); ok {
		objectRequest.Data = data
		if _, ok = data["guid"].(string); ok {
			id = data["guid"].(string)
		}
	}
	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{id},
				TableSlug:    c.Param("collection"),
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

// CreateItems godoc
// @Security ApiKeyAuth
// @ID create_items
// @Router /v2/items/{collection}/multiple-insert [POST]
// @Summary Create items
// @Description Create items
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param object body models.MultipleInsertItems true "CreateItemsRequestBody"
// @Success 201 {object} status_http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateItems(c *gin.Context) {
	var (
		objectRequest               models.MultipleInsertItems
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

	userId, _ := c.Get("user_id")

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

	request := make(map[string]interface{})
	request["company_service_project_id"] = resource.GetProjectId()
	request["company_service_environment_id"] = resource.GetEnvironmentId()
	request["items"] = objectRequest

	structData, err := helper.ConvertMapToStruct(request)

	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("collection"), "", "CREATE_MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			TableSlug:    c.Param("collection"),
			ObjectData:   request,
			Method:       "CREATE_MANY",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	logReq := &models.CreateVersionHistoryRequest{
		Services:     services,
		NodeType:     resource.NodeType,
		ProjectId:    resource.ResourceEnvironmentId,
		ActionSource: c.Request.URL.String(),
		ActionType:   "CREATE ITEM",
		UsedEnvironments: map[string]bool{
			cast.ToString(environmentId): true,
		},
		UserInfo:  cast.ToString(userId),
		Request:   structData,
		TableSlug: c.Param("collection"),
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().Create(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		// this logic for custom error message, object builder service may be return 400, 404, 500
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
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().Create(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	var items []interface{}
	if itemsFromResp, ok := resp.Data.AsMap()["items"].([]interface{}); ok {
		items = itemsFromResp
	}
	var ids = make([]string, 0, len(items))
	for _, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if id, ok := itemMap["guid"].(string); ok {
				ids = append(ids, id)
			}
		}
	}
	if len(afterActions) > 0 {
		functionName, err := DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          ids,
				TableSlug:    c.Param("collection"),
				ObjectData:   request,
				Method:       "CREATE_MANY",
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

// GetSingleItem godoc
// @Security ApiKeyAuth
// @ID get_item_by_id
// @Router /v2/items/{collection}/{id} [GET]
// @Summary Get item by id
// @Description Get item by id
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetSingleItem(c *gin.Context) {
	var (
		object     models.CommonMessage
		statusHttp = status_http.GrpcStatusToHTTP["Ok"]
	)

	object.Data = make(map[string]interface{})

	objectID := c.Param("id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, status_http.InvalidArgument, "id is an invalid uuid")
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().GetSingle(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
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
		resp, err := services.GoObjectBuilderService().Items().GetSingle(
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

		statusHttp.CustomMessage = resp.GetCustomMessage()
		h.handleResponse(c, statusHttp, resp)
	}
}

// GetAllItems godoc
// @Security ApiKeyAuth
// @ID get_list_items
// @Router /v2/items/{collection} [GET]
// @Summary Get all items
// @Description Get all items
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param language_setting query string false "language_setting"
// @Param data query string false "data"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllItems(c *gin.Context) {
	var (
		resp       *obs.CommonMessage
		statusHttp = status_http.GrpcStatusToHTTP["Ok"]
		queryData  string
	)

	queryParams := c.Request.URL.Query()
	if ok := queryParams.Has("data"); ok {
		queryData = queryParams.Get("data")
	} else {
		queryData = "{}"
	}

	objectRequest := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryData), &objectRequest)
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
			objectRequest["tables"] = tokenInfo.GetTables()
		}
		objectRequest["user_id_from_token"] = tokenInfo.GetUserId()
		objectRequest["role_id_from_token"] = tokenInfo.GetRoleId()
		objectRequest["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
	}
	objectRequest["language_setting"] = c.DefaultQuery("language_setting", "")

	structData, err := helper.ConvertMapToStruct(objectRequest)
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

	if viewId, ok := objectRequest["builder_service_view_id"].(string); ok {
		if util.IsValidUUID(viewId) {
			switch resource.ResourceType {
			case pb.ResourceType_MONGODB:
				redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("collection"), structData.String(), resource.ResourceEnvironmentId))), resource.ProjectId, resource.NodeType)
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
						TableSlug: c.Param("collection"),
						Data:      structData,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)

				if err == nil {
					if resp.IsCached {
						jsonData, _ := resp.GetData().MarshalJSON()
						err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("collection"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, resource.ProjectId, resource.NodeType)
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
						TableSlug: c.Param("collection"),
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
			// start := time.Now()

			redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("collection"), structData.String(), resource.ResourceEnvironmentId))), resource.ProjectId, resource.NodeType)
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

			resp, err = services.GetBuilderServiceByType(resource.NodeType).ItemsService().GetList(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: c.Param("collection"),
					Data:      structData,
					ProjectId: resource.ResourceEnvironmentId,
				},
			)

			if err == nil {
				if resp.IsCached {
					jsonData, _ := resp.GetData().MarshalJSON()
					err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("collection"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, resource.ProjectId, resource.NodeType)
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
			resp, err = services.PostgresBuilderService().ObjectBuilder().GetList(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: c.Param("collection"),
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

// UpdateItem godoc
// @Security ApiKeyAuth
// @ID update_item
// @Router /v2/items/{collection} [PUT]
// @Summary Update item
// @Description Update item
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param item body models.CommonMessage true "UpdateItemRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "Item data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateItem(c *gin.Context) {
	var (
		objectRequest               models.CommonMessage
		resp, singleObject          *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Ok"]
		actionErr                   error
		functionName                string
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
		objectRequest.Data["guid"] = c.Param("id")
		id = c.Param("id")

		if id == "" {
			h.handleResponse(c, status_http.BadRequest, "guid is required")
			return
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

	userId, _ := c.Get("user_id")

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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		singleObject, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().GetSingleSlim(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
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
		single, err := services.GoObjectBuilderService().Items().GetSingle(
			context.Background(),
			&nb.CommonMessage{
				TableSlug: c.Param("collection"),
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
		
		err = helper.MarshalToStruct(single, &singleObject)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("collection"), "", "UPDATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{id},
			TableSlug:    c.Param("collection"),
			ObjectData:   objectRequest.Data,
			Method:       "UPDATE",
			ActionType:   "BEFORE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "UPDATE ITEM",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &structData,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else if actionErr != nil {
			logReq.Response = actionErr.Error() + " in " + functionName
			h.handleResponse(c, status_http.InvalidArgument, actionErr.Error()+" in "+functionName)
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().Update(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
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
			return
		}
	case pb.ResourceType_POSTGRESQL:
		body, err := services.GoObjectBuilderService().Items().Update(
			context.Background(),
			&nb.CommonMessage{
				TableSlug:      c.Param("collection"),
				Data:           structData,
				ProjectId:      resource.ResourceEnvironmentId,
				BlockedBuilder: cast.ToBool(c.DefaultQuery("block_builder", "false")),
			},
		)
		if err != nil {
			return
		}

		err = helper.MarshalToStruct(body, &resp)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}
	if len(afterActions) > 0 {
		functionName, actionErr = DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents:           afterActions,
				IDs:                    []string{id},
				TableSlug:              c.Param("collection"),
				ObjectData:             objectRequest.Data,
				Method:                 "UPDATE",
				ObjectDataBeforeUpdate: singleObject.Data.AsMap(),
				ActionType:             "AFTER",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
}

// MultipleUpdateItems godoc
// @Security ApiKeyAuth
// @ID multiple_update_items
// @Router /v2/items/{collection} [PATCH]
// @Summary Multiple Update items
// @Description Multiple Update items
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param items body models.MultipleUpdateItems true "MultipleItemsRequesUpdatetBody"
// @Success 201 {object} status_http.Response{data=models.CommonMessage} "Items data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) MultipleUpdateItems(c *gin.Context) {
	var (
		objectRequest               models.MultipleUpdateItems
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Created"]
		actionErr                   error
		functionName                string
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

	userId, _ := c.Get("user_id")

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
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("collection"), "", "MULTIPLE_UPDATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          objectRequest.Ids,
			TableSlug:    c.Param("collection"),
			ObjectData:   objectRequest.Data,
			Method:       "MULTIPLE_UPDATE",
			ActionType:   "BEFORE",
		},
			c,
			h,
		)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error()+" in "+functionName)
			return
		}
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "UPDATE ITEM",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &structData,
			TableSlug: c.Param("collection"),
		}
	)

	var resp *obs.CommonMessage

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else if actionErr != nil {
			logReq.Response = actionErr.Error() + " in " + functionName
			h.handleResponse(c, status_http.InvalidArgument, actionErr.Error()+" in "+functionName)
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.MultipleUpdate(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
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
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = service.MultipleUpdate(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			return
		}
	}

	if len(afterActions) > 0 {
		functionName, actionErr = DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          objectRequest.Ids,
				TableSlug:    c.Param("collection"),
				ObjectData:   objectRequest.Data,
				Method:       "MULTIPLE_UPDATE",
				ActionType:   "AFTER",
			},
			c, // gin context
			h, // handler
		)
		if err != nil {
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
}

// DeleteItem godoc
// @Security ApiKeyAuth
// @ID delete_item
// @Router /v2/items/{collection}/{id} [DELETE]
// @Summary Delete item
// @Description Delete item
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteItem(c *gin.Context) {
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

	objectID := c.Param("id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, status_http.InvalidArgument, "item id is an invalid uuid")
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

	objectRequest.Data["id"] = objectID
	objectRequest.Data["company_service_project_id"] = projectId.(string)
	objectRequest.Data["company_service_environment_id"] = environmentId.(string)

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("collection"), "", "DELETE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{objectID},
			TableSlug:    c.Param("collection"),
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

	logReq := &models.CreateVersionHistoryRequest{
		Services:     services,
		NodeType:     resource.NodeType,
		ProjectId:    resource.ResourceEnvironmentId,
		ActionSource: c.Request.URL.String(),
		ActionType:   "DELETE ITEM",
		UsedEnvironments: map[string]bool{
			cast.ToString(environmentId): true,
		},
		UserInfo:  cast.ToString(userId),
		Request:   structData,
		TableSlug: c.Param("collection"),
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().Delete(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
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
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		logReq.Response = resp
		go h.versionHistory(c, logReq)
	case pb.ResourceType_POSTGRESQL:
		new, err := services.GoObjectBuilderService().Items().Delete(
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

		err = helper.MarshalToStruct(new, &resp)
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
				TableSlug:    c.Param("collection"),
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

// DeleteManyObject godoc
// @Security ApiKeyAuth
// @ID delete_items
// @Router /v2/items/{collection} [DELETE]
// @Summary Delete many itmes
// @Description Delete many itmes
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param object body models.Ids true "DeleteManyItemRequestBody"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteItems(c *gin.Context) {
	var (
		objectRequest               models.Ids
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["NoContent"]
		data                        = make(map[string]interface{})
		actionErr                   error
		functionName                string
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

	userId, _ := c.Get("user_id")

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

	data["company_service_project_id"] = projectId.(string)
	data["company_service_environment_id"] = environmentId.(string)
	data["ids"] = objectRequest.Ids

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
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
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("collection"), "", "DELETE_MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          objectRequest.Ids,
			TableSlug:    c.Param("collection"),
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

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "DELETE ITEM",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &structData,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else if actionErr != nil {
			logReq.Response = actionErr.Error() + " in " + functionName
			h.handleResponse(c, status_http.InvalidArgument, actionErr.Error()+" in "+functionName)
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = service.DeleteMany(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("collection"),
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
				TableSlug: c.Param("collection"),
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
		functionName, actionErr = DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          objectRequest.Ids,
				TableSlug:    c.Param("collection"),
				ObjectData:   data,
				Method:       "DELETE_MANY",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			return
		}
	}

	statusHttp.CustomMessage = resp.GetCustomMessage()
}

// DeleteManyToMany godoc
// @Security ApiKeyAuth
// @ID v2_delete_many2many
// @Router /v2/items/many-to-many [DELETE]
// @Summary Delete Many2Many items
// @Description Delete Many2Many items
// @Tags Items
// @Accept json
// @Produce json
// @Param object body obs.ManyToManyMessage true "DeleteManyToManyBody"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteManyToMany(c *gin.Context) {
	var (
		m2mMessage                  obs.ManyToManyMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["NoContent"]
		actionErr                   error
		functionName                string
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

	userId, _ := c.Get("user_id")

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
		resource.GetProjectId(),
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

	m2mMessage.ProjectId = resource.ResourceEnvironmentId
	fromOfs := c.Query("from-ofs")
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(m2mMessage.TableFrom, "", "DELETE_MANY2MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{m2mMessage.IdFrom},
			TableSlug:    m2mMessage.TableFrom,
			ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_to": m2mMessage.TableTo},
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

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "DELETE ITEM",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &m2mMessage,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else if actionErr != nil {
			logReq.Response = actionErr.Error() + " in " + functionName
			h.handleResponse(c, status_http.InvalidArgument, actionErr.Error()+" in "+functionName)
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

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
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().ManyToManyDelete(
			context.Background(),
			&m2mMessage,
		)

		if err != nil {
			return
		}
	}

	if len(afterActions) > 0 {
		functionName, actionErr = DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{m2mMessage.IdFrom},
				TableSlug:    m2mMessage.TableFrom,
				ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_from": m2mMessage.TableTo},
				Method:       "DELETE_MANY2MANY",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
}

// AppendManyToMany godoc
// @Security ApiKeyAuth
// @ID v2_append_many2many
// @Router /v2/items/many-to-many [PUT]
// @Summary Append many-to-many items
// @Description Append many-to-many items
// @Tags Items
// @Accept json
// @Produce json
// @Param object body obs.ManyToManyMessage true "UpdateMany2ManyRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Object data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) AppendManyToMany(c *gin.Context) {
	var (
		m2mMessage                  obs.ManyToManyMessage
		resp                        *obs.CommonMessage
		beforeActions, afterActions []*obs.CustomEvent
		statusHttp                  = status_http.GrpcStatusToHTTP["Ok"]
		actionErr                   error
		functionName                string
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

	userId, _ := c.Get("user_id")

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
		beforeActions, afterActions, err = GetListCustomEvents(m2mMessage.TableFrom, "", "APPEND_MANY2MANY", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
	}
	if len(beforeActions) > 0 {
		functionName, err := DoInvokeFuntion(DoInvokeFuntionStruct{
			CustomEvents: beforeActions,
			IDs:          []string{m2mMessage.IdFrom},
			TableSlug:    m2mMessage.TableFrom,
			ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_to": m2mMessage.TableTo},
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

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "UPDATE ITEM",
			UsedEnvironments: map[string]bool{
				cast.ToString(environmentId): true,
			},
			UserInfo:  cast.ToString(userId),
			Request:   &m2mMessage,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else if actionErr != nil {
			logReq.Response = actionErr.Error() + " in " + functionName
			h.handleResponse(c, status_http.InvalidArgument, actionErr.Error()+" in "+functionName)
		} else {
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

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
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().ManyToManyAppend(
			context.Background(),
			&m2mMessage,
		)
		if err != nil {
			return
		}
	}

	if len(afterActions) > 0 {
		functionName, actionErr = DoInvokeFuntion(
			DoInvokeFuntionStruct{
				CustomEvents: afterActions,
				IDs:          []string{m2mMessage.IdFrom},
				TableSlug:    m2mMessage.TableFrom,
				ObjectData:   map[string]interface{}{"id_to": m2mMessage.IdTo, "table_to": m2mMessage.TableTo},
				Method:       "APPEND_MANY2MANY",
			},
			c, // gin context,
			h, // handler
		)
		if err != nil {
			return
		}
	}
	statusHttp.CustomMessage = resp.GetCustomMessage()
}

// GetListAggregation godoc
// @Security ApiKeyAuth
// @ID get_list_aggregation
// @Router /v2/items/{collection}/aggregation [POST]
// @Summary Get List Aggregation
// @Description Get List Aggregation
// @Tags Items
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param object body models.CommonMessage true "GetListAggregation"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetListAggregation(c *gin.Context) {

	var (
		reqBody models.CommonMessage
	)

	err := c.ShouldBindJSON(&reqBody)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
	}

	key, err := json.Marshal(reqBody.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(reqBody.Data)
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

	if reqBody.IsCached {
		redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("collection"), string(key), resource.ResourceEnvironmentId))), projectId.(string), resource.NodeType)
		if err == nil {
			resp := make(map[string]interface{})
			m := make(map[string]interface{})
			err = json.Unmarshal([]byte(redisResp), &m)
			if err != nil {
				h.log.Error("Error while unmarshal redis in items aggregation", logger.Error(err))
			} else {
				resp["data"] = m
				h.handleResponse(c, status_http.OK, resp)
				return
			}
		}
	}

	resp, err := service.GetListAggregation(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("collection"),
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if reqBody.IsCached {
		jsonData, _ := resp.GetData().MarshalJSON()
		err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("collection"), string(key), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second, projectId.(string), resource.NodeType)
		if err != nil {
			h.log.Error("Error while setting redis in items aggregation", logger.Error(err))
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}
