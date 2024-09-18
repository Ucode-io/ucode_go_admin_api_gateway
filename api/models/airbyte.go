package models

import "ucode/ucode_go_api_gateway/genproto/company_service"

type AirByteRequest struct {
	Data company_service.GetListAirbyteRequest `json:"data"`
}
