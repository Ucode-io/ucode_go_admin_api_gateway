package tests

import (
	"encoding/json"
	"testing"

	sdk "github.com/golanguzb70/ucode-sdk"
	"github.com/google/uuid"
	"github.com/manveru/faker"
	"github.com/stretchr/testify/assert"
)

func TestEmailRegister(t *testing.T) {
	faker1, err := faker.New("en")
	assert.NoError(t, err, "faker error")

	body, err := UcodeApi.DoRequest(BaseUrlAuth+"/v2/register?project-id="+ProjectId, "POST",
		map[string]interface{}{
			"data": map[string]interface{}{"type": "email", "client_type_id": ClientTypeId, "role_id": RoleId, "email": Email, "name": faker1.Name()},
		},
		map[string]string{"Resource-Id": ResourceId, "Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	var registerResponse RegisterResponse
	err = json.Unmarshal(body, &registerResponse)
	assert.NoError(t, err)

	userId := registerResponse.Data.UserID

	body, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/send-code", "POST", map[string]interface{}{
		"recipient": Email,
		"text":      "This is your code",
		"type":      "EMAIL",
	}, map[string]string{"Resource-Id": ResourceId, "Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	var smsResponse SmsResponse
	err = json.Unmarshal(body, &smsResponse)
	assert.NoError(t, err)

	smsId := smsResponse.Data.SmsID

	_, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/login/with-option?project-id="+ProjectId, "POST", map[string]interface{}{
		"data": map[string]interface{}{
			"sms_id":         smsId,
			"otp":            "111111",
			"email":          Email,
			"client_type_id": ClientTypeId,
			"role_id":        RoleId,
		},
		"login_strategy": "EMAIL_OTP",
	}, map[string]string{"Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	_, err = UcodeApi.Delete(&sdk.Argument{
		TableSlug: "user_email",
		Request: sdk.Request{Data: map[string]interface{}{
			"guid": userId,
		}},
	})
	assert.NoError(t, err)
}

func TestPhoneRegister(t *testing.T) {
	faker1, err := faker.New("en")
	assert.NoError(t, err, "faker error")
	phone := "+998946236953"

	body, err := UcodeApi.DoRequest(BaseUrlAuth+"/v2/register?project-id="+ProjectId, "POST",
		map[string]interface{}{
			"data": map[string]interface{}{"type": "phone", "client_type_id": ClientTypeId, "role_id": RoleId, "phone": phone, "name": faker1.Name()},
		},
		map[string]string{"Resource-Id": ResourceId, "Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	var registerResponse RegisterResponse
	err = json.Unmarshal(body, &registerResponse)
	assert.NoError(t, err)

	userId := registerResponse.Data.UserID

	body, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/send-code", "POST", map[string]interface{}{
		"recipient": phone,
		"text":      "This is your code",
		"type":      "PHONE",
	}, map[string]string{"Resource-Id": ResourceId, "Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	var smsResponse SmsResponse
	err = json.Unmarshal(body, &smsResponse)
	assert.NoError(t, err)

	smsId := smsResponse.Data.SmsID

	_, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/login/with-option?project-id="+ProjectId, "POST", map[string]interface{}{
		"data": map[string]interface{}{
			"sms_id":         smsId,
			"otp":            "111111",
			"phone":          phone,
			"client_type_id": ClientTypeId,
			"role_id":        RoleId,
		},
		"login_strategy": "PHONE_OTP",
	}, map[string]string{"Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	_, err = UcodeApi.Delete(&sdk.Argument{
		TableSlug: "user_email",
		Request: sdk.Request{Data: map[string]interface{}{
			"guid": userId,
		}},
	})
	assert.NoError(t, err)
}

func TestLoginRegister(t *testing.T) {
	var (
		guid  = uuid.New().String()
		login = "test_login123"
	)

	_, _, err := UcodeApi.CreateObject(&sdk.Argument{
		TableSlug: "employee",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid":           guid,
				"login":          login,
				"password":       login,
				"email":          Email,
				"role_id":        EmployeeRoleId,
				"client_type_id": EmployeeClientTypeId,
			},
		},
		DisableFaas: true,
	})
	assert.NoError(t, err)

	_, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/login/with-option?project-id="+ProjectId, "POST", map[string]interface{}{
		"data": map[string]interface{}{
			"username":       login,
			"password":       login,
			"client_type_id": EmployeeClientTypeId,
			"role_id":        EmployeeRoleId,
		},
		"login_strategy": "LOGIN_PWD",
	}, map[string]string{"Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	userResp, _, err := UcodeApi.GetSingleSlim(&sdk.Argument{
		TableSlug:   "employee",
		DisableFaas: true,
		Request: sdk.Request{
			Data: map[string]interface{}{"guid": guid},
		},
	})
	assert.NoError(t, err)

	_, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/reset-password", "PUT", map[string]interface{}{
		"password": "12345678",
		"user_id":  userResp.Data.Data.Response["user_id_auth"],
	}, map[string]string{})
	assert.NoError(t, err)

	_, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/login/with-option?project-id="+ProjectId, "POST", map[string]interface{}{
		"data": map[string]interface{}{
			"username":       login,
			"password":       "12345678",
			"client_type_id": EmployeeClientTypeId,
			"role_id":        EmployeeRoleId,
		},
		"login_strategy": "LOGIN_PWD",
	}, map[string]string{"Environment-Id": EnvironmentId, "X-API-KEY": UcodeApi.Config().AppId},
	)
	assert.NoError(t, err)

	_, err = UcodeApi.DoRequest(BaseUrlAuth+"/v2/forgot-password", "POST", map[string]interface{}{"login": login}, map[string]string{})
	assert.NoError(t, err)

	_, err = UcodeApi.Delete(&sdk.Argument{
		TableSlug: "employee",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid": guid,
			},
		},
	})
	assert.NoError(t, err)
}
