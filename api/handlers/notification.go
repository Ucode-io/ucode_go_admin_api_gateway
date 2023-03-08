package handlers

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	npb "ucode/ucode_go_api_gateway/genproto/notification_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateUserToken godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create-user-token
// @Router /v1/notification/user-fcmtoken [POST]
// @Summary Create User Token
// @Description Create User Token
// @Tags Notification
// @Accept json
// @Produce json
// @Param Note_Folder body npb.CreateUserTokenRequest true "Request Body"
// @Success 201 {object} status_http.Response{data=tmp.FolderNote} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateUserFCMToken(c *gin.Context) {

	var (
		req npb.CreateUserTokenRequest
	)
	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	EnvironmentId, _ := c.Get("environment_id")
	if !util.IsValidUUID(EnvironmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	ProjectId := c.Query("project-id")
	if !util.IsValidUUID(ProjectId) {
		h.handleResponse(c, status_http.BadRequest, "project-id not found")
		return
	}
	resp, err := services.NotificationService().Notification().CreateUserToken(c.Request.Context(), &req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
