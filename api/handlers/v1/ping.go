package v1

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	fc "ucode/ucode_go_api_gateway/genproto/new_function_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// Ping godoc
// @ID ping
// @Router /ping [GET]
// @Summary returns "pong" message
// @Description this returns "pong" messsage to show service is working
// @Accept json
// @Produce json
// @Param service query string false "service"
// @Param environment_id query string false "environment_id"
// @Param project_id query string false "project_id"
// @Success 200 {object} status_http.Response{data=string} "Response data"
// @Failure 500 {object} status_http.Response{}
func (h *HandlerV1) Ping(c *gin.Context) {
	fmt.Println("config.PingRequest: ", config.CountReq)

	service := c.Query("service")
	fmt.Println("SERVICENAME: ", service)

	limit := 10
	offset := 0

	if service == "company_service" {
		_, err := h.companyServices.CompanyPing().Ping(context.Background(), &pb.PingRequest{})
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		fmt.Println("Ping to Company Service")
	} else if service == "auth_service" {

		_, err := h.authService.AuthPing().Ping(context.Background(), &auth_service.PingRequest{})
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}

		fmt.Println("Ping To Auth Service")
	} else if service == "object_builder_service" || service == "function_service" {

		projectId := c.Query("project_id")

		environmentId := c.Query("environment_id")

		if service == "object_builder_service" {
			resource, err := h.companyServices.ServiceResource().GetSingle(
				c.Request.Context(),
				&pb.GetSingleServiceResourceReq{
					ProjectId:     projectId,
					EnvironmentId: environmentId,
					ServiceType:   pb.ServiceType_BUILDER_SERVICE,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			services, err := h.GetProjectSrvc(
				c.Request.Context(),
				projectId,
				resource.NodeType,
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			_, err = services.GetBuilderServiceByType(resource.NodeType).Function().GetList(
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

			fmt.Println("Ping to Object Builder Service")
		} else if service == "function_service" {
			resource, err := h.companyServices.ServiceResource().GetSingle(
				c.Request.Context(),
				&pb.GetSingleServiceResourceReq{
					ProjectId:     projectId,
					EnvironmentId: environmentId,
					ServiceType:   pb.ServiceType_FUNCTION_SERVICE,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			environment, err := h.companyServices.Environment().GetById(
				context.Background(),
				&pb.EnvironmentPrimaryKey{
					Id: environmentId,
				},
			)
			if err != nil {
				err = errors.New("error getting resource environment id")
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			services, err := h.GetProjectSrvc(
				c.Request.Context(),
				projectId,
				resource.NodeType,
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			_, err = services.FunctionService().FunctionService().GetList(
				context.Background(),
				&fc.GetAllFunctionsRequest{
					Search:        c.DefaultQuery("search", ""),
					Limit:         int32(limit),
					Offset:        int32(offset),
					ProjectId:     resource.ResourceEnvironmentId,
					EnvironmentId: environment.GetId(),
					Type:          FUNCTION,
				},
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			fmt.Println("Ping to Function Service")
		}
	}

	h.handleResponse(c, status_http.OK, "pong")
}

// GetConfig godoc
// @ID get_config
// @Router /config [GET]
// @Summary get config data on the debug mode
// @Description show service config data when the service environment set to debug mode
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=config.Config} "Response data"
// @Failure 400 {object} status_http.Response{}
func (h *HandlerV1) GetConfig(c *gin.Context) {
	switch h.baseConf.Environment {
	case config.DebugMode:
		h.handleResponse(c, status_http.OK, h.baseConf)
		return
	case config.TestMode:
		h.handleResponse(c, status_http.OK, h.baseConf)
		return
	case config.ReleaseMode:
		h.handleResponse(c, status_http.OK, "private data")
		return
	}

	h.handleResponse(c, status_http.BadEnvironment, "wrong environment value passed")
}
