package v1

import (
	"context"
	"fmt"
	"log"
	"strings"
	"ucode/ucode_go_api_gateway/config"

	"ucode/ucode_go_api_gateway/api/models"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/google/uuid"
)

func createBackendFromPlan(ctx context.Context, plan *models.ArchitectPlan, projectId, envId string, service services.ServiceManagerI) error {
	log.Printf("[ai_messaging_backend] Starting backend creation for project %s (env: %s)", projectId, envId)

	var errs []string

	for _, tablePlan := range plan.Tables {
		var fields []*nb.CreateFieldsRequest
		for _, fieldPlan := range tablePlan.Fields {
			if fieldPlan.Slug == "created_at" || fieldPlan.Slug == "updated_at" || fieldPlan.Slug == "deleted_at" || fieldPlan.Slug == "guid" {
				continue
			}

			fieldType := fieldPlan.Type
			// Map Architect types to object-builder-service types
			switch fieldType {
			case "BOOLEAN", "SWITCH":
				fieldType = "CHECKBOX"
			case "TEXT":
				fieldType = "MULTI_LINE"
			case "IMAGE":
				fieldType = "PHOTO"
			case "DATE":
				fieldType = "DATE"
			case "DATE_TIME":
				fieldType = "DATE_TIME"
			case "JSON":
				fieldType = "JSON"
			default:
				if fieldType == "" {
					fieldType = "SINGLE_LINE"
				}
			}

			attr, _ := helper.ConvertMapToStruct(nil)
			fields = append(fields, &nb.CreateFieldsRequest{
				Id:         uuid.NewString(),
				Label:      fieldPlan.Label,
				Slug:       fieldPlan.Slug,
				Type:       fieldType,
				Attributes: attr,
			})
		}

		summarySection := &nb.Section{
			Id:               uuid.NewString(),
			Label:            "Summary",
			Order:            1,
			Column:           "SINGLE",
			IsSummarySection: true,
		}

		attributesMap := map[string]interface{}{
			"label_undefined": tablePlan.Label,
		}
		attributes, _ := helper.ConvertMapToStruct(attributesMap)

		tableReq := &nb.CreateTableRequest{
			Label:             tablePlan.Label,
			Slug:              tablePlan.Slug,
			ProjectId:         projectId,
			EnvId:             envId,
			MenuId:            config.MainMenuID,
			ViewId:            uuid.NewString(),
			LayoutId:          uuid.NewString(),
			ShowInMenu:        true,
			Fields:            fields,
			Sections:          []*nb.Section{summarySection},
			Attributes:        attributes,
			SubtitleFieldSlug: "",
		}

		tableResp, err := service.GoObjectBuilderService().Table().Create(ctx, tableReq)
		if err != nil {
			errs = append(errs, fmt.Sprintf("table %s creation failed: %v", tablePlan.Slug, err))
			continue
		}

		log.Printf("[ai_messaging_backend] Created table: %s (id: %s) with %d fields", tablePlan.Slug, tableResp.GetId(), len(fields))

		for i, mockRow := range tablePlan.MockData {
			structData, err := helper.ConvertMapToStruct(mockRow)
			if err != nil {
				errs = append(errs, fmt.Sprintf("mock %s[%d] convert: %v", tablePlan.Slug, i, err))
				continue
			}

			_, err = service.GoObjectBuilderService().Items().Create(
				ctx, &nb.CommonMessage{
					TableSlug: tablePlan.Slug,
					ProjectId: projectId,
					Data:      structData,
				},
			)
			if err != nil {
				errs = append(errs, fmt.Sprintf("mock %s[%d] insert: %v", tablePlan.Slug, i, err))
			}
		}
	}

	if len(errs) > 0 {
		log.Printf("[ai_messaging_backend] Completed with %d errors", len(errs))
		return fmt.Errorf("backend creation had %d errors: %s", len(errs), strings.Join(errs, "; "))
	}

	log.Printf("[ai_messaging_backend] Successfully completed backend creation")
	return nil
}
