package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/function"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	auth "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	gb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

type HandlerInterface interface {
	AuthService() services.AuthServiceManagerI
	BaseConf() config.BaseConfig
	HandleResponse(c *gin.Context, status status_http.Status, data any)
	GetAuthInfo(c *gin.Context) (result *auth.V2HasAccessUserRes, err error)
	GetProjectSrvc(c context.Context, projectId string, nodeType string) (services.ServiceManagerI, error)
}

func DoInvokeFunction(request models.DoInvokeFunctionStruct, c *gin.Context, h HandlerInterface) (functionName string, err error) {
	apiKeys, err := h.AuthService().ApiKey().GetList(context.Background(), &auth_service.GetListReq{
		EnvironmentId: request.Resource.EnvironmentId,
		ProjectId:     request.Resource.ProjectId,
	})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, "error getting api keys by environment id")
		return
	}

	var appId string
	if len(apiKeys.Data) > 0 {
		appId = apiKeys.Data[0].AppId
	} else {
		h.HandleResponse(c, status_http.GRPCError, "error no app id for this environment")
		return
	}

	authInfo, _ := h.GetAuthInfo(c)

	for _, customEvent := range request.CustomEvents {
		var (
			invokeFunction models.NewInvokeFunctionRequest
			path           = customEvent.GetFunctions()[0].GetPath()
			name           = customEvent.GetFunctions()[0].GetName()
			requestType    = customEvent.GetFunctions()[0].GetRequestType()

			funcError error

			status      = "success"
			duration    int64
			sendTime    time.Time
			completedAt time.Time
		)

		if customEvent.Path != "" {
			path = customEvent.Path
		}

		data, err := helper.ConvertStructToResponse(customEvent.Attributes)
		if err != nil {
			return customEvent.GetFunctions()[0].Name, err
		}

		data["object_ids"] = request.IDs
		data["table_slug"] = request.TableSlug
		data["object_data"] = request.ObjectData
		data["object_data_before_update"] = request.ObjectDataBeforeUpdate
		data["method"] = request.Method
		data["action_type"] = request.ActionType
		data["app_id"] = appId
		data["user_id"] = authInfo.GetUserId()
		data["session_id"] = authInfo.GetId()
		data["project_id"] = request.Resource.ProjectId
		data["environment_id"] = request.Resource.EnvironmentId
		invokeFunction.Data = data
		invokeFunction.OpenFaaSURL = h.BaseConf().OpenFaaSBaseUrl
		invokeFunction.KnativeURL = h.BaseConf().KnativeBaseUrl
		invokeFunction.AutomationURL = h.BaseConf().AutomationURL

		switch requestType {
		case "", "ASYNC":
			sendTime = time.Now()
			functionName, funcError = function.FuncHandlers[customEvent.Functions[0].Type](path, name, invokeFunction)
			if funcError != nil {
				status = "error"
			}

			completedAt = time.Now()
			duration = time.Since(sendTime).Milliseconds()

		case "SYNC":
			go func(customEvent *obs.CustomEvent) {
				_, _ = function.FuncHandlers[customEvent.Functions[0].Type](path, name, invokeFunction)
			}(customEvent)
		}

		go func() {
			if request.Resource.ResourceType == pb.ResourceType_POSTGRESQL {
				_, err = request.Services.VersionHistory().CreateFunctionLog(c, &gb.FunctionLogReq{
					ProjectId:     request.Resource.ResourceEnvironmentId,
					FunctionId:    customEvent.Functions[0].GetId(),
					TableSlug:     request.TableSlug,
					RequestMethod: request.Method,
					ActionType:    request.ActionType,
					SendAt:        sendTime.Format(time.DateTime),
					CompletedAt:   completedAt.Format(time.DateTime),
					Duration:      duration,
					Status:        status,
					ReturnSize:    0,
				})
				if err != nil {
					log.Println("error creating function log:", err)
					return
				}
			}
		}()

		if funcError != nil {
			return functionName, funcError
		}

	}
	return
}

func GetListCustomEvents(request models.GetListCustomEventsStruct, c *gin.Context, h HandlerInterface) (beforeEvents, afterEvents []*obs.CustomEvent, err error) {
	var (
		res   *obs.GetCustomEventsListResponse
		goRes *nb.GetCustomEventsListResponse
	)

	services2, err := h.GetProjectSrvc(c.Request.Context(), request.Resource.ProjectId, request.Resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch request.Resource.ResourceType {
	case pb.ResourceType_MONGODB:
		res, err = services2.GetBuilderServiceByType(request.Resource.NodeType).CustomEvent().GetList(
			c.Request.Context(), &obs.GetCustomEventsListRequest{
				TableSlug: request.TableSlug,
				Method:    request.Method,
				RoleId:    request.RoleId,
				ProjectId: request.Resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		goRes, err = services2.GoObjectBuilderService().CustomEvent().GetList(
			c.Request.Context(), &nb.GetCustomEventsListRequest{
				TableSlug: request.TableSlug,
				Method:    request.Method,
				RoleId:    request.RoleId,
				ProjectId: request.Resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		if err = helper.MarshalToStruct(goRes, &res); err != nil {
			h.HandleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
	}

	if res != nil {
		for _, customEvent := range res.CustomEvents {

			switch customEvent.ActionType {
			case config.BEFORE:
				beforeEvents = append(beforeEvents, customEvent)
			case config.AFTER:
				afterEvents = append(afterEvents, customEvent)
			}
		}
	}

	return
}

// =======================
// MCP AI UI INVOCATION
// =======================

func DoInvokeMCPAIUI(c *gin.Context, h HandlerInterface, payload map[string]any,) (any, error) {

	mcpBaseURL := h.BaseConf().MCPServerURL
	if mcpBaseURL == "" {
		return nil, errors.New("MCP service URL is not configured")
	}

	log.Println("MCP URL:", mcpBaseURL)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		mcpBaseURL+"/ai/ui",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Println("HERE ERROR 1111")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("HERE ERROR 2222")
		return nil, err
	}
	defer resp.Body.Close()

	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return result, errors.New("MCP AI UI invocation failed")
	}

	return result, nil
}
