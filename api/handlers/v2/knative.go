package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"ucode/ucode_go_api_gateway/api/models"
	_ "ucode/ucode_go_api_gateway/api/models"
	status "ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

type ApiKey struct {
	AppId string `json:"app_id"`
}

// InvokeFunctionByPath godoc
// @Security ApiKeyAuth
// @Param function-path path string true "function-path"
// @ID v2_invoke_function_by_path
// @Router /v2/invoke_function/{function-path} [POST]
// @Summary Invoke Function By Path
// @Description Invoke Function By Path
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionByPathRequest body models.CommonMessage true "InvokeFunctionByPathRequest"
// @Success 201 {object} status_http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) InvokeFunctionByPath(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

func (h *HandlerV2) InvokeFunctionByApiPath(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}

func (h *HandlerV2) InvokeInAdmin(c *gin.Context) {
	var (
		invokeFunction models.CommonMessage
		path           = c.Param("function-path")
		apiKey         ApiKey
	)

	if err := c.ShouldBindJSON(&invokeFunction); err != nil {
		h.handleResponse(c, status.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status.BadRequest, err)
		return
	}

	resourceBody, exist := h.cache.Get(fmt.Sprintf("project:%s:env:%s", projectId.(string), environmentId.(string)))
	if !exist {
		resource, err := h.companyServices.ServiceResource().GetSingle(
			c.Request.Context(),
			&pb.GetSingleServiceResourceReq{
				ProjectId:     projectId.(string),
				EnvironmentId: environmentId.(string),
				ServiceType:   pb.ServiceType_BUILDER_SERVICE,
			},
		)
		if err != nil {
			h.handleResponse(c, status.GRPCError, err.Error())
			return
		}

		apiKeys, err := h.authService.ApiKey().GetList(c.Request.Context(), &as.GetListReq{
			EnvironmentId: environmentId.(string),
			ProjectId:     resource.ProjectId,
			Limit:         1,
			Offset:        0,
		})
		if err != nil {
			h.handleResponse(c, status.GRPCError, err.Error())
			return
		}
		if len(apiKeys.Data) < 1 {
			h.handleResponse(c, status.InvalidArgument, "Api key not found")
			return
		}

		appIdByte, err := json.Marshal(ApiKey{AppId: apiKeys.GetData()[0].GetAppId()})
		if err != nil {
			h.handleResponse(c, status.InvalidArgument, err.Error())
			return
		}

		h.cache.Add(fmt.Sprintf("project:%s:env:%s", projectId.(string), environmentId.(string)), appIdByte, config.REDIS_KEY_TIMEOUT)
	} else {
		if err := json.Unmarshal(resourceBody, &apiKey); err != nil {
			h.handleResponse(c, status.InvalidArgument, err.Error())
			return
		}
	}

	authInfo, _ := h.GetAuthInfo(c)

	invokeFunction.Data["user_id"] = authInfo.GetUserId()
	invokeFunction.Data["project_id"] = authInfo.GetProjectId()
	invokeFunction.Data["environment_id"] = authInfo.GetEnvId()
	invokeFunction.Data["app_id"] = apiKey.AppId
	request := models.NewInvokeFunctionRequest{Data: invokeFunction.Data}

	resp, err := h.ExecKnative(path, request)
	if err != nil {
		h.handleResponse(c, status.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status.Created, resp)
}

func (h *HandlerV2) ExecKnative(path string, req models.NewInvokeFunctionRequest) (models.InvokeFunctionResponse, error) {
	url := fmt.Sprintf("http://%s.%s", path, config.KnativeBaseUrl)
	resp, err := util.DoRequest(url, http.MethodPost, req)
	if err != nil {
		return models.InvokeFunctionResponse{}, err
	}

	return resp, nil
}

func (h *HandlerV2) InvokeInAdminWithoutAuth(c *gin.Context) {
	var (
		invokeFunction models.CommonMessage
		path           = c.Param("function-path")
	)

	if err := c.ShouldBindJSON(&invokeFunction); err != nil {
		h.handleResponse(c, status.BadRequest, err.Error())
		return
	}

	projectId := c.Query("project_id")

	environmentId := c.Query("environment_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId,
			EnvironmentId: environmentId,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status.GRPCError, err.Error())
		return
	}

	apiKeys, err := h.authService.ApiKey().GetList(c.Request.Context(), &as.GetListReq{
		EnvironmentId: environmentId,
		ProjectId:     resource.ProjectId,
		Limit:         1,
		Offset:        0,
	})
	if err != nil {
		h.handleResponse(c, status.GRPCError, err.Error())
		return
	}
	if len(apiKeys.Data) < 1 {
		h.handleResponse(c, status.InvalidArgument, "Api key not found")
		return
	}

	invokeFunction.Data["project_id"] = projectId
	invokeFunction.Data["environment_id"] = environmentId
	invokeFunction.Data["app_id"] = apiKeys.GetData()[0].GetAppId()
	request := models.NewInvokeFunctionRequest{Data: invokeFunction.Data}

	resp, err := h.ExecKnative(path, request)
	if err != nil {
		h.handleResponse(c, status.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status.Created, resp)
}

func (h *HandlerV2) InvokeInAdminWithoutData(c *gin.Context) {
	var (
		invokeFunction models.CommonMessage
		path           = c.Param("function-path")
	)

	if err := c.ShouldBindJSON(&invokeFunction); err != nil {
		h.handleResponse(c, status.BadRequest, err.Error())
		return
	}

	request := models.NewInvokeFunctionRequest{Data: invokeFunction.Data}

	resp, err := h.ExecKnative(path, request)
	if err != nil {
		h.handleResponse(c, status.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status.Created, resp)
}

func (h *HandlerV2) InvokeInAdminAuthData(c *gin.Context) {
	var (
		invokeFunction models.CommonMessage
		path           = c.Param("function-path")
	)

	if err := c.ShouldBindJSON(&invokeFunction); err != nil {
		h.handleResponse(c, status.BadRequest, err.Error())
		return
	}

	authInfo, _ := h.GetAuthInfo(c)

	invokeFunction.Data["user_id"] = authInfo.GetUserId()
	invokeFunction.Data["project_id"] = authInfo.GetProjectId()
	invokeFunction.Data["environment_id"] = authInfo.GetEnvId()
	request := models.NewInvokeFunctionRequest{Data: invokeFunction.Data}

	resp, err := h.ExecKnative(path, request)
	if err != nil {
		h.handleResponse(c, status.InvalidArgument, err.Error())
		return
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		h.handleResponse(c, status.InvalidArgument, errStr)
		return
	}

	h.handleResponse(c, status.Created, resp)
}

func (h *HandlerV2) InvokeInAdminProxyWithoutAuth(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}
