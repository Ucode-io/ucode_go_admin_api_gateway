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
// @Param body body npb.CreateUserTokenRequest true "Request Body"
// @Success 201 {object} status_http.Response{data=npb.CreateUserTokenResponse} "Response Body"
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

	// EnvironmentId, _ := c.Get("environment_id")
	// if !util.IsValidUUID(EnvironmentId.(string)) {
	// 	h.handleResponse(c, status_http.BadRequest, "environment_id not found")
	// 	return
	// }

	// ProjectId := c.Query("project-id")
	// if !util.IsValidUUID(ProjectId) {
	// 	h.handleResponse(c, status_http.BadRequest, "project-id not found")
	// 	return
	// }
	resp, err := services.NotificationService().Notification().CreateUserToken(c.Request.Context(), &req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// CreateNotificationUsers godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create-user-notifications
// @Router /v1/notification [POST]
// @Summary Create User Notifications
// @Description Create User Notifications
// @Tags Notification
// @Accept json
// @Produce json
// @Param project-id query string false "project-id"
// @Param body body npb.CreateNotificationManyUserRequest true "Request Body"
// @Success 201 {object} status_http.Response{data=npb.NotificationUsers} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateNotificationUsers(c *gin.Context) {

	req := &npb.CreateNotificationManyUserRequest{}

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

	hasAccess, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	err = c.ShouldBindJSON(req)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	req.SenderId = hasAccess.GetUserId()
	req.EnvironmentId = EnvironmentId.(string)
	req.ProjectId = ProjectId

	resp, err := services.NotificationService().Notification().CreateNotificationUsers(
		c.Request.Context(),
		req,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetAllNotification godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get-all-notifications
// @Router /v1/notification [GET]
// @Summary Get All Notifications
// @Description Get All Notifications
// @Tags Notification
// @Accept json
// @Produce json
// @Param category-id query string false "category-id"
// @Param offset query int false "offset"
// @Param limit query int false "limit"
// @Success 200 {object} status_http.Response{data=npb.GetAllNotificationsResponse} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllNotifications(c *gin.Context) {

	var (
		req    npb.GetAllNotificationsRequest
		offset int
		limit  int
	)

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

	offset, err = h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	limit, err = h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	req.Limit = int32(limit)
	req.Offset = int32(offset)
	req.ProjectId = ProjectId
	req.EnvironmentId = EnvironmentId.(string)
	req.CategoryId = c.Query("category-id")

	resp, err := services.NotificationService().Notification().GetAllNotification(c.Request.Context(), &req)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetNotificationById godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID get-notification-by-id
// @Router /v1/notification/{id} [GET]
// @Summary Get Notification By Id
// @Description Get Notification By Id
// @Tags Notification
// @Accept json
// @Produce json
// @Param id path string true "Notification Id"
// @Success 201 {object} status_http.Response{data=npb.Notification} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetNotificationById(c *gin.Context) {

	// get notification
	notificationId := c.Param("id")
	if !util.IsValidUUID(notificationId) {
		h.handleResponse(c, status_http.BadRequest, "notification id not found")
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

	req := &npb.GetSingleNotificationRequest{
		Id: notificationId,
	}

	resp, err := services.NotificationService().Notification().GetSingleNotification(c.Request.Context(), req)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
