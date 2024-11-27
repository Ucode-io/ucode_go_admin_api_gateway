package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// CreateFunction godoc
// @Security ApiKeyAuth
// @ID create_function
// @Router /v1/function [POST]
// @Summary Create Function
// @Description Create Function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.CreateFunctionRequest true "CreateFunctionRequestBody"
// @Success 201 {object} status_http.Response{data=models.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateFunction(c *gin.Context) {
	var function models.CreateFunctionRequest

	err := c.ShouldBindJSON(&function)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	// structData, err := helper.ConvertMapToStruct(function.Body)

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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Function().Create(
		context.Background(),
		&obs.CreateFunctionRequest{
			Path:        function.Path,
			Name:        function.Name,
			Description: function.Description,
			// Body:        structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetFunctionByID godoc
// @Security ApiKeyAuth
// @ID get_function_by_id
// @Router /v1/function/{function_id} [GET]
// @Summary Get Function by id
// @Description Get Function by id
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 200 {object} status_http.Response{data=models.Function} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetFunctionByID(c *gin.Context) {
	functionID := c.Param("function_id")

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
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

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Function().GetSingle(
		context.Background(),
		&obs.FunctionPrimaryKey{
			Id:        functionID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllFunctions godoc
// @Security ApiKeyAuth
// @ID get_all_functions
// @Router /v1/function [GET]
// @Summary Get all functions
// @Description Get all functions
// @Tags Function
// @Accept json
// @Produce json
// @Param filters query obs.GetAllFunctionsRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "FunctionBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllFunctions(c *gin.Context) {

	limit, err := h.getLimitParam(c)
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Function().GetList(
			context.Background(),
			&obs.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Function().GetList(
			context.Background(),
			&nb.GetAllFunctionsRequest{
				Search:    c.DefaultQuery("search", ""),
				Limit:     int32(limit),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	}
}

// UpdateFunction godoc
// @Security ApiKeyAuth
// @ID update_function
// @Router /v1/function [PUT]
// @Summary Update function
// @Description Update function
// @Tags Function
// @Accept json
// @Produce json
// @Param Function body models.Function  true "UpdateFunctionRequestBody"
// @Success 200 {object} status_http.Response{data=models.Function} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateFunction(c *gin.Context) {
	var function models.Function

	err := c.ShouldBindJSON(&function)
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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Function().Update(
		context.Background(),
		&obs.Function{
			Id:          function.ID,
			Description: function.Description,
			Name:        function.Name,
			Path:        function.Path,
			// Body:        structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteFunction godoc
// @Security ApiKeyAuth
// @ID delete_function
// @Router /v1/function/{function_id} [DELETE]
// @Summary Delete Function
// @Description Delete Function
// @Tags Function
// @Accept json
// @Produce json
// @Param function_id path string true "function_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteFunction(c *gin.Context) {
	functionID := c.Param("function_id")

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, status_http.InvalidArgument, "function id is an invalid uuid")
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

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Function().Delete(
		context.Background(),
		&obs.FunctionPrimaryKey{
			Id:        functionID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}

// InvokeFunction godoc
// @Security ApiKeyAuth
// @ID invoke_function
// @Router /v1/invoke_function [POST]
// @Summary Invoke Function
// @Description Invoke Function
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionRequest body models.InvokeFunctionRequest true "InvokeFunctionRequest"
// @Success 201 {object} status_http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) InvokeFunction(c *gin.Context) {
	var invokeFunction models.InvokeFunctionRequest

	err := c.ShouldBindJSON(&invokeFunction)
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

	function := &obs.Function{}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		function, err = services.GetBuilderServiceByType(resource.NodeType).Function().GetSingle(
			c.Request.Context(),
			&obs.FunctionPrimaryKey{
				Id:        invokeFunction.FunctionID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		newFunction, err := services.GoObjectBuilderService().Function().GetSingle(
			context.Background(),
			&nb.FunctionPrimaryKey{
				Id:        invokeFunction.FunctionID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		err = helper.MarshalToStruct(newFunction, &function)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	apiKeys, err := h.authService.ApiKey().GetList(context.Background(), &auth_service.GetListReq{
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

	if invokeFunction.Attributes == nil {
		invokeFunction.Attributes = make(map[string]interface{}, 0)
	}
	authInfo, _ := h.GetAuthInfo(c)

	resp, err := util.DoRequest("https://ofs.u-code.io/function/"+function.Path, "POST", models.NewInvokeFunctionRequest{
		Data: map[string]interface{}{
			"object_ids":     invokeFunction.ObjectIDs,
			"app_id":         apiKeys.GetData()[0].GetAppId(),
			"attributes":     invokeFunction.Attributes,
			"user_id":        authInfo.GetUserId(),
			"project_id":     projectId,
			"environment_id": environmentId,
			"action_type":    "HTTP",
			"table_slug":     invokeFunction.TableSlug,
		},
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
	if c.Query("form_input") != "true" && c.Query("use_no_limit") != "true" {
		switch resource.ResourceType {
		case pb.ResourceType_MONGODB:
			_, err = services.GetBuilderServiceByType(resource.NodeType).CustomEvent().UpdateByFunctionId(
				context.Background(),
				&obs.UpdateByFunctionIdRequest{
					FunctionId: invokeFunction.FunctionID,
					ObjectIds:  invokeFunction.ObjectIDs,
					FieldSlug:  function.Path + "_disable",
					ProjectId:  resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case pb.ResourceType_POSTGRESQL:
			_, err = services.GoObjectBuilderService().CustomEvent().UpdateByFunctionId(
				context.Background(),
				&nb.UpdateByFunctionIdRequest{
					FunctionId: invokeFunction.FunctionID,
					ObjectIds:  invokeFunction.ObjectIDs,
					FieldSlug:  function.Path + "_disable",
					ProjectId:  resource.ResourceEnvironmentId,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	}
	h.handleResponse(c, status_http.Created, resp)
}
