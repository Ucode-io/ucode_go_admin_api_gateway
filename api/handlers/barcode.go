package handlers

import (
	"context"
	"errors"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"ucode/ucode_go_api_gateway/api/status_http"
	"github.com/gin-gonic/gin"
)

// GetNewGeneratedBarCode godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
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

	resp, err := services.BarcodeService().Generate(
		context.Background(),
		&obs.BarcodeGenerateReq{
			TableSlug: tableSlug,
			ProjectId: resourceId.(string),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
