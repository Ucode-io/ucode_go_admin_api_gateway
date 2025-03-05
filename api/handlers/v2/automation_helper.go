package v2

import (
	"context"
	"encoding/json"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/function"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
)

func GetListCustomEvents(request models.GetListCustomEventsStruct, c *gin.Context, h *HandlerV2) (beforeEvents, afterEvents []*obs.CustomEvent, err error) {
	var (
		res   *obs.GetCustomEventsListResponse
		gores *nb.GetCustomEventsListResponse
		body  []byte
	)

	namespace := c.GetString("namespace")

	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	switch request.Resource.ResourceType {
	case pb.ResourceType_MONGODB:
		res, err = services.GetBuilderServiceByType(request.Resource.NodeType).CustomEvent().GetList(
			c.Request.Context(), &obs.GetCustomEventsListRequest{
				TableSlug: request.TableSlug,
				Method:    request.Method,
				RoleId:    request.RoleId,
				ProjectId: request.Resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		gores, err = services.GoObjectBuilderService().CustomEvent().GetList(
			c.Request.Context(), &nb.GetCustomEventsListRequest{
				TableSlug: request.TableSlug,
				Method:    request.Method,
				RoleId:    request.RoleId,
				ProjectId: request.Resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		body, err = json.Marshal(gores)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}

		if err = json.Unmarshal(body, &res); err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
	}

	if res != nil {
		for _, customEvent := range res.CustomEvents {
			if customEvent.ActionType == config.BEFORE {
				beforeEvents = append(beforeEvents, customEvent)
			} else if customEvent.ActionType == config.AFTER {
				afterEvents = append(afterEvents, customEvent)
			}
		}
	}
	return
}

func DoInvokeFuntion(request models.DoInvokeFuntionStruct, c *gin.Context, h *HandlerV2) (functionName string, err error) {
	apiKeys, err := h.authService.ApiKey().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: request.Resource.EnvironmentId,
		ProjectId:     request.Resource.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, "error getting api keys by environment id")
		return
	}

	var appId string
	if len(apiKeys.Data) > 0 {
		appId = apiKeys.Data[0].AppId
	} else {
		h.handleResponse(c, status_http.GRPCError, "error no app id for this environment")
		return
	}

	authInfo, _ := h.GetAuthInfo(c)
	for _, customEvent := range request.CustomEvents {
		var (
			path           = customEvent.GetFunctions()[0].GetPath()
			name           = customEvent.GetFunctions()[0].GetName()
			requestType    = customEvent.GetFunctions()[0].GetRequestType()
			invokeFunction models.NewInvokeFunctionRequest
		)

		data, err := helper.ConvertStructToResponse(customEvent.Attributes)
		if err != nil {
			return name, err
		}

		data["object_ids"] = request.IDs
		data["table_slug"] = request.TableSlug
		data["object_data"] = request.ObjectData
		data["object_data_before_update"] = request.ObjectDataBeforeUpdate
		data["method"] = request.Method
		data["app_id"] = appId
		data["user_id"] = authInfo.GetUserId()
		data["session_id"] = authInfo.GetId()
		data["project_id"] = request.Resource.ProjectId
		data["environment_id"] = request.Resource.EnvironmentId
		data["action_type"] = request.ActionType
		invokeFunction.Data = data

		if requestType == "" || requestType == "ASYNC" {
			functionName, err = function.FuncHandlers[customEvent.Functions[0].Type](path, name, invokeFunction)
			if err != nil {
				return functionName, err
			}
		} else if requestType == "SYNC" {
			go func(customEvent *obs.CustomEvent) {
				function.FuncHandlers[customEvent.Functions[0].Type](path, name, invokeFunction)
			}(customEvent)
		}
	}
	return
}
