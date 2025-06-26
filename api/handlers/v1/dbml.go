package v1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obj "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ucode-io/dbml-go/parser"
	"github.com/ucode-io/dbml-go/scanner"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	enumMap = map[string][]*structpb.Value{}
)

type DbmlToUcodeRequest struct {
	Dbml    string              `json:"dbml"`
	Options map[string]string   `json:"options"`
	Menus   map[string][]string `json:"menus"`
}

func (h *HandlerV1) DbmlToUcode(c *gin.Context) {
	var (
		req         DbmlToUcodeRequest
		tableFieldM = make(map[string]map[string]string)
		tableMenuM  = make(map[string]string)
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	qqqqq, _ := json.Marshal(req)
	fmt.Println("DbmlToUcode request:", string(qqqqq))

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(), &pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType          = resource.ResourceType
		nodeType              = resource.NodeType
	)

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resourceCreds := resourceCreds{
		services:              services,
		resourceEnvironmentId: resourceEnvironmentId,
		environmentId:         resource.EnvironmentId,
		resourceType:          resourceType,
		nodeType:              nodeType,
	}

	currentDir, _ := os.Getwd()
	tmpFileName := filepath.Join(currentDir, uuid.NewString()+".dbml")
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}
	defer func() {
		tmpFile.Close()
		_ = os.Remove(tmpFileName)
	}()

	_, err = tmpFile.WriteString(req.Dbml)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	f, _ := os.Open(tmpFileName)
	s := scanner.NewScanner(f)
	parser := parser.NewParser(s)
	dbml, err := parser.Parse()
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	for _, enum := range dbml.Enums {
		options := make([]*structpb.Value, len(enum.Values))
		for i, value := range enum.Values {
			option, _ := structpb.NewStruct(map[string]any{
				"value": value.Name,
				"icon":  "",
				"color": "",
				"label": value.Name,
			})
			options[i] = structpb.NewStructValue(option)
		}
		enumMap[strings.ToLower(enum.Name)] = options
	}

	skipTables := map[string]bool{"role": true, "client_type": true}
	skipFields := map[string]bool{
		"guid":       true,
		"folder_id":  true,
		"created_at": true,
		"updated_at": true,
		"deleted_at": true}
	skipTypes := map[string]bool{"uuid": true, "uuid[]": true}

	for key, tables := range req.Menus {
		menuId := uuid.NewString()
		err = createMenu(c, &createMenuReq{
			resourceCreds: resourceCreds,
			id:            menuId,
			label:         key,
		})
		if err != nil {
			h.log.Error("Failed to create menu:", logger.Error(err))
			continue
		}

		for _, table := range tables {
			tableMenuM[table] = menuId
		}
	}

	for _, table := range dbml.Tables {
		if skipTables[table.Name] {
			continue
		}

		tableId, err := createTable(c, &createTableReq{
			resourceCreds: resourceCreds,
			label:         table.Name,
			menuId:        tableMenuM[table.Name],
		})
		if err != nil {
			h.log.Error("Failed to create table:", logger.Error(err))
			continue
		}

		for _, field := range table.Columns {
			field.Type = strings.ToLower(field.Type)
			if skipFields[field.Name] || skipTypes[field.Type] || field.Settings.PK {
				continue
			}

			fieldId := uuid.NewString()

			if field.Settings.Ref.Type == 0 {
				err := createField(c, &createFieldReq{
					resourceCreds: resourceCreds,
					id:            fieldId,
					tableId:       tableId,
					fieldType:     field.Type,
					label:         field.Name,
				})
				if err != nil {
					h.handleResponse(c, status_http.InternalServerError, err)
					continue
				}

				if _, ok := tableFieldM[table.Name]; !ok {
					tableFieldM[table.Name] = make(map[string]string)
				}
				tableFieldM[table.Name][field.Name] = fieldId
			} else {
				toParts := strings.Split(field.Settings.Ref.To, ".")
				err := createRelation(c, &createRelationReq{
					resourceCreds: resourceCreds,
					tableFrom:     table.Name,
					tableTo:       toParts[0],
					viewFieldId:   tableFieldM[toParts[0]][req.Options[toParts[0]]],
				})
				if err != nil {
					h.handleResponse(c, status_http.InternalServerError, err)
					continue
				}
			}

		}
	}

	for _, ref := range dbml.Refs {
		for _, relation := range ref.Relationships {
			fromParts := strings.Split(relation.From, ".")
			toParts := strings.Split(relation.To, ".")

			err := createRelation(c, &createRelationReq{
				resourceCreds: resourceCreds,
				tableFrom:     fromParts[0],
				tableTo:       toParts[0],
				viewFieldId:   tableFieldM[toParts[0]][req.Options[toParts[0]]],
			})
			if err != nil {
				h.handleResponse(c, status_http.InternalServerError, err)
				continue
			}
		}
	}

	h.handleResponse(c, status_http.OK, nil)
}

