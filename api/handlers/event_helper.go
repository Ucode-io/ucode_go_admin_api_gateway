package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

type DoInvokeFuntionStruct struct {
	CustomEvents []*obs.CustomEvent
	IDs          []string
	TableSlug    string
	ObjectData   map[string]interface{}
	Method       string
}

func GetListCustomEvents(tableSlug, roleId, method string, c *gin.Context, h *Handler) (beforeEvents, afterEvents []*obs.CustomEvent, err error) {

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.CustomEventService().GetList(
		context.Background(),
		&obs.GetCustomEventsListRequest{
			TableSlug: tableSlug,
			Method:    method,
			RoleId:    roleId,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		return
	}
	if res != nil {
		for _, customEvent := range res.CustomEvents {
			if err != nil {
				return nil, nil, err
			}
			if customEvent.ActionType == "before" {
				beforeEvents = append(beforeEvents, customEvent)
			} else if customEvent.ActionType == "after" {
				afterEvents = append(afterEvents, customEvent)
			}
		}
	}
	return
}

func DoInvokeFuntion(request DoInvokeFuntionStruct, c *gin.Context, h *Handler) (functionName string, err error) {
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	apiKeys, err  := h.authService.ApiKeyService().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: environmentId.(string),
	})
	if err != nil {
		err = errors.New("error getting api keys by environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	var appId string
	if len(apiKeys.Data) > 0 {
		appId = apiKeys.Data[0].AppId
	} else {
		err = errors.New("error no app id for this environment")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}


	for _, customEvent := range request.CustomEvents {
		//this is new invoke function request for befor and after actions
		var invokeFunction models.NewInvokeFunctionRequest
		data, err := helper.ConvertStructToResponse(customEvent.Attributes)
		if err != nil {
			return customEvent.Functions[0].Name, err
		}
		data["object_ids"] = request.IDs
		data["table_slug"] = request.TableSlug
		data["object_data"] = request.ObjectData
		data["method"] = request.Method
		data["api_key"] = appId
		invokeFunction.Data = data

		resp, err := util.DoRequest("https://ofs.medion.udevs.io/function/"+customEvent.Functions[0].Path, "POST", invokeFunction)
		if err != nil {
			return customEvent.Functions[0].Name, err
		} else if resp.Status == "error" {
			var errStr = resp.Status
			if resp.Data != nil && resp.Data["message"] != nil {
				errStr = resp.Data["message"].(string)
			}
			return customEvent.Functions[0].Name, errors.New(errStr)
		}
		_, err = services.CustomEventService().UpdateByFunctionId(context.Background(), &obs.UpdateByFunctionIdRequest{
			FunctionId: customEvent.Functions[0].Id,
			ObjectIds:  request.IDs,
			FieldSlug:  customEvent.Functions[0].Path + "_disable",
			ProjectId:  resourceEnvironment.GetId(),
		})
		if err != nil {
			return customEvent.Functions[0].Name, err
		}
	}
	return
}
