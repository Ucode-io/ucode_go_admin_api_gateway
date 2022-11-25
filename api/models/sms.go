package models

import (
	pb "ucode/ucode_go_api_gateway/genproto/auth_service"
	pbObject "ucode/ucode_go_api_gateway/genproto/object_builder_service"
)

type Sms struct {
	Text       string `json:"text"`
	Recipient  string `json:"recipient"`
	ClientType string `json:"client_type"`
}

type SendCodeResponse struct {
	SmsId string                    `json:"sms_id"`
	Data  *pbObject.V2LoginResponse `json:"data"`
}

type Verify struct {
	Data   *pbObject.V2LoginResponse `json:"data"`
	Tables []*pb.Object              `json:"tables"`
}

type RegisterOtp struct {
	Data map[string]interface{} `json:"data"`
}

type Email struct {
	Email      string `json:"email"`
	ClientType string `json:"client_type"`
}
