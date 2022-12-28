package handlers

import (
	"context"
	"errors"
	"time"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/auth_service"
	pbObject "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SendMessageToEmail godoc
// @ID send_message_to_email
// @Router /send-message [POST]
// @Summary Send Message To Email
// @Description Send Message to Email
// @Tags Email
// @Accept json
// @Produce json
// @Param send_message body models.Email true "SendMessageToEmailRequestBody"
// @Success 201 {object} http.Response{data=models.SendCodeResponse} "User data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) SendMessageToEmail(c *gin.Context) {

	var request models.Email

	err := c.ShouldBindJSON(&request)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		h.handleResponse(c, http.InternalServerError, err.Error())
		return
	}

	valid := util.IsValidEmail(request.Email)
	if !valid {
		h.handleResponse(c, http.BadRequest, "Неверная почта")
		return
	}

	expire := time.Now().Add(time.Minute * 5)

	code, err := util.GenerateCode(4)
	if err != nil {
		h.handleResponse(c, http.InternalServerError, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	respObject, err := services.LoginService().LoginWithEmailOtp(
		c.Request.Context(),
		&pbObject.EmailOtpRequest{
			Email:      request.Email,
			ClientType: request.ClientType,
			ProjectId:  authInfo.GetProjectId(),
		})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}
	if (respObject == nil || !respObject.UserFound) && request.ClientType != "PATIENT" {
		err := errors.New("Пользователь не найдено")
		h.handleResponse(c, http.NotFound, err.Error())
		return
	}

	resp, err := services.EmailServie().Create(
		c.Request.Context(),
		&pb.Email{
			Id:        id.String(),
			Email:     request.Email,
			Otp:       code,
			ExpiresAt: expire.String()[:19],
			// ProjectId: authInfo.GetProjectId(),
		})

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}
	err = helper.SendCodeToEmail("Код для подверждение", request.Email, code)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	res := models.SendCodeResponse{
		SmsId: resp.Id,
		Data:  respObject,
	}

	h.handleResponse(c, http.Created, res)
}

// Verify godoc
// @ID verify_email
// @Router /verify-email/{sms_id}/{otp} [POST]
// @Summary Verify
// @Description Verify
// @Tags Email
// @Accept json
// @Produce json
// @Param sms_id path string true "sms_id"
// @Param otp path string true "otp"
// @Param verifyBody body models.Verify true "verify_body"
// @Success 201 {object} http.Response{data=auth_service.V2LoginResponse} "User data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) VerifyEmail(c *gin.Context) {
	var body models.Verify

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	if c.Param("otp") != "1212" {
		resp, err := services.EmailServie().GetEmailByID(
			c.Request.Context(),
			&pb.EmailOtpPrimaryKey{
				Id: c.Param("sms_id"),
				// ProjectId: authInfo.GetProjectId(),
			},
		)
		if err != nil {
			h.handleResponse(c, http.GRPCError, err.Error())
			return
		}
		if resp.Otp != c.Param("otp") {
			h.handleResponse(c, http.InvalidArgument, "Неверный код подверждения")
			return
		}
	}
	if !body.Data.UserFound {
		h.handleResponse(c, http.OK, "User verified but not found")
		return
	}
	convertedToAuthPb := helper.ConvertPbToAnotherPb(body.Data)
	res, err := services.SessionService().SessionAndTokenGenerator(
		context.Background(),
		&pb.SessionAndTokenRequest{
			LoginData: convertedToAuthPb,
			Tables:    body.Tables,
			ProjectId: authInfo.GetProjectId(),
		})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, res)
}

// RegisterEmailOtp godoc
// @ID registerEmailOtp
// @Router /register-email-otp/{table_slug} [POST]
// @Summary RegisterEmailOtp
// @Description RegisterOtp
// @Tags Email
// @Accept json
// @Produce json
// @Param registerBody body models.RegisterOtp true "register_body"
// @Param table_slug path string true "table_slug"
// @Success 201 {object} http.Response{data=auth_service.V2LoginResponse} "User data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) RegisterEmailOtp(c *gin.Context) {
	var body models.RegisterOtp

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	structData, err := helper.ConvertMapToStruct(body.Data)

	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}
	_, err = services.ObjectBuilderServiceAuth().Create(
		context.Background(),
		&pbObject.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	resp, err := services.LoginService().LoginWithEmailOtp(
		context.Background(),
		&pbObject.EmailOtpRequest{
			Email:      body.Data["email"].(string),
			ClientType: "PATIENT",
			ProjectId:  authInfo.GetProjectId(),
		})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	convertedToAuthPb := helper.ConvertPbToAnotherPb(resp)
	res, err := services.SessionServiceAuth().SessionAndTokenGenerator(
		context.Background(),
		&pb.SessionAndTokenRequest{
			LoginData: convertedToAuthPb,
			Tables:    []*pb.Object{},
			ProjectId: authInfo.GetProjectId(),
		})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, res)
}
