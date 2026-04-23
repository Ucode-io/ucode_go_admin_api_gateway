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
					goto afterLoginBlock
				}

				log.Printf("[client_type_update] SUCCESS: Step 7 — client_type guid=%s updated with table_slug='%s'", clientTypeId, tablePlan.Slug)

				// ──────────────────────────────────────────────────────────────────────
				// Step 8-9: Migrate existing users from "user" table → new login table.
				//
				// Why: project creator was inserted into "user" (system table) at signup.
				// After client_type.table_slug now points to the new login table, the auth
				// service reads from the new table and the original user becomes invisible.
				//
				// Strategy: re-create each user row on the new table using
				//   from_auth_service=true  → skips SyncUserService.CreateUser()
				//                              (auth record already exists)
				//   already_hashed=true     → skips bcrypt hashing of stored hash
				//
				// Root cause of the original error:
				//   body["password"] was null in the DB (admin user created without one).
				//   Items.Create reads body["password"] → empty string → passes it to
				//   InsertPersonTable → auth service rejects: "must be at least 6 chars".
				//
				// Fix applied here: we detect a null/empty password and skip InsertPersonTable
				// password requirement by passing a dummy non-empty placeholder. The user's
				// actual auth record already exists in the auth service (user_id_auth is set),
				// so the placeholder is never used for real authentication.
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

						// ── PASSWORD HANDLING ──────────────────────────────────────────────
						// The original error:
						//   "password field must be at least 6 characters long"
						//
						// Cause: the admin user in "user" table has password=null (the user
						// was created by the auth system, password stored only in auth service,
						// not in the object builder DB).
						//
						// Items.Create with from_auth_service=true still calls InsertPersonTable,
						// which forwards the password to the auth service for person record sync.
						// Passing an empty string fails the auth service's length validation.
						//
						// Fix: we check if password exists and is non-empty.
						// If it's null/empty, we use a sentinel value that satisfies the length
						// check. Since this user's auth record already exists (user_id_auth is
						// set), they log in via user_id_auth lookup — the password field in
						// the person table is never used for actual authentication here.
						// ──────────────────────────────────────────────────────────────────
						existingPassword := safeString(existingUser["password"])
						if existingPassword == "" {
							// Sentinel: a valid-looking bcrypt hash placeholder (60 chars, correct
							// prefix). This satisfies InsertPersonTable's length check without
							// granting any real login ability via password.
							existingPassword = "$2a$10$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
							log.Printf("[migrate_user] INFO: Step 9 — user[%d] guid=%s has no password in DB (expected for auth-created users), using sentinel",
								idx, existingUserGuid)
						}

						// ── PAYLOAD ────────────────────────────────────────────────────────
						// Items.Create with from_auth_service=true remaps keys as follows:
						//   body["login"]          → body[authInfo.Login]
						//   body["password"]       → body[authInfo.Password]  (kept as-is when already_hashed=true)
						//   body["email"]          → body[authInfo.Email]
						//   body["phone"]          → body[authInfo.Phone]
						//   body["role_id"]        → body[authInfo.RoleID]
						//   body["client_type_id"] → body[authInfo.ClientTypeID]
						//
						// The "user" system table stores phone as "phone_number", not "phone".
						// ──────────────────────────────────────────────────────────────────
						migratePayload := map[string]any{
							// Control flags — must be bool, not string
							"from_auth_service": true,
							"already_hashed":    true,

							// Identity — used verbatim by InsertPersonTable
							"guid":         existingUserGuid,
							"user_id_auth": existingUserIdAuth,

							// Auth credentials — remapped inside Items.Create via authInfo slugs.
							// "password" is the stored bcrypt hash (or sentinel if null).
							"login":    existingUser["login"],
							"password": existingPassword,
							"email":    existingUser["email"],
							"phone":    existingUser["phone_number"], // "user" table field name

							// Roles — remapped to authInfo.RoleID / authInfo.ClientTypeID
							"role_id":        existingUser["role_id"],
							"client_type_id": clientTypeId, // just-updated client_type guid

							// Profile — forwarded to InsertPersonTable as FullName / Image
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
			// Auth fields (login, email, phone, password, tin) are managed by the auth
			// system on login tables — creating them as custom fields causes duplicate
			// column errors and breaks auth validation.
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

		// Insert Mock Data via Items service.
		// Login tables are skipped: mock rows don't carry auth credentials
		// (role_id, client_type_id, login/password) and always fail auth validation.
		// Real users for login tables are migrated from "user" table in Steps 8-9 above.
		if tablePlan.IsLoginTable {
			if len(tablePlan.MockData) > 0 {
				log.Printf("[ai_messaging_backend] Skipping %d mock data row(s) for login table '%s' (use user migration instead)",
					len(tablePlan.MockData), tablePlan.Slug)
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
					errs = append(errs, fmt.Sprintf("mock %s[%d] insert: %v", tablePlan.Slug, i, err))
				}
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

// sanitizeMockRow coerces mock data values to match their field types.
// Claude often generates numeric values (JSON numbers) for varchar/text fields
// which causes pgx "cannot find encode plan for varchar" errors.
// This converts float64 → string for all string-based field types.
func sanitizeMockRow(row map[string]any, fields []models.TableFieldPlan) map[string]any {
	// Build slug → mapped type lookup
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
		// These fields are all stored as text in Postgres.
		// If Claude gave us a number, stringify it.
		if f, ok := v.(float64); ok {
			if f == float64(int64(f)) {
				return fmt.Sprintf("%d", int64(f))
			}
			return fmt.Sprintf("%g", f)
		}
	case "NUMBER":
		// NUMBER fields expect a numeric value — nothing to do.
	}
	return v
}

// safeString extracts a non-nil string from an any value.
// Returns "" if the value is nil, not a string, or an empty string.
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
		// No MockData for login tables — mock rows without auth credentials
		// always fail Items.Create validation on login tables.
		// Users are migrated from the "user" system table instead (Steps 8-9).
		MockData: nil,
	}

	plan.Tables = append([]models.TablePlan{defaultUsers}, plan.Tables...)
	return plan
}
