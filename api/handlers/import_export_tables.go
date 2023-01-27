package handlers

import (
	"context"
	"errors"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// ExportToJSON godoc
// @Security ApiKeyAuth
// @ID export_to_json
// @Router /v1/export-to-json [POST]
// @Summary export to json
// @Description  export to json
// @Tags ExportToJSON
// @Accept json
// @Produce json
// @Param export_to_json body object_builder_service.ExportToJSONRequest true "ExportToJSONRequestBody"
// @Success 201 {object} status_http.Response{data=object_builder_service.ExportToJSONReponse} "Link"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) ExportToJSON(c *gin.Context) {
	var body object_builder_service.ExportToJSONRequest

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id: " + err.Error())
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	body.ProjectId = resourceEnvironment.GetId()

	response, err := services.TableHelpersService().ExportToJSON(c.Request.Context(), &body)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

// ImportFromJSON godoc
// @Security ApiKeyAuth
// @ID import_from_json
// @Router /v1/import-from-json [POST]
// @Summary import from json
// @Description  import from json
// @Tags ExportToJSON
// @Accept json
// @Produce json
// @Param export_to_json body object_builder_service.ImportFromJSONRequest true "ImportFromJSONRequestBody"
// @Success 201 {object} status_http.Response{data=string} "Response"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) ImportFromJSON(c *gin.Context) {
	var body object_builder_service.ImportFromJSONRequest

	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	body.ProjectId = resourceEnvironment.GetId()

	ctx, cnlFunc := context.WithTimeout(c.Request.Context(), time.Minute*3)
	defer cnlFunc()
	_, err = services.TableHelpersService().ImportFromJSON(ctx, &body)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, "Tables Successfully Created")
}
