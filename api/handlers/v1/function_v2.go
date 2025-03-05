package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	_ "ucode/ucode_go_api_gateway/genproto/new_function_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// CreateNewFunction godoc
// @Security ApiKeyAuth
// @ID create_new_function
// @Router /v2/function [POST]
// @Summary Create New Function
// @Description Create New Function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.CreateFunctionRequest true "CreateFunctionRequestBody"
// @Success 201 {object} status_http.Response{data=new_function_service.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateNewFunction(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// GetNewFunctionByID godoc
// @Security ApiKeyAuth
// @ID get_new_function_by_id
// @Router /v2/function/{function_id} [GET]
// @Summary Get Function by id
// @Description Get Function by id
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 200 {object} status_http.Response{data=new_function_service.Function} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetNewFunctionByID(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// GetAllNewFunctions godoc
// @Security ApiKeyAuth
// @ID get_all_new_functions
// @Router /v2/function [GET]
// @Summary Get all functions
// @Description Get all functions
// @Tags Function
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllNewFunctions(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// UpdateNewFunction godoc
// @Security ApiKeyAuth
// @ID update_new_function
// @Router /v2/function [PUT]
// @Summary Update new function
// @Description Update new function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.Function  true "UpdateFunctionRequestBody"
// @Success 200 {object} status_http.Response{data=models.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateNewFunction(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// DeleteNewFunction godoc
// @Security ApiKeyAuth
// @ID delete_new_function
// @Router /v2/function/{function_id} [DELETE]
// @Summary Delete New Function
// @Description Delete New Function
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteNewFunction(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

// InvokeFunctionByPath godoc
// @Security ApiKeyAuth
// @Param function-path path string true "function-path"
// @ID invoke_function_by_path
// @Router /v1/invoke_function/{function-path} [POST]
// @Summary Invoke Function By Path
// @Description Invoke Function By Path
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionByPathRequest body models.CommonMessage true "InvokeFunctionByPathRequest"
// @Success 201 {object} status_http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) InvokeFunctionByPath(c *gin.Context) {
	var invokeFunction models.CommonMessage

	if err := c.ShouldBindJSON(&invokeFunction); err != nil {
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
		err := errors.New("error getting environment id | not valid")
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

	apiKeys, err := h.authService.ApiKey().GetList(c.Request.Context(), &auth_service.GetListReq{
		EnvironmentId: environmentId.(string),
		ProjectId:     resource.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if len(apiKeys.Data) < 1 {
		h.handleResponse(c, status_http.InvalidArgument, "Api key not found")
		return
	}

	authInfo, _ := h.GetAuthInfo(c)

	invokeFunction.Data["user_id"] = authInfo.GetUserId()
	invokeFunction.Data["project_id"] = authInfo.GetProjectId()
	invokeFunction.Data["environment_id"] = authInfo.GetEnvId()
	invokeFunction.Data["app_id"] = apiKeys.GetData()[0].GetAppId()

	functionPath := map[string]string{
		"wayll-main-report":                  "wayll-main-report-knative",
		"wayll-profit-report":                "wayll-profit-report-knative",
		"wayll-create-required-pockets":      "wayll-create-required-pockets-knative",
		"wayll-candle-chart":                 "wayll-candle-chart-knative",
		"wayll-get-projects":                 "wayll-get-projects-knative",
		"wayll-payment":                      "wayll-payment-knative",
		"wayll-payme-integration":            "wayll-payme-integration-knative",
		"wayll-generate-doc":                 "wayll-generate-doc-knative",
		"wayll-create-investor-legal-entity": "wayll-create-investor-legal-entity-knative",
		"wayll-get-profile":                  "wayll-get-profile-knative",
		"wayll-get-comments":                 "wayll-get-comments-knative",
		"wayll-get-followers-list":           "wayll-get-followers-list-knative",
		"wayll-get-project":                  "wayll-get-project-knative",
	}

	knativeFunctionPath := c.Param("function-path")
	if val, ok := functionPath[knativeFunctionPath]; ok {
		knativeFunctionPath = val
	}

	url := fmt.Sprintf("http://%s.%s", knativeFunctionPath, config.KnativeBaseUrl)
	resp, err := util.DoRequest(url, http.MethodPost, models.NewInvokeFunctionRequest{
		Data: invokeFunction.Data,
	})
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status_http.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// FunctionRun godoc
// @Security ApiKeyAuth
// @ID function_run
// @Router /v2/functions/{function-id}/run [POST]
// @Summary Function Run
// @Description Function Run
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionRequest body models.InvokeFunctionRequest true "InvokeFunctionRequest"
// @Success 200 {object} status_http.Response{data=models.InvokeFunctionResponse} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) FunctionRun(c *gin.Context) {
	var (
		requestData    models.HttpRequest
		invokeFunction models.InvokeFunctionRequest
	)

	bodyReq, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.log.Error("cant parse body or an empty body received", logger.Any("req", c.Request))
		return
	}

	err = json.Unmarshal(bodyReq, &invokeFunction)
	if err != nil {
		h.log.Error("cant parse body or an empty body received", logger.Any("req", c.Request))
		return
	}

	if cast.ToBool(c.GetHeader("/v1/functions/")) {
		var authData = models.AuthData{}
		err = json.Unmarshal([]byte(c.GetHeader("auth")), &authData)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, "cant get auth info")
			return
		}

		c.Set("auth", authData)
		c.Set("resource_id", c.GetHeader("resource_id"))
		c.Set("environment_id", c.GetHeader("environment_id"))
		c.Set("project_id", c.GetHeader("project_id"))
		c.Set("resource", c.GetHeader("resource"))
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
		return
	}

	var resource *pb.ServiceResourceModel
	resourceBody, ok := c.Get("resource")
	if resourceBody != "" && ok {
		var resourceList *pb.GetResourceByEnvIDResponse

		if err = json.Unmarshal([]byte(resourceBody.(string)), &resourceList); err != nil {
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
			c.Request.Context(), &pb.GetSingleServiceResourceReq{
				ProjectId:     projectId.(string),
				EnvironmentId: environmentId.(string),
				ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	authInfoAny, ok := c.Get("auth")
	if !ok {
		h.handleResponse(c, status_http.InvalidArgument, "cant get auth info")
		return
	}

	authInfo := authInfoAny.(models.AuthData)
	requestData.Method = c.Request.Method
	requestData.Headers = c.Request.Header
	requestData.Path = c.Request.URL.Path
	requestData.Params = c.Request.URL.Query()
	requestData.Body = bodyReq

	var functionResponse = &obs.Function{}

	if util.IsValidUUID(c.Param("function-id")) {
		services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		switch resource.ResourceType {
		case pb.ResourceType_MONGODB:
			functionResponse, err = services.GetBuilderServiceByType(resource.NodeType).Function().GetSingle(
				c.Request.Context(), &obs.FunctionPrimaryKey{
					Id:        c.Param("function-id"),
					ProjectId: resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case pb.ResourceType_POSTGRESQL:
			resp, err := services.GoObjectBuilderService().Function().GetSingle(
				c.Request.Context(), &nb.FunctionPrimaryKey{
					Id:        c.Param("function-id"),
					ProjectId: resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			if err = helper.MarshalToStruct(resp, &functionResponse); err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	} else {
		functionResponse.Path = c.Param("function-id")
	}

	var resp = models.InvokeFunctionResponse{}

	resp, err = util.DoRequest(fmt.Sprintf("http://%s.%s", functionResponse.GetPath(), config.KnativeBaseUrl), http.MethodPost,
		models.FunctionRunV2{
			Auth:        models.AuthData{},
			RequestData: requestData,
			Data: map[string]any{
				"object_ids": invokeFunction.ObjectIDs,
				"attributes": invokeFunction.Attributes,
				"app_id":     authInfo.Data["app_id"],
			},
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status_http.InvalidArgument, errStr)
		return
	}

	if isOwnData, ok := resp.Attributes["is_own_data"].(bool); ok {
		if isOwnData {
			if _, ok := resp.Data["code"]; ok {
				c.JSON(cast.ToInt(resp.Data["code"]), resp.Data)
				return
			}

			c.JSON(http.StatusOK, resp.Data)
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllNewFunctionsForApp godoc
// @Security ApiKeyAuth
// @ID get_all_new_functions_for_app
// @Router /v2/function-for-app [GET]
// @Summary Get all functions
// @Description Get all functions
// @Tags Function
// @Accept json
// @Produce json
// @Param limit query number false "limit"
// @Param offset query number false "offset"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllNewFunctionsForApp(c *gin.Context) {
	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
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
		h.handleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(), &pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Function().GetList(
			c.Request.Context(), &obs.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
				Type:      []string{config.FUNCTION, config.KNATIVE},
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Function().GetList(
			c.Request.Context(), &nb.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
				Type:      []string{config.FUNCTION, config.KNATIVE},
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	}
}
