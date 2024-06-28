package gpt

import (
	"context"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/spf13/cast"
)

func CreateMenu(reqBody *models.CreateMenuAI) (*models.CreateVersionHistoryRequest, error) {

	attributes, err := helper.ConvertMapToStruct(map[string]interface{}{
		"label":    "",
		"label_en": reqBody.Label,
	})
	if err != nil {
		return nil, err
	}

	var (
		mongoRequest = &obs.CreateMenuRequest{
			Label:      reqBody.Label,
			ParentId:   "c57eedc3-a954-4262-a0af-376c65b5a284",
			Type:       "FOLDER",
			ProjectId:  reqBody.Resource.ResourceEnvironmentId,
			Attributes: attributes,
		}
		psqlRequest = &nb.CreateMenuRequest{
			Label:      reqBody.Label,
			ParentId:   "c57eedc3-a954-4262-a0af-376c65b5a284",
			Type:       "FOLDER",
			ProjectId:  reqBody.Resource.ResourceEnvironmentId,
			Attributes: attributes,
		}
		logReq = &models.CreateVersionHistoryRequest{
			Services:     reqBody.Service,
			NodeType:     reqBody.Resource.NodeType,
			ProjectId:    reqBody.Resource.ResourceEnvironmentId,
			ActionSource: "MENU",
			ActionType:   "CREATE MENU",
			UserInfo:     cast.ToString(reqBody.UserId),
			Request:      &mongoRequest,
			TableSlug:    "Menu",
		}
	)

	// ? there in each ResourceType should write to db or redis
	switch reqBody.Resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := reqBody.Service.GetBuilderServiceByType(reqBody.Resource.NodeType).Menu().Create(
			context.Background(),
			mongoRequest,
		)

		if err != nil {
			logReq.Response = err.Error()
			return nil, err
		} else {
			logReq.Response = resp
			logReq.Current = resp
		}

	case pb.ResourceType_POSTGRESQL:
		resp, err := reqBody.Service.GoObjectBuilderService().Menu().Create(
			context.Background(),
			psqlRequest,
		)

		if err != nil {
			logReq.Response = err.Error()
			return nil, err
		} else {
			logReq.Response = resp
			logReq.Current = resp
		}
	}

	return logReq, nil
}
