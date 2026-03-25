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
	log.Printf("[ai_messaging_backend] Starting sequential backend creation for project %s (env: %s)", projectId, envId)

	var errs []string

	for _, tablePlan := range plan.Tables {
		// 1. Create the Table metadata & physical structure
		attributesMap := map[string]interface{}{
			"label":    "",
			"label_en": tablePlan.Label,
		}
		attributes, _ := helper.ConvertMapToStruct(attributesMap)

		tableReq := &nb.CreateTableRequest{
			Label:      tablePlan.Label,
			Slug:       tablePlan.Slug,
			ProjectId:  projectId,
			EnvId:      envId,
			MenuId:     config.MainMenuID,
			ViewId:     uuid.NewString(), // Server generates if empty, but good to be explicit
			LayoutId:   uuid.NewString(),
			ShowInMenu: true,
			Attributes: attributes,
		}

		tableResp, err := service.GoObjectBuilderService().Table().Create(ctx, tableReq)
		if err != nil {
			errs = append(errs, fmt.Sprintf("table %s creation failed: %v", tablePlan.Slug, err))
			continue
		}

		tableId := tableResp.GetId()
		log.Printf("[ai_messaging_backend] Created table: %s (id: %s)", tablePlan.Slug, tableId)

		// 2. Create each Field individually (triggers DB alter, permissions, and UI placement)
		for _, fieldPlan := range tablePlan.Fields {
			if isSystemField(fieldPlan.Slug) {
				continue
			}

			mappedType := mapFieldType(fieldPlan.Type)
			
			fieldAttrMap := map[string]interface{}{
				"label":    "",
				"label_en": fieldPlan.Label,
			}
			fieldAttr, _ := helper.ConvertMapToStruct(fieldAttrMap)

			fieldReq := &nb.CreateFieldRequest{
				Id:         uuid.NewString(),
				TableId:    tableId,
				Label:      fieldPlan.Label,
				Slug:       fieldPlan.Slug,
				Type:       mappedType,
				Attributes: fieldAttr,
				ProjectId:  projectId,
				Index:      "string", // Consistent with existing patterns
				IsVisible:  true,
			}

			_, err = service.GoObjectBuilderService().Field().Create(ctx, fieldReq)
			if err != nil {
				errs = append(errs, fmt.Sprintf("field %s.%s creation failed: %v", tablePlan.Slug, fieldPlan.Slug, err))
			}
		}

		// 3. Insert Mock Data via Items service
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

func isSystemField(slug string) bool {
	systemSlugs := map[string]bool{
		"guid":       true,
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
	}
	return systemSlugs[slug]
}

func mapFieldType(aiType string) string {
	// Verified mapping against ucode_go_object_builder_service/pkg/helper/convert.go
	switch strings.ToUpper(aiType) {
	case "BOOLEAN", "SWITCH", "CHECKBOX":
		return "CHECKBOX"
	case "TEXT", "LONGTEXT", "MARKDOWN", "MULTI_LINE":
		return "MULTI_LINE"
	case "IMAGE", "PHOTO", "AVATAR":
		return "PHOTO"
	case "DATE":
		return "DATE"
	case "DATE_TIME", "DATETIME":
		return "DATE_TIME"
	case "JSON", "OBJECT", "ARRAY", "MAP":
		return "JSON"
	case "NUMBER", "INTEGER", "FLOAT", "DECIMAL", "INT":
		return "NUMBER"
	case "EMAIL":
		return "EMAIL"
	case "URL", "LINK":
		return "SINGLE_LINE" // Ucode usually uses SINGLE_LINE for URLs
	case "PHONE", "TEL":
		return "PHONE"
	case "INTERNATIONAL_PHONE":
		return "INTERNATION_PHONE" // Note: misspelled in Ucode source but used consistently
	case "PASSWORD":
		return "PASSWORD"
	case "COLOR":
		return "COLOR"
	case "UUID":
		return "UUID"
	case "PICK_LIST", "SELECT", "DROPDOWN":
		return "PICK_LIST"
	default:
		return "SINGLE_LINE"
	}
}
