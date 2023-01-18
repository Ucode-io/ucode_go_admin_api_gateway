package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// GetNewGeneratedBarCode godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID generate_new_barcode_for_items
// @Router /v1/barcode-generator/{table_slug} [GET]
// @Summary get barcode
// @Description Get new barcode for items
// @Tags Barcode
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Success 200 {object} status_http.Response{data=object_builder_service.BarcodeGenerateRes} "Barcode"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetNewGeneratedBarCode(c *gin.Context) {
	tableSlug := c.Param("table_slug")

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

	resourceEnvironment, err := services.ResourceService().GetResourceEnvironment(
		context.Background(),
		&company_service.GetResourceEnvironmentReq{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.BarcodeService().Generate(
		context.Background(),
		&obs.BarcodeGenerateReq{
			TableSlug: tableSlug,
			ProjectId: resourceEnvironment.GetId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
