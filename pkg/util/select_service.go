package util

import (
	"ucode/ucode_go_api_gateway/services"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

func SelectService(s services.ServiceManagerI, resourceType pb.ResourceType) interface{} {

	if resourceType == 1 {
		return s.BuilderService()
	}

	return nil
}
