package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	ars "ucode/ucode_go_api_gateway/genproto/api_reference_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateCategory godoc
// @Security ApiKeyAuth
// @ID create_category
// @Router /v1/category [POST]
// @Summary Create category
// @Description Create category
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param app body models.CreateCategory true "CreateApiReferenceRequestBody"
// @Success 201 {object} status_http.Response{data=models.Category} "Categoryç data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateCategory(c *gin.Context) {
	var category ars.CreateCategoryRequest

	err := c.ShouldBindJSON(&category)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if !util.IsValidUUID(category.ProjectId) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid"))
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	// app.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	// attributes, err := helper.ConvertMapToStruct(category.Attributes)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err)
	// 	return
	// }

	resp, err := services.CategoryService().Create(
		context.Background(), &category,
		// &ars.CreateCategoryRequest{
		// 	Name:       category.Name,
		// 	BaseUrl:    category.BaseUrl,
		// 	ProjectId:  category.ProjectID,
		// 	Attributes: attributes,
		// },
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetApiCategoryByID godoc
// @Security ApiKeyAuth
// @ID get_category_by_id
// @Router /v1/category/{category_id} [GET]
// @Summary Get api reference by id
// @Description Get api reference by id
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param category_id path string true "category_id"
// @Success 200 {object} status_http.Response{data=models.ApiReference} "AppBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetApiCategoryByID(c *gin.Context) {
	id := c.Param("category_id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "api reference id is an invalid uuid")
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

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	resp, err := services.ApiReferenceService().Get(
		context.Background(),
		&ars.GetApiReferenceRequest{
			Guid: id,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetAllCategories godoc
// @Security ApiKeyAuth
// @ID get_all_category
// @Router /v1/category [GET]
// @Summary Get all categories
// @Description Get all categories
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param filters query api_reference_service.GetListCategoryRequest true "filters"
// @Success 200 {object} status_http.Response{data=models.GetAllCategoriesResponse} "GetAllCategoriesBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllCategories(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	if !util.IsValidUUID(c.Query("project_id")) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid"))
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

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	resp, err := services.CategoryService().GetList(
		context.Background(),
		&ars.GetListCategoryRequest{
			Limit:     int64(limit),
			Offset:    int64(offset),
			ProjectId: c.Query("project_id"),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateCategory godoc
// @Security ApiKeyAuth
// @ID update_reference
// @Router /v1/category [PUT]
// @Summary Update category
// @Description Update category
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param api_reference body models.Category  true "UpdateCategoryRequestBody"
// @Success 200 {object} status_http.Response{data=models.Category} "Category data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateCategory(c *gin.Context) {
	var category models.Category

	err := c.ShouldBindJSON(&category)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if !util.IsValidUUID(category.ProjectID) {
		h.handleResponse(c, status_http.BadRequest, errors.New("project id is invalid uuid"))
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	// app.ProjectId = resourceId.(string)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	attributes, err := helper.ConvertMapToStruct(category.Attributes)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resp, err := services.CategoryService().Update(
		context.Background(),
		&ars.Category{
			Guid:       category.Guid,
			Name:       category.Name,
			BaseUrl:    category.BaseUrl,
			ProjectId:  category.ProjectID,
			Attributes: attributes,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteCategory godoc
// @Security ApiKeyAuth
// @ID delete_api_reference
// @Router /v1/category/{category_id} [DELETE]
// @Summary Delete App
// @Description Delete App
// @Tags ApiReference
// @Accept json
// @Produce json
// @Param category_id path string true "category_id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteCategory(c *gin.Context) {
	id := c.Param("category_id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "app id is an invalid uuid")
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

	// resourceId, ok := c.Get("resource_id")
	// if !ok {
	// 	err = errors.New("error getting resource id")
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	resp, err := services.CategoryService().Delete(
		context.Background(),
		&ars.DeleteCategoryRequest{
			Guid: id,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
}
