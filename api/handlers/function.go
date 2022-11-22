package handlers

import (
	"context"
	"fmt"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

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
// @Success 201 {object} http.Response{data=models.Function} "Function data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateFunction(c *gin.Context) {
	var function models.CreateFunctionRequest

	err := c.ShouldBindJSON(&function)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	structData, err := helper.ConvertMapToStruct(function.Body)
	fmt.Println("err", err)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FunctionService().Create(
		context.Background(),
		&obs.CreateFunctionRequest{
			Path:        function.Path,
			Name:        function.Name,
			Description: function.Description,
			Body:        structData,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
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
// @Success 200 {object} http.Response{data=models.Function} "FunctionBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetFunctionByID(c *gin.Context) {
	functionID := c.Param("function_id")

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, http.InvalidArgument, "function id is an invalid uuid")
		return
	}

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FunctionService().GetSingle(
		context.Background(),
		&obs.FunctionPrimaryKey{
			Id: functionID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Param filters query object_builder_service.GetAllFunctionsRequest true "filters"
// @Success 200 {object} http.Response{data=string} "FunctionBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllFunctions(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FunctionService().GetList(
		context.Background(),
		&obs.GetAllFunctionsRequest{
			Search: c.DefaultQuery("search", ""),
			Limit:  int32(limit),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Success 200 {object} http.Response{data=models.Function} "Function data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateFunction(c *gin.Context) {
	var function models.Function

	err := c.ShouldBindJSON(&function)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	structData, err := helper.ConvertMapToStruct(function.Body)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FunctionService().Update(
		context.Background(),
		&obs.Function{
			Id:          function.ID,
			Description: function.Description,
			Name:        function.Name,
			Path:        function.Path,
			Body:        structData,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
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
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteFunction(c *gin.Context) {
	functionID := c.Param("function_id")

	if !util.IsValidUUID(functionID) {
		h.handleResponse(c, http.InvalidArgument, "function id is an invalid uuid")
		return
	}

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.FunctionService().Delete(
		context.Background(),
		&obs.FunctionPrimaryKey{
			Id: functionID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
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
// @Success 201 {object} http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) InvokeFunction(c *gin.Context) {
	var invokeFunction models.InvokeFunctionRequest

	err := c.ShouldBindJSON(&invokeFunction)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetHeader("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	function, err := services.FunctionService().GetSingle(
		context.Background(),
		&obs.FunctionPrimaryKey{
			Id: invokeFunction.FunctionID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	resp, err := util.DoRequest("https://ofs.medion.udevs.io/function/"+function.Path, "POST", invokeFunction)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}
	_, err = services.CustomEventService().UpdateByFunctionId(context.Background(), &obs.UpdateByFunctionIdRequest{
		FunctionId: invokeFunction.FunctionID,
		ObjectIds:  invokeFunction.ObjectIDs,
		FieldSlug:  function.Path + "_disable",
	})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}
	h.handleResponse(c, http.Created, resp)
}
