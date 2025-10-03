package v1

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/setupintent"

	"ucode/ucode_go_api_gateway/api/status_http"
)

type createPaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	// Optional: metadata or description can be extended later
}

// CreatePaymentIntent calls Stripe to create a real SetupIntent (uses configured Stripe key)
func (h *HandlerV1) CreatePaymentIntent(c *gin.Context) {
	var req createPaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, "Invalid JSON")
		return
	}

	// For SetupIntents, amount/currency are not required; ignoring provided values

	stripe.Key = "sk_test_51QvC6qCx1p2EqOQp37eMRD73jmsECnITZ1eYTn4BbYv8uLNUfGOJUf3X0j14fyjhAvcoZYucz9oCy1aEJrg7Yyp300ScU9kgfh"

	params := &stripe.SetupIntentParams{
		PaymentMethodTypes: []*string{stripe.String("card")},
	}

	si, err := setupintent.New(params)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	// return the Stripe SetupIntent directly for maximum fidelity
	h.handleResponse(c, status_http.Created, si)
}

func (h *HandlerV1) StripeWebhook(c *gin.Context) {
	req := make(map[string]any)
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, "Invalid JSON")
		return
	}

	asd, _ := json.Marshal(req)
	fmt.Println("Stripe Webhook received: ", string(asd))

	h.handleResponse(c, status_http.Created, status_http.OK)
}
