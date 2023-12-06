package helper

import (
	"context"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/services"
)

func GetMenuTemplateById(id string, compService services.CompanyServiceI) (*obs.MenuTemplate, error) {
	var resp *obs.MenuTemplate
	global, err := compService.Company().GetMenuTemplateById(context.Background(), &pb.GetMenuTemplateRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}
	resp = &obs.MenuTemplate{
		Id:               global.GetId(),
		Background:       global.GetBackground(),
		ActiveBackground: global.GetActiveBackground(),
		Text:             global.GetText(),
		ActiveText:       global.GetActiveText(),
		Title:            global.GetTitle(),
	}
	return resp, nil
}
