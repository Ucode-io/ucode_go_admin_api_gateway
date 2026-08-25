package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
)

type createIpakPaymentRequest struct {
	// Amount in UZS (the frontend converts the entered USD via cbu.uz before sending).
	Amount float64 `json:"amount"`
}

// CreateIpakPayment opens a Visa/Mastercard hosted-page top-up and returns the bank
// payment_url for the frontend to redirect the customer to. JWT-protected.
func (h *HandlerV1) CreateIpakPayment(c *gin.Context) {
	var request createIpakPaymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleError(c, status_http.BadRequest, err)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleError(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return
	}

	uid, _ := c.Get("user_id")
	userId, _ := uid.(string)

	resp, err := h.companyServices.Billing().CreateIpakPayment(c, &pb.CreateIpakPaymentRequest{
		ProjectId: projectId.(string),
		UserId:    userId,
		Amount:    request.Amount,
	})
	if err != nil {
		h.handleError(c, status_http.GRPCError, err)
		return
	}

	h.HandleResponse(c, status_http.Created, resp)
}

// GetIpakPaymentStatus lets the frontend return page poll a top-up's status. If the
// payment is still pending it actively re-checks the bank (transfer.get). JWT-protected.
func (h *HandlerV1) GetIpakPaymentStatus(c *gin.Context) {
	transferId := c.Param("transfer_id")
	if transferId == "" {
		h.HandleResponse(c, status_http.BadRequest, "transfer_id is required")
		return
	}

	resp, err := h.companyServices.Billing().GetIpakPaymentStatus(c, &pb.ConfirmIpakPaymentRequest{TransferId: transferId})
	if err != nil {
		h.handleError(c, status_http.GRPCError, err)
		return
	}

	h.HandleResponse(c, status_http.OK, resp)
}

type ipakCallbackPayload struct {
	TransactionId string  `json:"transactionId"`
	OrderId       string  `json:"orderId"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount"`
	NetAmount     float64 `json:"netAmount"`
	SourceAccount string  `json:"sourceAccount"`
}

// IpakYuliPaymentWebhook receives the bank's payment callback. It is public (the
// bank calls it) and authenticated with a shared bearer secret. The callback body
// is only a trigger: settlement is verified server-side via transfer.get inside
// ConfirmIpakPayment, so a spoofed body cannot credit anyone. The bank sends the
// callback exactly once and never retries, so we always acknowledge with 200 and
// let the reconcile cron backstop any failure here.
func (h *HandlerV1) IpakYuliPaymentWebhook(c *gin.Context) {
	expected := h.baseConf.IpakPaymentCallbackBearer
	if expected == "" || c.GetHeader("Authorization") != "Bearer "+expected {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}

	var payload ipakCallbackPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}

	if payload.TransactionId != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		if _, err := h.companyServices.Billing().ConfirmIpakPayment(ctx, &pb.ConfirmIpakPaymentRequest{
			TransferId: payload.TransactionId,
		}); err != nil {
			h.log.Error("ipak callback confirm failed", logger.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
