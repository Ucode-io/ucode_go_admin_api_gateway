package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"

	"ucode/ucode_go_api_gateway/genproto/auth_service"

	"github.com/saidamir98/udevs_pkg/util"

	"github.com/gin-gonic/gin"
)

// V2CreateClientPlatform godoc
// @ID create_client_platform
// @Router /v2/client-platform [POST]
// @Summary Create ClientPlatform
// @Description Create ClientPlatform
// @Tags V2_ClientPlatform
// @Accept json
// @Produce json
// @Param client-platform body auth_service.CreateClientPlatformRequest true "CreateClientPlatformRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "ClientPlatform data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2CreateClientPlatform(c *gin.Context) {
	var client_platform auth_service.CreateClientPlatformRequest

	err := c.ShouldBindJSON(&client_platform)
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

	resp, err := services.ClientServiceAuth().V2CreateClientPlatform(
		context.Background(),
		&client_platform,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2GetClientPlatformList godoc
// @ID get_client_platform_list
// @Router /v2/client-platform [GET]
// @Summary Get ClientPlatform List
// @Description  Get ClientPlatform List
// @Tags V2_ClientPlatform
// @Accept json
// @Produce json
// @Param offset query integer false "offset"
// @Param limit query integer false "limit"
// @Param search query string false "search"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "GetClientPlatformListResponseBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetClientPlatformList(c *gin.Context) {
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
	resp, err := services.ClientServiceAuth().V2GetClientPlatformList(
		c.Request.Context(),
		&auth_service.GetClientPlatformListRequest{
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

// V2GetClientPlatformByID godoc
// @ID get_client_platform_by_id
// @Router /v2/client-platform/{client-platform-id} [GET]
// @Summary Get ClientPlatform By ID
// @Description Get ClientPlatform By ID
// @Tags V2_ClientPlatform
// @Accept json
// @Produce json
// @Param client-platform-id path string true "client-platform-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "ClientPlatformBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetClientPlatformByID(c *gin.Context) {
	client_platformID := c.Param("client-platform-id")

	if !util.IsValidUUID(client_platformID) {
		h.handleResponse(c, http.InvalidArgument, "client_platform id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ClientService().V2GetClientPlatformByID(
		c.Request.Context(),
		&auth_service.ClientPlatformPrimaryKey{
			Id: client_platformID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// v2GetClientPlatformByID godoc
// @ID get_client_platform_detailed_by_id
// @Router /v2/client-platform-detailed/{client-platform-id} [GET]
// @Summary Get ClientPlatform By ID Detailed
// @Description Get ClientPlatform By ID Detailed
// @Tags V2_ClientPlatform
// @Accept json
// @Produce json
// @Param client-platform-id path string true "client-platform-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "ClientPlatformDetailedBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetClientPlatformByIDDetailed(c *gin.Context) {
	client_platformID := c.Param("client-platform-id")

	if !util.IsValidUUID(client_platformID) {
		h.handleResponse(c, http.InvalidArgument, "client_platform id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ClientService().V2GetClientPlatformByIDDetailed(
		c.Request.Context(),
		&auth_service.ClientPlatformPrimaryKey{
			Id: client_platformID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2UpdateClientPlatform godoc
// @ID update_client_platform
// @Router /v2/client-platform [PUT]
// @Summary Update ClientPlatform
// @Description Update ClientPlatform
// @Tags V2_ClientPlatform
// @Accept json
// @Produce json
// @Param client-platform body auth_service.UpdateClientPlatformRequest true "UpdateClientPlatformRequestBody"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "ClientPlatform data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateClientPlatform(c *gin.Context) {
	var client_platform auth_service.UpdateClientPlatformRequest

	err := c.ShouldBindJSON(&client_platform)
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
	resp, err := services.ClientService().V2UpdateClientPlatform(
		c.Request.Context(),
		&client_platform,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2DeleteClientPlatform godoc
// @ID delete_client_platform
// @Router /v2/client-platform/{client-platform-id} [DELETE]
// @Summary Delete ClientPlatform
// @Description Get ClientPlatform
// @Tags V2_ClientPlatform
// @Accept json
// @Produce json
// @Param client-platform-id path string true "client-platform-id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2DeleteClientPlatform(c *gin.Context) {
	client_platformID := c.Param("client-platform-id")

	if !util.IsValidUUID(client_platformID) {
		h.handleResponse(c, http.InvalidArgument, "client_platform id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.ClientService().V2DeleteClientPlatform(
		c.Request.Context(),
		&auth_service.ClientPlatformPrimaryKey{
			Id: client_platformID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2CreateClientType godoc
// @ID create_client_type
// @Router /v2/client-type [POST]
// @Summary Create ClientType
// @Description Create ClientType
// @Tags V2_ClientType
// @Accept json
// @Produce json
// @Param client-type body models.CreateClientTypeRequest true "CreateClientTypeRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "ClientType data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2CreateClientType(c *gin.Context) {
	var client_type auth_service.CreateClientTypeRequest

	err := c.ShouldBindJSON(&client_type)
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
	resp, err := services.ClientService().V2CreateClientType(
		c.Request.Context(),
		&client_type,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2GetClientTypeList godoc
// @ID get_client_type_list
// @Router /v2/client-type [GET]
// @Summary Get ClientType List
// @Description  Get ClientType List
// @Tags V2_ClientType
// @Accept json
// @Produce json
// @Param offset query integer false "offset"
// @Param limit query integer false "limit"
// @Param search query string false "search"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "GetClientTypeListResponseBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetClientTypeList(c *gin.Context) {
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
	resp, err := services.ClientService().V2GetClientTypeList(
		c.Request.Context(),
		&auth_service.GetClientTypeListRequest{
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

// V2GetClientTypeByID godoc
// @ID get_client_type_by_id
// @Router /v2/client-type/{client-type-id} [GET]
// @Summary Get ClientType By ID
// @Description Get ClientType By ID
// @Tags V2_ClientType
// @Accept json
// @Produce json
// @Param client-type-id path string true "client-type-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "ClientTypeBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetClientTypeByID(c *gin.Context) {
	client_typeID := c.Param("client-type-id")

	if !util.IsValidUUID(client_typeID) {
		h.handleResponse(c, http.InvalidArgument, "client_type id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.ClientService().V2GetClientTypeByID(
		c.Request.Context(),
		&auth_service.ClientTypePrimaryKey{
			Id: client_typeID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2UpdateClientType godoc
// @ID update_client_type
// @Router /v2/client-type [PUT]
// @Summary Update ClientType
// @Description Update ClientType
// @Tags V2_ClientType
// @Accept json
// @Produce json
// @Param client-type body models.UpdateClientTypeRequest true "UpdateClientTypeRequestBody"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "ClientType data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateClientType(c *gin.Context) {
	var client_type auth_service.UpdateClientTypeRequest

	err := c.ShouldBindJSON(&client_type)
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
	resp, err := services.ClientService().V2UpdateClientType(
		c.Request.Context(),
		&client_type,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2DeleteClientType godoc
// @ID delete_client_type
// @Router /v2/client-type/{client-type-id} [DELETE]
// @Summary Delete ClientType
// @Description Get ClientType
// @Tags V2_ClientType
// @Accept json
// @Produce json
// @Param client-type-id path string true "client-type-id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2DeleteClientType(c *gin.Context) {
	client_typeID := c.Param("client-type-id")

	if !util.IsValidUUID(client_typeID) {
		h.handleResponse(c, http.InvalidArgument, "client_type id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.ClientService().V2DeleteClientType(
		c.Request.Context(),
		&auth_service.ClientTypePrimaryKey{
			Id: client_typeID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2AddClient godoc
// @ID create_client
// @Router /v2/client [POST]
// @Summary Create Client
// @Description Create Client
// @Tags V2_Client
// @Accept json
// @Produce json
// @Param client body auth_service.AddClientRequest true "AddClientRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "Client data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2AddClient(c *gin.Context) {
	var client auth_service.AddClientRequest

	err := c.ShouldBindJSON(&client)
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
	resp, err := services.ClientService().V2AddClient(
		c.Request.Context(),
		&client,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2GetClientMatrix godoc
// @ID get_client_matrix
// @Router /v2/client/{project-id} [GET]
// @Summary Get Client Matrix
// @Description Get Client Matrix
// @Tags V2_Client
// @Accept json
// @Produce json
// @Param project-id path string true "project-id"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "GetClientMatrixBody"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2GetClientMatrix(c *gin.Context) {
	project_id := c.Param("project-id")

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.ClientService().V2GetClientMatrix(
		c.Request.Context(),
		&auth_service.GetClientMatrixRequest{
			ProjectId: project_id,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2UpdateClient godoc
// @ID update_client
// @Router /v2/client [PUT]
// @Summary Update Client
// @Description Update Client
// @Tags V2_Client
// @Accept json
// @Produce json
// @Param client body auth_service.UpdateClientRequest true "UpdateClientRequestBody"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "Client data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateClient(c *gin.Context) {
	var client auth_service.UpdateClientRequest

	err := c.ShouldBindJSON(&client)
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
	resp, err := services.ClientService().V2UpdateClient(
		c.Request.Context(),
		&client,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2RemoveClient godoc
// @ID remove_client
// @Router /v2/client [DELETE]
// @Summary Delete Client
// @Description Get Client
// @Tags V2_Client
// @Accept json
// @Produce json
// @Param remove-client body auth_service.ClientPrimaryKey true "RemoveClientBody"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2RemoveClient(c *gin.Context) {
	var removeClient auth_service.ClientPrimaryKey

	err := c.ShouldBindJSON(&removeClient)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	if !util.IsValidUUID(removeClient.ClientPlatformId) {
		h.handleResponse(c, http.InvalidArgument, "client platform id is an invalid uuid")
		return
	}

	if !util.IsValidUUID(removeClient.ClientTypeId) {
		h.handleResponse(c, http.InvalidArgument, "client type id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.ClientService().V2RemoveClient(
		c.Request.Context(),
		&removeClient,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2AddRelation godoc
// @ID create_relation
// @Router /v2/relation [POST]
// @Summary Create Relation
// @Description Create Relation
// @Tags V2_Relation
// @Accept json
// @Produce json
// @Param relation body auth_service.AddRelationRequest true "AddRelationRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "Relation data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2AddRelation(c *gin.Context) {
	var relation auth_service.AddRelationRequest

	err := c.ShouldBindJSON(&relation)
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
	resp, err := services.ClientService().V2AddRelation(
		c.Request.Context(),
		&relation,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2UpdateRelation godoc
// @ID update_relation
// @Router /v2/relation [PUT]
// @Summary Update Relation
// @Description Update Relation
// @Tags V2_Relation
// @Accept json
// @Produce json
// @Param relation body auth_service.UpdateRelationRequest true "UpdateRelationRequestBody"
// @Success 200 {object} http.Response{data=auth_service.CommonMessage} "Relation data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateRelation(c *gin.Context) {
	var relation auth_service.UpdateRelationRequest

	err := c.ShouldBindJSON(&relation)
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
	resp, err := services.ClientService().V2UpdateRelation(
		c.Request.Context(),
		&relation,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2RemoveRelation godoc
// @ID delete_relation
// @Router /v2/relation/{relation-id} [DELETE]
// @Summary Delete Relation
// @Description Get Relation
// @Tags V2_Relation
// @Accept json
// @Produce json
// @Param relation-id path string true "relation-id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2RemoveRelation(c *gin.Context) {
	relationID := c.Param("relation-id")

	if !util.IsValidUUID(relationID) {
		h.handleResponse(c, http.InvalidArgument, "relation id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}
	resp, err := services.ClientService().V2RemoveRelation(
		c.Request.Context(),
		&auth_service.RelationPrimaryKey{
			Id: relationID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// V2AddUserInfoField godoc
// @ID create_user_info_field
// @Router /v2/user-info-field [POST]
// @Summary Create UserInfoField
// @Description Create UserInfoField
// @Tags V2_UserInfoField
// @Accept json
// @Produce json
// @Param user-info-field body auth_service.AddUserInfoFieldRequest true "AddUserInfoFieldRequestBody"
// @Success 201 {object} http.Response{data=auth_service.CommonMessage} "UserInfoField data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2AddUserInfoField(c *gin.Context) {
	var user_info_field auth_service.AddUserInfoFieldRequest

	err := c.ShouldBindJSON(&user_info_field)
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
	resp, err := services.ClientService().V2AddUserInfoField(
		c.Request.Context(),
		&user_info_field,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// V2UpdateUserInfoField godoc
// @ID update_user_info_field
// @Router /v2/user-info-field [PUT]
// @Summary Update UserInfoField
// @Description Update UserInfoField
// @Tags V2_UserInfoField
// @Accept json
// @Produce json
// @Param user-info-field body auth_service.UpdateUserInfoFieldRequest true "UpdateUserInfoFieldRequestBody"
// @Success 200 {object} http.Response{data=string} "UserInfoField data"
// @Response 400 {object} http.Response{data=auth_service.CommonMessage} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2UpdateUserInfoField(c *gin.Context) {
	var user_info_field auth_service.UpdateUserInfoFieldRequest

	err := c.ShouldBindJSON(&user_info_field)
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
	resp, err := services.ClientService().V2UpdateUserInfoField(
		c.Request.Context(),
		&user_info_field,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// V2RemoveUserInfoField godoc
// @ID delete_user_info_field
// @Router /v2/user-info-field/{user-info-field-id} [DELETE]
// @Summary Delete UserInfoField
// @Description Get UserInfoField
// @Tags V2_UserInfoField
// @Accept json
// @Produce json
// @Param user-info-field-id path string true "user-info-field-id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) V2RemoveUserInfoField(c *gin.Context) {
	userInfoFieldID := c.Param("user-info-field-id")

	if !util.IsValidUUID(userInfoFieldID) {
		h.handleResponse(c, http.InvalidArgument, "user info field id is an invalid uuid")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ClientService().V2RemoveUserInfoField(
		c.Request.Context(),
		&auth_service.UserInfoFieldPrimaryKey{
			Id: userInfoFieldID,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}
