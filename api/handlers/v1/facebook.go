package v1

import (
	"net/http"

	"ucode/ucode_go_api_gateway/api/models"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) FacebookWebhookVerify(c *gin.Context) {
	var (
		mode      = c.Query("hub.mode")
		token     = c.Query("hub.verify_token")
		challenge = c.Query("hub.challenge")
	)

	if mode == "subscribe" && token == h.baseConf.FacebookWebhookVerifyToken && token != "" {
		c.String(http.StatusOK, challenge)
		return
	}

	c.String(http.StatusForbidden, "verification failed")
}

func (h *HandlerV1) FacebookWebhookReceive(c *gin.Context) {
	var event models.FacebookLeadWebhookEvent

	_ = c.ShouldBindJSON(&event)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
