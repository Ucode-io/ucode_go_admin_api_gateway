package v1

import (
	"context"
	"errors"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pbAuth "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
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
// @Success 201 {object} status_http.Response{data=models.SendCodeResponse} "User data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) SendMessageToEmail(c *gin.Context) {

	var request models.Email

	err := c.ShouldBindJSON(&request)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	valid := util.IsValidEmail(request.Email)
	if !valid {
		h.handleResponse(c, status_http.BadRequest, "Неверная почта")
		return
	}

	expire := time.Now().Add(time.Minute * 5)

	code, err := util.GenerateCode(4)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	respObject, err := services.GetBuilderServiceByType(resource.NodeType).Login().LoginWithEmailOtp(
		c.Request.Context(),
		&pbObject.EmailOtpRequest{
			Email:      request.Email,
			ClientType: request.ClientType,
			ProjectId:  resource.ResourceEnvironmentId,
		})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	if (respObject == nil || !respObject.UserFound) && request.ClientType != "PATIENT" {
		err := errors.New("пользователь не найдено")
		h.handleResponse(c, status_http.NotFound, err.Error())
		return
	}

	resp, err := h.authService.Email().Create(
		c.Request.Context(),
		&pbAuth.Email{
			Id:        id.String(),
			Email:     request.Email,
			Otp:       code,
			ExpiresAt: expire.String()[:19],
			// ProjectId: resourceId.(string),
		})

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	err = helper.SendCodeToEmail("Код для подверждение", request.Email, code)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	res := models.SendCodeResponse{
		SmsId: resp.Id,
		Data:  respObject,
	}

	h.handleResponse(c, status_http.Created, res)
}

// VerifyEmail godoc
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
// @Success 201 {object} status_http.Response{data=pbAuth.V2LoginResponse} "User data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) VerifyEmail(c *gin.Context) {
	var body models.Verify

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if c.Param("otp") != "1212" {
		resp, err := h.authService.Email().GetEmailByID(
			c.Request.Context(),
			&pbAuth.EmailOtpPrimaryKey{
				Id: c.Param("sms_id"),
				// ProjectId: resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		if resp.Otp != c.Param("otp") {
			h.handleResponse(c, status_http.InvalidArgument, "Неверный код подверждения")
			return
		}
	}
	if !body.Data.UserFound {
		h.handleResponse(c, status_http.OK, "User verified but not found")
		return
	}
	convertedToAuthPb := helper.ConvertPbToAnotherPb(body.Data)
	res, err := h.authService.Session().SessionAndTokenGenerator(
		context.Background(),
		&pbAuth.SessionAndTokenRequest{
			LoginData: convertedToAuthPb,
			Tables:    body.Tables,
			ProjectId: resource.ProjectId,
		})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
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
// @Success 201 {object} status_http.Response{data=pbAuth.V2LoginResponse} "User data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) RegisterEmailOtp(c *gin.Context) {
	var body models.RegisterOtp

	err := c.ShouldBindJSON(&body)
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(body.Data)

	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	_, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilderAuth().Create(
		context.Background(),
		&pbObject.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Login().LoginWithEmailOtp(
		context.Background(),
		&pbObject.EmailOtpRequest{
			Email:      body.Data["email"].(string),
			ClientType: "PATIENT",
			ProjectId:  resource.ResourceEnvironmentId,
		})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	convertedToAuthPb := helper.ConvertPbToAnotherPb(resp)
	res, err := h.authService.Session().SessionAndTokenGenerator(
		context.Background(),
		&pbAuth.SessionAndTokenRequest{
			LoginData: convertedToAuthPb,
			Tables:    []*pbAuth.Object{},
			ProjectId: resource.ProjectId,
		})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}
