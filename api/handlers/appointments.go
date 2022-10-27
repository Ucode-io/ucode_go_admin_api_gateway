package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	ps "ucode/ucode_go_api_gateway/genproto/pos_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// GetAllOfflineAppointments godoc
// @Security ApiKeyAuth
// @ID get_all_offline_appointments
// @Router /v1/offline_appointment [GET]
// @Summary Get all offline appointments
// @Description Get all offline appointments
// @Tags Appointment
// @Accept json
// @Produce json
// @Param filters query pos_service.GetAllOfflineAppointmentsRequest true "filters"
// @Success 200 {object} http.Response{data=pos_service.GetAllOfflineAppointmentsResponse} "OfflineAppointmentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllOfflineAppointments(c *gin.Context) {
	authBody := h.GetAuthInfo(c)
	cashboxId := ""
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}
	if authBody.Tables != nil {
		if authBody.Tables[0].TableSlug == "cashbox" {
			cashboxId = authBody.Tables[0].ObjectId
		} else {
			cashboxId = "554990c9-03a1-42f4-9a85-301c1588dc10"
		}
	} else {
		cashboxId = "554990c9-03a1-42f4-9a85-301c1588dc10"
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.services.OfflineAppointmentService().GetList(
		context.Background(),
		&ps.GetAllOfflineAppointmentsRequest{
			Limit:         int32(limit),
			Offset:        int32(offset),
			Search:        c.DefaultQuery("search", ""),
			PaymentStatus: c.DefaultQuery("payment_status", ""),
			Date:          c.DefaultQuery("date", ""),
			CashboxId:     cashboxId,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetAllBookedAppointments godoc
// @Security ApiKeyAuth
// @ID get_all_booked_appointments
// @Router /v1/booked_appointment [GET]
// @Summary Get all booked appointments
// @Description Get all booked appointments
// @Tags Appointment
// @Accept json
// @Produce json
// @Param filters query pos_service.GetAllBookedAppointmentsRequest true "filters"
// @Success 200 {object} http.Response{data=pos_service.GetAllBookedAppointmentsResponse} "BookedAppointmentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllBookedAppointments(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}
	resp, err := h.services.BookedAppointmentService().GetList(
		context.Background(),
		&ps.GetAllBookedAppointmentsRequest{
			Limit:         int32(limit),
			Offset:        int32(offset),
			Search:        c.DefaultQuery("search", ""),
			PaymentStatus: c.DefaultQuery("payment_status", ""),
			Date:          c.DefaultQuery("date", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetSingleOfflineAppointment godoc
// @Security ApiKeyAuth
// @ID get_offline_appointment_by_id
// @Router /v1/offline_appointment/{offline_appointment_id} [GET]
// @Summary Get single offline appointment
// @Description Get single offline appointment
// @Tags Appointment
// @Accept json
// @Produce json
// @Param offline_appointment_id path string true "offline_appointment_id"
// @Success 200 {object} http.Response{data=pos_service.GetSingleOfflineAppointmentResponse} "OfflineAppointmentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleOfflineAppointment(c *gin.Context) {
	offlineAppointmentID := c.Param("offline_appointment_id")

	if !util.IsValidUUID(offlineAppointmentID) {
		h.handleResponse(c, http.InvalidArgument, "offline_appointment_id id is an invalid uuid")
		return
	}
	resp, err := h.services.OfflineAppointmentService().GetSingle(
		context.Background(),
		&ps.OfflineAppointmentPrimaryKey{
			Id: offlineAppointmentID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetSingleBookedAppointment godoc
// @Security ApiKeyAuth
// @ID get_booked_appointment_by_id
// @Router /v1/booked_appointment/{booked_appointment_id} [GET]
// @Summary Get single booked appointment
// @Description Get single booked appointment
// @Tags Appointment
// @Accept json
// @Produce json
// @Param booked_appointment_id path string true "booked_appointment_id"
// @Success 200 {object} http.Response{data=pos_service.GetSingleBookedAppointmentResponse} "BookedAppointmentBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingleBookedAppointment(c *gin.Context) {
	bookedAppointmentID := c.Param("booked_appointment_id")

	if !util.IsValidUUID(bookedAppointmentID) {
		h.handleResponse(c, http.InvalidArgument, "booked_appointment_id id is an invalid uuid")
		return
	}
	resp, err := h.services.BookedAppointmentService().GetSingle(
		context.Background(),
		&ps.BookedAppointmentPrimaryKey{
			Id: bookedAppointmentID,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateAppointmentPaymentStatus godoc
// @Security ApiKeyAuth
// @ID update_appointment_payment_status
// @Router /v1/payment_status/{appointment_id} [PUT]
// @Summary Update appointment payment status
// @Description Update appointment payment status
// @Tags Appointment
// @Accept json
// @Produce json
// @Param view body pos_service.UpdatePaymentStatusBody true "UpdateAppointmentStatus"
// @Success 200 {object} http.Response{data=pos_service.OfflineAppointment} "Appointment data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateAppointmentPaymentStatus(c *gin.Context) {
	authBody := h.GetAuthInfo(c)
	var paymentBody ps.UpdatePaymentStatusBody

	err := c.ShouldBindJSON(&paymentBody)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	if authBody.Tables != nil {
		if authBody.Tables[0].TableSlug == "cashbox" {
			paymentBody.CashboxId = authBody.Tables[0].ObjectId
		} else {
			paymentBody.CashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
		}
	} else {
		paymentBody.CashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
	}

	resp, err := h.services.OfflineAppointmentService().UpdatePaymentStatus(
		context.Background(),
		&paymentBody,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// CloseCashbox godoc
// @Security ApiKeyAuth
// @ID close_cashbox_info
// @Router /v1/close-cashbox [GET]
// @Summary Get close cashbox
// @Description Get close cashbox
// @Tags Appointment
// @Accept json
// @Produce json
// @Success 200 {object} http.Response{data=pos_service.CashboxResponse} "Cashbox data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCloseCashboxInfo(c *gin.Context) {
	authBody := h.GetAuthInfo(c)
	var cashbox ps.CashboxRequestBody

	if authBody.Tables != nil {
		if authBody.Tables[0].TableSlug == "cashbox" {
			cashbox.CashboxId = authBody.Tables[0].ObjectId
		} else {
			cashbox.CashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
		}
	} else {
		cashbox.CashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
	}

	resp, err := h.services.OfflineAppointmentService().GetCloseCashboxInfo(
		context.Background(),
		&cashbox,
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// OpenCashbox godoc
// @Security ApiKeyAuth
// @ID open_cashbox_info
// @Router /v1/open-cashbox [GET]
// @Summary Get open cashbox
// @Description Get open cashbox
// @Tags Appointment
// @Accept json
// @Produce json
// @Success 200 {object} http.Response{data=pos_service.CashboxResponse} "Cashbox data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetOpenCashboxInfo(c *gin.Context) {
	authBody := h.GetAuthInfo(c)
	var cashbox ps.CashboxRequestBody

	if authBody.Tables != nil {
		if authBody.Tables[0].TableSlug == "cashbox" {
			cashbox.CashboxId = authBody.Tables[0].ObjectId
		} else {
			cashbox.CashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
		}
	} else {
		cashbox.CashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
	}

	resp, err := h.services.OfflineAppointmentService().GetOpenCashboxInfo(
		context.Background(),
		&cashbox,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)

}

// CashboxTransaction godoc
// @Security ApiKeyAuth
// @ID create_cashbox_transaction
// @Router /v1/cashbox_transaction [POST]
// @Summary Create cashbox transaction
// @Description Create cashbox transaction
// @Tags Cashbox
// @Accept json
// @Produce json
// @Param app body models.CreateCashboxTransactionRequest true "CreateTransactionBody"
// @Success 201
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CashboxTransaction(c *gin.Context) {
	authBody := h.GetAuthInfo(c)
	var cashboxTransactionRequest ps.CreateCashboxTransactionRequest
	cashboxId := ""

	if authBody.Tables != nil {
		if authBody.Tables[0].TableSlug == "cashbox" {
			cashboxId = authBody.Tables[0].ObjectId
		} else {
			cashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
		}
	} else {
		cashboxId = "d20c4fcc-0f6b-408c-bc4c-2566344c3e58"
	}

	err := c.ShouldBindJSON(&cashboxTransactionRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	_, err = h.services.OfflineAppointmentService().CreateCashboxTransaction(context.Background(), &ps.CreateCashboxTransactionRequest{
		CashboxId:     cashboxId,
		AmountOfMoney: cashboxTransactionRequest.AmountOfMoney,
		Comment:       cashboxTransactionRequest.Comment,
		Status:        cashboxTransactionRequest.Status,
	})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}
	h.handleResponse(c, http.Created, "Cashbox Transaction Created")
}
