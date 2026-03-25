package v1

import (
	"context"
	"log"

	"ucode/ucode_go_api_gateway/api/models"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"
)

func createBackendFromPlan(ctx context.Context, plan *models.ArchitectPlan, projectId string, service services.ServiceManagerI) error {
	log.Printf("[ai_messaging_backend] Starting backend creation from plan for project %s", projectId)

	for _, tablePlan := range plan.Tables {
		// 1. Create Table
		tableReq := &nb.CreateTableRequest{
			Label:      tablePlan.Label,
			Slug:       tablePlan.Slug,
			ProjectId:  projectId,
			ShowInMenu: true,
		}

		tableResp, err := service.GoObjectBuilderService().Table().Create(ctx, tableReq)
		if err != nil {
			log.Printf("[ai_messaging_backend] Failed to create table %s: %v", tablePlan.Slug, err)
			continue
		}
		log.Printf("[ai_messaging_backend] Created table %s", tablePlan.Slug)

		// 2. Create Fields
		for _, fieldPlan := range tablePlan.Fields {
			fieldReq := &nb.CreateFieldRequest{
				Label:     fieldPlan.Label,
				Slug:      fieldPlan.Slug,
				Type:      fieldPlan.Type,
				TableId:   tableResp.Id,
				ProjectId: projectId,
			}
			_, err = service.GoObjectBuilderService().Field().Create(ctx, fieldReq)
			if err != nil {
				log.Printf("[ai_messaging_backend] Failed to create field %s for table %s: %v", fieldPlan.Slug, tablePlan.Slug, err)
			}
		}

		// 3. Insert Mock Data
		for _, mockRow := range tablePlan.MockData {
			structData, err := helper.ConvertMapToStruct(mockRow)
			if err != nil {
				log.Printf("[ai_messaging_backend] Failed to convert mock data struct for table %s: %v", tablePlan.Slug, err)
				continue
			}

			_, err = service.GoObjectBuilderService().Items().Create(ctx, &nb.CommonMessage{
				TableSlug: tablePlan.Slug,
				ProjectId: projectId,
				Data:      structData,
			})
			if err != nil {
				log.Printf("[ai_messaging_backend] Failed to insert mock data item for table %s: %v", tablePlan.Slug, err)
			}
		}
	}

	log.Printf("[ai_messaging_backend] Successfully completed backend creation")
	return nil
}
