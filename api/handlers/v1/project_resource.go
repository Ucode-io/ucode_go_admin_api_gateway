package v1

import (
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// AddResourceToProject godoc
// @Security ApiKeyAuth
// @ID add_resource_to_project
// @Router /v2/company/project/resource [POST]
// @Summary Add rosource to project
// @Description Add rosource to project
// @Tags Project resource
// @Accept json
// @Produce json
// @Param data body pb.AddResourceToProjectRequest true "AddResourceToProjectRequest"
// @Success 201 {object} status_http.Response{data=pb.ProjectResource} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) AddResourceToProject(c *gin.Context) {
	var (
		request = &pb.AddResourceToProjectRequest{}
		resp    = &pb.ProjectResource{}
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return
	}

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)

	if request.GetType() == pb.ResourceType_GOOGLE_DRIVE {
		if request.GetSettings() == nil || request.GetSettings().GetGoogleDrive() == nil || strings.TrimSpace(request.GetSettings().GetGoogleDrive().GetRefreshToken()) == "" {
			h.HandleResponse(c, status_http.BadRequest, "use /v1/google-drive/connect to connect Google Drive")
			return
		}

		if request.Name == "" {
			request.Name = "Google Drive"
		}
		request.Settings = sanitizeGoogleDriveSettingsForStorage(request.GetSettings(), h.baseConf.GoogleDriveVisibility)
	}

	resp, err := h.companyServices.Resource().AddResourceToProject(c.Request.Context(), request)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if request.GetType() == pb.ResourceType_GOOGLE_DRIVE {
		resp.Settings = request.GetSettings()
		resp.ResourceType = int32(request.GetType())
		sanitizeGoogleDriveResourceForResponse(resp)
	}

	h.HandleResponse(c, status_http.Created, resp)
}

// UpdateProjectResource godoc
// @Security ApiKeyAuth
// @ID update_project_resource
// @Router /v2/company/project/resource [PUT]
// @Summary Update Project resource
// @Description update Project resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param Company body pb.ProjectResource  true "ProjectResource"
// @Success 200 {object} status_http.Response{data=pb.Empty} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateProjectResource(c *gin.Context) {
	var (
		request = &pb.ProjectResource{}
		resp    = &pb.Empty{}
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(), &pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)

	if shouldSanitizeGoogleDriveSettings(request) {
		current, err := h.companyServices.Resource().GetSingleProjectResouece(c.Request.Context(), &pb.PrimaryKeyProjectResource{
			Id:            request.GetId(),
			ProjectId:     request.GetProjectId(),
			EnvironmentId: request.GetEnvironmentId(),
		})
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		if current.GetType() == pb.ResourceType_GOOGLE_DRIVE.String() || request.GetType() == pb.ResourceType_GOOGLE_DRIVE.String() || request.GetResourceType() == int32(pb.ResourceType_GOOGLE_DRIVE) {
			var currentDrive *pb.GoogleDriveCredentials
			if current.GetSettings() != nil {
				currentDrive = current.GetSettings().GetGoogleDrive()
			}
			var requestDrive *pb.GoogleDriveCredentials
			if request.GetSettings() != nil {
				requestDrive = request.GetSettings().GetGoogleDrive()
			}

			folderID := ""
			if currentDrive != nil {
				folderID = currentDrive.GetFolderId()
			}
			if folderID == "" && requestDrive != nil {
				folderID = requestDrive.GetFolderId()
			}
			if strings.TrimSpace(folderID) == "" {
				h.HandleResponse(c, status_http.BadRequest, "google drive resource folder_id is empty")
				return
			}

			visibility := strings.TrimSpace(h.baseConf.GoogleDriveVisibility)
			if visibility == "" {
				visibility = "private"
			}
			if requestDrive != nil && strings.TrimSpace(requestDrive.GetVisibility()) != "" {
				visibility = requestDrive.GetVisibility()
			}

			refreshToken := ""
			authType := "oauth"
			if currentDrive != nil {
				refreshToken = currentDrive.GetRefreshToken()
				if strings.TrimSpace(currentDrive.GetAuthType()) != "" {
					authType = currentDrive.GetAuthType()
				}
			}
			if requestDrive != nil {
				if strings.TrimSpace(requestDrive.GetRefreshToken()) != "" {
					refreshToken = requestDrive.GetRefreshToken()
				}
				if strings.TrimSpace(requestDrive.GetAuthType()) != "" {
					authType = requestDrive.GetAuthType()
				}
			}

			request.Type = pb.ResourceType_GOOGLE_DRIVE.String()
			request.ResourceType = int32(pb.ResourceType_GOOGLE_DRIVE)
			request.Settings = &pb.Settings{
				GoogleDrive: &pb.GoogleDriveCredentials{
					AuthType:     authType,
					FolderId:     folderID,
					Visibility:   visibility,
					RefreshToken: refreshToken,
				},
			}
		}
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "UPDATE",
			UserInfo:     cast.ToString(userId),
			Request:      googleDriveProjectResourceForLog(request),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.HandleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.HandleResponse(c, status_http.OK, resp)
		}
		go h.versionHistory(logReq)
	}()

	resp, err = h.companyServices.Resource().UpdateProjectResource(c.Request.Context(), request)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, resp)
}