func createTable(c *gin.Context, req *createTableReq) (string, error) {
	tableReq := &obj.CreateTableRequest{
		Label:      formatString(req.label),
		Slug:       req.label,
		ShowInMenu: true,
		ViewId:     uuid.NewString(),
		LayoutId:   uuid.NewString(),
		MenuId:     req.menuId,
		Attributes: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"label_en": structpb.NewStringValue(formatString(req.label)),
			},
		},
		EnvId:     req.resourceCreds.environmentId,
		ProjectId: req.resourceCreds.resourceEnvironmentId,
	}

	switch req.resourceCreds.resourceType {
	case pb.ResourceType_MONGODB:
		tableResp, err := req.resourceCreds.services.GetBuilderServiceByType(req.resourceCreds.nodeType).Table().Create(
			c.Request.Context(),
			tableReq,
		)
		if err != nil {
			return "", err
		}
		return tableResp.Id, nil
	case pb.ResourceType_POSTGRESQL:
		pgTableReq := &nb.CreateTableRequest{}

		if err := helper.MarshalToStruct(&tableReq, &pgTableReq); err != nil {
			return "", err
		}

		tableResp, err := req.resourceCreds.services.GoObjectBuilderService().Table().Create(
			c.Request.Context(),
			pgTableReq,
		)
		if err != nil {
			return "", err
		}

		return tableResp.Id, nil
	}

	return "", nil
}

