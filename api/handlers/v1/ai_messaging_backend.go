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

func createBackendFromPlan(ctx context.Context, plan *models.ArchitectPlan, resourceEnvId, projectId, userId, envId string, service services.ServiceManagerI) error {
	log.Printf("[backend] creating tables for project %s", resourceEnvId)

	plan = ensureLoginTable(plan)

	var errs []string

	for _, tablePlan := range plan.Tables {

		attributesMap := map[string]any{
			"label":    "",
			"label_en": tablePlan.Label,
		}

		if tablePlan.IsLoginTable {
			attributesMap["auth_info"] = map[string]any{
				"login_strategy": []string{"login"},
			}
		}
		attributes, _ := helper.ConvertMapToStruct(attributesMap)

		tableReq := &nb.CreateTableRequest{
			Label:        tablePlan.Label,
			Slug:         tablePlan.Slug,
			ProjectId:    resourceEnvId,
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
			getListData, convertErr := helper.ConvertMapToStruct(map[string]any{"limit": 1, "offset": 0})
			if convertErr != nil {
				log.Printf("[backend] failed to build client_type query: %v", convertErr)
				goto afterLoginBlock
			}

			clientTypeResp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(
				ctx, &nb.CommonMessage{
					TableSlug: "client_type",
					Data:      getListData,
					ProjectId: resourceEnvId,
					EnvId:     envId,
				},
			)
			if err != nil {
				log.Printf("[backend] client_type GetList2 failed: %v", err)
				goto afterLoginBlock
			}

			if clientTypeResp == nil || clientTypeResp.GetData() == nil {
				log.Printf("[backend] client_type response is nil")
				goto afterLoginBlock
			}

			{
				respData := clientTypeResp.GetData().AsMap()
				dataItems, ok := respData["response"].([]any)
				if !ok || len(dataItems) == 0 {
					log.Printf("[backend] no client_type records found (env=%s)", envId)
					goto afterLoginBlock
				}

				firstItem, ok := dataItems[0].(map[string]any)
				if !ok {
					log.Printf("[backend] client_type record is invalid type")
					goto afterLoginBlock
				}

				clientTypeId, _ := firstItem["guid"].(string)
				firstItem["table_slug"] = tablePlan.Slug

				updateData, convertErr := helper.ConvertMapToStruct(firstItem)
				if convertErr != nil {
					log.Printf("[backend] failed to convert client_type update payload: %v", convertErr)
					goto afterLoginBlock
				}

				_, err = service.GoObjectBuilderService().Items().Update(
					ctx, &nb.CommonMessage{
						TableSlug: "client_type",
						Data:      updateData,
						ProjectId: resourceEnvId,
						EnvId:     envId,
					},
				)
				if err != nil {
					log.Printf("[backend] client_type update failed (guid=%s): %v", clientTypeId, err)
					goto afterLoginBlock
				}

				log.Printf("[backend] client_type updated → table_slug=%s", tablePlan.Slug)

				// Migrate existing users from "user" table → new login table.
				// The project creator is in "user" (system table) at signup; after
				// client_type.table_slug points to the new login table the auth
				// service reads from the new table and the original user becomes invisible.
				getUserListData, err := helper.ConvertMapToStruct(map[string]any{"limit": 50, "offset": 0})
				if err != nil {
					log.Printf("[backend] failed to build user list request: %v", err)
					goto afterLoginBlock
				}

				userListResp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(
					ctx, &nb.CommonMessage{
						TableSlug: "user",
						Data:      getUserListData,
						ProjectId: resourceEnvId,
						EnvId:     envId,
					},
				)
				if err != nil {
					log.Printf("[backend] failed to fetch users for migration: %v", err)
					goto afterLoginBlock
				}

				if userListResp == nil || userListResp.GetData() == nil {
					goto afterLoginBlock
				}

				{
					userRespData := userListResp.GetData().AsMap()
					userItems, ok := userRespData["response"].([]any)
					if !ok || len(userItems) == 0 {
						goto afterLoginBlock
					}

					log.Printf("[backend] migrating %d user(s) to login table %s", len(userItems), tablePlan.Slug)

					for idx, item := range userItems {
						existingUser, ok := item.(map[string]any)
						if !ok {
							continue
						}

						existingUserGuid := fmt.Sprintf("%v", existingUser["guid"])
						existingUserIdAuth := fmt.Sprintf("%v", existingUser["user_id_auth"])

						existingPassword := safeString(existingUser["password"])
						if existingPassword == "" {
							// Sentinel: satisfies InsertPersonTable length check.
							// Auth record already exists (user_id_auth is set) so the
							// placeholder is never used for actual authentication.
							existingPassword = "$2a$10$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
						}

						migratePayload := map[string]any{
							"from_auth_service": true,
							"already_hashed":    true,
							"guid":              existingUserGuid,
							"user_id_auth":      existingUserIdAuth,
							"login":             existingUser["login"],
							"password":          existingPassword,
							"email":             existingUser["email"],
							"phone":             existingUser["phone_number"],
							"role_id":           existingUser["role_id"],
							"client_type_id":    clientTypeId,
							"name":              existingUser["full_name"],
							"photo":             existingUser["photo"],
						}

						migrateData, err := helper.ConvertMapToStruct(migratePayload)
						if err != nil {
							errs = append(errs, fmt.Sprintf("user migration %s convert: %v", existingUserGuid, err))
							continue
						}

						_, err = service.GoObjectBuilderService().Items().Create(
							ctx, &nb.CommonMessage{
								TableSlug: tablePlan.Slug,
								Data:      migrateData,
								ProjectId: resourceEnvId,
								EnvId:     envId,
							},
						)
						if err != nil {
							log.Printf("[backend] user migration failed (idx=%d guid=%s): %v", idx, existingUserGuid, err)
							errs = append(errs, fmt.Sprintf("user migration %s: %v", existingUserGuid, err))
						}
					}
				}
			}
		}

	afterLoginBlock:

		tableId := tableResp.GetId()
		log.Printf("[backend] table created: %s (id=%s)", tablePlan.Slug, tableId)

		for _, fieldPlan := range tablePlan.Fields {
			if isSystemField(fieldPlan.Slug) {
				continue
			}
			if tablePlan.IsLoginTable && isAuthField(fieldPlan.Slug) {
				continue
			}

			mappedType := mapFieldType(fieldPlan.Type)

			fieldAttr, _ := helper.ConvertMapToStruct(map[string]any{
				"label":    "",
				"label_en": fieldPlan.Label,
			})

			fieldReq := &nb.CreateFieldRequest{
				Id:         uuid.NewString(),
				TableId:    tableId,
				Label:      fieldPlan.Label,
				Slug:       fieldPlan.Slug,
				Type:       mappedType,
				Attributes: fieldAttr,
				ProjectId:  resourceEnvId,
				Index:      "string",
				IsVisible:  true,
			}

			_, err = service.GoObjectBuilderService().Field().Create(ctx, fieldReq)
			if err != nil {
				errs = append(errs, fmt.Sprintf("field %s.%s: %v", tablePlan.Slug, fieldPlan.Slug, err))
			}
		}

		if tablePlan.IsLoginTable {
			if len(tablePlan.MockData) > 0 {
				log.Printf("[backend] skipping %d mock rows for login table %s", len(tablePlan.MockData), tablePlan.Slug)
			}
		} else {
			for i, mockRow := range tablePlan.MockData {
				sanitized := sanitizeMockRow(mockRow, tablePlan.Fields)
				structData, err := helper.ConvertMapToStruct(sanitized)
				if err != nil {
					errs = append(errs, fmt.Sprintf("mock %s[%d] convert: %v", tablePlan.Slug, i, err))
					continue
				}

				_, err = service.GoObjectBuilderService().Items().Create(
					ctx, &nb.CommonMessage{
						TableSlug: tablePlan.Slug,
						ProjectId: resourceEnvId,
						Data:      structData,
					},
				)
				if err != nil {
					errs = append(errs, fmt.Sprintf("mock %s[%d]: %v", tablePlan.Slug, i, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		log.Printf("[backend] completed with %d errors", len(errs))
		return fmt.Errorf("backend creation had %d errors: %s", len(errs), strings.Join(errs, "; "))
	}

	log.Printf("[backend] all tables created successfully")
	return nil
}

// sanitizeMockRow coerces mock data values to match their field types.
// Claude often generates numeric values (JSON numbers) for varchar/text fields
// which causes pgx "cannot find encode plan for varchar" errors.
func sanitizeMockRow(row map[string]any, fields []models.TableFieldPlan) map[string]any {
	fieldTypes := make(map[string]string, len(fields))
	for _, f := range fields {
		fieldTypes[f.Slug] = mapFieldType(f.Type)
	}
	out := make(map[string]any, len(row))
	for k, v := range row {
		out[k] = coerceMockValue(v, fieldTypes[k])
	}
	return out
}

// coerceMockValue converts v to the right Go type for the given mapped field type.
// JSON numbers come in as float64; varchar fields need string.
func coerceMockValue(v any, mappedType string) any {
	if v == nil {
		return v
	}
	switch mappedType {
	case "SINGLE_LINE", "MULTI_LINE", "EMAIL", "PHONE", "INTERNATION_PHONE",
		"PASSWORD", "COLOR", "PICK_LIST", "UUID", "DATE", "DATE_TIME":
		if f, ok := v.(float64); ok {
			if f == float64(int64(f)) {
				return fmt.Sprintf("%d", int64(f))
			}
			return fmt.Sprintf("%g", f)
		}
	}
	return v
}

func safeString(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func isSystemField(slug string) bool {
	return map[string]bool{
		"guid": true, "created_at": true, "updated_at": true, "deleted_at": true,
	}[slug]
}

func isAuthField(slug string) bool {
	return map[string]bool{
		"login": true, "email": true, "phone": true, "password": true, "tin": true,
	}[slug]
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

func ensureLoginTable(plan *models.ArchitectPlan) *models.ArchitectPlan {
	for _, t := range plan.Tables {
		if t.IsLoginTable {
			return plan
		}
	}

	log.Printf("[backend] WARNING: no login table in plan — injecting default users table")

	defaultUsers := models.TablePlan{
		Slug:          "users",
		Label:         "Users",
		IsLoginTable:  true,
		LoginStrategy: []string{"login"},
		Fields:        []models.TableFieldPlan{{Slug: "full_name", Label: "Full Name", Type: "SINGLE_LINE"}},
		MockData:      nil,
	}

	plan.Tables = append([]models.TablePlan{defaultUsers}, plan.Tables...)
	return plan
}

// unused — kept for reference by older callers in mcp.go path
var _ = json.Marshal
