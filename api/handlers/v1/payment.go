package v1

import (
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

	stripe.Key = "sk_test_51SDcleECKew23mDJnmG6yfVSLAVjHUskKFh27GdEeGujYsBfi4yLFOqyQGnNBIXHxV9pja1DXYitDShTbrCLZxUa00cvbfmFBb"

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
