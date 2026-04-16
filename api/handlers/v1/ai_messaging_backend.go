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
	log.Printf("[ai_messaging_backend] Starting sequential backend creation for project %s (env: %s)", resourceEnvId, envId)

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
			log.Printf("[client_type_update] ========== START client_type update for login table: %s ==========", tablePlan.Slug)
			log.Printf("[client_type_update] Using resourceEnvId=%s, envId=%s", resourceEnvId, envId)

			// Step 1: Build getListData for client_type query
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

			// Step 2: Fetch client_type list
			log.Printf("[client_type_update] Step 2 — calling GetList2 on table 'client_type' with resourceEnvId=%s envId=%s limit=1 offset=0", resourceEnvId, envId)

			clientTypeResp, err := service.GoObjectBuilderService().ObjectBuilder().GetList2(
				ctx, &nb.CommonMessage{
					TableSlug: "client_type",
					Data:      getListData,
					ProjectId: resourceEnvId,
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

				rawJSON, _ := json.Marshal(respData)
				log.Printf("[client_type_update] Step 3 — full raw response: %s", string(rawJSON))

				// Step 4: Parse response array
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

				// Step 5: Extract first client_type record
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

				// Step 6: Set table_slug on client_type record to point to new login table
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

				// Step 7: Update client_type record
				log.Printf("[client_type_update] Step 7 — calling Items().Update() on 'client_type' resourceEnvId=%s envId=%s guid=%s", resourceEnvId, envId, clientTypeId)

				_, err = service.GoObjectBuilderService().Items().Update(
					ctx, &nb.CommonMessage{
						TableSlug: "client_type",
						Data:      updateData,
						ProjectId: resourceEnvId,
						EnvId:     envId,
					},
				)
				if err != nil {
					log.Printf("[client_type_update] ERROR: Step 7 — Items().Update() failed for guid=%s: %v", clientTypeId, err)
					log.Printf("[client_type_update] ========== END client_type update (FAILED at Step 7) ==========")
					// client_type not updated — no point migrating users, auth will still be broken
					goto afterLoginBlock
				}

				log.Printf("[client_type_update] SUCCESS: Step 7 — client_type guid=%s updated with table_slug='%s'", clientTypeId, tablePlan.Slug)

				// ──────────────────────────────────────────────────────────────────────
				// Step 8: Migrate existing users from "user" table to the new login table.
				//
				// Problem: Before this flow, project creator was written to the "user" table.
				// Now that client_type.table_slug points to the new login table, auth reads
				// from the new table and the old user is invisible to the login system.
				//
				// Fix: fetch all records from "user" table, re-create each one on the new
				// login table using from_auth_service=true so that:
				//   • SyncUserService().CreateUser() is NOT called (user already exists in auth)
				//   • InsertPersonTable() IS called (syncs person table for auth lookups)
				//   • A row IS inserted into the new login table (so the user is visible in UI)
				//
				// already_hashed=true prevents double-hashing the stored bcrypt password.
				// ──────────────────────────────────────────────────────────────────────
				log.Printf("[migrate_user] ========== START user migration to login table '%s' ==========", tablePlan.Slug)

				getUserListData, getUserConvertErr := helper.ConvertMapToStruct(map[string]any{
					"limit":  50,
					"offset": 0,
				})
				if getUserConvertErr != nil {
					log.Printf("[migrate_user] ERROR: Step 8 — failed to build user list request: %v", getUserConvertErr)
					goto afterLoginBlock
				}

				userListResp, userListErr := service.GoObjectBuilderService().ObjectBuilder().GetList2(
					ctx, &nb.CommonMessage{
						TableSlug: "user",
						Data:      getUserListData,
						ProjectId: resourceEnvId,
						EnvId:     envId,
					},
				)
				if userListErr != nil {
					log.Printf("[migrate_user] ERROR: Step 8 — GetList2 on 'user' table failed: %v", userListErr)
					goto afterLoginBlock
				}

				if userListResp == nil || userListResp.GetData() == nil {
					log.Printf("[migrate_user] WARNING: Step 8 — empty or nil response from 'user' table, skipping migration")
					goto afterLoginBlock
				}

				{
					userRespData := userListResp.GetData().AsMap()

					rawUsersJSON, _ := json.Marshal(userRespData)
					log.Printf("[migrate_user] Step 8 — raw 'user' table response: %s", string(rawUsersJSON))

					userItems, usersOk := userRespData["response"].([]any)
					if !usersOk || len(userItems) == 0 {
						log.Printf("[migrate_user] WARNING: Step 8 — no records found in 'user' table, nothing to migrate")
						goto afterLoginBlock
					}

					log.Printf("[migrate_user] Step 8 — found %d user(s) to migrate from 'user' table to '%s'", len(userItems), tablePlan.Slug)

					// Step 9: Re-create each user on the new login table
					for idx, item := range userItems {
						existingUser, userOk := item.(map[string]any)
						if !userOk {
							log.Printf("[migrate_user] WARNING: Step 9 — user[%d] is not a map[string]any (type=%T), skipping", idx, item)
							continue
						}

						existingUserGuid := fmt.Sprintf("%v", existingUser["guid"])
						existingUserIdAuth := fmt.Sprintf("%v", existingUser["user_id_auth"])

						log.Printf("[migrate_user] Step 9 — migrating user[%d]: guid=%s user_id_auth=%s to login table '%s'",
							idx, existingUserGuid, existingUserIdAuth, tablePlan.Slug)

						// Build migration payload.
						// Keys follow the Items.Create from_auth_service contract:
						//   "login", "password", "email", "phone" are read from body and then
						//   remapped to authInfo.Login / authInfo.Password / etc. field slugs.
						//   "role_id" and "client_type_id" are also remapped via authInfo.
						//   "guid" and "user_id_auth" go directly into InsertPersonTable.
						//   "name" and "photo" populate FullName / Image in InsertPersonTable.
						//
						// Note: phone field in "user" table is "phone_number" (matches
						//   CreateUser call: Phone: cast.ToString(data["phone_number"])).
						migratePayload := map[string]any{
							// Flags
							"from_auth_service": true,
							"already_hashed":    true,
							// Identity
							"guid":         existingUserGuid,
							"user_id_auth": existingUserIdAuth,
							// Auth credentials (remapped to authInfo slugs inside Items.Create)
							"login":    existingUser["login"],
							"password": existingUser["password"], // stored bcrypt hash, not re-hashed
							"email":    existingUser["email"],
							"phone":    existingUser["phone_number"], // "user" table uses phone_number
							// Auth roles (remapped to authInfo.RoleID / authInfo.ClientTypeID slugs)
							"role_id":        existingUser["role_id"],
							"client_type_id": clientTypeId, // the just-updated client_type
							// Extra profile fields passed to InsertPersonTable
							"name":  existingUser["full_name"],
							"photo": existingUser["photo"],
						}

						rawMigrateJSON, _ := json.Marshal(migratePayload)
						log.Printf("[migrate_user] Step 9 — migrate payload for user[%d]: %s", idx, string(rawMigrateJSON))

						migrateData, migrateConvertErr := helper.ConvertMapToStruct(migratePayload)
						if migrateConvertErr != nil {
							log.Printf("[migrate_user] ERROR: Step 9 — failed to convert payload for user guid=%s: %v",
								existingUserGuid, migrateConvertErr)
							errs = append(errs, fmt.Sprintf("user migration %s convert: %v", existingUserGuid, migrateConvertErr))
							continue
						}

						_, migrateErr := service.GoObjectBuilderService().Items().Create(
							ctx, &nb.CommonMessage{
								TableSlug: tablePlan.Slug,
								Data:      migrateData,
								ProjectId: resourceEnvId,
								EnvId:     envId,
							},
						)
						if migrateErr != nil {
							log.Printf("[migrate_user] ERROR: Step 9 — Items().Create() failed for user guid=%s on table '%s': %v",
								existingUserGuid, tablePlan.Slug, migrateErr)
							errs = append(errs, fmt.Sprintf("user migration %s insert: %v", existingUserGuid, migrateErr))
						} else {
							log.Printf("[migrate_user] SUCCESS: Step 9 — user guid=%s (user_id_auth=%s) migrated to login table '%s'",
								existingUserGuid, existingUserIdAuth, tablePlan.Slug)
						}
					}
				}

				log.Printf("[migrate_user] ========== END user migration ==========")
			}

			log.Printf("[client_type_update] ========== END client_type update ==========")
		afterLoginBlock:
		}

		tableId := tableResp.GetId()
		log.Printf("[ai_messaging_backend] Created table: %s (id: %s)", tablePlan.Slug, tableId)

		// Create each Field individually
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
				ProjectId:  resourceEnvId,
				Index:      "string",
				IsVisible:  true,
			}

			_, err = service.GoObjectBuilderService().Field().Create(ctx, fieldReq)
			if err != nil {
				errs = append(errs, fmt.Sprintf("field %s.%s creation failed: %v", tablePlan.Slug, fieldPlan.Slug, err))
			}
		}

		// Insert Mock Data via Items service
		for i, mockRow := range tablePlan.MockData {
			structData, err := helper.ConvertMapToStruct(mockRow)
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
