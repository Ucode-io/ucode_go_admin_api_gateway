package handlers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"strconv"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/template_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateTemplateFolder godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	var folder tmp.CreateFolderReq

	err := c.ShouldBindJSON(&folder)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	folder.ProjectId = resourceEnvironment.GetId()

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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	folderId := c.Param("template-folder-id")

	if !util.IsValidUUID(folderId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().Template().GetSingleFolder(
		context.Background(),
		&tmp.GetSingleFolderReq{
			Id:        folderId,
			ProjectId: resourceEnvironment.GetId(),
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	var folder tmp.UpdateFolderReq

	err := c.ShouldBindJSON(&folder)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	folder.ProjectId = resourceEnvironment.GetId()

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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	folderId := c.Param("template-folder-id")

	if !util.IsValidUUID(folderId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().Template().DeleteFolder(
		context.Background(),
		&tmp.DeleteFolderReq{
			Id:        folderId,
			ProjectId: resourceEnvironment.GetId(),
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().Template().GetListFolder(
		context.Background(),
		&tmp.GetListFolderReq{
			ProjectId: resourceEnvironment.GetId(),
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	var template tmp.CreateTemplateReq

	err := c.ShouldBindJSON(&template)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	template.ProjectId = resourceEnvironment.GetId()

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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	templateId := c.Param("template-id")

	if !util.IsValidUUID(templateId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().Template().GetSingleTemplate(
		context.Background(),
		&tmp.GetSingleTemplateReq{
			Id:        templateId,
			ProjectId: resourceEnvironment.GetId(),
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	var template tmp.UpdateTemplateReq

	err := c.ShouldBindJSON(&template)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	template.ProjectId = resourceEnvironment.GetId()

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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
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
	templateId := c.Param("template-id")

	if !util.IsValidUUID(templateId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().Template().DeleteTemplate(
		context.Background(),
		&tmp.DeleteTemplateReq{
			Id:        templateId,
			ProjectId: resourceEnvironment.GetId(),
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
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
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get_list_template
// @Router /v1/template [GET]
// @Summary Get List template
// @Description Get List template
// @Tags Template
// @Accept json
// @Produce json
// @Param folder-id query string true "folder-id"
// @Param limit query string true "limit"
// @Param offset query string true "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListFolderRes} "FolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListTemplate(c *gin.Context) {
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
	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	res, err := services.TemplateService().Template().GetListTemplate(
		context.Background(),
		&tmp.GetListTemplateReq{
			ProjectId: resourceEnvironment.GetId(),
			VersionId: "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			FolderId:  c.DefaultQuery("folder-id", ""),
			Limit:     int32(limit),
			Offset:    int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
