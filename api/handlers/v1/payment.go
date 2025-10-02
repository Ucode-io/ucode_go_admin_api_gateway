package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"

	"ucode/ucode_go_api_gateway/api/status_http"
)

type createPaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	// Optional: metadata or description can be extended later
}

// CreatePaymentIntent calls Stripe to create a real PaymentIntent (uses env STRIPE_SECRET_KEY or header X-Stripe-Key)
func (h *HandlerV1) CreatePaymentIntent(c *gin.Context) {
	var req createPaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, "Invalid JSON")
		return
	}

	if req.Amount <= 0 {
		h.handleResponse(c, status_http.InvalidArgument, "amount must be > 0")
		return
	}
	if req.Currency == "" {
		req.Currency = "usd"
	}

	stripe.Key = "sk_test_51SDcleECKew23mDJnmG6yfVSLAVjHUskKFh27GdEeGujYsBfi4yLFOqyQGnNBIXHxV9pja1DXYitDShTbrCLZxUa00cvbfmFBb"

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	// return the Stripe PaymentIntent directly for maximum fidelity
	h.handleResponse(c, status_http.Created, pi)
}
