package v1

import (
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) SleepApi(c *gin.Context) {
	time.Sleep(5 * time.Minute)
	h.handleResponse(c, status_http.OK, "hey")
}
