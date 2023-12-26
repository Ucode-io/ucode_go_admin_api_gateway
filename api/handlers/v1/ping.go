package v1

import (
	"context"
	"fmt"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

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
// @Success 200 {object} status_http.Response{data=string} "Response data"
// @Failure 500 {object} status_http.Response{}
func (h *HandlerV1) Ping(c *gin.Context) {
	fmt.Println("config.PingRequest: ", config.CountReq)

	service := c.Query("service")
	fmt.Println("SERVICENAME: ", service)
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

	_, err = h.companyServices.Company().GetListWithProjects(
		context.Background(),
		&company_service.GetListWithProjectsRequest{
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	fmt.Println("Connected to company service")


	_, err = h.authService.User().GetUserProjects(context.Background(), &auth_service.UserPrimaryKey{
		Id: "",
	})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	fmt.Println("Connected to auth service")

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
