package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	authPb "ucode/ucode_go_api_gateway/genproto/auth_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"

	"github.com/gin-gonic/gin"
)

// UpsertPermissionsByAppId godoc
// @Security ApiKeyAuth
// @ID upsert_permission
// @Router /v1/permission-upsert/{app_id} [POST]
// @Summary Upsert permissions
// @Description Upsert permissions
// @Tags Permission
// @Accept json
// @Produce json
// @Param app_id path string true "app_id"
// @Param object body models.CommonMessage true "UpsertPermissionRequestBody"
// @Success 201 {object} http.Response{data=models.CommonMessage} "Upsert Permission data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpsertPermissionsByAppId(c *gin.Context) {
	var objectRequest models.CommonMessage

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.services.PermissionService().UpsertPermissionsByAppId(
		context.Background(),
		&obs.UpsertPermissionsByAppIdRequest{
			AppId: c.Param("app_id"),
			Data:  structData,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}
	if objectRequest.Data["role_id"] == nil {
		err := errors.New("role id must be have in update permission")
		h.handleResponse(c, http.BadRequest, err.Error())
	}
	_, err = h.services.SessionService().UpdateSessionsByRoleId(context.Background(), &authPb.UpdateSessionByRoleIdRequest{
		RoleId:    objectRequest.Data["role_id"].(string),
		IsChanged: true,
	})
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetAllPermissionByRoleId godoc
// @Security ApiKeyAuth
// @ID get_all_permission_by_role_id
// @Router /v1/permission-get-all/{role_id} [GET]
// @Summary Get all permissions by role id
// @Description Get all permissions by role id
// @Tags Permission
// @Accept json
// @Produce json
// @Param role_id path string true "role_id"
// @Success 200 {object} http.Response{data=models.CommonMessage} "Get All Permission data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllPermissionByRoleId(c *gin.Context) {

	resp, err := h.services.PermissionService().GetAllPermissionsByRoleId(
		context.Background(),
		&obs.GetAllPermissionRequest{
			RoleId: c.Param("role_id"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetFieldPermissions godoc
// @Security ApiKeyAuth
// @ID get_all_field_permission
// @Router /v1/field-permission/{role_id}/{table_slug} [GET]
// @Summary Get all field permissions
// @Description Get all field permissions
// @Tags Permission
// @Accept json
// @Produce json
// @Param role_id path string true "role_id"
// @Param table_slug path string true "table_slug"
// @Success 200 {object} http.Response{data=models.CommonMessage} "Get All Field Permission data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetFieldPermissions(c *gin.Context) {

	resp, err := h.services.PermissionService().GetFieldPermissions(
		context.Background(),
		&obs.GetFieldPermissionRequest{
			RoleId:    c.Param("role_id"),
			TableSlug: c.Param(("table_slug")),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
