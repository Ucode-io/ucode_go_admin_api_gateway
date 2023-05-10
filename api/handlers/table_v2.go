package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	postgresObs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// V2CreateTable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID v2_create_table
// @Router /v2/table [POST]
// @Summary v2 Create table
// @Description v2 Create table
// @Tags V2Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table body models.CreateTableRequest true "CreateTableRequestBody"
// @Success 201 {object} status_http.Response{data=postgres_object_builder_service.Table} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) V2CreateTable(c *gin.Context) {
	var tableRequest models.CreateTableRequest

	err := c.ShouldBindJSON(&tableRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get resource_id"))
	//	return
	//}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
			ProjectId:     projectId,
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
	//		ResourceId:    resource.ResourceEnvironmentId,
	//	},
	//)
	//if err != nil {
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	var fields []*postgresObs.CreateFieldsRequest
	for _, field := range tableRequest.Fields {
		attributes, err := helper.ConvertMapToStruct(field.Attributes)
		if err != nil {
			h.handleResponse(c, status_http.InvalidArgument, err.Error())
			return
		}
		var tempField = postgresObs.CreateFieldsRequest{
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

		tempField.ProjectId = resource.ResourceEnvironmentId

		fields = append(fields, &tempField)
	}

	var table = postgresObs.CreateTableRequest{
		Label:             tableRequest.Label,
		Description:       tableRequest.Description,
		Slug:              tableRequest.Slug,
		ShowInMenu:        tableRequest.ShowInMeny,
		Icon:              tableRequest.Icon,
		Fields:            fields,
		SubtitleFieldSlug: tableRequest.SubtitleFieldSlug,
		AppId:             tableRequest.AppID,
		IncrementId: &postgresObs.IncrementID{
			WithIncrementId: tableRequest.IncrementID.WithIncrementID,
			DigitNumber:     tableRequest.IncrementID.DigitNumber,
			Prefix:          tableRequest.IncrementID.Prefix,
		},
		AuthorId:   authInfo.GetUserId(),
		Name:       fmt.Sprintf("Auto Created Commit Create table - %s", time.Now().Format(time.RFC1123)),
		CommitType: config.COMMIT_TYPE_TABLE,
	}

	table.ProjectId = resource.ResourceEnvironmentId

	resp, err := services.PostgresBuilderService().Table().Create(
		context.Background(),
		&table,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// V2GetTableByID godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID v2_get_table_by_id
// @Router /v2/table/{table_id} [GET]
// @Summary V2 Get table by id
// @Description V2 Get table by id
// @Tags V2Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table_id path string true "table_id"
// @Success 200 {object} status_http.Response{data=models.CreateTableResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) V2GetTableByID(c *gin.Context) {
	tableID := c.Param("table_id")

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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get resource_id"))
	//	return
	//}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
			ProjectId:     projectId,
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
	//		ResourceId:    resource.ResourceEnvironmentId,
	//	},
	//)
	//if err != nil {
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	resp, err := services.PostgresBuilderService().Table().GetByID(
		context.Background(),
		&postgresObs.TablePrimaryKey{
			Id:        tableID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllTables godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID v2_get_all_tables
// @Router /v2/table [GET]
// @Summary V2 Get all tables
// @Description V2 Get all tables
// @Tags V2Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param filters query object_builder_service.GetAllTablesRequest true "filters"
// @Success 200 {object} status_http.Response{data=postgres_object_builder_service.GetAllTablesResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) V2GetAllTables(c *gin.Context) {
	var (
	//resourceEnvironment *company_service.ResourceEnvironment
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

	//_, err = h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get resource_id"))
	//	return
	//}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&company_service.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&company_service.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId,
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	fmt.Println("resourceId:::::::::", resource.ResourceId)
	fmt.Println("environmentId::::::", environmentId)
	fmt.Println("resourceEnvironment", resource.ResourceEnvironmentId)

	resp, err := services.PostgresBuilderService().Table().GetAll(
		context.Background(),
		&postgresObs.GetAllTablesRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// V2UpdateTable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID v2_update_table
// @Router /v2/table [PUT]
// @Summary Update table
// @Description Update table
// @Tags V2Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table body postgres_object_builder_service.Table  true "UpdateTableRequestBody"
// @Success 200 {object} status_http.Response{data=postgres_object_builder_service.Table} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateTable(c *gin.Context) {
	var (
		table postgresObs.UpdateTableRequest
		//resourceEnvironment *company_service.ResourceEnvironment
	)

	err := c.ShouldBindJSON(&table)
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

	//if !util.IsValidUUID(table.GetProjectId()) {
	//	h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
	//	return
	//}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get resource_id"))
	//	return
	//}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
			ProjectId:     projectId,
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&company_service.GetResourceEnvironmentReq{
	//			EnvironmentId: environmentId.(string),
	//			ResourceId:    resourceId.(string),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//} else {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetDefaultResourceEnvironment(
	//		c.Request.Context(),
	//		&company_service.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  table.GetProjectId(),
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	table.ProjectId = resource.ResourceEnvironmentId
	table.AuthorId = authInfo.GetUserId()
	table.Name = fmt.Sprintf("Auto Created Commit Update table - %s", time.Now().Format(time.RFC1123))
	table.CommitType = config.COMMIT_TYPE_TABLE

	resp, err := services.PostgresBuilderService().Table().Update(
		context.Background(),
		&table,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// V2DeleteTable godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID v2_delete_table
// @Router /v2/table/{table_id} [DELETE]
// @Summary V2 Delete Table
// @Description V2 Delete Table
// @Tags V2Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table_id path string true "table_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) V2DeleteTable(c *gin.Context) {
	tableID := c.Param("table_id")

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

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	h.handleResponse(c, status_http.BadRequest, errors.New("cant get resource_id"))
	//	return
	//}

	projectId := c.Query("project-id")
	if !util.IsValidUUID(projectId) {
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
			ProjectId:     projectId,
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
	//		ResourceId:    resource.ResourceEnvironmentId,
	//	},
	//)
	//if err != nil {
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	resp, err := services.PostgresBuilderService().Table().Delete(
		context.Background(),
		&postgresObs.TablePrimaryKey{
			Id:        tableID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