func shouldSanitizeGoogleDriveSettings(resource *pb.ProjectResource) bool {
	if resource == nil {
		return false
	}
	if resource.GetType() == pb.ResourceType_GOOGLE_DRIVE.String() || resource.GetResourceType() == int32(pb.ResourceType_GOOGLE_DRIVE) {
		return true
	}
	return resource.GetSettings() != nil && resource.GetSettings().GetGoogleDrive() != nil
}

func sanitizeGoogleDriveSettingsForStorage(settings *pb.Settings, defaultVisibility string) *pb.Settings {
	if settings == nil || settings.GetGoogleDrive() == nil {
		return &pb.Settings{}
	}

	visibility := strings.TrimSpace(defaultVisibility)
	if visibility == "" {
		visibility = "private"
	}
	if settings != nil && settings.GetGoogleDrive() != nil && strings.TrimSpace(settings.GetGoogleDrive().GetVisibility()) != "" {
		visibility = settings.GetGoogleDrive().GetVisibility()
	}

	drive := settings.GetGoogleDrive()
	authType := strings.TrimSpace(drive.GetAuthType())
	if authType == "" {
		authType = "oauth"
	}

	return &pb.Settings{
		GoogleDrive: &pb.GoogleDriveCredentials{
			AuthType:     authType,
			FolderId:     strings.TrimSpace(drive.GetFolderId()),
			Visibility:   visibility,
			RefreshToken: strings.TrimSpace(drive.GetRefreshToken()),
		},
	}
}

func sanitizeGoogleDriveResourceForResponse(resource *pb.ProjectResource) {
	if resource == nil || resource.GetSettings() == nil || resource.GetSettings().GetGoogleDrive() == nil {
		return
	}

	drive := resource.GetSettings().GetGoogleDrive()
	resource.Settings.GoogleDrive = &pb.GoogleDriveCredentials{
		AuthType:   drive.GetAuthType(),
		FolderId:   drive.GetFolderId(),
		Visibility: drive.GetVisibility(),
	}
}

func googleDriveProjectResourceForLog(resource *pb.ProjectResource) *pb.ProjectResource {
	if resource == nil || resource.GetSettings() == nil || resource.GetSettings().GetGoogleDrive() == nil {
		return resource
	}

	safe := *resource
	drive := resource.GetSettings().GetGoogleDrive()
	safe.Settings = &pb.Settings{
		GoogleDrive: &pb.GoogleDriveCredentials{
			AuthType:   drive.GetAuthType(),
			FolderId:   drive.GetFolderId(),
			Visibility: drive.GetVisibility(),
		},
	}
	return &safe
}

// GetListProjectResource godoc
// @Security ApiKeyAuth
// @ID get_list_project_resource
// @Router /v2/company/project/resource [GET]
// @Summary Get list project resource
// @Description Get list project resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param type query string false "type"
// @Success 200 {object} status_http.Response{data=pb.ListProjectResource} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListProjectResourceList(c *gin.Context) {
	var request = &pb.GetProjectResourceListRequest{}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return
	}

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)

	if c.DefaultQuery("type", "") != "" && pb.ResourceType(pb.ResourceType_value[c.DefaultQuery("type", "")]) != 0 {
		request.Type = pb.ResourceType(pb.ResourceType_value[c.DefaultQuery("type", "")])
	}

	resp, err := h.companyServices.Resource().GetProjectResourceList(c.Request.Context(), request)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, resource := range resp.GetResources() {
		sanitizeGoogleDriveResourceForResponse(resource)
	}

	h.HandleResponse(c, status_http.OK, resp)
}

// GetProjectResourceByID godoc
// @Security ApiKeyAuth
// @ID get_single_project_resource
// @Router /v2/company/project/resource/{id} [GET]
// @Summary Get single variable resource
// @Description Get single variable resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=pb.ProjectResource} "ProjectResource"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingleProjectResource(c *gin.Context) {
	request := &pb.PrimaryKeyProjectResource{}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return
	}

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)
	request.Id = c.Param("id")

	resp, err := h.companyServices.Resource().GetSingleProjectResouece(c.Request.Context(), request)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	sanitizeGoogleDriveResourceForResponse(resp)

	h.HandleResponse(c, status_http.OK, resp)
}

// DeletevariableResource godoc
// @Security ApiKeyAuth
// @ID delete_project_resource
// @Router /v2/company/project/resource/{id} [DELETE]
// @Summary Delete variable resource
// @Description Delete variable resource
// @Tags Project resource
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteProjectResource(c *gin.Context) {
	var (
		request = &pb.PrimaryKeyProjectResource{}
		resp    = &pb.Empty{}
	)

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, "error getting environment id | not valid")
		return
	}

	userId, _ := c.Get("user_id")

	request.ProjectId = projectId.(string)
	request.EnvironmentId = environmentId.(string)
	request.Id = c.Param("id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(), &pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		logReq = &models.CreateVersionHistoryRequest{
			Services:     services,
			NodeType:     resource.NodeType,
			ProjectId:    resource.ResourceEnvironmentId,
			ActionSource: c.Request.URL.String(),
			ActionType:   "DELETE",
			UserInfo:     cast.ToString(userId),
		}
	)

	defer func() {
		if err != nil {
			logReq.Response = err.Error()
			h.HandleResponse(c, status_http.GRPCError, err.Error())
		} else {
			h.HandleResponse(c, status_http.NoContent, resp)
		}
		go h.versionHistory(logReq)
	}()

	resp, err = h.companyServices.Resource().DeleteProjectResource(c.Request.Context(), request)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.NoContent, nil)
}
