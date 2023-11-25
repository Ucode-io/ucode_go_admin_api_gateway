package handlers

import (
	"context"
	"errors"
	"strconv"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/template_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateTemplateFolder godoc
// @Security ApiKeyAuth
// @ID create_template_folder
// @Router /v1/template-folder [POST]
// @Summary Create template folder
// @Description Create template folder
// @Tags Template
// @Accept json
// @Produce json
// @Param template_folder body tmp.CreateFolderReq true "CreateFolderReq"
// @Success 201 {object} status_http.Response{data=tmp.Folder} "Template folder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateTemplateFolder(c *gin.Context) {
	var (
		folder tmp.CreateFolderReq
		//resourceEnvironment *obs.ResourceEnvironment
	)

	err := c.ShouldBindJSON(&folder)
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	folder.ProjectId = projectId.(string)
	folder.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	folder.CommitId = uuID.String()
	folder.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().Template().CreateFolder(
		context.Background(),
		&folder,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleTemplateFolder godoc
// @Security ApiKeyAuth
// @ID get_single_template_folder
// @Router /v1/template-folder/{template-folder-id} [GET]
// @Summary Get single template folder
// @Description Get single template folder
// @Tags Template
// @Accept json
// @Produce json
// @Param template-folder-id path string true "template-folder-id"
// @Success 200 {object} status_http.Response{data=tmp.GetSingleFolderRes} "TemplateFolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleTemplateFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	folderId := c.Param("template-folder-id")

	if !util.IsValidUUID(folderId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().GetSingleFolder(
		context.Background(),
		&tmp.GetSingleFolderReq{
			Id:         folderId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateTemplateFolder godoc
// @Security ApiKeyAuth
// @ID update_template_folder
// @Router /v1/template-folder [PUT]
// @Summary Update template folder
// @Description Update template folder
// @Tags Template
// @Accept json
// @Produce json
// @Param folder body tmp.UpdateFolderReq true "UpdateFolderReqBody"
// @Success 200 {object} status_http.Response{data=tmp.Folder} "Folder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateTemplateFolder(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		folder tmp.UpdateFolderReq
	)

	err := c.ShouldBindJSON(&folder)
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	folder.ProjectId = projectId.(string)
	folder.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	folder.CommitId = uuID.String()
	folder.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().Template().UpdateFolder(
		context.Background(),
		&folder,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteTemplateFolder godoc
// @Security ApiKeyAuth
// @ID delete_template_folder
// @Router /v1/template-folder/{template-folder-id} [DELETE]
// @Summary Delete template folder
// @Description Delete template folder
// @Tags Template
// @Accept json
// @Produce json
// @Param template-folder-id path string true "template-folder-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteTemplateFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	folderId := c.Param("template-folder-id")

	if !util.IsValidUUID(folderId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().DeleteFolder(
		context.Background(),
		&tmp.DeleteFolderReq{
			Id:         folderId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListTemplateFolder godoc
// @Security ApiKeyAuth
// @ID get_list_template_folder
// @Router /v1/template-folder [GET]
// @Summary Get List template folder
// @Description Get List template folder
// @Tags Template
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=tmp.GetListFolderRes} "FolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListTemplateFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().GetListFolder(
		context.Background(),
		&tmp.GetListFolderReq{
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// GetTemplateFolderCommits godoc
// @Security ApiKeyAuth
// @ID get_commits_template_folder
// @Router /v1/template-folder/commits/{template-folder-id} [GET]
// @Summary Get Commits template folder
// @Description Get Commits template folder
// @Tags Template
// @Accept json
// @Produce json
// @Param template-folder-id path string true "template-folder-id"
// @Success 200 {object} status_http.Response{data=tmp.GetListTemplateRes} "GetListTemplateRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTemplateFolderCommits(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().GetTemplateFolderObjectCommits(
		context.Background(),
		&tmp.GetTemplateFolderObjectCommitsReq{
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			Id:         c.Param("template-folder-id"),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// CreateTemplate godoc
// @Security ApiKeyAuth
// @ID create_template
// @Router /v1/template [POST]
// @Summary Create template
// @Description Create template
// @Tags Template
// @Accept json
// @Produce json
// @Param template body tmp.CreateTemplateReq true "CreateTemplateReq"
// @Success 201 {object} status_http.Response{data=tmp.Template} "Template data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateTemplate(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		template tmp.CreateTemplateReq
	)

	err := c.ShouldBindJSON(&template)
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	template.ProjectId = projectId.(string)
	template.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	template.CommitId = uuID.String()
	template.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().Template().CreateTemplate(
		context.Background(),
		&template,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleTemplate godoc
// @Security ApiKeyAuth
// @ID get_single_template
// @Router /v1/template/{template-id} [GET]
// @Summary Get single template
// @Description Get single template
// @Tags Template
// @Accept json
// @Produce json
// @Param template-id path string true "template-id"
// @Success 200 {object} status_http.Response{data=tmp.Template} "TemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleTemplate(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	templateId := c.Param("template-id")

	if !util.IsValidUUID(templateId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().GetSingleTemplate(
		context.Background(),
		&tmp.GetSingleTemplateReq{
			Id:         templateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateTemplate godoc
// @Security ApiKeyAuth
// @ID update_template
// @Router /v1/template [PUT]
// @Summary Update template
// @Description Update template
// @Tags Template
// @Accept json
// @Produce json
// @Param template body tmp.UpdateTemplateReq true "UpdateTemplateReqBody"
// @Success 200 {object} status_http.Response{data=tmp.Template} "Template data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateTemplate(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		template tmp.UpdateTemplateReq
	)

	err := c.ShouldBindJSON(&template)
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	template.ProjectId = projectId.(string)
	template.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	template.CommitId = uuID.String()
	template.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().Template().UpdateTemplate(
		context.Background(),
		&template,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteTemplate godoc
// @Security ApiKeyAuth
// @ID delete_template
// @Router /v1/template/{template-id} [DELETE]
// @Summary Delete template
// @Description Delete template
// @Tags Template
// @Accept json
// @Produce json
// @Param template-id path string true "template-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteTemplate(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	templateId := c.Param("template-id")

	if !util.IsValidUUID(templateId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().DeleteTemplate(
		context.Background(),
		&tmp.DeleteTemplateReq{
			Id:         templateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListTemplate godoc
// @Security ApiKeyAuth
// @ID get_list_template
// @Router /v1/template [GET]
// @Summary Get List template
// @Description Get List template
// @Tags Template
// @Accept json
// @Produce json
// @Param folder-id query string true "folder-id"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListFolderRes} "FolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListTemplate(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().GetListTemplate(
		context.Background(),
		&tmp.GetListTemplateReq{
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			FolderId:   c.DefaultQuery("folder-id", ""),
			Limit:      int32(limit),
			Offset:     int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// GetTemplateCommits godoc
// @Security ApiKeyAuth
// @ID get_commits_template
// @Router /v1/template/commits/{template-id} [GET]
// @Summary Get Commits template
// @Description Get Commits template
// @Tags Template
// @Accept json
// @Produce json
// @Param template-id path string true "template-id"
// @Success 200 {object} status_http.Response{data=tmp.GetListTemplateRes} "GetListTemplateRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetTemplateCommits(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().Template().GetTemplateObjectCommits(
		context.Background(),
		&tmp.GetTemplateObjectCommitsReq{
			ProjectId:  projectId.(string),
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			Id:         c.Param("template-id"),
			ResourceId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// ConvertHtmlToPdfV2 godoc
// @Security ApiKeyAuth
// @ID convert_html_to_pdfV2
// @Router /v1/html-to-pdfV2 [POST]
// @Summary Convert html to pdf
// @Description Convert html to pdf
// @Tags Template
// @Accept json
// @Produce json
// @Param template body models.HtmlBody true "HtmlBody"
// @Success 201 {object} status_http.Response{data=tmp.PdfBody} "PdfBody data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) ConvertHtmlToPdfV2(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		html models.HtmlBody
	)

	err := c.ShouldBindJSON(&html)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	structData, err := helper.ConvertMapToStruct(html.Data)

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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	resp, err := services.TemplateService().Template().ConvertHtmlToPdf(
		context.Background(),
		&tmp.HtmlBody{
			Data:       structData,
			Html:       html.Html,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// ConvertTemplateToHtmlV2 godoc
// @Security ApiKeyAuth
// @ID convert_template_to_htmlV2
// @Router /v1/template-to-htmlV2 [POST]
// @Summary Convert template to html
// @Description Convert template to html
// @Tags Template
// @Accept json
// @Produce json
// @Param view body models.HtmlBody true "TemplateBody"
// @Success 201 {object} status_http.Response{data=models.HtmlBody} "HtmlBody data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) ConvertTemplateToHtmlV2(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		html models.HtmlBody
	)

	err := c.ShouldBindJSON(&html)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(html.Data)

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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	resp, err := services.TemplateService().Template().ConvertTemplateToHtml(
		context.Background(),
		&tmp.HtmlBody{
			Data:       structData,
			Html:       html.Html,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
