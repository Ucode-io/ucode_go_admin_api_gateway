package v2

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetRelationCascading godoc
// @Security ApiKeyAuth
// @ID get_relation_cascading
// @Router /v2/relations/{collection}/cascading [GET]
// @Security ApiKeyAuth
// @Summary Get relation cascading
// @Description Get relation cascading
// @Tags V2_Relation
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Success 200 {object} status_http.Response{data=string} "CascadingBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetRelationCascading(c *gin.Context) {
	var (
		resp *obs.CommonMessage
	)

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
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Cascading().GetCascadings(
			context.Background(),
			&obs.GetCascadingRequest{
				TableSlug: c.Param("collection"),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Cascading().GetCascadings(
			context.Background(),
			&obs.GetCascadingRequest{
				TableSlug: c.Param("collection"),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetByIdRelation godoc
// @ID v2_get_by_id_relation
// @Router /v2/relations/{collection}/{id} [GET]
// @Security ApiKeyAuth
// @Summary Get relation by id
// @Description Get relation by id
// @Tags V2_Relation
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param id path string true "id"
// Success 200 {object} status_http.Response{data=obs.RelationForGetAll} "RelationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetByIdRelation(c *gin.Context) {
	var (
		relationId string = c.Param("id")
		resp       *obs.RelationForGetAll
	)

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
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Relation().GetByID(
			context.Background(),
			&obs.RelationPrimaryKey{
				Id:        relationId,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Relation().GetByID(
			context.Background(),
			&obs.RelationPrimaryKey{
				Id:        relationId,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// CreateRelation godoc
// @ID create_relations_V2
// @Router /v2/relations/{collection} [POST]
// @Security ApiKeyAuth
// @Summary Create relation
// @Description Create relation
// @Tags Relation
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param table body obs.CreateRelationRequest true "CreateRelationRequestBody"
// @Success 201 {object} status_http.Response{data=string} "Relation data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateRelation(c *gin.Context) {
	var (
		relation obs.CreateRelationRequest
		resp     *obs.CreateRelationRequest
		err      error
	)

	err = c.ShouldBindJSON(&relation)
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	relation.RelationFieldId = uuid.NewString()
	relation.RelationToFieldId = uuid.NewString()
	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "RELATION",
			ActionType:   "CREATE RELATION",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			Request:   &relation,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.Created, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	relation.ProjectId = resource.ResourceEnvironmentId
	relation.EnvId = resource.EnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Relation().Create(
			context.Background(),
			&relation,
		)
		if err != nil {
			return
		}
		relation.Id = resp.Id
		logReq.Request = &relation
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Relation().Create(
			context.Background(),
			&relation,
		)
		if err != nil {
			return
		}
	}
}

// GetAllRelations godoc
// @Security ApiKeyAuth
// @ID get_all_relations
// @Router /v1/relation [GET]
// @Security ApiKeyAuth
// @Summary Get all relations
// @Description Get all relations
// @Tags Relation
// @Accept json
// @Produce json
// @Param filters query obs.GetAllRelationsRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "RelationBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllRelations(c *gin.Context) {
	var (
		resp *obs.GetAllRelationsResponse
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
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Relation().GetAll(
			context.Background(),
			&obs.GetAllRelationsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				TableSlug: c.DefaultQuery("table_slug", ""),
				TableId:   c.DefaultQuery("table_id", ""),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Relation().GetAll(
			context.Background(),
			&obs.GetAllRelationsRequest{
				Limit:     int32(limit),
				Offset:    int32(offset),
				TableSlug: c.DefaultQuery("table_slug", ""),
				TableId:   c.DefaultQuery("table_id", ""),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateRelation godoc
// @Security ApiKeyAuth
// @ID update_relations_v2
// @Router /v2/relations/:collection [PUT]
// @Security ApiKeyAuth
// @Summary Update relation
// @Description Update relation
// @Tags Relation
// @Accept json
// @Produce json
// @Param collection path string true "collection"
// @Param relation body obs.UpdateRelationRequest  true "UpdateRelationRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Relation data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateRelation(c *gin.Context) {
	var (
		relation obs.UpdateRelationRequest
		resp     *obs.RelationForGetAll
	)

	err := c.ShouldBindJSON(&relation)
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		oldRelation = &obs.RelationForGetAll{}
		logReq      = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "RELATION",
			ActionType:   "UPDATE RELATION",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			Request:   &relation,
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		logReq.Previous = oldRelation
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			logReq.Response = resp
			logReq.Current = resp
			h.handleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	oldRelation, err = services.GetBuilderServiceByType(resource.NodeType).Relation().GetByID(
		context.Background(),
		&obs.RelationPrimaryKey{
			Id:        relation.Id,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}

	relation.ProjectId = resource.ResourceEnvironmentId
	relation.EnvId = resource.EnvironmentId
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Relation().Update(
			context.Background(),
			&relation,
		)
		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Relation().Update(
			context.Background(),
			&relation,
		)
		if err != nil {
			return
		}
	}
}

// DeleteRelation godoc
// @Security ApiKeyAuth
// @ID delete_relations_v2
// @Router /v2/relations/{collection}/{relation_id} [DELETE]
// @Security ApiKeyAuth
// @Summary Delete Relation
// @Description Delete Relation
// @Tags Relation
// @Accept json
// @Produce json
// @Param relation_id path string true "relation_id"
// @Param collection path string true "collection"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteRelation(c *gin.Context) {
	var (
		resp *emptypb.Empty
	)
	relationID := c.Param("id")

	if !util.IsValidUUID(relationID) {
		h.handleResponse(c, status_http.InvalidArgument, "relation id is an invalid uuid")
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
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		oldRelation = &obs.RelationForGetAll{}
		logReq      = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: "RELATION",
			ActionType:   "DELETE RELATION",
			// UsedEnvironments: map[string]bool{
			// 	cast.ToString(environmentId): true,
			// },
			UserInfo:  cast.ToString(userId),
			TableSlug: c.Param("collection"),
		}
	)

	defer func() {
		logReq.Previous = oldRelation
		if err != nil {
			logReq.Response = err.Error()
			h.handleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.handleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(c, logReq)
	}()

	oldRelation, err = services.GetBuilderServiceByType(resource.NodeType).Relation().GetByID(
		context.Background(),
		&obs.RelationPrimaryKey{
			Id:        relationID,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Relation().Delete(
			context.Background(),
			&obs.RelationPrimaryKey{
				Id:        relationID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Relation().Delete(
			context.Background(),
			&obs.RelationPrimaryKey{
				Id:        relationID,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			return
		}
	}
}

// GetRelationCascaders godoc
// @Security ApiKeyAuth
// @ID get_relation_cascaders
// @Router /v1/get-relation-cascading/{table_slug} [GET]
// @Security ApiKeyAuth
// @Summary Get all relations
// @Description Get all relations
// @Tags Relation
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Success 200 {object} status_http.Response{data=string} "CascaderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetRelationCascaders(c *gin.Context) {
	var (
		resp *obs.CommonMessage
	)

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
		resp, err = services.GetBuilderServiceByType(resource.NodeType).Cascading().GetCascadings(
			context.Background(),
			&obs.GetCascadingRequest{
				TableSlug: c.Param("table_slug"),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Cascading().GetCascadings(
			context.Background(),
			&obs.GetCascadingRequest{
				TableSlug: c.Param("table_slug"),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}
