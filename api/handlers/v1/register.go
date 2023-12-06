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
	pbSms "ucode/ucode_go_api_gateway/genproto/sms_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SendCode godoc
// @ID sendCode
// @Router /send-code [POST]
// @Summary SendCode
// @Description SendCode
// @Tags register
// @Accept json
// @Produce json
// @Param login body models.Sms true "SendCode"
// @Success 201 {object} status_http.Response{data=models.SendCodeResponse} "User data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) SendCode(c *gin.Context) {

	var request models.Sms

	err := c.ShouldBindJSON(&request)
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

	id, err := uuid.NewRandom()
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	valid := util.IsValidPhone(request.Recipient)
	if !valid {
		h.handleResponse(c, status_http.BadRequest, "Неверный номер телефона, он должен содержать двенадцать цифр и +")
		return
	}

	expire := time.Now().Add(time.Minute * 5)

	code, err := util.GenerateCode(4)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	phone := helper.ConverPhoneNumberToMongoPhoneFormat(request.Recipient)

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	//resourceEnvironment, err := h.companyServices.Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	respObject, err := services.GetBuilderServiceByType(resource.NodeType).Login().LoginWithOtp(
		c.Request.Context(),
		&pbObject.PhoneOtpRequst{
			PhoneNumber: phone,
			ClientType:  request.ClientType,
			ProjectId:   resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if (respObject == nil || !respObject.UserFound) && request.ClientType != "PATIENT" {
		err := errors.New("пользователь не найдено")
		h.handleResponse(c, status_http.NotFound, err.Error())
		return
	}

	resp, err := services.SmsService().SmsService().Send(
		c.Request.Context(),
		&pbSms.Sms{
			Id:        id.String(),
			Text:      request.Text,
			Otp:       code,
			Recipient: request.Recipient,
			ExpiresAt: expire.String()[:19],
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res := models.SendCodeResponse{
		SmsId: resp.SmsId,
		Data:  respObject,
	}

	h.handleResponse(c, status_http.Created, res)

}

// Verify godoc
// @ID verify
// @Router /verify/{sms_id}/{otp} [POST]
// @Summary Verify
// @Description Verify
// @Tags register
// @Accept json
// @Produce json
// @Param sms_id path string true "sms_id"
// @Param otp path string true "otp"
// @Param verifyBody body models.Verify true "verify_body"
// @Success 201 {object} status_http.Response{data=pbAuth.V2LoginResponse} "User data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) Verify(c *gin.Context) {
	var body models.Verify

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	_, err = services.SmsService().SmsService().ConfirmOtp(
		c.Request.Context(),
		&pbSms.ConfirmOtpRequest{
			SmsId: c.Param("sms_id"),
			Otp:   c.Param("otp"),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if !body.Data.UserFound {
		h.handleResponse(c, status_http.OK, "User verified but not found")
		return
	}
	convertedToAuthPb := helper.ConvertPbToAnotherPb(body.Data)

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	_, err = h.companyServices.ServiceResource().GetSingle(
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

	//resourceEnvironment, err := h.companyServices.Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	res, err := h.authService.Session().SessionAndTokenGenerator(
		context.Background(),
		&pbAuth.SessionAndTokenRequest{
			LoginData: convertedToAuthPb,
			Tables:    body.Tables,
			ProjectId: projectId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// RegisterOtp godoc
// @ID registerOtp
// @Router /register-otp/{table_slug} [POST]
// @Summary RegisterOtp
// @Description RegisterOtp
// @Tags register
// @Accept json
// @Produce json
// @Param registerBody body models.RegisterOtp true "register_body"
// @Param table_slug path string true "table_slug"
// @Success 201 {object} status_http.Response{data=pbAuth.V2LoginResponse} "User data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) RegisterOtp(c *gin.Context) {
	var body models.RegisterOtp

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(body.Data)

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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	//resourceEnvironment, err := h.companyServices.Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

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

	resp, err := services.GetBuilderServiceByType(resource.NodeType).Login().LoginWithOtp(
		context.Background(),
		&pbObject.PhoneOtpRequst{
			PhoneNumber: body.Data["phone"].(string),
			ClientType:  "PATIENT",
			ProjectId:   resource.ResourceEnvironmentId,
		},
	)
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
			ProjectId: projectId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}
