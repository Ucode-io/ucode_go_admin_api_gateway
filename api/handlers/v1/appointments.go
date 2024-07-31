package v1

import (
	"context"
	ps "ucode/ucode_go_api_gateway/genproto/pos_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

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
// @Param filters query ps.GetAllOfflineAppointmentsRequest true "filters"
// @Success 200 {object} status_http.Response{data=ps.GetAllOfflineAppointmentsResponse} "OfflineAppointmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllOfflineAppointments(c *gin.Context) {
	authBody, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	cashboxId := ""
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().OfflineAppointment().GetList(
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
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Param filters query ps.GetAllBookedAppointmentsRequest true "filters"
// @Success 200 {object} status_http.Response{data=ps.GetAllBookedAppointmentsResponse} "BookedAppointmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllBookedAppointments(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().BookedAppointment().GetList(
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
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Success 200 {object} status_http.Response{data=ps.GetSingleOfflineAppointmentResponse} "OfflineAppointmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingleOfflineAppointment(c *gin.Context) {
	offlineAppointmentID := c.Param("offline_appointment_id")

	if !util.IsValidUUID(offlineAppointmentID) {
		h.handleResponse(c, status_http.InvalidArgument, "offline_appointment_id id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().OfflineAppointment().GetSingle(
		context.Background(),
		&ps.OfflineAppointmentPrimaryKey{
			Id: offlineAppointmentID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Success 200 {object} status_http.Response{data=ps.GetSingleBookedAppointmentResponse} "BookedAppointmentBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingleBookedAppointment(c *gin.Context) {
	bookedAppointmentID := c.Param("booked_appointment_id")

	if !util.IsValidUUID(bookedAppointmentID) {
		h.handleResponse(c, status_http.InvalidArgument, "booked_appointment_id id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().BookedAppointment().GetSingle(
		context.Background(),
		&ps.BookedAppointmentPrimaryKey{
			Id: bookedAppointmentID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Param view body ps.UpdatePaymentStatusBody true "UpdateAppointmentStatus"
// @Success 200 {object} status_http.Response{data=ps.OfflineAppointment} "Appointment data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateAppointmentPaymentStatus(c *gin.Context) {
	authBody, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	var paymentBody ps.UpdatePaymentStatusBody

	err = c.ShouldBindJSON(&paymentBody)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().OfflineAppointment().UpdatePaymentStatus(
		context.Background(),
		&paymentBody,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetCloseCashboxInfo godoc
// @Security ApiKeyAuth
// @ID close_cashbox_info
// @Router /v1/close-cashbox [GET]
// @Summary Get close cashbox
// @Description Get close cashbox
// @Tags Appointment
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=ps.CashboxResponse} "Cashbox data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetCloseCashboxInfo(c *gin.Context) {
	authBody, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().OfflineAppointment().GetCloseCashboxInfo(
		context.Background(),
		&cashbox,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetOpenCashboxInfo godoc
// @Security ApiKeyAuth
// @ID open_cashbox_info
// @Router /v1/open-cashbox [GET]
// @Summary Get open cashbox
// @Description Get open cashbox
// @Tags Appointment
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=ps.CashboxResponse} "Cashbox data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetOpenCashboxInfo(c *gin.Context) {
	authBody, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.PosService().OfflineAppointment().GetOpenCashboxInfo(
		context.Background(),
		&cashbox,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)

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
// @Param app body ps.CreateCashboxTransactionRequest true "CreateTransactionBody"
// @Success 201
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CashboxTransaction(c *gin.Context) {
	authBody, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
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

	err = c.ShouldBindJSON(&cashboxTransactionRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	_, err = services.PosService().OfflineAppointment().CreateCashboxTransaction(context.Background(), &ps.CreateCashboxTransactionRequest{
		CashboxId:     cashboxId,
		AmountOfMoney: cashboxTransactionRequest.AmountOfMoney,
		Comment:       cashboxTransactionRequest.Comment,
		Status:        cashboxTransactionRequest.Status,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	h.handleResponse(c, status_http.Created, "Cashbox Transaction Created")
}
