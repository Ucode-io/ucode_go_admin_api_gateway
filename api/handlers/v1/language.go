package v1

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) CreateLanguage(c *gin.Context) {
	var (
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
		createLanguage        = &obs.CreateLanguageRequest{}
	)

	err := c.ShouldBindJSON(&createLanguage)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType
	createLanguage.ProjectId = resourceEnvironmentId

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), nodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Language().Create(
			c, createLanguage,
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.Created, resp)
	case pb.ResourceType_POSTGRESQL:
		createLanguagePg := nb.CreateLanguageRequest{}

		if err = helper.MarshalToStruct(&createLanguage, &createLanguagePg); err != nil {
			h.handleError(c, status_http.InternalServerError, err)
			return
		}
		resp, err := services.GoObjectBuilderService().Language().Create(
			c, &createLanguagePg,
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.Created, resp)
	}
}

func (h *HandlerV1) GetByIdLanguage(c *gin.Context) {
	var (
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
		languageId            string
	)

	languageId = c.Param("id")
	if languageId == "" {
		h.handleResponse(c, status_http.BadRequest, "language id is required")
		return
	}

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), nodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Language().GetById(
			c, &obs.PrimaryKey{
				Id:        languageId,
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Language().GetById(
			c, &nb.PrimaryKey{
				Id:        languageId,
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

func (h *HandlerV1) GetListLanguage(c *gin.Context) {
	var (
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
	)

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), nodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Language().GetList(
			c, &obs.GetListLanguagesRequest{
				Offset:    int32(offset),
				Limit:     int32(limit),
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Language().GetList(
			c, &nb.GetListLanguagesRequest{
				Offset:    int32(offset),
				Limit:     int32(limit),
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

func (h *HandlerV1) UpdateLanguage(c *gin.Context) {
	var (
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
		language              = &obs.UpdateLanguageRequest{}
	)

	err := c.ShouldBindJSON(&language)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType
	language.ProjectId = resourceEnvironmentId

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), nodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Language().Update(
			c, language,
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		languagePg := nb.UpdateLanguageRequest{}

		if err = helper.MarshalToStruct(&language, &languagePg); err != nil {
			h.handleError(c, status_http.InternalServerError, err)
			return
		}
		resp, err := services.GoObjectBuilderService().Language().Update(
			c, &languagePg,
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}
}

func (h *HandlerV1) DeleteLanguage(c *gin.Context) {
	var (
		resourceEnvironmentId string
		resourceType          pb.ResourceType
		nodeType              string
		languageId            string
	)

	languageId = c.Param("id")
	if languageId == "" {
		h.handleResponse(c, status_http.BadRequest, "language id is required")
		return
	}

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

	resourceEnvironmentId = resource.ResourceEnvironmentId
	resourceType = resource.ResourceType
	nodeType = resource.NodeType

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), nodeType)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	switch resourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).Language().Delete(
			c, &obs.PrimaryKey{
				Id:        languageId,
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.NoContent, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().Language().Delete(
			c, &nb.PrimaryKey{
				Id:        languageId,
				ProjectId: resourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err)
			return
		}
		h.handleResponse(c, status_http.NoContent, resp)
	}
}
