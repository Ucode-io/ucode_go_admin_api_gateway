package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/function"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

func GetListCustomEvents(request models.GetListCustomEventsStruct, c *gin.Context, h *HandlerV1) (beforeEvents, afterEvents []*obs.CustomEvent, err error) {
	var (
		res   *obs.GetCustomEventsListResponse
		goRes *nb.GetCustomEventsListResponse
	)

	services, err := h.GetProjectSrvc(c.Request.Context(), request.Resource.ProjectId, request.Resource.NodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
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
		goRes, err = services.GoObjectBuilderService().CustomEvent().GetList(
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

		if err = helper.MarshalToStruct(goRes, &res); err != nil {
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

func DoInvokeFuntion(request models.DoInvokeFuntionStruct, c *gin.Context, h *HandlerV1) (functionName string, err error) {
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
		//this is new invoke function request for befor and after actions
		var (
			invokeFunction models.NewInvokeFunctionRequest
			path           = customEvent.GetFunctions()[0].GetPath()
			name           = customEvent.GetFunctions()[0].GetName()
			requestType    = customEvent.GetFunctions()[0].GetRequestType()
		)

		data, err := helper.ConvertStructToResponse(customEvent.Attributes)
		if err != nil {
			return customEvent.GetFunctions()[0].Name, err
		}

		data["object_ids"] = request.IDs
		data["table_slug"] = request.TableSlug
		data["object_data"] = request.ObjectData
		data["object_data_before_update"] = request.ObjectDataBeforeUpdate
		data["method"] = request.Method
		data["app_id"] = appId
		data["user_id"] = authInfo.GetUserId()
		data["project_id"] = request.Resource.ProjectId
		data["environment_id"] = request.Resource.EnvironmentId
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

func DoInvokeFuntionForGetList(request models.DoInvokeFuntionStruct, c *gin.Context, h *HandlerV1) (functionName string, data map[string]interface{}, err error) {
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
		//this is new invoke function request for befor and after actions
		var invokeFunction models.NewInvokeFunctionRequest
		data, err := helper.ConvertStructToResponse(customEvent.Attributes)
		if err != nil {
			return customEvent.GetFunctions()[0].Name, nil, err
		}
		data["object_ids"] = request.IDs
		data["table_slug"] = request.TableSlug
		data["object_data"] = request.ObjectData
		data["object_data_before_update"] = request.ObjectDataBeforeUpdate
		data["method"] = request.Method
		data["app_id"] = appId
		data["user_id"] = authInfo.GetUserId()
		data["project_id"] = request.Resource.ProjectId
		data["environment_id"] = request.Resource.EnvironmentId
		invokeFunction.Data = data

		if customEvent.GetFunctions()[0].RequestType == "" || customEvent.GetFunctions()[0].RequestType == "ASYNC" {
			resp, err := util.DoRequest("https://ofs.u-code.io/function/"+customEvent.GetFunctions()[0].Path, "POST", invokeFunction)
			if err != nil {
				return customEvent.GetFunctions()[0].Name, nil, err
			} else if resp.Status == "error" {
				var errStr = resp.Status
				if resp.Data != nil && resp.Data["message"] != nil {
					errStr = resp.Data["message"].(string)
				}
				return customEvent.GetFunctions()[0].Name, nil, errors.New(errStr)
			}
			return customEvent.GetFunctions()[0].Name, resp.Data, nil
		} else if customEvent.GetFunctions()[0].RequestType == "SYNC" {
			go func(customEvent *obs.CustomEvent) {
				resp, err := util.DoRequest("https://ofs.u-code.io/function/"+customEvent.GetFunctions()[0].Path, "POST", invokeFunction)
				if err != nil {
					h.log.Error("ERROR FROM OFS", logger.Any("err", err.Error()))
					return
				} else if resp.Status == "error" {
					var errStr = resp.Status
					if resp.Data != nil && resp.Data["message"] != nil {
						errStr = resp.Data["message"].(string)
						h.log.Error("ERROR FROM OFS", logger.Any("err", errStr))
						return
					}

					h.log.Error("ERROR FROM OFS "+customEvent.GetFunctions()[0].Path, logger.Any("err", errStr))
					return
				}
			}(customEvent)
		}

	}
	return
}
