package v1

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/customer"
	"github.com/stripe/stripe-go/v83/paymentmethod"
	"github.com/stripe/stripe-go/v83/setupintent"

	"ucode/ucode_go_api_gateway/api/status_http"
)

type createPaymentIntentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
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

	params := &stripe.CustomerParams{}
	if req.Email != "" {
		params.Email = stripe.String(req.Email)
	}
	if req.Phone != "" {
		params.Phone = stripe.String(req.Phone)
	}

	cus, err := customer.New(params)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	pm := &stripe.PaymentMethodParams{
		// Customer: stripe.String(cus.ID),
		Card: &stripe.PaymentMethodCardParams{
			Token: stripe.String("tok_visa"), // Using a test token; in real scenarios, use Stripe.js to create tokens securely
		},
		Type: stripe.String("card"),
	}

	paymentMethod, err := paymentmethod.New(pm)
	if err != nil {
		fmt.Println("paymentMethod err", err)
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	_, err = paymentmethod.Attach(paymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(cus.ID),
	})
	if err != nil {
		fmt.Println("attach err", err)
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	// card.List(&stripe.CardListParams{
	// 	Customer: stripe.String(cus.ID),
	// })

	intentParams := &stripe.SetupIntentParams{
		Customer:      stripe.String(cus.ID),
		PaymentMethod: &paymentMethod.ID,
	}

	setI, err := setupintent.New(intentParams)
	if err != nil {
		fmt.Println("setupintent err", err)
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	// paymentIntentParams := &stripe.PaymentIntentParams{
	// 	Amount:        stripe.Int64(req.Amount),
	// 	Currency:      stripe.String(string(stripe.CurrencyUSD)),
	// 	Customer:      stripe.String(cus.ID),
	// 	PaymentMethod: &paymentMethod.ID,
	// 	Confirm:       stripe.Bool(true),
	// 	OffSession:    stripe.Bool(true),
	// 	// ReceiptEmail:  stripe.String(req.Email),
	// }
	// paymentItent, err := paymentintent.New(paymentIntentParams)
	// if err != nil {
	// 	fmt.Println("paymentintent err", err)
	// 	h.handleResponse(c, status_http.InternalServerError, err.Error())
	// 	return
	// }

	// return the Stripe SetupIntent directly for maximum fidelity
	h.handleResponse(c, status_http.Created, setI)
}

func (h *HandlerV1) StripeWebhook(c *gin.Context) {
	// var payload []byte
	var payload = make(map[string]any)
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.handleResponse(c, status_http.BadRequest, "Invalid JSON")
		return
	}

	asd, _ := json.Marshal(payload)
	fmt.Println("payload", string(asd))

	// endpointSecret := "whsec_cOGBaP6EVo4kRUCfeKXuSWg0JAL2avRg"
	// signatureHeader := c.GetHeader("Stripe-Signature")
	// event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	// if err != nil {
	// 	h.handleResponse(c, status_http.InternalServerError, "Webhook signature verification failed."+err.Error())
	// 	return
	// }

	// fmt.Println("Received event: ", string(event.Data.Raw))

	// switch event.Type {
	// case "payment_intent.succeeded":
	// 	var paymentIntent stripe.PaymentIntent
	// 	err := json.Unmarshal(event.Data.Raw, &paymentIntent)
	// 	if err != nil {
	// 		h.log.Error("Error parsing webhook JSON: ", logger.Error(err))
	// 		h.handleResponse(c, status_http.InternalServerError, err.Error())
	// 		return
	// 	}

	// 	log.Printf("Successful payment for %d.", paymentIntent.Amount)
	// 	// Then define and call a func to handle the successful payment intent.
	// 	// handlePaymentIntentSucceeded(paymentIntent)
	// case "payment_method.attached":
	// 	var paymentMethod stripe.PaymentMethod
	// 	err := json.Unmarshal(event.Data.Raw, &paymentMethod)
	// 	if err != nil {
	// 		h.log.Error("Error parsing webhook JSON: ", logger.Error(err))
	// 		h.handleResponse(c, status_http.InternalServerError, err.Error())
	// 		return
	// 	}
	// 	// Then define and call a func to handle the successful attachment of a PaymentMethod.
	// 	// handlePaymentMethodAttached(paymentMethod)
	// default:
	// 	fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	// }

	h.handleResponse(c, status_http.Created, status_http.OK)
}
