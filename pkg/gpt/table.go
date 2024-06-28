package gpt

import (
	"context"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func CreateTable(reqBody *models.CreateTableAI) (*models.CreateVersionHistoryRequest, error) {

	attributes, err := helper.ConvertMapToStruct(map[string]interface{}{
		"label":    "",
		"label_en": reqBody.Label,
	})
	if err != nil {
		return nil, err
	}

	var (
		resource   = reqBody.Resource
		services   = reqBody.Service
		tableMongo = &obs.CreateTableRequest{
			Label:      reqBody.Label,
			Slug:       reqBody.Label,
			ShowInMenu: true,
			Name:       fmt.Sprintf("Auto Created Commit Create table - %s", time.Now().Format(time.RFC1123)),
			CommitType: config.COMMIT_TYPE_TABLE,
			OrderBy:    false,
			Attributes: attributes,
			EnvId:      reqBody.EnvironmentId,
			ViewId:     uuid.NewString(),
			LayoutId:   uuid.NewString(),
			ProjectId:  resource.ResourceEnvironmentId,
		}
		tablePsql = &nb.CreateTableRequest{
			Label:      reqBody.Label,
			Slug:       reqBody.Label,
			ShowInMenu: true,
			Name:       fmt.Sprintf("Auto Created Commit Create table - %s", time.Now().Format(time.RFC1123)),
			CommitType: config.COMMIT_TYPE_TABLE,
			OrderBy:    false,
			Attributes: attributes,
			EnvId:      reqBody.EnvironmentId,
			ViewId:     uuid.NewString(),
			LayoutId:   uuid.NewString(),
			ProjectId:  resource.ResourceEnvironmentId,
		}
		logReq = &models.CreateVersionHistoryRequest{
			Services:     reqBody.Service,
			NodeType:     reqBody.Resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "TABLE",
			ActionType:   "CREATE TABLE",
			UserInfo:     cast.ToString(reqBody.UserId),
			Request:      tableMongo,
			TableSlug:    reqBody.TableSlug,
		}
	)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(reqBody.Resource.NodeType).Table().Create(
			context.Background(),
			tableMongo,
		)
		if err != nil {
			logReq.Response = err.Error()
			return logReq, err
		} else {
			tableMongo.Id = resp.Id
			logReq.Current = &tableMongo
			logReq.Response = &tableMongo
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Table().Create(
			context.Background(),
			tablePsql,
		)
		if err != nil {
			logReq.Response = err.Error()
			return logReq, err
		} else {
			tablePsql.Id = resp.Id
			logReq.Current = &tablePsql
			logReq.Response = &tablePsql
		}
	}

	return logReq, nil
}
