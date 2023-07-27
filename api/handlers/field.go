package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// CreateField godoc
// @Security ApiKeyAuth
// @ID create_field
// @Router /v1/field [POST]
// @Summary Create field
// @Description Create field
// @Tags Field
// @Accept json
// @Produce json
// @Param table body models.CreateFieldRequest true "CreateFieldRequestBody"
// @Success 201 {object} status_http.Response{data=models.Field} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateField(c *gin.Context) {
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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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
	if fieldRequest.EnableMultilanguage {
		if fieldRequest.Type == "SINGLE_LINE" || fieldRequest.Type == "MULTI_LINE" {
			languages, err := services.CompanyService().Project().GetById(context.Background(), &pb.GetProjectByIdRequest{
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		for _, field := range fields {
			resp, err = services.BuilderService().Field().Create(
				context.Background(),
				field,
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	case pb.ResourceType_POSTGRESQL:
		for _, field := range fields {
			resp, err = services.PostgresBuilderService().Field().Create(
				context.Background(),
				field,
			)
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		}
	}
	h.handleResponse(c, status_http.Created, resp)
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
// @ID get_all_fields
// @Router /v1/field [GET]
// @Summary Get all fields
// @Description Get all fields
// @Tags Field
// @Accept json
// @Produce json
// @Param filters query obs.GetAllFieldsRequest true "filters"
// @Success 200 {object} status_http.Response{data=models.GetAllFieldsResponse} "FieldBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllFields(c *gin.Context) {
	var (
		resp *obs.GetAllFieldsResponse
	)
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}
	limit = 100
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Field().GetAll(
			context.Background(),
			&obs.GetAllFieldsRequest{
				Limit:            int32(limit),
				Offset:           int32(offset),
				Search:           c.DefaultQuery("search", ""),
				TableId:          c.DefaultQuery("table_id", ""),
				TableSlug:        c.DefaultQuery("table_slug", ""),
				WithManyRelation: withManyRelation,
				WithOneRelation:  withOneRelation,
				ProjectId:        resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Field().GetAll(
			context.Background(),
			&obs.GetAllFieldsRequest{
				Limit:            int32(limit),
				Offset:           int32(offset),
				Search:           c.DefaultQuery("search", ""),
				TableId:          c.DefaultQuery("table_id", ""),
				TableSlug:        c.DefaultQuery("table_slug", ""),
				WithManyRelation: withManyRelation,
				WithOneRelation:  withOneRelation,
				ProjectId:        resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateField godoc
// @Security ApiKeyAuth
// @ID update_field
// @Router /v1/field [PUT]
// @Summary Update field
// @Description Update field
// @Tags Field
// @Accept json
// @Produce json
// @Param relation body models.Field  true "UpdateFieldRequestBody"
// @Success 200 {object} status_http.Response{data=models.Field} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateField(c *gin.Context) {
	var (
		fieldRequest models.Field
		resp         *emptypb.Empty
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
		Id:            fieldRequest.ID,
		Default:       fieldRequest.Default,
		Type:          fieldRequest.Type,
		Index:         fieldRequest.Index,
		Label:         fieldRequest.Label,
		Slug:          fieldRequest.Slug,
		TableId:       fieldRequest.TableID,
		Required:      fieldRequest.Required,
		Attributes:    attributes,
		IsVisible:     fieldRequest.IsVisible,
		AutofillField: fieldRequest.AutoFillField,
		AutofillTable: fieldRequest.AutoFillTable,
		RelationId:    fieldRequest.RelationId,
		Automatic:     fieldRequest.Automatic,
		Unique:        fieldRequest.Unique,
		RelationField: fieldRequest.RelationField,
		ShowLabel:     fieldRequest.ShowLabel,
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	if !fieldRequest.EnableMultilanguage {
		field.EnableMultilanguage = false
		fields, err := services.BuilderService().Field().GetAll(context.Background(), &obs.GetAllFieldsRequest{
			TableId:   fieldRequest.TableID,
			ProjectId: resource.ResourceEnvironmentId,
		})
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		for _, value := range fields.GetFields() {
			if fieldRequest.Slug[:len(fieldRequest.Slug)-3] == value.GetSlug()[:len(value.GetSlug())-3] && fieldRequest.Slug != value.GetSlug() {
				go func(arg *obs.Field) {
					_, err := services.BuilderService().Field().Delete(context.Background(), &obs.FieldPrimaryKey{
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
	} else {
		field.EnableMultilanguage = true
		languaegs, err := services.CompanyService().Project().GetById(context.Background(), &pb.GetProjectByIdRequest{
			ProjectId: resource.GetProjectId(),
		})
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		for _, value := range languaegs.GetLanguage() {
			if fieldRequest.Slug != fieldRequest.Slug[:len(fieldRequest.Slug)-3]+"_"+value.GetShortName() {
				go func(arg *pb.Language, project_id string) {
					id, _ := uuid.NewRandom()
					_, err := services.BuilderService().Field().Create(context.Background(), &obs.CreateFieldRequest{
						Id:                  id.String(),
						Default:             fieldRequest.Default,
						Type:                fieldRequest.Type,
						Index:               fieldRequest.Index,
						Label:               fieldRequest.Label,
						Slug:                fieldRequest.Slug[:len(fieldRequest.Slug)-3] + "_" + arg.ShortName,
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
						EnableMultilanguage: true,
					})
					if err != nil {
						h.handleResponse(c, status_http.GRPCError, err.Error())
						return
					}
				}(value, resource.ResourceEnvironmentId)
			}
		}
	}

	field.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Field().Update(
			context.Background(),
			&field,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Field().Update(
			context.Background(),
			&field,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteField godoc
// @Security ApiKeyAuth
// @ID delete_field
// @Router /v1/field/{field_id} [DELETE]
// @Summary Delete Field
// @Description Delete Field
// @Tags Field
// @Accept json
// @Produce json
// @Param field_id path string true "field_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteField(c *gin.Context) {
	fieldID := c.Param("field_id")
	var (
		resp *emptypb.Empty
	)

	if !util.IsValidUUID(fieldID) {
		h.handleResponse(c, status_http.InvalidArgument, "field id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Field().Delete(
			context.Background(),
			&obs.FieldPrimaryKey{
				Id:        fieldID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Field().Delete(
			context.Background(),
			&obs.FieldPrimaryKey{
				Id:        fieldID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
