package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/notification_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateCategory godoc
// @Security ApiKeyAuth
// @ID create_category_for_notification
// @Router /v1/notification/category [POST]
// @Summary Create category
// @Description Create category
// @Tags Notification-Category
// @Accept json
// @Produce json
// @Param app body models.CreateCategory true "Request body"
// @Success 201 {object} status_http.Response{data=models.Category} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateCategoryNotification(c *gin.Context) {
	var (
		req models.CreateCategory
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resp, err := services.NotificationService().Category().Create(
		context.Background(),
		&pb.CreateCategoryRequest{
			Name:          req.Name,
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			ParentId:      req.ParentID,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetCategory godoc
// @Security ApiKeyAuth
// @ID get_single_category_for_notification
// @Router /v1/notification/category/{id} [GET]
// @Summary Get single category
// @Description Get single category
// @Tags Notification-Category
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=models.Category} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetCategoryNotification(c *gin.Context) {

	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "id is invalid uuid")
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resp, err := services.NotificationService().Category().Get(
		context.Background(),
		&pb.GetCategoryRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			Guid:          id,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListCategory godoc
// @Security ApiKeyAuth
// @ID get_list_category_for_notification
// @Router /v1/notification/category [GET]
// @Summary Get list category
// @Description Get list category
// @Tags Notification-Category
// @Accept json
// @Produce json
// @Param offset query string false "offset"
// @Param limit query string false "limit"
// @Param page query string false "page"
// @Success 200 {object} status_http.Response{data=models.GetAllCategoriesResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListCategoryNotification(c *gin.Context) {

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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resp, err := services.NotificationService().Category().GetList(
		c.Request.Context(),
		&pb.GetListCategoryRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateCategory godoc
// @Security ApiKeyAuth
// @ID update_category_for_notification
// @Router /v1/notification/category [PUT]
// @Summary Update category
// @Description Update category
// @Tags Notification-Category
// @Accept json
// @Produce json
// @Param body body models.UpdateCategoryRequest true "body"
// @Success 200 {object} status_http.Response{data=models.Category} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateCategoryNotification(c *gin.Context) {

	var req models.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.NotificationService().Category().Update(
		context.Background(),
		&pb.Category{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			Guid:          req.Id,
			Name:          req.Name,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteCategory godoc
// @Security ApiKeyAuth
// @ID delete_category_for_notification
// @Router /v1/notification/category/{id} [DELETE]
// @Summary Delete category
// @Description Delete category
// @Tags Notification-Category
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=string} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteCategoryNotification(c *gin.Context) {

	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "id is invalid uuid")
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

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resp, err := services.NotificationService().Category().Delete(
		c.Request.Context(),
		&pb.DeleteCategoryRequest{
			ProjectId:     projectId.(string),
			EnvironmentId: EnvironmentId.(string),
			Guid:          id,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
