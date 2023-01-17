package handlers

import (
	"ucode/ucode_go_api_gateway/config"

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
// @Success 200 {object} status_http.Response{data=string} "Response data"
// @Failure 500 {object} status_http.Response{}
func (h *Handler) Ping(c *gin.Context) {
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
func (h *Handler) GetConfig(c *gin.Context) {
	switch h.cfg.Environment {
	case config.DebugMode:
		h.handleResponse(c, status_http.OK, h.cfg)
		return
	case config.TestMode:
		h.handleResponse(c, status_http.OK, h.cfg.Environment)
		return
	case config.ReleaseMode:
		h.handleResponse(c, status_http.OK, "private data")
		return
	}

	h.handleResponse(c, status_http.BadEnvironment, "wrong environment value passed")
}
