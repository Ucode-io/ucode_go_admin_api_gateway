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

func createBackendFromPlan(ctx context.Context, plan *models.ArchitectPlan, resourceEnvId, projectId, userId, envId string, service services.ServiceManagerI, emit ProgressEmitter) error {
	log.Printf("[backend] creating tables for project %s", resourceEnvId)

	plan = ensureLoginTable(plan)

	var errs []string

	// tableIdMap stores the ucode table ID (returned by Table.Create) keyed by table slug.
	// Used in Phase 2 to create RELATION fields on the correct table.
	tableIdMap := make(map[string]string, len(plan.Tables))

	// relFieldSlugsToSkip holds FK field slugs ({table_to}_id) that must NOT be created
	// as plain SINGLE_LINE fields in Phase 1 — they are created as RELATION type in Phase 2.
	relFieldSlugsToSkip := make(map[string]bool, len(plan.Relations))
	for _, rel := range plan.Relations {
		relFieldSlugsToSkip[rel.TableTo+"_id"] = true
	}

	// ─── Phase 1: Tables + Fields ───────────────────────────────────────────────
	// Mock data is deferred to Phase 3 so FK columns exist before inserts.

	for _, tablePlan := range plan.Tables {

		emit.Emit(
			SSEEvent{
				Type:    EvTableStart,
				Icon:    "database",
				Message: "Создаю таблицу",
				Value:   tablePlan.Label,
				Data:    map[string]any{"table": tablePlan.Slug, "label": tablePlan.Label},
			},
		)

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
				if len(plan.ClientTypes) > 0 {
					firstItem["name"] = plan.ClientTypes[0]
				}

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

				if len(plan.ClientTypes) > 0 && clientTypeId != "" {
					createRoleForClientType(ctx, plan.ClientTypes[0], clientTypeId, resourceEnvId, envId, service)

					for _, typeName := range plan.ClientTypes[1:] {
						ctGUID := uuid.NewString()
						ctData, ctConvertErr := helper.ConvertMapToStruct(map[string]any{
							"guid":       ctGUID,
							"name":       typeName,
							"table_slug": tablePlan.Slug,
						})
						if ctConvertErr != nil {
							log.Printf("[backend] client_type %q convert failed: %v", typeName, ctConvertErr)
							continue
						}
						if _, ctErr := service.GoObjectBuilderService().Items().Create(ctx, &nb.CommonMessage{
							TableSlug: "client_type",
							Data:      ctData,
							ProjectId: resourceEnvId,
							EnvId:     envId,
						}); ctErr != nil {
							log.Printf("[backend] client_type %q create failed: %v", typeName, ctErr)
							continue
						}
						log.Printf("[backend] client_type %q created (guid=%s)", typeName, ctGUID)
						createRoleForClientType(ctx, typeName, ctGUID, resourceEnvId, envId, service)
					}
				} else if clientTypeId != "" {
					// Architect didn't specify ClientTypes — create a role using the existing client_type name.
					ctName, _ := firstItem["name"].(string)
					if ctName == "" {
						ctName = tablePlan.Label
					}
					createRoleForClientType(ctx, ctName, clientTypeId, resourceEnvId, envId, service)
				}
			}
		}

	afterLoginBlock:

		tableId := tableResp.GetId()
		tableIdMap[tablePlan.Slug] = tableId
		log.Printf("[backend] table created: %s (id=%s)", tablePlan.Slug, tableId)

		emit.Emit(
			SSEEvent{
				Type:    EvTableDone,
				Icon:    "database",
				Message: "Таблица создана",
				Value:   tablePlan.Label,
				Data:    map[string]any{"table": tablePlan.Slug, "label": tablePlan.Label, "fields": len(tablePlan.Fields)},
			},
		)

		userFields := 0
		for _, fp := range tablePlan.Fields {
			if !isSystemField(fp.Slug) && !(tablePlan.IsLoginTable && isAuthField(fp.Slug)) {
				userFields++
			}
		}
		if userFields > 0 {
			emit.Emit(SSEEvent{
				Type:    EvProgress,
				Icon:    "columns",
				Message: fmt.Sprintf("Добавляю поля в %s", tablePlan.Label),
				Value:   fmt.Sprintf("%d полей", userFields),
			})
		}

		for _, fieldPlan := range tablePlan.Fields {
			if isSystemField(fieldPlan.Slug) {
				continue
			}
			if tablePlan.IsLoginTable && isAuthField(fieldPlan.Slug) {
				continue
			}
			// Skip FK slugs — created as proper RELATION type fields in Phase 2.
			if relFieldSlugsToSkip[fieldPlan.Slug] {
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
	}

	// ─── Phase 2: Relations ──────────────────────────────────────────────────────
	// Calls Relation().Create() which internally creates the RELATION type field
	// on table_from. ViewFields must be non-empty — ObtainRandomOne fetches a real
	// field id from table_from. This mirrors the proven pattern in table.go:1378-1404.

	log.Printf("[backend] Phase 2: %d relations to create (tableIdMap has %d entries)", len(plan.Relations), len(tableIdMap))

	if len(plan.Relations) > 0 {
		emit.Emit(SSEEvent{
			Type:    EvProgress,
			Icon:    "link",
			Message: fmt.Sprintf("Создаю связи между таблицами (%d)", len(plan.Relations)),
			Value:   fmt.Sprintf("%d связей", len(plan.Relations)),
		})

		for _, rel := range plan.Relations {
			// FK column slug: {table_to}_id (e.g. orders→customers → "customers_id" on orders)
			relFieldSlug := rel.TableTo + "_id"

			log.Printf("[backend] relation %s→%s: tableFrom_id=%q tableTo exists=%v", rel.TableFrom, rel.TableTo, tableIdMap[rel.TableFrom], tableIdMap[rel.TableTo] != "")

			// Sanity-check: source table must have been created successfully.
			if tableIdMap[rel.TableFrom] == "" {
				msg := fmt.Sprintf("relation %s→%s: source table not in tableIdMap (was it created?)", rel.TableFrom, rel.TableTo)
				errs = append(errs, msg)
				log.Printf("[backend] ⚠️ %s", msg)
				emit.Emit(SSEEvent{Type: EvProgress, Icon: "alert-triangle", Message: msg})
				continue
			}

			// ObtainRandomOne fetches any existing field id from table_from.
			// Relation.Create requires ViewFields to be non-empty — without it the
			// relation is silently rejected. This mirrors table.go:1378-1404.
			viewField, obtainErr := service.GoObjectBuilderService().Field().ObtainRandomOne(ctx, &nb.ObtainRandomRequest{
				TableSlug: rel.TableFrom,
				ProjectId: resourceEnvId,
				EnvId:     envId,
			})
			if obtainErr != nil {
				log.Printf("[backend] ObtainRandomOne %s failed: %v — skipping relation %s→%s", rel.TableFrom, obtainErr, rel.TableFrom, rel.TableTo)
				errs = append(errs, fmt.Sprintf("relation %s→%s: ObtainRandomOne failed: %v", rel.TableFrom, rel.TableTo, obtainErr))
				continue
			}

			relAttr, _ := helper.ConvertMapToStruct(map[string]any{
				"label_en":    slugToLabel(rel.TableTo),
				"label_to_en": slugToLabel(rel.TableFrom),
			})
			_, relErr := service.GoObjectBuilderService().Relation().Create(ctx, &nb.CreateRelationRequest{
				Id:                uuid.NewString(),
				TableFrom:         rel.TableFrom,
				TableTo:           rel.TableTo,
				Type:              "Many2One",
				RelationTableSlug: rel.TableTo,
				RelationFieldSlug: relFieldSlug,
				RelationFieldId:   uuid.NewString(),
				RelationToFieldId: uuid.NewString(),
				ProjectId:         resourceEnvId,
				EnvId:             envId,
				ViewFields:        []string{viewField.GetId()},
				Attributes:        relAttr,
			})
			if relErr != nil {
				msg := fmt.Sprintf("relation %s→%s failed: %v", rel.TableFrom, rel.TableTo, relErr)
				errs = append(errs, msg)
				log.Printf("[backend] ⚠️ Relation.Create %s→%s failed: %v", rel.TableFrom, rel.TableTo, relErr)
				emit.Emit(SSEEvent{Type: EvProgress, Icon: "alert-triangle", Message: fmt.Sprintf("Ошибка связи %s→%s", rel.TableFrom, rel.TableTo), Value: relErr.Error()})
			} else {
				log.Printf("[backend] ✅ relation Many2One %s→%s created (fk_slug=%s view_field=%s)", rel.TableFrom, rel.TableTo, relFieldSlug, viewField.GetId())
				emit.Emit(SSEEvent{Type: EvProgress, Icon: "link", Message: fmt.Sprintf("Связь создана: %s → %s", rel.TableFrom, rel.TableTo), Value: relFieldSlug})
			}
		}
	}

	// ─── Phase 3: Mock Data ──────────────────────────────────────────────────────
	// Inserted after relations so FK columns exist for any relation fields in mock rows.
	// Relation field slugs ({table_to}_id) are stripped to avoid FK constraint violations
	// since mock GUIDs don't point to real records.

	relFieldSlugs := make(map[string]bool, len(plan.Relations))
	for _, rel := range plan.Relations {
		relFieldSlugs[rel.TableTo+"_id"] = true
	}

	for _, tablePlan := range plan.Tables {
		if tablePlan.IsLoginTable {
			if len(tablePlan.MockData) > 0 {
				log.Printf("[backend] skipping %d mock rows for login table %s", len(tablePlan.MockData), tablePlan.Slug)
			}
			continue
		}

		for i, mockRow := range tablePlan.MockData {
			sanitized := sanitizeMockRow(mockRow, tablePlan.Fields)
			for slug := range relFieldSlugs {
				delete(sanitized, slug)
			}

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

	if len(errs) > 0 {
		log.Printf("[backend] completed with %d errors", len(errs))
		return fmt.Errorf("backend creation had %d errors: %s", len(errs), strings.Join(errs, "; "))
	}

	log.Printf("[backend] all tables, relations and mock data created successfully")
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
	case "RELATION", "LOOKUP", "FOREIGN_KEY":
		return "RELATION"
	default:
		return "SINGLE_LINE"
	}
}

// slugToLabel converts a snake_case slug to a Title Case label (e.g. "product_categories" → "Product Categories").
func slugToLabel(slug string) string {
	parts := strings.Split(slug, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
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

func createRoleForClientType(ctx context.Context, name, clientTypeId, resourceEnvId, envId string, service services.ServiceManagerI) {
	roleData, err := helper.ConvertMapToStruct(map[string]any{
		"guid":               uuid.NewString(),
		"name":               name,
		"client_type_id":     clientTypeId,
		"client_platform_id": uuid.NewString(),
	})
	if err != nil {
		log.Printf("[backend] role %q convert failed: %v", name, err)
		return
	}
	if _, err = service.GoObjectBuilderService().Items().Create(ctx, &nb.CommonMessage{
		TableSlug: "role",
		Data:      roleData,
		ProjectId: resourceEnvId,
		EnvId:     envId,
	}); err != nil {
		log.Printf("[backend] role %q create failed: %v", name, err)
		return
	}
	log.Printf("[backend] role %q created (client_type=%s)", name, clientTypeId)
}

// unused — kept for reference by older callers in mcp.go path
var _ = json.Marshal
