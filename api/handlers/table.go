package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateTable godoc
// @Security ApiKeyAuth
// @ID create_table
// @Router /v1/table [POST]
// @Summary Create table
// @Description Create table
// @Tags Table
// @Accept json
// @Produce json
// @Param table body models.CreateTableRequest true "CreateTableRequestBody"
// @Success 201 {object} status_http.Response{data=obs.Table} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateTable(c *gin.Context) {
	var (
		tableRequest          models.CreateTableRequest
		resp                  *obs.CreateTableResponse
		err                   error
		resourceEnvironmentId string
		resourceType          pb.ResourceType
	)

	err = c.ShouldBindJSON(&tableRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if tableRequest.Attributes == nil {
		tableRequest.Attributes = make(map[string]interface{})
	}

	attributes, err := helper.ConvertMapToStruct(tableRequest.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
	}

	var fields []*obs.CreateFieldsRequest
	for _, field := range tableRequest.Fields {
		attributes, err := helper.ConvertMapToStruct(field.Attributes)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
		var tempField = obs.CreateFieldsRequest{
			Id:         field.ID,
			Default:    field.Default,
			Type:       field.Type,
			Index:      field.Index,
			Label:      field.Label,
			Slug:       field.Slug,
			Attributes: attributes,
			IsVisible:  field.IsVisible,
			Unique:     field.Unique,
			Automatic:  field.Automatic,
		}

		tempField.ProjectId = resourceEnvironmentId

		fields = append(fields, &tempField)
	}

	var table = obs.CreateTableRequest{
		Label:             tableRequest.Label,
		Description:       tableRequest.Description,
		Slug:              tableRequest.Slug,
		ShowInMenu:        tableRequest.ShowInMeny,
		Icon:              tableRequest.Icon,
		Fields:            fields,
		SubtitleFieldSlug: tableRequest.SubtitleFieldSlug,
		Sections:          tableRequest.Sections,
		Layouts:           tableRequest.Layouts,
		AppId:             tableRequest.AppID,
		IncrementId: &obs.IncrementID{
			WithIncrementId: tableRequest.IncrementID.WithIncrementID,
			DigitNumber:     tableRequest.IncrementID.DigitNumber,
			Prefix:          tableRequest.IncrementID.Prefix,
		},
		AuthorId:   authInfo.GetUserId(),
		Name:       fmt.Sprintf("Auto Created Commit Create table - %s", time.Now().Format(time.RFC1123)),
		CommitType: config.COMMIT_TYPE_TABLE,
		OrderBy:    tableRequest.OrderBy,
		Attributes: attributes,
	}

	table.ProjectId = resourceEnvironmentId

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Table().Create(
			context.Background(),
			&table,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Table().Create(
			context.Background(),
			&table,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetTableByID godoc
// @Security ApiKeyAuth
// @ID get_table_by_id
// @Router /v1/table/{table_id} [GET]
// @Summary Get table by id
// @Description Get table by id
// @Tags Table
// @Accept json
// @Produce json
// @Param table_id path string true "table_id"
// @Success 200 {object} status_http.Response{data=models.CreateTableResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTableByID(c *gin.Context) {
	tableID := c.Param("table_id")
	var (
		resp                  *obs.Table
		err                   error
		resourceEnvironmentId string
		resourceType          pb.ResourceType
	)

	if !util.IsValidUUID(tableID) {
		h.handleResponse(c, status_http.InvalidArgument, "table id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Table().GetByID(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Table().GetByID(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}
	h.handleResponse(c, status_http.OK, resp)
}

// GetAllTables godoc
// @Security ApiKeyAuth
// @ID get_all_tables
// @Router /v1/table [GET]
// @Summary Get all tables
// @Description Get all tables
// @Tags Table
// @Accept json
// @Produce json
// @Param filters query models.GetAllTablesRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllTablesResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllTables(c *gin.Context) {
	var (
		//resourceEnvironment *company_service.ResourceEnvironment
		resp                  *obs.GetAllTablesResponse
		resourceEnvironmentId string
		resourceType          pb.ResourceType
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
	}
	var isLoginTable bool
	var isLoginTableStr = c.Query("is_login_table")
	if isLoginTableStr == "true" {
		isLoginTable = true
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Table().GetAll(
			context.Background(),
			&obs.GetAllTablesRequest{
				Limit:        int32(limit),
				Offset:       int32(offset),
				Search:       c.DefaultQuery("search", ""),
				ProjectId:    resourceEnvironmentId,
				FolderId:     c.Query("folder_id"),
				IsLoginTable: isLoginTable,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Table().GetAll(
			context.Background(),
			&obs.GetAllTablesRequest{
				Limit:        int32(limit),
				Offset:       int32(offset),
				Search:       c.DefaultQuery("search", ""),
				ProjectId:    resourceEnvironmentId,
				IsLoginTable: isLoginTable,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateTable godoc
// @Security ApiKeyAuth
// @ID update_table
// @Router /v1/table [PUT]
// @Summary Update table
// @Description Update table
// @Tags Table
// @Accept json
// @Produce json
// @Param table body models.UpdateTableRequest  true "UpdateTableRequestBody"
// @Success 200 {object} status_http.Response{data=obs.Table} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateTable(c *gin.Context) {
	var (
		table models.UpdateTableRequest
		//resourceEnvironment *company_service.ResourceEnvironment
		resp                  *emptypb.Empty
		resourceEnvironmentId string
		resourceType          pb.ResourceType
	)

	err := c.ShouldBindJSON(&table)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	table.AuthorId = authInfo.GetUserId()
	table.Name = fmt.Sprintf("Auto Created Commit Update table - %s", time.Now().Format(time.RFC1123))
	table.CommitType = config.COMMIT_TYPE_TABLE

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
	}
	structData, err := helper.ConvertMapToStruct(table.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err)
		return
	}

	table.ProjectId = resourceEnvironmentId
	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Table().Update(
			context.Background(),
			&obs.UpdateTableRequest{
				Id:                table.Id,
				Description:       table.Description,
				Label:             table.Label,
				Slug:              table.Slug,
				ShowInMenu:        table.ShowInMenu,
				Icon:              table.Icon,
				SubtitleFieldSlug: table.SubtitleFieldSlug,
				IsVisible:         table.IsVisible,
				IsOwnTable:        table.IsOwnTable,
				IncrementId: &obs.IncrementID{
					WithIncrementId: table.IncrementId.WithIncrementID,
					DigitNumber:     table.IncrementId.DigitNumber,
					Prefix:          table.IncrementId.Prefix,
				},
				ProjectId:    table.ProjectId,
				FolderId:     table.FolderId,
				AuthorId:     table.AuthorId,
				CommitType:   table.CommitType,
				Name:         table.Name,
				IsCached:     table.IsCached,
				IsLoginTable: table.IsLoginTable,
				Attributes:   structData,
				OrderBy:      table.OrderBy,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Table().Update(
			context.Background(),
			&obs.UpdateTableRequest{
				Id:                table.Id,
				Description:       table.Description,
				Slug:              table.Slug,
				ShowInMenu:        table.ShowInMenu,
				Icon:              table.Icon,
				SubtitleFieldSlug: table.SubtitleFieldSlug,
				IsVisible:         table.IsVisible,
				IsOwnTable:        table.IsOwnTable,
				IncrementId: &obs.IncrementID{
					WithIncrementId: table.IncrementId.WithIncrementID,
					DigitNumber:     table.IncrementId.DigitNumber,
					Prefix:          table.IncrementId.Prefix,
				},
				ProjectId:    table.ProjectId,
				FolderId:     table.FolderId,
				AuthorId:     table.AuthorId,
				CommitType:   table.CommitType,
				Name:         table.Name,
				IsCached:     table.IsCached,
				IsLoginTable: table.IsLoginTable,
				Attributes:   structData,
				OrderBy:      table.OrderBy,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteTable godoc
// @Security ApiKeyAuth
// @ID delete_table
// @Router /v1/table/{table_id} [DELETE]
// @Summary Delete Table
// @Description Delete Table
// @Tags Table
// @Accept json
// @Produce json
// @Param table_id path string true "table_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteTable(c *gin.Context) {
	tableID := c.Param("table_id")
	var (
		resp                  *emptypb.Empty
		resourceEnvironmentId string
		resourceType          pb.ResourceType
	)

	if !util.IsValidUUID(tableID) {
		h.handleResponse(c, status_http.InvalidArgument, "table id is an invalid uuid")
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, resourceIdOk := c.Get("resource_id")

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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
	} else {
		resourceEnvironment, err := services.CompanyService().Resource().GetResourceEnvironment(
			c.Request.Context(),
			&pb.GetResourceEnvironmentReq{
				EnvironmentId: environmentId.(string),
				ResourceId:    resourceId.(string),
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resourceEnvironmentId = resourceEnvironment.GetId()
		resourceType = pb.ResourceType(resourceEnvironment.ResourceType)
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.BuilderService().Table().Delete(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:         tableID,
				ProjectId:  resourceEnvironmentId,
				AuthorId:   authInfo.GetUserId(),
				Name:       fmt.Sprintf("Auto Created Commit Delete table - %s", time.Now().Format(time.RFC1123)),
				CommitType: config.COMMIT_TYPE_TABLE,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Table().Delete(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}
	h.handleResponse(c, status_http.NoContent, resp)
}

// GetTableHistories godoc
// @Security ApiKeyAuth
// @ID get_table_histories
// @Router /v1/table-history/list/{table_id} [GET]
// @Summary Get table histories
// @Description Get table histories
// @Tags Table
// @Accept json
// @Produce json
// @Param table_id path string true "table_id"
// @Success 200 {object} status_http.Response{data=object_builder_service.GetTableHistoryResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListTableHistory(c *gin.Context) {
	var (
	//resourceEnvironment *company_service.ResourceEnvironment
	)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resp, err := services.BuilderService().Table().GetListTableHistory(
		context.Background(),
		&obs.GetTableHistoryRequest{
			ProjectId: resource.ResourceEnvironmentId,
			TableId:   c.Param("table_id"),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetTableHistoryById godoc
// @Security ApiKeyAuth
// @ID get_table_history_by_id
// @Router /v1/table-history/{id} [GET]
// @Summary Get table history by id
// @Description Get table history by id
// @Tags Table
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=object_builder_service.Table} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTableHistoryById(c *gin.Context) {
	var (
	//resourceEnvironment *company_service.ResourceEnvironment
	)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resp, err := services.BuilderService().Table().GetTableHistoryById(
		context.Background(),
		&obs.TableHistoryPrimaryKey{
			ProjectId: resource.ResourceEnvironmentId,
			Id:        c.Param("id"),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// RevertTableHistory godoc
// @Security ApiKeyAuth
// @ID revert_table_history
// @Router /v1/table-history/revert [PUT]
// @Summary Get table history by id
// @Description Get table history by id
// @Tags Table
// @Accept json
// @Produce json
// @Param table body models.RevertHistoryRequest  true "UpdateTableRequestBody"
// @Success 200 {object} status_http.Response{data=object_builder_service.TableHistory} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RevertTableHistory(c *gin.Context) {
	var (
		body models.RevertHistoryRequest
	)

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error getting auth info: %w", err).Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resp, err := services.BuilderService().Table().RevertTableHistory(
		context.Background(),
		&obs.RevertTableHistoryRequest{
			ProjectId:  resource.ResourceEnvironmentId,
			Id:         body.Id,
			AuthorId:   authInfo.GetUserId(),
			CommitType: config.COMMIT_TYPE_TABLE,
			Name:       fmt.Sprintf("Auto Created Commit Revert table - %s", time.Now().Format(time.RFC1123)),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// InsetrVersionsIdsToTableHistory godoc
// @Security ApiKeyAuth
// @ID insert into table history
// @Router /v1/table-history [PUT]
// @Summary Get table history by id
// @Description Get table history by id
// @Tags Table
// @Accept json
// @Produce json
// @Param table body models.InsertVersionsToCommitRequest  true "UpdateTableRequestBody"
// @Success 200 {object} status_http.Response{data=object_builder_service.TableHistory} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) InsetrVersionsIdsToTableHistory(c *gin.Context) {
	var (
		body models.InsertVersionsToCommitRequest
	)

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
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

	resp, err := services.BuilderService().Table().InsertVersionsToCommit(
		context.Background(),
		&obs.InsertVersionsToCommitRequest{
			ProjectId:  resource.ResourceEnvironmentId,
			Id:         body.Id,
			VersionIds: body.Version_ids,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
