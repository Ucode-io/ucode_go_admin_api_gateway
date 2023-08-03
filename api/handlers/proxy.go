package handlers

import (
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Proxy(c *gin.Context) {
	h.handleResponse(c, status_http.OK, "PROXY response")
}
