package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ucode/ucode_go_api_gateway/api/status_http"
)

func (h *Handler) Proxy(c *gin.Context) {
	fmt.Println("PROXY:::::::::: in function")
	h.handleResponse(c, status_http.OK, "PROXY response")
}
