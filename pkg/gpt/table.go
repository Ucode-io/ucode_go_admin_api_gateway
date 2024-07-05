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

func CreateTable(reqBody *models.CreateTableAI) ([]models.CreateVersionHistoryRequest, error) {

	attributes, err := helper.ConvertMapToStruct(map[string]interface{}{
		"label":    "",
		"label_en": reqBody.Label,
	})
	if err != nil {
		return nil, err
	}

	var (
		respLogReq = []models.CreateVersionHistoryRequest{}
		resource   = reqBody.Resource
		services   = reqBody.Service
		tableMongo = &obs.CreateTableRequest{
			Label:      reqBody.Label,
			Slug:       reqBody.TableSlug,
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
			Slug:       reqBody.TableSlug,
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
		logReq = models.CreateVersionHistoryRequest{
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
			respLogReq = append(respLogReq, logReq)
			return respLogReq, err
		} else {
			tableMongo.Id = resp.Id
			logReq.Current = &tableMongo
			logReq.Response = &tableMongo
			respLogReq = append(respLogReq, logReq)
		}

		if reqBody.Menu != "c57eedc3-a954-4262-a0af-376c65b5a284" {
			menus, err := services.GetBuilderServiceByType(resource.NodeType).Menu().GetByLabel(
				context.Background(),
				&obs.MenuPrimaryKey{Label: reqBody.Menu, ProjectId: reqBody.Resource.ResourceEnvironmentId},
			)
			if err != nil {
				return respLogReq, err
			}

			for _, menu := range menus.Menus {
				var (
					mongoRequest = &obs.CreateMenuRequest{
						Label:      reqBody.Label,
						ParentId:   menu.Id,
						Type:       "TABLE",
						ProjectId:  reqBody.Resource.ResourceEnvironmentId,
						Attributes: attributes,
						TableId:    resp.Id,
					}
					logReq = models.CreateVersionHistoryRequest{
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

				resp, err := reqBody.Service.GetBuilderServiceByType(reqBody.Resource.NodeType).Menu().Create(
					context.Background(),
					mongoRequest,
				)

				if err != nil {
					logReq.Response = err.Error()
					respLogReq = append(respLogReq, logReq)
					return nil, err
				} else {
					logReq.Response = resp
					logReq.Current = resp
					respLogReq = append(respLogReq, logReq)
				}
			}
		} else {
			var (
				mongoRequest = &obs.CreateMenuRequest{
					Label:      reqBody.Label,
					ParentId:   "c57eedc3-a954-4262-a0af-376c65b5a284",
					Type:       "TABLE",
					ProjectId:  reqBody.Resource.ResourceEnvironmentId,
					Attributes: attributes,
					TableId:    resp.Id,
				}
				logReq = models.CreateVersionHistoryRequest{
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

			resp, err := reqBody.Service.GetBuilderServiceByType(reqBody.Resource.NodeType).Menu().Create(
				context.Background(),
				mongoRequest,
			)

			if err != nil {
				logReq.Response = err.Error()
				respLogReq = append(respLogReq, logReq)
				return nil, err
			} else {
				logReq.Response = resp
				logReq.Current = resp
				respLogReq = append(respLogReq, logReq)
			}
		}

	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Table().Create(
			context.Background(),
			tablePsql,
		)
		if err != nil {
			logReq.Response = err.Error()
			respLogReq = append(respLogReq, logReq)
			return respLogReq, err
		} else {
			tablePsql.Id = resp.Id
			logReq.Current = &tablePsql
			logReq.Response = &tablePsql
			respLogReq = append(respLogReq, logReq)
		}
	}

	return respLogReq, nil
}

func UpdateTable(req *models.UpdateTableAI) ([]models.CreateVersionHistoryRequest, error) {

	var (
		resource   = req.Resource
		services   = req.Service
		respLogReq = []models.CreateVersionHistoryRequest{}
	)

	attributes, err := helper.ConvertMapToStruct(map[string]interface{}{
		"label":    "",
		"label_en": req.NewLabel,
	})
	if err != nil {
		return respLogReq, err
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		tables, err := services.GetBuilderServiceByType(resource.NodeType).Table().GetTablesByLabel(context.Background(), &obs.GetTablesByLabelReq{
			ProjectId: resource.ResourceEnvironmentId,
			Label:     req.OldLabel,
		})
		if err != nil {
			return nil, err
		}

		for _, table := range tables.Tables {

			var (
				updateTable = &obs.UpdateTableRequest{}
				logReq      = models.CreateVersionHistoryRequest{
					Services:     services,
					NodeType:     resource.NodeType,
					ProjectId:    resource.ResourceEnvironmentId,
					ActionSource: "TABLE",
					ActionType:   "UPDATE TABLE",
					UserInfo:     cast.ToString(req.UserId),
					TableSlug:    table.Slug,
				}
			)

			table.Label = req.NewLabel

			err = helper.MarshalToStruct(table, &updateTable)
			if err != nil {
				return respLogReq, err
			}

			updateTable.ProjectId = resource.ResourceEnvironmentId
			updateTable.Attributes = attributes

			logReq.Request = &updateTable

			resp, err := services.GetBuilderServiceByType(resource.NodeType).Table().Update(
				context.Background(),
				updateTable,
			)
			if err != nil {
				logReq.Response = err.Error()
				respLogReq = append(respLogReq, logReq)
				return respLogReq, err
			} else {
				logReq.Response = resp
				logReq.Current = resp
				respLogReq = append(respLogReq, logReq)
			}
		}

		menus, err := services.GetBuilderServiceByType(resource.NodeType).Menu().GetByLabel(
			context.Background(),
			&obs.MenuPrimaryKey{Label: req.OldLabel, ProjectId: resource.ResourceEnvironmentId},
		)
		if err != nil {
			return respLogReq, err
		}

		for _, menu := range menus.Menus {
			var (
				updateMenu = &obs.Menu{
					ProjectId:       resource.ResourceEnvironmentId,
					Label:           req.NewLabel,
					Id:              menu.Id,
					Icon:            menu.Icon,
					TableId:         menu.TableId,
					LayoutId:        menu.LayoutId,
					ParentId:        menu.ParentId,
					Type:            menu.Type,
					MicrofrontendId: menu.MicrofrontendId,
					IsStatic:        menu.IsStatic,
				}
				logReq = models.CreateVersionHistoryRequest{
					Services:     services,
					NodeType:     resource.NodeType,
					ProjectId:    resource.ResourceEnvironmentId,
					ActionSource: "MENU",
					ActionType:   "UPDATE MENU",
					UserInfo:     cast.ToString(req.UserId),
					TableSlug:    "Menu",
					Request:      updateMenu,
				}
			)

			resp, err := services.GetBuilderServiceByType(resource.NodeType).Menu().Update(
				context.Background(),
				updateMenu,
			)
			if err != nil {
				logReq.Response = err.Error()
				respLogReq = append(respLogReq, logReq)
				return respLogReq, err
			} else {
				logReq.Response = resp
				logReq.Current = resp
				respLogReq = append(respLogReq, logReq)
			}
		}

	case pb.ResourceType_POSTGRESQL:

		tables, err := services.GoObjectBuilderService().Table().GetTablesByLabel(context.Background(), &nb.GetTablesByLabelReq{
			ProjectId: resource.ResourceEnvironmentId,
			Label:     req.OldLabel,
		})

		for _, table := range tables.Tables {
			var (
				updateTable = &nb.UpdateTableRequest{}
				logReq      = models.CreateVersionHistoryRequest{
					Services:     services,
					NodeType:     resource.NodeType,
					ProjectId:    resource.ResourceEnvironmentId,
					ActionSource: "TABLE",
					ActionType:   "UPDATE TABLE",
					UserInfo:     cast.ToString(req.UserId),
					TableSlug:    table.Slug,
				}
			)

			table.Label = req.NewLabel

			err = helper.MarshalToStruct(table, &updateTable)
			if err != nil {
				return respLogReq, err
			}

			updateTable.ProjectId = resource.ResourceEnvironmentId
			updateTable.Attributes = attributes

			logReq.Request = &updateTable

			resp, err := services.GoObjectBuilderService().Table().Update(
				context.Background(),
				updateTable,
			)
			if err != nil {
				logReq.Response = err.Error()
				respLogReq = append(respLogReq, logReq)
				return respLogReq, err
			} else {
				logReq.Response = resp
				logReq.Current = resp
				respLogReq = append(respLogReq, logReq)
			}
		}
	}

	return respLogReq, nil
}