func createMenu(c *gin.Context, req *createMenuReq) error {
	menuReq := &obj.CreateMenuRequest{
		Id:       req.id,
		Label:    formatString(req.label),
		Type:     "FOLDER",
		ParentId: "c57eedc3-a954-4262-a0af-376c65b5a284",
		Attributes: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"label":    structpb.NewStringValue(formatString(formatString(req.label))),
				"label_en": structpb.NewStringValue(formatString(formatString(req.label))),
			},
		},
		ProjectId: req.resourceCreds.resourceEnvironmentId,
		EnvId:     req.resourceCreds.environmentId,
	}

	switch req.resourceCreds.resourceType {
	case pb.ResourceType_MONGODB:
		_, err := req.resourceCreds.services.GetBuilderServiceByType(req.resourceCreds.nodeType).Menu().Create(
			c.Request.Context(),
			menuReq,
		)
		if err != nil {
			return err
		}
	case pb.ResourceType_POSTGRESQL:
		pgMenuReq := &nb.CreateMenuRequest{}

		if err := helper.MarshalToStruct(&menuReq, &pgMenuReq); err != nil {
			return err
		}

		_, err := req.resourceCreds.services.GoObjectBuilderService().Menu().Create(
			c.Request.Context(),
			pgMenuReq,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func createField(c *gin.Context, req *createFieldReq) error {
	ucodeType := getFieldType(req.fieldType)

	fieldReq := &obj.CreateFieldRequest{
		Id:      req.id,
		TableId: req.tableId,
		Type:    ucodeType,
		Label:   formatString(req.label),
		Slug:    req.label,
		Attributes: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"label_en": structpb.NewStringValue(formatString(req.label)),
			},
		},
		ProjectId: req.resourceCreds.resourceEnvironmentId,
		EnvId:     req.resourceCreds.environmentId,
	}

	if ucodeType == "MULTISELECT" {
		fieldReq.Attributes.Fields["options"] = structpb.NewListValue(&structpb.ListValue{
			Values: enumMap[req.fieldType],
		})
	}

	switch req.resourceCreds.resourceType {
	case pb.ResourceType_MONGODB:
		_, err := req.resourceCreds.services.GetBuilderServiceByType(req.resourceCreds.nodeType).Field().Create(
			c.Request.Context(),
			fieldReq,
		)
		if err != nil {
			return err
		}
	case pb.ResourceType_POSTGRESQL:
		pgFieldReq := &nb.CreateFieldRequest{}

		if err := helper.MarshalToStruct(&fieldReq, &pgFieldReq); err != nil {
			return err
		}
		_, err := req.resourceCreds.services.GoObjectBuilderService().Field().Create(
			c.Request.Context(),
			pgFieldReq,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func createRelation(c *gin.Context, req *createRelationReq) error {
	relationReq := &obj.CreateRelationRequest{
		Id:        uuid.NewString(),
		Type:      "Many2One",
		TableFrom: req.tableFrom,
		TableTo:   req.tableTo,
		Attributes: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"label_en":    structpb.NewStringValue(formatString(req.tableTo)),
				"label_to_en": structpb.NewStringValue(formatString(req.tableFrom)),
			},
		},
		RelationFieldId:   uuid.NewString(),
		RelationToFieldId: uuid.NewString(),
		ProjectId:         req.resourceCreds.resourceEnvironmentId,
		EnvId:             req.resourceCreds.environmentId,
		ViewFields:        []string{req.viewFieldId},
	}

	switch req.resourceCreds.resourceType {
	case pb.ResourceType_MONGODB:
		_, err := req.resourceCreds.services.GetBuilderServiceByType(req.resourceCreds.nodeType).Relation().Create(
			c.Request.Context(),
			relationReq,
		)
		if err != nil {
			return err
		}
	case pb.ResourceType_POSTGRESQL:
		pgRelationReq := &nb.CreateRelationRequest{}

		if err := helper.MarshalToStruct(&relationReq, &pgRelationReq); err != nil {
			return err
		}
		_, err := req.resourceCreds.services.GoObjectBuilderService().Relation().Create(
			c.Request.Context(),
			pgRelationReq,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

type resourceCreds struct {
	services              services.ServiceManagerI
	resourceEnvironmentId string
	environmentId         string
	resourceType          pb.ResourceType
	nodeType              string
}

type createTableReq struct {
	resourceCreds resourceCreds
	label         string
	menuId        string
}

type createMenuReq struct {
	resourceCreds resourceCreds
	id            string
	label         string
}

type createFieldReq struct {
	resourceCreds resourceCreds
	id            string
	tableId       string
	fieldType     string
	label         string
}

type createRelationReq struct {
	resourceCreds resourceCreds
	tableFrom     string
	tableTo       string
	viewFieldId   string
}

var FIELD_TYPES = map[string]string{
	"character varying": "SINGLE_LINE",
	"varchar":           "SINGLE_LINE",
	"text":              "MULTI_LINE",
	"enum":              "SINGLE_LINE",
	"bytea":             "SINGLE_LINE",
	"citext":            "SINGLE_LINE",

	"jsonb": "JSON",
	"json":  "JSON",

	"int":              "FLOAT",
	"float":            "FLOAT",
	"smallint":         "FLOAT",
	"integer":          "FLOAT",
	"bigint":           "FLOAT",
	"numeric":          "FLOAT",
	"decimal":          "FLOAT",
	"real":             "FLOAT",
	"double precision": "FLOAT",
	"smallserial":      "FLOAT",
	"serial":           "FLOAT",
	"bigserial":        "FLOAT",
	"money":            "FLOAT",
	"int2":             "FLOAT",
	"int4":             "FLOAT",
	"float8":           "FLOAT",

	"timestamp":                   "DATE_TIME",
	"timestamptz":                 "DATE_TIME",
	"timestamp without time zone": "DATE_TIME_WITHOUT_TIME_ZONE",
	"timestamp with time zone":    "DATE_TIME",
	"date":                        "DATE",

	"boolean": "CHECKBOX",

	"uuid": "UUID",

	"point":   "MAP",
	"polygon": "POLYGON",
}

func getFieldType(fieldType string) string {
	if _, ok := enumMap[fieldType]; ok {
		return "MULTISELECT"
	}
	if _, ok := FIELD_TYPES[fieldType]; !ok {
		return "SINGLE_LINE"
	}

	return FIELD_TYPES[fieldType]
}

func formatString(input string) string {
	// Replace underscores with spaces
	input = strings.ReplaceAll(input, "_", " ")

	// Capitalize the first letter of the string
	runes := []rune(input)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}

	return string(runes)
}
