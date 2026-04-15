package v1

import (
	"context"
	"encoding/json"
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

	plan = ensureLoginTable(plan)

	var errs []string

	for _, tablePlan := range plan.Tables {

		attributesMap := map[string]any{
			"label":    "",
			"label_en": tablePlan.Label,
		}

		if tablePlan.IsLoginTable {
			loginStrategy := tablePlan.LoginStrategy
			if len(loginStrategy) == 0 {
				loginStrategy = []string{"login"}
			}
			attributesMap["auth_info"] = map[string]any{
				"login_strategy": loginStrategy,
			}
		}
		attributes, _ := helper.ConvertMapToStruct(attributesMap)

		tableReq := &nb.CreateTableRequest{
			Label:        tablePlan.Label,
			Slug:         tablePlan.Slug,
			ProjectId:    projectId,
			EnvId:        envId,
			MenuId:       config.MainMenuID,
			ViewId:       uuid.NewString(),
			LayoutId:     uuid.NewString(),
			ShowInMenu:   true,
			Attributes:   attributes,
			IsLoginTable: tablePlan.IsLoginTable,
		}

		tableResp, err := service.GoObjectBuilderService().Table().Create(ctx, tableReq)
		if err != nil {
			errs = append(errs, fmt.Sprintf("table %s creation failed: %v", tablePlan.Slug, err))
			continue
		}

		if tablePlan.IsLoginTable {
			log.Printf("[client_type_update] ========== START client_type update for login table: %s ==========", tablePlan.Slug)
			log.Printf("[client_type_update] Using projectId=%s, envId=%s", projectId, envId)

			// Step 1: Convert getListData
			getListData, convertErr := helper.ConvertMapToStruct(
				map[string]any{
					"limit":  1,
					"offset": 0,
				},
			)
			if convertErr != nil {
				log.Printf("[client_type_update] ERROR: Step 1 — failed to convert getListData to struct: %v", convertErr)
			} else {
				log.Printf("[client_type_update] Step 1 — getListData converted successfully")
			}

			// Step 2: Call GetList2
			log.Printf("[client_type_update] Step 2 — calling GetList2 on table 'client_type' with projectId=%s envId=%s limit=1 offset=0", projectId, envId)

			clientTypeResp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(
				ctx, &nb.CommonMessage{
					TableSlug: "client_type",
					Data:      getListData,
					ProjectId: projectId,
					EnvId:     envId,
				},
			)
			if err != nil {
				log.Printf("[client_type_update] ERROR: Step 2 — GetList2 transport/gRPC error: %v", err)
			} else {
				log.Printf("[client_type_update] Step 2 — GetList2 returned without transport error")
			}

			// Step 3: Validate response
			if clientTypeResp == nil {
				log.Printf("[client_type_update] ERROR: Step 3 — clientTypeResp is nil (service returned empty response)")
				log.Printf("[client_type_update] ========== END client_type update (FAILED at Step 3) ==========")
				goto afterLoginBlock
			}

			if clientTypeResp.GetData() == nil {
				log.Printf("[client_type_update] ERROR: Step 3 — clientTypeResp.GetData() is nil (response has no data field at all)")
				log.Printf("[client_type_update] ========== END client_type update (FAILED at Step 3) ==========")
				goto afterLoginBlock
			}

			log.Printf("[client_type_update] Step 3 — response is not nil, converting to map...")

			{
				respData := clientTypeResp.GetData().AsMap()

				// Log full raw response
				rawJSON, _ := json.Marshal(respData)
				log.Printf("[client_type_update] Step 3 — full raw response: %s", string(rawJSON))

				// Step 4: Parse data array
				dataItems, ok := respData["response"].([]any)
				if !ok {
					log.Printf("[client_type_update] ERROR: Step 4 — respData[\"response\"] type assertion to []any failed — actual type: %T, value: %v", respData["response"], respData["response"])
					log.Printf("[client_type_update] ========== END client_type update (FAILED at Step 4) ==========")
					goto afterLoginBlock
				}

				log.Printf("[client_type_update] Step 4 — data array parsed, count=%d", len(dataItems))

				if len(dataItems) == 0 {
					log.Printf("[client_type_update] WARNING: Step 4 — dataItems is empty, no client_type records found")
					log.Printf("[client_type_update] Possible causes: wrong envId, client_type table is not seeded, or records filtered by env")
					log.Printf("[client_type_update] ========== END client_type update (SKIPPED — no records) ==========")
					goto afterLoginBlock
				}

				// Step 5: Parse first item
				firstItem, ok := dataItems[0].(map[string]any)
				if !ok {
					log.Printf("[client_type_update] ERROR: Step 5 — dataItems[0] is not map[string]any — actual type: %T, value: %v", dataItems[0], dataItems[0])
					log.Printf("[client_type_update] ========== END client_type update (FAILED at Step 5) ==========")
					goto afterLoginBlock
				}

				clientTypeId, hasGuid := firstItem["guid"].(string)
				if !hasGuid || clientTypeId == "" {
					log.Printf("[client_type_update] WARNING: Step 5 — firstItem has no valid 'guid' field — full item: %v", firstItem)
				} else {
					log.Printf("[client_type_update] Step 5 — first client_type record: guid=%s", clientTypeId)
				}

				// Step 6: Set table_slug and build update payload
				log.Printf("[client_type_update] Step 6 — setting table_slug='%s' on client_type guid=%s", tablePlan.Slug, clientTypeId)
				firstItem["table_slug"] = tablePlan.Slug

				payloadJSON, _ := json.Marshal(firstItem)
				log.Printf("[client_type_update] Step 6 — update payload: %s", string(payloadJSON))

				updateData, convertErr := helper.ConvertMapToStruct(firstItem)
				if convertErr != nil {
					log.Printf("[client_type_update] ERROR: Step 6 — failed to convert update payload to struct: %v", convertErr)
					log.Printf("[client_type_update] ========== END client_type update (FAILED at Step 6) ==========")
					goto afterLoginBlock
				}

				log.Printf("[client_type_update] Step 6 — payload converted to struct successfully")

				// Step 7: Call Items().Update()
				log.Printf("[client_type_update] Step 7 — calling Items().Update() on 'client_type' projectId=%s envId=%s guid=%s", projectId, envId, clientTypeId)

				_, err = service.GoObjectBuilderService().Items().Update(
					ctx, &nb.CommonMessage{
						TableSlug: "client_type",
						Data:      updateData,
						ProjectId: projectId,
						EnvId:     envId,
					},
				)
				if err != nil {
					log.Printf("[client_type_update] ERROR: Step 7 — Items().Update() failed for guid=%s: %v", clientTypeId, err)
				} else {
					log.Printf("[client_type_update] SUCCESS: Step 7 — client_type guid=%s updated with table_slug='%s'", clientTypeId, tablePlan.Slug)
				}
			}

			log.Printf("[client_type_update] ========== END client_type update ==========")
		afterLoginBlock:
		}

		tableId := tableResp.GetId()
		log.Printf("[ai_messaging_backend] Created table: %s (id: %s)", tablePlan.Slug, tableId)

		// 2. Create each Field individually
		for _, fieldPlan := range tablePlan.Fields {
			if isSystemField(fieldPlan.Slug) {
				continue
			}
			if tablePlan.IsLoginTable && isAuthField(fieldPlan.Slug) {
				continue
			}

			mappedType := mapFieldType(fieldPlan.Type)

			fieldAttrMap := map[string]any{
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
				Index:      "string",
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
		return "SINGLE_LINE"
	case "PHONE", "TEL":
		return "PHONE"
	case "INTERNATIONAL_PHONE":
		return "INTERNATION_PHONE"
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

func isAuthField(slug string) bool {
	return map[string]bool{
		"login":    true,
		"email":    true,
		"phone":    true,
		"password": true,
		"tin":      true,
	}[slug]
}

func ensureLoginTable(plan *models.ArchitectPlan) *models.ArchitectPlan {
	for _, t := range plan.Tables {
		if t.IsLoginTable {
			return plan
		}
	}

	log.Printf("[ai_messaging_backend] WARNING: no login table found in plan — injecting default users table")

	defaultUsers := models.TablePlan{
		Slug:          "users",
		Label:         "Users",
		IsLoginTable:  true,
		LoginStrategy: []string{"login"},
		Fields: []models.TableFieldPlan{
			{
				Slug:  "full_name",
				Label: "Full Name",
				Type:  "SINGLE_LINE",
			},
		},
		MockData: []map[string]any{
			{"full_name": "Admin User"},
		},
	}

	plan.Tables = append([]models.TablePlan{defaultUsers}, plan.Tables...)
	return plan
}
