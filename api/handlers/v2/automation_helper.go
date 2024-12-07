package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

type DoInvokeFuntionStruct struct {
	CustomEvents           []*obs.CustomEvent
	IDs                    []string
	TableSlug              string
	ObjectData             map[string]interface{}
	Method                 string
	ActionType             string
	ObjectDataBeforeUpdate map[string]interface{}
}

func GetListCustomEvents(tableSlug, roleId, method string, c *gin.Context, h *HandlerV2) (beforeEvents, afterEvents []*obs.CustomEvent, err error) {
	var (
		res   *obs.GetCustomEventsListResponse
		gores *nb.GetCustomEventsListResponse
		body  []byte
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		res, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().GetList(
			ctx,
			&obs.GetCustomEventsListRequest{
				TableSlug: tableSlug,
				Method:    method,
				RoleId:    roleId,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:
		gores, err = services.GoObjectBuilderService().CustomEvent().GetList(
			ctx,
			&nb.GetCustomEventsListRequest{
				TableSlug: tableSlug,
				Method:    method,
				RoleId:    roleId,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			return
		}

		body, err = json.Marshal(gores)
		if err != nil {
			return
		}

		if err = json.Unmarshal(body, &res); err != nil {
			return
		}
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

func DoInvokeFuntion(request DoInvokeFuntionStruct, c *gin.Context, h *HandlerV2) (functionName string, err error) {
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

	apiKeys, err := h.authService.ApiKey().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     resource.ProjectId,
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
	authInfo, _ := h.GetAuthInfo(c)
	for _, customEvent := range request.CustomEvents {
		var (
			path        = customEvent.GetFunctions()[0].GetPath()
			name        = customEvent.GetFunctions()[0].GetName()
			requestType = customEvent.GetFunctions()[0].GetRequestType()
		)
		var invokeFunction models.NewInvokeFunctionRequest
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
		data["project_id"] = projectId
		data["environment_id"] = environmentId
		data["action_type"] = request.ActionType
		invokeFunction.Data = data

		if requestType == "" || requestType == "ASYNC" {
			return funcHandlers[customEvent.Functions[0].Type](path, name, invokeFunction)
		} else if requestType == "SYNC" {
			go func(customEvent *obs.CustomEvent) {
				funcHandlers[customEvent.Functions[0].Type](path, name, invokeFunction)
			}(customEvent)
		}
	}
	return
}

type handlerFunc func(string, string, models.NewInvokeFunctionRequest) (string, error)

var funcHandlers = map[string]handlerFunc{
	"FUNCTION": ExecOpenFaaS,
	"KNATIVE":  ExecKnative,
}

func ExecOpenFaaS(path, name string, req models.NewInvokeFunctionRequest) (string, error) {
	url := fmt.Sprintf("%s%s", config.OpenFaaSBaseUrl, path)
	resp, err := util.DoRequest(url, http.MethodPost, req)
	if err != nil {
		return name, err
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		return name, errors.New(errStr)
	}

	return "", nil
}

func ExecKnative(path, name string, req models.NewInvokeFunctionRequest) (string, error) {
	url := fmt.Sprintf("http://%s.%s", path, config.KnativeBaseUrl)
	resp, err := util.DoRequest(url, http.MethodPost, req)
	if err != nil {
		return name, err
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		return name, errors.New(errStr)
	}

	return "", nil
}
