package v2

import (
	"context"
	"errors"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"google.golang.org/protobuf/types/known/structpb"
)

// CreateField godoc
// @Security ApiKeyAuth
// @ID v2_create_field
// @Router /v2/fields/{collection} [POST]
// @Summary Create field
// @Description Create field
// @Tags V2_Field
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param table body models.CreateFieldRequest true "CreateFieldRequestBody"
// @Success 201 {object} status_http.Response{data=models.Field} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateField(c *gin.Context) {
	var (
		fieldRequest models.CreateFieldRequest
		resp         *obs.Field
		fields       []*obs.CreateFieldRequest
	)

	err := c.ShouldBindJSON(&fieldRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(fieldRequest.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if fieldRequest.EnableMultilanguage {
		if fieldRequest.Type == "SINGLE_LINE" || fieldRequest.Type == "MULTI_LINE" {
			languages, err := h.companyServices.Project().GetById(context.Background(), &pb.GetProjectByIdRequest{
				ProjectId: resource.GetProjectId(),
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			if len(languages.GetLanguage()) > 1 {
				for _, value := range languages.GetLanguage() {
					id, _ := uuid.NewRandom()
					fieldRequest.ID = id.String()
					fields = append(fields, SetTitlePrefix(fieldRequest, value.ShortName, resource.ResourceEnvironmentId, attributes, true, false))
				}
			} else {
				fields = append(fields, SetTitlePrefix(fieldRequest, "", resource.ResourceEnvironmentId, attributes, true, false))
			}
		} else {
			fields = append(fields, SetTitlePrefix(fieldRequest, "", resource.ResourceEnvironmentId, attributes, false, false))
		}
	} else {
		fields = append(fields, SetTitlePrefix(fieldRequest, "", resource.ResourceEnvironmentId, attributes, false, false))
	}

	logReq := &models.CreateVersionHistoryRequest{
		Services:     services,
		NodeType:     resource.NodeType,
		ProjectId:    resource.ResourceEnvironmentId,
		ActionSource: "FIELD",
		ActionType:   "CREATE FIELD",
		// UsedEnvironments: map[string]bool{
		// 	cast.ToString(environmentId): true,
		// },
		UserInfo:  cast.ToString(userId),
		TableSlug: c.Param("collection"),
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		for _, field := range fields {
			resp, err = services.GetBuilderServiceByType(resource.NodeType).Field().Create(
				context.Background(),
				field,
			)
			if err != nil {
				logReq.Request = field
				logReq.Response = err.Error()
				go h.versionHistory(c, logReq)
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			logReq.Request = field
			logReq.Response = resp
			logReq.Current = resp
			go h.versionHistory(c, logReq)
		}

		h.handleResponse(c, status_http.Created, resp)
	case pb.ResourceType_POSTGRESQL:
		resp := &nb.Field{}

		for _, field := range fields {

			newReq := nb.CreateFieldRequest{}

			err = helper.MarshalToStruct(&field, &newReq)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}

			resp, err = services.GoObjectBuilderService().Field().Create(
				context.Background(),
				&newReq,
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}

		h.handleResponse(c, status_http.Created, resp)
	}

}

func SetTitlePrefix(fieldRequest models.CreateFieldRequest, prefix, project_id string, attributes *structpb.Struct, enable, hide bool) *obs.CreateFieldRequest {
	if prefix != "" {
		return &obs.CreateFieldRequest{
			Id:                  fieldRequest.ID,
			Default:             fieldRequest.Default,
			Type:                fieldRequest.Type,
			Index:               fieldRequest.Index,
			Label:               fieldRequest.Label,
			Slug:                fieldRequest.Slug + "_" + prefix,
			TableId:             fieldRequest.TableID,
			Attributes:          attributes,
			IsVisible:           fieldRequest.IsVisible,
			AutofillTable:       fieldRequest.AutoFillTable,
			AutofillField:       fieldRequest.AutoFillField,
			RelationField:       fieldRequest.RelationField,
			Automatic:           fieldRequest.Automatic,
			ShowLabel:           fieldRequest.ShowLabel,
			Unique:              fieldRequest.Unique,
			ProjectId:           project_id,
			EnableMultilanguage: enable,
			HideMultilanguage:   hide,
		}
	} else {
		return &obs.CreateFieldRequest{
			Id:                  fieldRequest.ID,
			Default:             fieldRequest.Default,
			Type:                fieldRequest.Type,
			Index:               fieldRequest.Index,
			Label:               fieldRequest.Label,
			Slug:                fieldRequest.Slug,
			TableId:             fieldRequest.TableID,
			Attributes:          attributes,
			IsVisible:           fieldRequest.IsVisible,
			AutofillTable:       fieldRequest.AutoFillTable,
			AutofillField:       fieldRequest.AutoFillField,
			RelationField:       fieldRequest.RelationField,
			Automatic:           fieldRequest.Automatic,
			ShowLabel:           fieldRequest.ShowLabel,
			Unique:              fieldRequest.Unique,
			ProjectId:           project_id,
			EnableMultilanguage: enable,
			HideMultilanguage:   hide,
		}
	}
}

// GetAllFields godoc
// @Security ApiKeyAuth
// @ID v2_get_all_fields
// @Router /v2/fields/{collection} [GET]
// @Summary Get all fields
// @Description Get all fields
// @Tags V2_Field
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param filters query obs.GetAllFieldsRequest true "filters"
// @Success 200 {object} status_http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllFields(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var withManyRelation, withOneRelation = false, false
	if c.Query("with_many_relation") == "true" {
		withManyRelation = true
	}

	if c.Query("with_one_relation") == "true" {
		withOneRelation = true
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	limit := 100
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.BuilderService().Field().GetAll(
			context.Background(),
			&obs.GetAllFieldsRequest{
				Limit:            int32(limit),
				Offset:           int32(offset),
				Search:           c.DefaultQuery("search", ""),
				TableId:          c.DefaultQuery("table_id", ""),
				TableSlug:        c.Param("collection"),
				WithManyRelation: withManyRelation,
				WithOneRelation:  withOneRelation,
				ProjectId:        resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Field().GetAll(
			context.Background(),
			&nb.GetAllFieldsRequest{
				Limit:            int32(limit),
				Offset:           int32(offset),
				Search:           c.DefaultQuery("search", ""),
				TableId:          c.DefaultQuery("table_id", ""),
				TableSlug:        c.Param("collection"),
				WithManyRelation: withManyRelation,
				WithOneRelation:  withOneRelation,
				ProjectId:        resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	}
}

// UpdateField godoc
// @Security ApiKeyAuth
// @ID v2_update_field
// @Router /v2/fields/{collection} [PUT]
// @Summary Update field
// @Description Update field
// @Tags V2_Field
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param relation body models.Field  true "UpdateFieldRequestBody"
// @Success 200 {object} status_http.Response{data=models.Field} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateField(c *gin.Context) {
	var (
		fieldRequest models.Field
	)

	err := c.ShouldBindJSON(&fieldRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	attributes, err := helper.ConvertMapToStruct(fieldRequest.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	var field = obs.Field{
		Id:                  fieldRequest.ID,
		Default:             fieldRequest.Default,
		Type:                fieldRequest.Type,
		Index:               fieldRequest.Index,
		Label:               fieldRequest.Label,
		Slug:                fieldRequest.Slug,
		TableId:             fieldRequest.TableID,
		Required:            fieldRequest.Required,
		Attributes:          attributes,
		IsVisible:           fieldRequest.IsVisible,
		AutofillField:       fieldRequest.AutoFillField,
		AutofillTable:       fieldRequest.AutoFillTable,
		RelationId:          fieldRequest.RelationId,
		Automatic:           fieldRequest.Automatic,
		Unique:              fieldRequest.Unique,
		RelationField:       fieldRequest.RelationField,
		ShowLabel:           fieldRequest.ShowLabel,
		EnableMultilanguage: fieldRequest.EnableMultilanguage,
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if !fieldRequest.EnableMultilanguage {
		field.EnableMultilanguage = false
		fields, err := services.GetBuilderServiceByType(resource.NodeType).Field().GetAll(context.Background(), &obs.GetAllFieldsRequest{
			TableId:   fieldRequest.TableID,
			ProjectId: resource.ResourceEnvironmentId,
		})
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		for _, value := range fields.GetFields() {
			if len(value.GetSlug()) > 3 && len(fieldRequest.Slug) > 3 && (value.GetType() == "SINGLE_LINE" || value.GetType() == "MULTI_LINE") {
				if fieldRequest.Slug[:len(fieldRequest.Slug)-3] == value.GetSlug()[:len(value.GetSlug())-3] && fieldRequest.Slug != value.GetSlug() {
					go func(arg *obs.Field) {
						_, err := services.GetBuilderServiceByType(resource.NodeType).Field().Delete(context.Background(), &obs.FieldPrimaryKey{
							Id:        arg.GetId(),
							ProjectId: resource.GetResourceEnvironmentId(),
						})
						if err != nil {
							h.handleResponse(c, status_http.GRPCError, err.Error())
							return
						}
					}(value)
				}
			}
		}
	}

	if fieldRequest.EnableMultilanguage {
		if fieldRequest.Type == "SINGLE_LINE" || fieldRequest.Type == "MULTI_LINE" {
			allFields, err := services.GetBuilderServiceByType(resource.NodeType).Field().GetAll(context.Background(), &obs.GetAllFieldsRequest{
				TableId:   fieldRequest.TableID,
				ProjectId: resource.ResourceEnvironmentId,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			languages, err := h.companyServices.Project().GetById(context.Background(), &pb.GetProjectByIdRequest{
				ProjectId: resource.GetProjectId(),
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
			var langs []string
			if len(languages.GetLanguage()) > 1 {
				for _, value := range languages.GetLanguage() {
					langs = append(langs, value.ShortName)
				}
				newFields := SeparateMultilangField(allFields, langs, resource.ResourceEnvironmentId, field)
				for _, field := range newFields {
					_, err = services.GetBuilderServiceByType(resource.NodeType).Field().Create(
						context.Background(),
						field,
					)
					if err != nil {
						h.handleResponse(c, status_http.GRPCError, err.Error())
						return
					}
				}
			}
		}
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "FIELD",
			ActionType:   "UPDATE FIELD",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			TableSlug: c.Param("collection"),
		}
	)

	field.ProjectId = resource.ResourceEnvironmentId
	field.EnvId = resource.EnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:

		oldField, deferErr := services.GetBuilderServiceByType(resource.NodeType).Field().GetByID(
			context.Background(),
			&obs.FieldPrimaryKey{
				Id:        field.Id,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if deferErr != nil {
			return
		}

		resp, deferErr := services.GetBuilderServiceByType(resource.NodeType).Field().Update(
			context.Background(),
			&field,
		)
		logReq.Request = &field
		logReq.Previous = oldField
		if deferErr != nil {
			logReq.Response = deferErr.Error()
			h.handleResponse(c, status_http.GRPCError, deferErr.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)

		h.handleResponse(c, status_http.Created, resp)
	case pb.ResourceType_POSTGRESQL:

		newReq := nb.Field{}

		err = helper.MarshalToStruct(&field, &newReq)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		oldField, deferErr := services.GoObjectBuilderService().Field().GetByID(
			context.Background(),
			&nb.FieldPrimaryKey{
				Id:        field.Id,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if deferErr != nil {
			return
		}

		resp, deferErr := services.GoObjectBuilderService().Field().Update(
			context.Background(),
			&newReq,
		)
		logReq.Request = &field
		logReq.Previous = oldField
		if deferErr != nil {
			logReq.Response = deferErr.Error()
			h.handleResponse(c, status_http.GRPCError, deferErr.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)

		h.handleResponse(c, status_http.Created, resp)
	}
}

func (h *HandlerV2) UpdateSearch(c *gin.Context) {

	var (
		searchRequest obs.SearchUpdateRequest
	)
	err := c.ShouldBindJSON(&searchRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	searchRequest.ProjectId = resource.ResourceEnvironmentId
	searchRequest.TableSlug = c.Param("collection")
	resp, err := services.GetBuilderServiceByType(resource.NodeType).Field().UpdateSearch(
		context.Background(),
		&searchRequest,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteField godoc
// @Security ApiKeyAuth
// @ID v2_delete_field
// @Router /v2/fields/{collection}/{id} [DELETE]
// @Summary Delete Field
// @Description Delete Field
// @Tags V2_Field
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteField(c *gin.Context) {
	fieldID := c.Param("id")

	if !util.IsValidUUID(fieldID) {
		h.handleResponse(c, status_http.InvalidArgument, "field id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "FIELD",
			ActionType:   "DELETE FIELD",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			TableSlug: c.Param("collection"),
		}
	)

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:

		oldField, err := services.GetBuilderServiceByType(resource.NodeType).Field().GetByID(
			context.Background(),
			&obs.FieldPrimaryKey{
				Id:        fieldID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			return
		}

		resp, err := services.GetBuilderServiceByType(resource.NodeType).Field().Delete(
			context.Background(),
			&obs.FieldPrimaryKey{
				Id:        fieldID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			logReq.Previous = oldField
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Previous = oldField
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)

		h.handleResponse(c, status_http.NoContent, resp)
	case pb.ResourceType_POSTGRESQL:

		oldField, err := services.GoObjectBuilderService().Field().GetByID(
			context.Background(),
			&nb.FieldPrimaryKey{
				Id:        fieldID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			return
		}

		resp, err := services.GoObjectBuilderService().Field().Delete(
			context.Background(),
			&nb.FieldPrimaryKey{
				Id:        fieldID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			logReq.Previous = oldField
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Previous = oldField
			logReq.Response = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)

		h.handleResponse(c, status_http.NoContent, resp)
	}

}

// GetAllFieldsWithDetails godoc
// @Security ApiKeyAuth
// @ID v2_get_all_fields_with_details
// @Router /v2/fields/{collection}/details [GET]
// @Summary Get all fields with details
// @Description Get all fields with details
// @Tags V2_Field
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param with_relations query string false "with_relations"
// @Param language_setting query string false "language_setting"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "FieldBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllFieldsWithDetails(c *gin.Context) {
	var (
		resp   *obs.AllFields
		roleId string
	)

	var withRelations = false
	if c.Query("with_relations") == "true" {
		withRelations = true
	}

	autoInfo, _ := h.GetAuthInfo(c)
	if autoInfo != nil {
		if autoInfo.GetRoleId() != "" {
			roleId = autoInfo.GetRoleId()
		}
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Field().GetAllForItems(
			context.Background(),
			&obs.GetAllFieldsForItemsRequest{
				Collection:       c.Param("collection"),
				ProjectId:        resource.ResourceEnvironmentId,
				LanguageSettings: c.DefaultQuery("language_setting", ""),
				WithRelations:    withRelations,
				RoleId:           roleId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Field().GetAllForItems(
			context.Background(),
			&obs.GetAllFieldsForItemsRequest{
				Collection:       c.Param("collection"),
				ProjectId:        resource.ResourceEnvironmentId,
				LanguageSettings: c.DefaultQuery("language_setting", ""),
				WithRelations:    withRelations,
				RoleId:           roleId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

func SeparateMultilangField(req *obs.GetAllFieldsResponse, langs []string, project_id string, reqField obs.Field) []*obs.CreateFieldRequest {
	var (
		multilangFields []*obs.Field
		response        []*obs.CreateFieldRequest
		newField        *obs.Field
		newFieldSlug    []string
		splitted        []string
	)
	for _, field := range req.Fields {
		if field.EnableMultilanguage {
			spliteed := strings.Split(field.Slug, "_")
			newFieldSlug = strings.Split(reqField.Slug, "_")
			if spliteed[0] == newFieldSlug[0] {
				newField = field
				multilangFields = append(multilangFields, field)
			}
		}
	}

	for _, lang := range langs {
		found := false
		for _, field := range multilangFields {
			splitted = strings.Split(field.Slug, "_")
			if splitted[len(splitted)-1] == lang {
				found = true
				break
			}
		}

		if !found {
			newStruct := obs.CreateFieldRequest{
				Default:             newField.Default,
				Type:                newField.Type,
				Index:               newField.Index,
				Label:               newField.Label,
				Slug:                newFieldSlug[0] + "_" + lang,
				TableId:             req.Fields[0].TableId,
				Attributes:          newField.Attributes,
				IsVisible:           newField.IsVisible,
				RelationField:       newField.RelationField,
				Automatic:           newField.Automatic,
				ShowLabel:           newField.ShowLabel,
				Unique:              newField.Unique,
				ProjectId:           project_id,
				EnableMultilanguage: true,
				HideMultilanguage:   false,
			}
			response = append(response, &obs.CreateFieldRequest{
				Default:             newField.Default,
				Type:                newField.Type,
				Index:               newField.Index,
				Label:               newField.Label,
				Slug:                newStruct.Slug,
				TableId:             req.Fields[0].TableId,
				Attributes:          newField.Attributes,
				IsVisible:           newField.IsVisible,
				RelationField:       newField.RelationField,
				Automatic:           newField.Automatic,
				ShowLabel:           newField.ShowLabel,
				Unique:              newField.Unique,
				ProjectId:           project_id,
				EnableMultilanguage: true,
				HideMultilanguage:   false,
			})
		}
	}
	//
	return response
}
