package v2

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV2) CreateFolderGroup(c *gin.Context) {
	var (
		folderGroup nb.CreateFolderGroupRequest
	)

	err := c.ShouldBindJSON(&folderGroup)
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		h.handleResponse(c, status_http.InternalServerError, "For Mongo Coming soon")
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().FolderGroup().Create(
			context.Background(),
			&nb.CreateFolderGroupRequest{
				TableId:   folderGroup.TableId,
				Name:      folderGroup.Name,
				Comment:   folderGroup.Comment,
				Code:      folderGroup.Code,
				ProjectId: resource.ResourceEnvironmentId,
				ParentId:  folderGroup.ParentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.Created, resp)
	}
}

func (h *HandlerV2) GetFolderGroupById(c *gin.Context) {
	folderGroupId := c.Param("id")

	if !util.IsValidUUID(folderGroupId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder group id is an invalid uuid")
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
		h.handleResponse(c, status_http.InternalServerError, "For Mongo Coming soon")
	case pb.ResourceType_POSTGRESQL:
		pgResp, err := services.GoObjectBuilderService().FolderGroup().GetByID(
			context.Background(),
			&nb.FolderGroupPrimaryKey{
				Id:        folderGroupId,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, pgResp)
	}
}

func (h *HandlerV2) GetAllFolderGroups(c *gin.Context) {
	var objectRequest models.CommonMessage

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

	parentId := c.DefaultQuery("parent_id", "")

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

	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	if tokenInfo != nil {
		objectRequest.Data = make(map[string]any)
		if tokenInfo.Tables != nil {
			objectRequest.Data["tables"] = tokenInfo.GetTables()
		}
		objectRequest.Data["user_id_from_token"] = tokenInfo.GetUserId()
		objectRequest.Data["role_id_from_token"] = tokenInfo.GetRoleId()
		objectRequest.Data["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
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

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		h.handleResponse(c, status_http.InternalServerError, "For Mongo Coming soon")
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().FolderGroup().GetAll(
			context.Background(),
			&nb.GetAllFolderGroupRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				ProjectId: resource.ResourceEnvironmentId,
				TableId:   c.DefaultQuery("table_id", ""),
				ParentId:  parentId,
				Data:      structData,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

func (h *HandlerV2) UpdateFolderGroup(c *gin.Context) {
	updateFolderGroupRequest := nb.UpdateFolderGroupRequest{}

	err := c.ShouldBindJSON(&updateFolderGroupRequest)
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	updateFolderGroupRequest.ProjectId = resource.ResourceEnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		h.handleResponse(c, status_http.InternalServerError, "For Mongo coming soon")
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().FolderGroup().Update(
			context.Background(),
			&updateFolderGroupRequest,
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

func (h *HandlerV2) DeleteFolderGroup(c *gin.Context) {
	folderGroupId := c.Param("id")

	if !util.IsValidUUID(folderGroupId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder group id is an invalid uuid")
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
		h.handleResponse(c, status_http.InternalServerError, "For Mongo coming soon")
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().FolderGroup().Delete(
			context.Background(),
			&nb.FolderGroupPrimaryKey{
				Id:        folderGroupId,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		h.handleResponse(c, status_http.NoContent, resp)
	}
}
