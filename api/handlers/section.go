package handlers

import (
	"context"
	"ucode/ucode_go_api_gateway/api/http"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// GetAllSections godoc
// @Security ApiKeyAuth
// @ID get_all_sections
// @Router /v1/section [GET]
// @Summary Get all sections
// @Description Get all sections
// @Tags Section
// @Accept json
// @Produce json
// @Param filters query object_builder_service.GetAllSectionsRequest true "filters"
// @Success 200 {object} http.Response{data=string} "FieldBody"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetAllSections(c *gin.Context) {

	//tokenInfo := h.GetAuthInfo

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}

	resp, err := services.SectionService().GetAll(
		context.Background(),
		&obs.GetAllSectionsRequest{
			TableId:   c.Query("table_id"),
			TableSlug: c.Query("table_slug"),
			RoleId:    authInfo.GetRoleId(),
			ProjectId: authInfo.GetProjectId(),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateSection godoc
// @Security ApiKeyAuth
// @ID update_section
// @Router /v1/section [PUT]
// @Summary Update section
// @Description Update section
// @Tags Section
// @Accept json
// @Produce json
// @Param table body object_builder_service.UpdateSectionsRequest  true "UpdateSectionRequestBody"
// @Success 200 {object} http.Response{data=string} "Section data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateSection(c *gin.Context) {
	var sections obs.UpdateSectionsRequest

	err := c.ShouldBindJSON(&sections)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		return
	}
	sections.ProjectId = authInfo.GetProjectId()

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	resp, err := services.SectionService().Update(
		context.Background(),
		&sections,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
