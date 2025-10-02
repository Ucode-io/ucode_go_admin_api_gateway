package v1

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ucode/ucode_go_api_gateway/api/status_http"
)

type createPaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	// Optional: metadata or description can be extended later
}

type paymentIntentResponse struct {
	ID           string                 `json:"id"`
	Object       string                 `json:"object"`
	Amount       int64                  `json:"amount"`
	Currency     string                 `json:"currency"`
	Status       string                 `json:"status"`
	ClientSecret string                 `json:"client_secret"`
	Created      int64                  `json:"created"`
	Livemode     bool                   `json:"livemode"`
	Metadata     map[string]string      `json:"metadata"`
	NextAction   map[string]interface{} `json:"next_action"`
}

// MockPaymentIntent creates a PaymentIntent-like response without calling any external API
func (h *HandlerV1) MockPaymentIntent(c *gin.Context) {
	var req createPaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, "Invalid JSON")
		return
	}

	if req.Amount <= 0 {
		req.Amount = 1000
	}
	if req.Currency == "" {
		req.Currency = "usd"
	}

	id := "pi_" + uuid.NewString()
	clientSecret := id + "_secret_" + uuid.NewString()

	resp := paymentIntentResponse{
		ID:           id,
		Object:       "payment_intent",
		Amount:       req.Amount,
		Currency:     req.Currency,
		Status:       "requires_payment_method",
		ClientSecret: clientSecret,
		Created:      time.Now().Unix(),
		Livemode:     false,
		Metadata:     map[string]string{},
		NextAction:   map[string]interface{}{},
	}

	h.handleResponse(c, status_http.Created, resp)
}
