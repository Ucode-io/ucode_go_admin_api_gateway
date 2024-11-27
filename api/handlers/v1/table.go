package v1

import (
	"context"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
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
func (h *HandlerV1) CreateTable(c *gin.Context) {
	var (
		tableRequest          models.CreateTableRequest
		err                   error
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

	if err = c.ShouldBindJSON(&tableRequest); err != nil {
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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
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
		Layouts:           tableRequest.Layouts,
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
		EnvId:      environmentId.(string),
		ViewId:     uuid.NewString(),
		LayoutId:   uuid.NewString(),
	}

	table.ProjectId = resourceEnvironmentId

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "TABLE",
			ActionType:   "CREATE TABLE",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			Request:   &table,
			TableSlug: tableRequest.Slug,
		}
	)

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(nodeType).Table().Create(
			context.Background(),
			&table,
		)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			table.Id = resp.Id
			logReq.Current = &table
			logReq.Response = &table
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistory(logReq)
	case pb.ResourceType_POSTGRESQL:

		newReq := nb.CreateTableRequest{}

		err = helper.MarshalToStruct(&table, &newReq)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		}

		resp, err := services.GoObjectBuilderService().Table().Create(
			context.Background(),
			&newReq,
		)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			table.Id = resp.Id
			logReq.Current = &table
			logReq.Response = &table
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistoryGo(c, logReq)
	}
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
func (h *HandlerV1) GetTableByID(c *gin.Context) {
	var (
		tableID               = c.Param("table_id")
		err                   error
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(nodeType).Table().GetByID(
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
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Table().GetByID(
			context.Background(),
			&nb.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
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
func (h *HandlerV1) GetAllTables(c *gin.Context) {
	var (
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit := 100

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	var isLoginTable bool
	var isLoginTableStr = c.Query("is_login_table")
	if isLoginTableStr == "true" {
		isLoginTable = true
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(nodeType).Table().GetAll(
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

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Table().GetAll(
			context.Background(),
			&nb.GetAllTablesRequest{
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

		h.handleResponse(c, status_http.OK, resp)
	}
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
func (h *HandlerV1) UpdateTable(c *gin.Context) {
	var (
		table                 models.UpdateTableRequest
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

	if err := c.ShouldBindJSON(&table); err != nil {
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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	structData, err := helper.ConvertMapToStruct(table.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err)
		return
	}

	services, _ := h.GetProjectSrvc(c.Request.Context(), projectId.(string), nodeType)

	table.ProjectId = resourceEnvironmentId

	var (
		updateTable = &obs.UpdateTableRequest{
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
			SoftDelete:   table.SoftDelete,
		}
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "TABLE",
			ActionType:   "UPDATE TABLE",
			UserInfo:     cast.ToString(userId),
			Request:      &updateTable,
			TableSlug:    table.Slug,
		}
	)

	switch resourceType {
	case pb.ResourceType_MONGODB:
		oldTable, err := services.GetBuilderServiceByType(nodeType).Table().GetByID(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:        table.Id,
				ProjectId: table.ProjectId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resp, err := services.GetBuilderServiceByType(nodeType).Table().Update(
			context.Background(),
			updateTable,
		)

		logReq.Previous = oldTable
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}

		go h.versionHistory(logReq)
	case pb.ResourceType_POSTGRESQL:
		oldTable, err := services.GoObjectBuilderService().Table().GetByID(
			context.Background(),
			&nb.TablePrimaryKey{
				Id:        table.Id,
				ProjectId: table.ProjectId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		newTable := nb.UpdateTableRequest{}

		err = helper.MarshalToStruct(&updateTable, &newTable)
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		}

		resp, err := services.GoObjectBuilderService().Table().Update(context.Background(), &newTable)

		logReq.Previous = oldTable
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistoryGo(c, logReq)
	}
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
func (h *HandlerV1) DeleteTable(c *gin.Context) {
	var (
		tableID               = c.Param("table_id")
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
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

	if !resourceIdOk {
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

		resourceEnvironmentId = resource.ResourceEnvironmentId
		resourceType = resource.ResourceType
		nodeType = resource.NodeType
	} else {
		resourceEnvironment, err := h.companyServices.Resource().GetResourceEnvironment(
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
		nodeType = resourceEnvironment.GetNodeType()
	}

	services, _ := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		nodeType,
	)

	var (
		oldTable = &obs.Table{}
		logReq   = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "TABLE",
			ActionType:   "DELETE TABLE",
			UserInfo:     cast.ToString(userId),
			TableSlug:    tableID,
		}
	)

	switch resourceType {
	case pb.ResourceType_MONGODB:

		oldTable, err = services.GetBuilderServiceByType(nodeType).Table().GetByID(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			return
		}

		resp, err := services.GetBuilderServiceByType(nodeType).Table().Delete(
			context.Background(),
			&obs.TablePrimaryKey{
				Id:         tableID,
				ProjectId:  resourceEnvironmentId,
				AuthorId:   authInfo.GetUserId(),
				Name:       fmt.Sprintf("Auto Created Commit Delete table - %s", time.Now().Format(time.RFC1123)),
				CommitType: config.COMMIT_TYPE_TABLE,
				EnvId:      environmentId.(string),
			},
		)

		logReq.Previous = oldTable
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(logReq)

	case pb.ResourceType_POSTGRESQL:

		oldTable, err := services.GoObjectBuilderService().Table().GetByID(
			context.Background(),
			&nb.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		resp, err := services.GoObjectBuilderService().Table().Delete(
			context.Background(),
			&nb.TablePrimaryKey{
				Id:        tableID,
				ProjectId: resourceEnvironmentId,
			},
		)

		logReq.Previous = oldTable
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistoryGo(c, logReq)
	}
}

// GetTableDetails godoc
// @Security ApiKeyAuth
// @ID get_table_details
// @Router /v1/table-details/{table_slug} [POST]
// @Summary Get table details
// @Description Get table details
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param language_setting query string false "language_setting"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetTableDetails(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	if tokenInfo != nil {
		if tokenInfo.Tables != nil {
			objectRequest.Data["tables"] = tokenInfo.GetTables()
		}
		objectRequest.Data["user_id_from_token"] = tokenInfo.GetUserId()
		objectRequest.Data["role_id_from_token"] = tokenInfo.GetRoleId()
		objectRequest.Data["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
	}
	objectRequest.Data["language_setting"] = c.DefaultQuery("language_setting", "")

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().GetTableDetails(
			context.Background(),
			&obs.CommonMessage{
				TableSlug: c.Param("table_slug"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		statusHttp.CustomMessage = resp.GetCustomMessage()
		h.handleResponse(c, statusHttp, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().ObjectBuilder().GetTableDetails(
			context.Background(),
			&nb.CommonMessage{
				TableSlug: c.Param("table_slug"),
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		statusHttp.CustomMessage = resp.GetCustomMessage()
		h.handleResponse(c, statusHttp, resp)
	}
}
