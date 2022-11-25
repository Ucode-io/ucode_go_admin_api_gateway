package handlers

import (
	"ucode/ucode_go_api_gateway/api/http"

	"ucode/ucode_go_api_gateway/genproto/auth_service"

	"github.com/saidamir98/udevs_pkg/util"

	"github.com/gin-gonic/gin"
)

// V2AddRole godoc
// @ID create_role
// @Router /v2/role [POST]
// @Summary Create Role
// @Description Create Role
// @Tags V2_Role
// @Accept json
// @Produce json
// @Param role body auth_service.AddRoleRequest true "AddRoleRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "Role data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2AddRole(c *gin.Context) {
	var role auth_service.AddRoleRequest

	err := c.ShouldBindJSON(&role)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.PermissionServiceAuth().V2AddRole(
		c.Request.Context(),
		&role,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2GetRoleById godoc
// @ID get_role_by_id
// @Router /v2/role/{role-id} [GET]
// @Summary Get Role By ID
// @Description Get Role By ID
// @Tags V2_Role
// @Accept json
// @Produce json
// @Param role-id path string true "role-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "ClientTypeBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetRoleByID(c *gin.Context) {
	roleId := c.Param("role-id")
	if !util.IsValidUUID(roleId) {
		h.handleResponse(c, http.InvalidArgument, "role id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2GetRoleById(c.Request.Context(), &auth_service.RolePrimaryKey{
		Id: roleId,
	})

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2GetRolesList godoc
// @ID get_roles_list
// @Router /v2/role [GET]
// @Summary Get Roles List
// @Description  Get Roles List
// @Tags V2_Role
// @Accept json
// @Produce json
// @Param offset query integer false "offset"
// @Param limit query integer false "limit"
// @Param client-platform-id query string false "client-platform-id"
// @Param client-type-id query string false "client-type-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "GetRolesListResponseBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetRolesList(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2GetRolesList(
		c.Request.Context(),
		&auth_service.GetRolesListRequest{
			Offset:           uint32(offset),
			Limit:            uint32(limit),
			ClientPlatformId: c.Query("client-platform-id"),
			ClientTypeId:     c.Query("client-type-id"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2UpdateRole godoc
// @ID update_role
// @Router /v2/role [PUT]
// @Summary Update Role
// @Description Update Role
// @Tags V2_Role
// @Accept json
// @Produce json
// @Param role body auth_service.UpdateRoleRequest true "UpdateRoleRequestBody"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "Role data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateRole(c *gin.Context) {
	var role auth_service.UpdateRoleRequest

	err := c.ShouldBindJSON(&role)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2UpdateRole(
		c.Request.Context(),
		&role,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2RemoveRole godoc
// @ID delete_role
// @Router /v2/role/{role-id} [DELETE]
// @Summary Delete Role
// @Description Get Role
// @Tags V2_Role
// @Accept json
// @Produce json
// @Param role-id path string true "role-id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2RemoveRole(c *gin.Context) {
	roleID := c.Param("role-id")

	if !util.IsValidUUID(roleID) {
		h.handleResponse(c, http.InvalidArgument, "role id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.PermissionServiceAuth().V2RemoveRole(
		c.Request.Context(),
		&auth_service.RolePrimaryKey{
			Id: roleID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2CreatePermission godoc
// @ID create_permission
// @Router /v2/permission [POST]
// @Summary Create Permission
// @Description Create Permission
// @Tags V2_Permission
// @Accept json
// @Produce json
// @Param permission body auth_service.CreatePermissionRequest true "CreatePermissionRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "Permission data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2CreatePermission(c *gin.Context) {
	var permission auth_service.CreatePermissionRequest

	err := c.ShouldBindJSON(&permission)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.PermissionServiceAuth().V2CreatePermission(
		c.Request.Context(),
		&permission,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2GetPermissionList godoc
// @ID get_permission_list
// @Router /v2/permission [GET]
// @Summary Get Permission List
// @Description  Get Permission List
// @Tags V2_Permission
// @Accept json
// @Produce json
// @Param offset query integer false "offset"
// @Param limit query integer false "limit"
// @Param search query string false "search"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "GetPermissionListResponseBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetPermissionList(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2GetPermissionList(
		c.Request.Context(),
		&auth_service.GetPermissionListRequest{
			Limit:  int32(limit),
			Offset: int32(offset),
			Search: c.Query("search"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2GetPermissionByID godoc
// @ID get_permission_by_id
// @Router /v2/permission/{permission-id} [GET]
// @Summary Get Permission By ID
// @Description Get Permission By ID
// @Tags V2_Permission
// @Accept json
// @Produce json
// @Param permission-id path string true "permission-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "PermissionBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetPermissionByID(c *gin.Context) {
	permissionID := c.Param("permission-id")

	if !util.IsValidUUID(permissionID) {
		h.handleResponse(c, http.InvalidArgument, "permission id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2GetPermissionByID(
		c.Request.Context(),
		&auth_service.PermissionPrimaryKey{
			Id: permissionID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2UpdatePermission godoc
// @ID update_permission
// @Router /v2/permission [PUT]
// @Summary Update Permission
// @Description Update Permission
// @Tags V2_Permission
// @Accept json
// @Produce json
// @Param permission body auth_service.UpdatePermissionRequest true "UpdatePermissionRequestBody"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "Permission data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdatePermission(c *gin.Context) {
	var permission auth_service.UpdatePermissionRequest

	err := c.ShouldBindJSON(&permission)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2UpdatePermission(
		c.Request.Context(),
		&permission,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2DeletePermission godoc
// @ID delete_permission
// @Router /v2/permission/{permission-id} [DELETE]
// @Summary Delete Permission
// @Description Get Permission
// @Tags V2_Permission
// @Accept json
// @Produce json
// @Param permission-id path string true "permission-id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2DeletePermission(c *gin.Context) {
	permissionID := c.Param("permission-id")

	if !util.IsValidUUID(permissionID) {
		h.handleResponse(c, http.InvalidArgument, "permission id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2DeletePermission(
		c.Request.Context(),
		&auth_service.PermissionPrimaryKey{
			Id: permissionID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2GetScopesList godoc
// @ID get_scopes_list
// @Router /v2/scope [GET]
// @Summary Get Scopes List
// @Description  Get Scopes List
// @Tags V2_Scope
// @Accept json
// @Produce json
// @Param offset query integer false "offset"
// @Param limit query integer false "limit"
// @Param client-platform-id query string true "client-platform-id"
// @Param search query string false "search"
// @Param order_by query string false "order_by"
// @Param order_type query string false "order_type"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "GetScopesListResponseBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetScopesList(c *gin.Context) {
	clientPlatformID := c.Query("client-platform-id")
	if !util.IsValidUUID(clientPlatformID) {
		h.handleResponse(c, http.InvalidArgument, "Client platform id is an invalid uuid")
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2GetScopeList(
		c.Request.Context(),
		&auth_service.GetScopeListRequest{
			Offset:           uint32(offset),
			Limit:            uint32(limit),
			Search:           c.Query("search"),
			OrderBy:          c.Query("order_by"),
			OrderType:        c.Query("order_type"),
			ClientPlatformId: clientPlatformID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2AddPermissionScope godoc
// @ID add_permission_scope
// @Router /v2/permission-scope [POST]
// @Summary Create PermissionScope
// @Description Create PermissionScope
// @Tags V2_PermissionScope
// @Accept json
// @Produce json
// @Param permission-scope body auth_service.AddPermissionScopeRequest true "AddPermissionScopeRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "PermissionScope data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2AddPermissionScope(c *gin.Context) {
	var permission_scope auth_service.AddPermissionScopeRequest

	err := c.ShouldBindJSON(&permission_scope)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2AddPermissionScope(
		c.Request.Context(),
		&permission_scope,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2RemovePermissionScope godoc
// @ID delete_permission_scope
// @Router /v2/permission-scope [DELETE]
// @Summary Delete PermissionScope
// @Description Get PermissionScope
// @Tags V2_PermissionScope
// @Accept json
// @Produce json
// @Param permission-scope body auth_service.PermissionScopePrimaryKey true "PermissionScopePrimaryKeyBody"
// @Success 204
// @Response 400 {object} http.Response{data=auth_service.CommonMessage} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2RemovePermissionScope(c *gin.Context) {
	var permission_scope auth_service.PermissionScopePrimaryKey

	err := c.ShouldBindJSON(&permission_scope)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2RemovePermissionScope(
		c.Request.Context(),
		&permission_scope,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2AddRolePermission godoc
// @ID add_role_permission
// @Router /v2/role-permission [POST]
// @Summary Create RolePermission
// @Description Create RolePermission
// @Tags V2_RolePermission
// @Accept json
// @Produce json
// @Param role-permission body auth_service.AddRolePermissionRequest true "AddRolePermissionRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "RolePermission data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2AddRolePermission(c *gin.Context) {
	var role_permission auth_service.AddRolePermissionRequest

	err := c.ShouldBindJSON(&role_permission)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2AddRolePermission(
		c.Request.Context(),
		&role_permission,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2RemoveRolePermission godoc
// @ID delete_role_permission
// @Router /v2/role-permission [DELETE]
// @Summary Delete RolePermission
// @Description Get RolePermission
// @Tags V2_RolePermission
// @Accept json
// @Produce json
// @Param role-permission body auth_service.RolePermissionPrimaryKey true "RolePermissionPrimaryKeyBody"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2RemoveRolePermission(c *gin.Context) {
	var role_permission auth_service.RolePermissionPrimaryKey

	err := c.ShouldBindJSON(&role_permission)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.PermissionServiceAuth().V2RemoveRolePermission(
		c.Request.Context(),
		&role_permission,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
