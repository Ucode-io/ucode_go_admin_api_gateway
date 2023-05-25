package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// CreateTableFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_table_folder
// @Router /v2/table-folder [POST]
// @Summary Create table folder
// @Description Create table folder
// @Tags Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table body models.CreateTableFolderRequest true "CreateTableFolderRequest"
// @Success 201 {object} status_http.Response{data=object_builder_service.TableFolder} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateTableFolder(c *gin.Context) {
	var tableFolder models.CreateTableFolderRequest
	var resourceEnvironmentId string

	err := c.ShouldBindJSON(&tableFolder)
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
	}

	resp, err := services.BuilderService().TableFolder().Create(
		context.Background(),
		&object_builder_service.TableFolderRequest{
			Title:     tableFolder.Title,
			ParentId:  tableFolder.ParentdId,
			ProjectId: resourceEnvironmentId,
			Icon:      tableFolder.Icon,
			AppId:     tableFolder.AppId,
		},
	)

	h.handleResponse(c, status_http.OK, resp)

}

// GetTableFolderByID godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_table_folder_by_id
// @Router /v2/table-folder/{id} [GET]
// @Summary Get table folder by id
// @Description Get table folder by id
// @Tags Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=object_builder_service.TableFolder} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTableFolderByID(c *gin.Context) {
	Id := c.Param("id")

	if !util.IsValidUUID(Id) {
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

	var resourceEnvironmentId string
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
	}

	resp, err := services.BuilderService().TableFolder().GetByID(
		context.Background(),
		&object_builder_service.TableFolderPrimaryKey{
			Id:        Id,
			ProjectId: resourceEnvironmentId,
		},
	)

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllTableFolders godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_all_table_folders
// @Router /v2/table-folder [GET]
// @Summary Get all table folders
// @Description Get all table folders
// @Tags Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param filters query models.GetAllTableFoldersRequest true "filters"
// @Success 200 {object} status_http.Response{data=object_builder_service.GetAllTablesResponse} "TableBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllTableFolders(c *gin.Context) {
	var (
		//resourceEnvironment *company_service.ResourceEnvironment
		resourceEnvironmentId string
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
	}

	resp, err := services.BuilderService().TableFolder().GetAll(
		context.Background(),
		&object_builder_service.GetAllTableFoldersRequest{
			Offset:    int32(offset),
			Limit:     int32(limit),
			ProjectId: resourceEnvironmentId,
			AppId:     c.Query("app_id"),
		},
	)

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateTableFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID update_table_folder
// @Router /v2/table-folder [PUT]
// @Summary Update table folder
// @Description Update table folder
// @Tags Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param table body object_builder_service.TableFolder  true "TableFolder"
// @Success 200 {object} status_http.Response{data=object_builder_service.TableFolder} "Table data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateTableFolder(c *gin.Context) {
	var (
		tableFolder object_builder_service.TableFolder
		//resourceEnvironment *company_service.ResourceEnvironment
		resourceEnvironmentId string
	)

	err := c.ShouldBindJSON(&tableFolder)
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
	}

	resp, err := services.BuilderService().TableFolder().Update(
		context.Background(),
		&object_builder_service.TableFolder{
			Title:     tableFolder.Title,
			ParentId:  tableFolder.ParentId,
			ProjectId: resourceEnvironmentId,
			Id:        tableFolder.Id,
			Icon:      tableFolder.Icon,
			AppId:     tableFolder.AppId,
		},
	)

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteTableFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID delete_table_folder
// @Router /v2/table-folder/{id} [DELETE]
// @Summary Delete Table Folder
// @Description Delete Table Folder
// @Tags Table
// @Accept json
// @Produce json
// @Param project-id query string true "project-id"
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteTableFolder(c *gin.Context) {
	Id := c.Param("id")
	resourceEnvironmentId := ""
	if !util.IsValidUUID(Id) {
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
	}

	resp, err := services.BuilderService().TableFolder().Delete(
		context.Background(),
		&obs.TableFolderPrimaryKey{
			Id:        Id,
			ProjectId: resourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
