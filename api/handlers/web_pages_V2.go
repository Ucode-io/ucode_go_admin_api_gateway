package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	vcs "ucode/ucode_go_api_gateway/genproto/versioning_service"
	tmp "ucode/ucode_go_api_gateway/genproto/web_page_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	_ "github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
)

// CreateWebPageFolder godoc
// @Security ApiKeyAuth
// @ID create_web_page_folder
// @Router /v1/webpage-folder [POST]
// @Summary Create web page folder
// @Description Create web page folder
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage_folder body models.CreateFolderReqModel true "CreateFolderReqModel"
// @Success 201 {object} status_http.Response{data=tmp.Folder} "WebPage folder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateWebPageFolder(c *gin.Context) {
	var (
		folder tmp.CreateFolderReq
		//resourceEnvironment *obs.ResourceEnvironment
	)

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	folder.EnvironmentId = environmentId.(string)
	folder.ProjectId = projectId.(string)
	folder.ResourceId = resource.ResourceEnvironmentId
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	//folder.ProjectId = resource.ResourceEnvironmentId

	res, err := services.WebPageService().Folder().CreateFolder(
		context.Background(),
		&folder,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleWebPageFolder godoc
// @Security ApiKeyAuth
// @ID get_single_web_page_folder
// @Router /v1/webpage-folder/{webpage-folder-id} [GET]
// @Summary Get single webpage folder
// @Description Get single webpage folder
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-folder-id path string true "webpage-folder-id"
// @Success 200 {object} status_http.Response{data=tmp.GetSingleFolderRes} "GetSingleFolderRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleWebPageFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	folderId := c.Param("webpage-folder-id")

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
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
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	//
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.WebPageService().Folder().GetSingleFolder(
		context.Background(),
		&tmp.GetSingleFolderReq{
			Id:            folderId,
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			EnvironmentId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateWebPageFolder godoc
// @Security ApiKeyAuth
// @ID update_web_page_folder
// @Router /v1/webpage-folder [PUT]
// @Summary Update webpage folder
// @Description Update webpage folder
// @Tags WebPage
// @Accept json
// @Produce json
// @Param folder body models.UpdateFolderReqModel true "UpdateFolderReqModel"
// @Success 200 {object} status_http.Response{data=tmp.Folder} "Folder data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateWebPageFolder(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		folder tmp.UpdateFolderReq
	)

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

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//
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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	folder.EnvironmentId = environmentId.(string)
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	folder.ProjectId = projectId.(string)
	folder.ResourceId = resource.ResourceEnvironmentId

	res, err := services.WebPageService().Folder().UpdateFolder(
		context.Background(),
		&folder,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteWebPageFolder godoc
// @Security ApiKeyAuth
// @ID delete_web_page_folder
// @Router /v1/webpage-folder/{webpage-folder-id} [DELETE]
// @Summary Delete webpage folder
// @Description Delete webpage folder
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-folder-id path string true "webpage-folder-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteWebPageFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	folderId := c.Param("webpage-folder-id")

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
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
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	//
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.WebPageService().Folder().DeleteFolder(
		context.Background(),
		&tmp.DeleteFolderReq{
			Id:            folderId,
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			EnvironmentId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListWebPageFolder godoc
// @Security ApiKeyAuth
// @ID get_list_web_page_folder
// @Router /v1/webpage-folder [GET]
// @Summary Get List webpage folder
// @Description Get List webpage folder
// @Tags WebPage
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=tmp.GetListFolderRes} "FolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListWebPageFolder(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	//
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.WebPageService().Folder().GetListFolder(
		context.Background(),
		&tmp.GetListFolderReq{
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			EnvironmentId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// CreateWebPageV2 godoc
// @Security ApiKeyAuth
// @ID create_web_pageV2
// @Router /v1/webpageV2 [POST]
// @Summary Create webpage
// @Description Create webpage
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage body models.CreateWebPageReqModel true "CreateWebPageReqModel"
// @Success 201 {object} status_http.Response{data=tmp.WebPage} "WebPage data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateWebPageV2(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		webpage tmp.CreateWebPageReq
	)


	err := c.ShouldBindJSON(&webpage)
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

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}
	//
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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	webpage.EnvironmentId = environmentId.(string)
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	webpage.ProjectId = projectId.(string)
	webpage.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	webpage.CommitId = uuID.String()
	webpage.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	webpage.CommitInfo = &tmp.CommitInfo{
		Id:         "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Create api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  webpage.GetProjectId(),
	}

	res, err := services.WebPageService().WebPage().CreateWebPage(
		context.Background(),
		&webpage,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleWebPageV2 godoc
// @Security ApiKeyAuth
// @ID get_single_web_pageV2
// @Router /v1/webpageV2/{webpage-id} [GET]
// @Summary Get single webpage
// @Description Get single webpage
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-id path string true "webpage-id"
// @Param commit-id query string false "commit-id"
// @Param version-id query string false "version-id"
// @Success 200 {object} status_http.Response{data=tmp.WebPage} "WebPage"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetSingleWebPageV2(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	webPageId := c.Param("webpage-id")
	commitId := c.Query("commit_id")
	versionId := c.Query("version_id")

	if !util.IsValidUUID(webPageId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
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
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	//
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.WebPageService().WebPage().GetSingleWebPage(
		context.Background(),
		&tmp.GetSingleWebPageReq{
			Id:            webPageId,
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			VersionId:     versionId,
			CommitId:      commitId,
			EnvironmentId: environmentId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	versions, err := services.VersioningService().Release().GetMultipleVersionInfo(context.Background(), &vcs.GetMultipleVersionInfoRequest{
		VersionIds: res.CommitInfo.VersionIds,
		ProjectId:  res.ProjectId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	versionInfos := make([]*tmp.VersionInfo, 0, len(res.GetCommitInfo().GetVersionIds()))
	for _, id := range res.CommitInfo.VersionIds {
		versionInfo, ok := versions.VersionInfos[id]
		if ok {
			versionInfos = append(versionInfos, &tmp.VersionInfo{
				AuthorId:  versionInfo.AuthorId,
				CreatedAt: versionInfo.CreatedAt,
				UpdatedAt: versionInfo.UpdatedAt,
				Desc:      versionInfo.Desc,
				IsCurrent: versionInfo.IsCurrent,
				Version:   versionInfo.Version,
				VersionId: versionInfo.VersionId,
			})
		}
	}
	res.CommitInfo.VersionInfos = versionInfos

	h.handleResponse(c, status_http.OK, res)
}

// UpdateWebPageV2 godoc
// @Security ApiKeyAuth
// @ID update_web_pageV2
// @Router /v1/webpageV2 [PUT]
// @Summary Update web page
// @Description Update web page
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage body models.UpdateWebPageReqModel true "UpdateWebPageReqModel"
// @Success 200 {object} status_http.Response{data=tmp.WebPage} "WebPage data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateWebPageV2(c *gin.Context) {
	var (
		//resourceEnvironment *obs.ResourceEnvironment
		webPage tmp.WebPage
	)

	err := c.ShouldBindJSON(&webPage)
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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	webPage.EnvironmentId = environmentId.(string)
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}
	webPage.ProjectId = projectId.(string)
	webPage.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	webPage.CommitId = uuID.String()
	webPage.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	//activeVersion, err := services.VersioningService().Release().GetCurrentActive(
	//	c.Request.Context(),
	//	&vcs.GetCurrentReleaseRequest{
	//		EnvironmentId: webPage.GetEnvironmentId(),
	//	},
	//)
	//if err != nil {
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	//webPage.VersionId = activeVersion.GetVersionId()
	webPage.CommitInfo = &tmp.CommitInfo{
		Id:         "",
		CommitType: config.COMMIT_TYPE_FIELD,
		Name:       fmt.Sprintf("Auto Created Commit Update api reference - %s", time.Now().Format(time.RFC1123)),
		AuthorId:   authInfo.GetUserId(),
		ProjectId:  webPage.GetProjectId(),
	}

	res, err := services.WebPageService().WebPage().UpdateWebPage(
		context.Background(),
		&webPage,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteWebPageV2 godoc
// @Security ApiKeyAuth
// @ID delete_web_pageV2
// @Router /v1/webpageV2/{webpage-id} [DELETE]
// @Summary Delete webpage
// @Description Delete webpage
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-id path string true "webpage-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteWebPageV2(c *gin.Context) {
	var (
	//resourceEnvironment *obs.ResourceEnvironment
	)
	webPageId := c.Param("webpage-id")

	if !util.IsValidUUID(webPageId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	//
	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.WebPageService().WebPage().DeleteWebPage(
		context.Background(),
		&tmp.DeleteWebPageReq{
			Id:            webPageId,
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			VersionId:     uuid.NewString(),
			EnvironmentId: environmentId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListWebPageV2 godoc
// @Security ApiKeyAuth
// @ID get_list_web_pageV2
// @Router /v1/webpageV2 [GET]
// @Summary Get List web page
// @Description Get List web page
// @Tags WebPage
// @Accept json
// @Produce json
// @Param folder-id query string true "folder-id"
// @Param app-id query string true "app-id"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListWebPageRes} "GetListWebPageRes"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListWebPageV2(c *gin.Context) {
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
	//

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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	//if util.IsValidUUID(environmentId.(string)) {
	//	resourceEnvironment, err = services.CompanyService().Resource().GetResourceEnvironment(
	//		c.Request.Context(),
	//		&obs.GetResourceEnvironmentReq{
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
	//		&obs.GetDefaultResourceEnvironmentReq{
	//			ResourceId: resourceId.(string),
	//			ProjectId:  projectId.(string)
	//		},
	//	)
	//	if err != nil {
	//		h.handleResponse(c, status_http.GRPCError, err.Error())
	//		return
	//	}
	//}

	res, err := services.WebPageService().WebPage().GetListWebPage(
		context.Background(),
		&tmp.GetListWebPageReq{
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			EnvironmentId: environmentId.(string),
			FolderId:      c.DefaultQuery("folder-id", ""),
			AppId:         c.DefaultQuery("app-id", ""),
			Limit:         int32(limit),
			Offset:        int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// GetWebPageHistory godoc
// @Security ApiKeyAuth
// @ID get_web_page_history
// @Router /v1/webpageV2/{webpage-id}/history [GET]
// @Summary Get Api webpage history
// @Description Get webpage history
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-id path string true "webpage-id"
// @Param limit query string true "limit"
// @Param offset query string true "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetWebPageHistoryRes} "GetWebPageHistoryRes"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetWebPageHistory(c *gin.Context) {
	id := c.Param("webpage-id")

	if !util.IsValidUUID(id) {
		err := errors.New("query is an invalid uuid")
		h.log.Error("query is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "query is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.log.Error("error getting limit param", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.log.Error("error getting offset param", logger.Error(err))
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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.WebPageService().WebPage().GetWebPageChanges(
		context.Background(),
		&tmp.GetWebPageHistoryReq{
			Id:            id,
			ProjectId:     projectId.(string),
			ResourceId:    resource.ResourceEnvironmentId,
			Offset:        int64(offset),
			Limit:         int64(limit),
			EnvironmentId: environmentId.(string),
		},
	)
	if err != nil {
		h.log.Error("error getting query history", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		versionIds []string
		// autherGuids []string
	)
	for _, item := range resp.GetWebPages() {
		versionIds = append(versionIds, item.GetCommitInfo().GetVersionIds()...)
	}
	multipleVersionResp, err := services.VersioningService().Release().GetMultipleVersionInfo(
		c.Request.Context(),
		&vcs.GetMultipleVersionInfoRequest{
			VersionIds: versionIds,
			ProjectId:  projectId.(string),
		},
	)
	if err != nil {
		h.log.Error("error getting multiple version infos", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	for _, item := range resp.GetWebPages() {
		for key := range item.GetVersionInfos() {

			versionInfoData := multipleVersionResp.GetVersionInfos()[key]

			item.VersionInfos[key] = &tmp.VersionInfo{
				VersionId: versionInfoData.GetVersionId(),
				AuthorId:  versionInfoData.GetAuthorId(),
				Version:   versionInfoData.GetVersion(),
				Desc:      versionInfoData.GetDesc(),
				CreatedAt: versionInfoData.GetCreatedAt(),
				UpdatedAt: versionInfoData.GetUpdatedAt(),
				IsCurrent: versionInfoData.GetIsCurrent(),
			}
		}
	}
	// reqAutherGuids, err := helper.ConvertMapToStruct(map[string]interface{}{
	// 	"guid": []string{},
	// })
	// if err != nil {
	// 	h.log.Error("error converting map to struct", logger.Error(err))
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	// // Get User Data
	// respAuthersData, err := services.BuilderService().ObjectBuilder().GetList(
	// 	c.Request.Context(),

	// 	&builder.CommonMessage{
	// 		TableSlug: "users",
	// 		Data:      reqAutherGuids,
	// 	},
	// )
	// if err != nil {
	// 	h.log.Error("error getting user data", logger.Error(err))
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// 	return
	// }

	h.handleResponse(c, status_http.OK, resp)
}

// RevertWebPage godoc
// @Security ApiKeyAuth
// @ID revert_web_pageV2
// @Router /v1/webpageV2/{webpage-id}/revert [POST]
// @Summary Revert webpage
// @Description Revert webpage
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-id path string true "webpage-id"
// @Param RevertWebPageReq body models.RevertWebPageReqModel true "Request Body"
// @Success 200 {object} status_http.Response{data=tmp.WebPage} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RevertWebPage(c *gin.Context) {

	var body tmp.RevertWebPageReq

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("error binding json", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	id := c.Param("webpage-id")

	if !util.IsValidUUID(id) {
		err := errors.New("webpage id is an invalid uuid")
		h.log.Error("webpage is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "webpage is an invalid uuid")
		return
	}

	if !util.IsValidUUID(body.GetProjectId()) {
		err := errors.New("project_id is an invalid uuid")
		h.log.Error("project_id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "project_id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	body.EnvironmentId = environmentId.(string)
	body.ProjectId = projectId.(string)
	body.ResourceId = resource.ResourceEnvironmentId

	if !util.IsValidUUID(body.GetEnvironmentId()) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}
	versionGuid, commitGuid, err := h.CreateAutoCommitForAdminChange(c, body.GetEnvironmentId(), config.COMMIT_TYPE_FIELD, body.ProjectId)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
		return
	}
	body.VersionId = versionGuid
	body.NewCommitId = commitGuid

	resp, err := services.WebPageService().WebPage().RevertWebPage(
		context.Background(),
		&body,
	)
	if err != nil {
		h.log.Error("error reverting webpage", logger.Error(err))
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// InsertManyVersionForWebPageService godoc
// @Security ApiKeyAuth
// @ID insert_many_web_pageV2
// @Router /v1/webpageV2/select-versions/{webpage-id} [POST]
// @Summary Insert Many webpageV2
// @Description Insert Many webpageV2
// @Tags WebPage
// @Accept json
// @Produce json
// @Param webpage-id path string true "webpage-id"
// @Param ManyVersionsModel body models.ManyVersionsModel true "Request Body"
// @Success 200 {object} status_http.Response{data=empty.Empty} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) InsertManyVersionForWebPageService(c *gin.Context) {
	var (
		body tmp.ManyVersions
	)

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	log.Printf("API->body: %+v", body)

	if !util.IsValidUUID(body.GetEnvironmentId()) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}

	webPageId := c.Param("webpage-id")
	if !util.IsValidUUID(webPageId) {
		err := errors.New("webpage-id is an invalid uuid")
		h.log.Error("webpage-id is an invalid uuid", logger.Error(err))
		h.handleResponse(c, status_http.InvalidArgument, "webpage-id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.log.Error("error getting service", logger.Error(err))
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
			ServiceType:   pb.ServiceType_WEB_PAGE_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	body.EnvironmentId = environmentId.(string)
	if !util.IsValidUUID(body.GetEnvironmentId()) {
		h.handleResponse(c, status_http.BadRequest, errors.New("environment id is invalid uuid").Error())
		return
	}
	body.ProjectId = projectId.(string)
	body.ResourceId = resource.ResourceEnvironmentId

	// _, commitId, err := h.CreateAutoCommitForAdminChange(c, environmentID.(string), config.COMMIT_TYPE_FIELD, body.GetProjectId())
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err).Error())
	// 	return
	// }

	resp, err := services.WebPageService().WebPage().CreateManyWebPage(c.Request.Context(), &body)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
