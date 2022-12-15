package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	authPb "ucode/ucode_go_api_gateway/genproto/auth_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create godoc
// @Security ApiKeyAuth
// @ID create_object
// @Router /v1/object/{table_slug}/ [POST]
// @Summary Create object
// @Description Create object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "CreateObjectRequestBody"
// @Success 201 {object} http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateObject(c *gin.Context) {
	var objectRequest models.CommonMessage

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo := h.GetAuthInfo(c)

	// THIS for loop is written to create child objects (right now it is used in the case of One2One relation)
	for key, value := range objectRequest.Data {
		if key[0] == '$' {

			interfaceToMap := value.(map[string]interface{})

			id, _ := uuid.NewRandom()
			interfaceToMap["guid"] = id

			mapToStruct, err := helper.ConvertMapToStruct(interfaceToMap)
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

			_, err = services.ObjectBuilderService().Create(
				context.Background(),
				&obs.CommonMessage{
					TableSlug: key[1:],
					Data:      mapToStruct,
					ProjectId: authInfo.GetProjectId(),
				},
			)

			if err != nil {
				h.handleResponse(c, http.GRPCError, err.Error())
				return
			}

			objectRequest.Data[key[1:]+"_id"] = id
		}
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)

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

	resp, err := services.ObjectBuilderService().Create(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetObjectByID godoc
// @Security ApiKeyAuth
// @ID get_object_by_id
// @Router /v1/object/{table_slug}/{object_id} [GET]
// @Summary Get object by id
// @Description Get object by id
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object_id path string true "object_id"
// @Success 200 {object} http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetSingle(c *gin.Context) {
	var object models.CommonMessage

	object.Data = make(map[string]interface{})

	objectID := c.Param("object_id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, http.InvalidArgument, "object_id is an invalid uuid")
		return
	}

	object.Data["id"] = objectID

	structData, err := helper.ConvertMapToStruct(object.Data)
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

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().GetSingle(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateObject godoc
// @Security ApiKeyAuth
// @ID update_object
// @Router /v1/object/{table_slug} [PUT]
// @Summary Update object
// @Description Update object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "UpdateObjectRequestBody"
// @Success 200 {object} http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateObject(c *gin.Context) {
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().Update(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	if c.Param("table_slug") == "record_permission" {
		if objectRequest.Data["role_id"] == nil {
			err := errors.New("role id must be have in update permission")
			h.handleResponse(c, http.BadRequest, err.Error())
			return
		}

		_, err = services.SessionService().UpdateSessionsByRoleId(
			context.Background(),
			&authPb.UpdateSessionByRoleIdRequest{
				RoleId:    objectRequest.Data["role_id"].(string),
				IsChanged: true,
			},
		)
		if err != nil {
			h.handleResponse(c, http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteObject godoc
// @Security ApiKeyAuth
// @ID delete_object
// @Router /v1/object/{table_slug}/{object_id} [DELETE]
// @Summary Delete object
// @Description Delete object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "DeleteObjectRequestBody"
// @Param object_id path string true "object_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteObject(c *gin.Context) {
	var objectRequest models.CommonMessage

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	objectID := c.Param("object_id")
	if !util.IsValidUUID(objectID) {
		h.handleResponse(c, http.InvalidArgument, "object id is an invalid uuid")
		return
	}
	objectRequest.Data["id"] = objectID

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
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

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().Delete(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetAllObjects godoc
// @Security ApiKeyAuth
// @ID get_list_objects
// @Router /v1/object/get-list/{table_slug} [POST]
// @Summary Get all objects
// @Description Get all objects
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetList(c *gin.Context) {
	var objectRequest models.CommonMessage

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}
	tokenInfo := h.GetAuthInfo
	objectRequest.Data["tables"] = tokenInfo(c).Tables
	objectRequest.Data["user_id_from_token"] = tokenInfo(c).UserId
	objectRequest.Data["role_id_from_token"] = tokenInfo(c).RoleId
	objectRequest.Data["client_type_id_from_token"] = tokenInfo(c).ClientTypeId
	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
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

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().GetList(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetAllObjects godoc
// @Security ApiKeyAuth
// @ID get_list_objects_in_excel
// @Router /v1/object/excel/{table_slug} [POST]
// @Summary Get all objects in excel
// @Description Get all objects in excel
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetListInExcel(c *gin.Context) {
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().GetListInExcel(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteManyToMany godoc
// @Security ApiKeyAuth
// @ID delete_many2many
// @Router /v1/many-to-many [DELETE]
// @Summary Delete Many2Many
// @Description Delete Many2Many
// @Tags Object
// @Accept json
// @Produce json
// @Param object body object_builder_service.ManyToManyMessage true "DeleteManyToManyBody"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteManyToMany(c *gin.Context) {
	var m2mMessage obs.ManyToManyMessage

	err := c.ShouldBindJSON(&m2mMessage)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
	}

	authInfo := h.GetAuthInfo(c)
	m2mMessage.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ObjectBuilderService().ManyToManyDelete(
		context.Background(),
		&m2mMessage,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// UpdateMany2Many godoc
// @Security ApiKeyAuth
// @ID append_many2many
// @Router /v1/many-to-many [PUT]
// @Summary Update many-to-many
// @Description Update many-to-many
// @Tags Object
// @Accept json
// @Produce json
// @Param object body object_builder_service.ManyToManyMessage true "UpdateMany2ManyRequestBody"
// @Success 200 {object} http.Response{data=string} "Object data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) AppendManyToMany(c *gin.Context) {
	var m2mMessage obs.ManyToManyMessage

	err := c.ShouldBindJSON(&m2mMessage)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
	}

	authInfo := h.GetAuthInfo(c)
	m2mMessage.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.ObjectBuilderService().ManyToManyAppend(
		context.Background(),
		&m2mMessage,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// GetObjectDetails godoc
// @Security ApiKeyAuth
// @ID get_object_details
// @Router /v1/object/object-details/{table_slug} [POST]
// @Summary Get object details
// @Description object details
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "GetObjectDetailsBody"
// @Success 201 {object} http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetObjectDetails(c *gin.Context) {
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().GetObjectDetails(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)

}

// UpsertObject godoc
// @Security ApiKeyAuth
// @ID upsert_object
// @Router /v1/object-upsert/{table_slug} [POST]
// @Summary Upsert object
// @Description Upsert object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.UpsertCommonMessage true "CreateObjectRequestBody"
// @Success 201 {object} http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpsertObject(c *gin.Context) {
	var objectRequest models.UpsertCommonMessage

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo := h.GetAuthInfo(c)

	// THIS for loop is written to create child objects (right now it is used in the case of One2One relation)
	for key, value := range objectRequest.Data {
		if key[0] == '$' {

			interfaceToMap := value.(map[string]interface{})

			id, _ := uuid.NewRandom()
			interfaceToMap["guid"] = id

			_, err := helper.ConvertMapToStruct(interfaceToMap)
			if err != nil {
				h.handleResponse(c, http.InvalidArgument, err.Error())
				return
			}
			// _, err = services.ObjectBuilderService().Create(
			// 	context.Background(),
			// 	&obs.CommonMessage{
			// 		TableSlug: key[1:],
			// 		Data:      mapToStruct,
			// 	},
			// )

			if err != nil {
				h.handleResponse(c, http.GRPCError, err.Error())
				return
			}

			objectRequest.Data[key[1:]+"_id"] = id
		}
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)

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

	resp, err := services.ObjectBuilderService().Batch(
		context.Background(),
		&obs.BatchRequest{
			TableSlug:     c.Param("table_slug"),
			Data:          structData,
			UpdatedFields: objectRequest.UpdatedFields,
			ProjectId:     authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	if c.Param("table_slug") == "record_permission" {
		role_id := objectRequest.Data["objects"].([]interface{})[0].(map[string]interface{})["role_id"]
		if role_id == nil {
			err := errors.New("role id must be have in upsert permission")
			h.handleResponse(c, http.BadRequest, err.Error())
			return
		}
		_, err = services.SessionService().UpdateSessionsByRoleId(
			context.Background(),
			&authPb.UpdateSessionByRoleIdRequest{
				RoleId:    objectRequest.Data["role_id"].(string),
				IsChanged: true,
			},
		)
		if err != nil {
			h.handleResponse(c, http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, http.Created, resp)
}

// MultipleUpdateObject godoc
// @Security ApiKeyAuth
// @ID multipe_update_object
// @Router /v1/object/multiple-update/{table_slug} [PUT]
// @Summary Multiple Update object
// @Description Multiple Update object
// @Tags Object
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Param object body models.CommonMessage true "MultipleUpdateObjectRequestBody"
// @Success 201 {object} http.Response{data=models.CommonMessage} "Object data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) MultipleUpdateObject(c *gin.Context) {
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

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	resp, err := services.ObjectBuilderService().MultipleUpdate(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: c.Param("table_slug"),
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}
