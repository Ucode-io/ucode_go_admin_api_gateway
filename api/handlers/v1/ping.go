package v1

import (
	"context"
	"fmt"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"ucode/ucode_go_api_gateway/api/status_http"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gin-gonic/gin"
)

var (
	pingSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ping_success",
			Help: "Number of successful pings",
		},
		[]string{"service"},
	)
	pingFail = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ping_fail",
			Help: "Number of failed pings",
		},
		[]string{"service"},
	)
)

func init() {
	prometheus.MustRegister(pingSuccess, pingFail)
}

// Ping godoc
// @ID ping
// @Router /ping [GET]
// @Summary returns "pong" message
// @Description this returns "pong" messsage to show service is working
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=string} "Response data"
// @Failure 500 {object} status_http.Response{}
func (h *HandlerV1) Ping(c *gin.Context) {
	fmt.Println("config.CountReq: ", config.CountReq)

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
		pingFail.WithLabelValues("company_service").Inc()
	}
	pingSuccess.WithLabelValues("company_service").Inc()


	_, err = h.authService.User().GetUserProjects(context.Background(), &auth_service.UserPrimaryKey{
		Id: "",
	})
	if err != nil {
		pingFail.WithLabelValues("auth_service").Inc()
	}

	pingSuccess.WithLabelValues("auth_service").Inc()

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
