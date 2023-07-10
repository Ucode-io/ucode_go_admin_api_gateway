package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	authPb "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"encoding/base64"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"google.golang.org/grpc/status"
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
func (h *Handler) CreateObject(c *gin.Context) {
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}
	//start := time.Now()
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	//fmt.Println("TIME_MANAGEMENT_LOGGING:::GetService", time.Since(start))

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

	//start = time.Now()
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
	//fmt.Println("TIME_MANAGEMENT_LOGGING:::GetResEnvByResIdEnvId", time.Since(start))

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

			_, err = services.BuilderService().ObjectBuilder().Create(
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
	//fmt.Println("TIME_MANAGEMENT_LOGGING:::Create child objects", time.Since(start))

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
	//fmt.Println("TIME_MANAGEMENT_LOGGING:::GetListCustomEvents", time.Since(start))
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
	// fmt.Println("TIME_MANAGEMENT_LOGGING:::DoInvokeFuntion", time.Since(start))

	//start = time.Now()
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().ObjectBuilder().Create(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("table_slug"),
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

	//fmt.Println("TIME_MANAGEMENT_LOGGING:::Create", time.Since(start))

	//start = time.Now()
	//fmt.Println("after action:::", afterActions)
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
	//fmt.Println("TIME_MANAGEMENT_LOGGING:::DoInvokeFuntion", time.Since(start))
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
func (h *Handler) GetSingle(c *gin.Context) {
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

	object.Data["id"] = objectID

	structData, err := helper.ConvertMapToStruct(object.Data)
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().ObjectBuilder().GetSingle(
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
func (h *Handler) GetSingleSlim(c *gin.Context) {
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
		h.log.Error("Error while getting redis", logger.Error(err))
	}

	resp, err := services.BuilderService().ObjectBuilder().GetSingleSlim(
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

	jsonData, _ := resp.GetData().MarshalJSON()
	err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second)
	if err != nil {
		h.log.Error("Error while setting redis", logger.Error(err))
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
func (h *Handler) UpdateObject(c *gin.Context) {
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

	fromOfs := c.Query("from-ofs")
	// fmt.Println("from-ofs::", fromOfs)
	if fromOfs != "true" {
		beforeActions, afterActions, err = GetListCustomEvents(c.Param("table_slug"), "", "UPDATE", c, h)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
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
		resp, err = services.BuilderService().ObjectBuilder().Update(
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

		_, err = services.AuthService().Session().UpdateSessionsByRoleId(
			context.Background(),
			&authPb.UpdateSessionByRoleIdRequest{
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
				IDs:          []string{id},
				TableSlug:    c.Param("table_slug"),
				ObjectData:   objectRequest.Data,
				Method:       "UPDATE",
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
func (h *Handler) DeleteObject(c *gin.Context) {
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
	objectRequest.Data["id"] = objectID

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
		resp, err = services.BuilderService().ObjectBuilder().Delete(
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

	if c.Param("table_slug") == "user" {
		log.Printf("\n\ndelete user -> userId: %s, projectId: %s\n\n", objectID, resource.ProjectId)
		_, err = services.AuthService().User().DeleteUser(
			c.Request.Context(),
			&authPb.UserPrimaryKey{
				Id:        objectID,
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
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetList(c *gin.Context) {
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		fmt.Println("begin:", time.Now())

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
			h.log.Error("Error while getting redis", logger.Error(err))
		}

		resp, err = services.BuilderService().ObjectBuilder().GetList(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("table_slug"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err == nil {
			if resp.IsCached == true {
				jsonData, _ := resp.GetData().MarshalJSON()
				err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 60*time.Second)
				if err != nil {
					h.log.Error("Error while setting redis", logger.Error(err))
				}
			}
		}
		fmt.Println("end:", time.Now())

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().ObjectBuilder().GetList(
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
func (h *Handler) GetListSlim(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		queryData     string
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
	)
	// queryParams := make(map[string]interface{})
	// err := c.ShouldBindQuery(&queryParams)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	// fmt.Println("::::	objectRequest::", queryParams)
	fmt.Println(":::test:::")

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
	// for key, values := range queryParams {
	// 	fmt.Println(values)
	// 	fmt.Println(len(values))
	// 	if len(values) == 1 {
	// 		queryMap[key] = values[0]
	// 	} else if len(values) > 1 {
	// 		queryMap[key] = values
	// 	}
	// }
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

	fmt.Println("query map:", queryMap)
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

	//redisResp, err := h.redis.Get(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))))
	//if err == nil {
	//	resp := make(map[string]interface{})
	//	m := make(map[string]interface{})
	//	err = json.Unmarshal([]byte(redisResp), &m)
	//	if err != nil {
	//		h.log.Error("Error while unmarshal redis", logger.Error(err))
	//	} else {
	//		resp["data"] = m
	//		h.handleResponse(c, status_http.OK, resp)
	//		return
	//	}
	//} else {
	//	h.log.Error("Error while getting redis", logger.Error(err))
	//}

	resp, err := services.BuilderService().ObjectBuilder().GetListSlim(
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

	jsonData, _ := resp.GetData().MarshalJSON()
	err = h.redis.SetX(context.Background(), base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%s-%s", c.Param("table_slug"), structData.String(), resource.ResourceEnvironmentId))), string(jsonData), 15*time.Second)
	if err != nil {
		h.log.Error("Error while setting redis", logger.Error(err))
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
func (h *Handler) GetListInExcel(c *gin.Context) {
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
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().ObjectBuilder().GetListInExcel(
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
		resp, err = services.PostgresBuilderService().ObjectBuilder().GetListInExcel(
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
func (h *Handler) DeleteManyToMany(c *gin.Context) {
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
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
		resp, err = services.BuilderService().ObjectBuilder().ManyToManyDelete(
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
func (h *Handler) AppendManyToMany(c *gin.Context) {
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

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
		resp, err = services.BuilderService().ObjectBuilder().ManyToManyAppend(
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
func (h *Handler) UpsertObject(c *gin.Context) {
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

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
			// _, err = services.BuilderService()..ObjectBuilde().Create(
			// 	context.Background(),
			// 	&obs.CommonMessage{
			// 		TableSlug: key[1:],
			// 		Data:      mapToStruct,
			// 	},
			// )

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
	var resp *obs.CommonMessage
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().ObjectBuilder().Batch(
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
		_, err = services.AuthService().Session().UpdateSessionsByRoleId(
			context.Background(),
			&authPb.UpdateSessionByRoleIdRequest{
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
func (h *Handler) MultipleUpdateObject(c *gin.Context) {
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

	objects := objectRequest.Data["objects"].([]interface{})
	editedObjects := make([]map[string]interface{}, 0, len(objects))
	var objectIds = make([]string, 0, len(objects))
	for _, object := range objects {
		newObjects := object.(map[string]interface{})
		guid, ok := newObjects["guid"]
		if ok {
			if guid.(string) == "" {
				guid, _ := uuid.NewRandom()
				newObjects["guid"] = guid.String()
				newObjects["is_new"] = true
			}
		}
		objectIds = append(objectIds, newObjects["guid"].(string))
		editedObjects = append(editedObjects, newObjects)
	}
	structData, err := helper.ConvertMapToStruct(objectRequest.Data)

	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	objectRequest.Data["objects"] = editedObjects

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
		resp, err = services.BuilderService().ObjectBuilder().MultipleUpdate(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("table_slug"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		log.Println("----mulltiple_update ---->", resp.GetData().AsMap())

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
		resp, err = services.BuilderService().ObjectBuilder().MultipleUpdate(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("table_slug"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		log.Println("----mulltiple_update ---->", resp.GetData().AsMap())

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
func (h *Handler) GetFinancialAnalytics(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

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

	resp, err := services.BuilderService().ObjectBuilder().GetFinancialAnalytics(
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
