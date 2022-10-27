package handlers

import (
	"context"
	"medion/medion_go_api_gateway/api/http"
	obs "medion/medion_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// GetSingleDocument godoc
// @Security ApiKeyAuth
// @ID generate_new_barcode_for_items
// @Router /v1/barcode-generator/{table_slug} [GET]
// @Summary get barcode
// @Description Get new barcode for items
// @Tags Barcode
// @Accept json
// @Produce json
// @Param table_slug path string true "table_slug"
// @Success 200 {object} http.Response{data=object_builder_service.BarcodeGenerateRes} "Barcode"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetNewGeneratedBarCode(c *gin.Context) {
	tableSlug := c.Param("table_slug")

	resp, err := h.services.BarcodeService().Generate(
		context.Background(),
		&obs.BarcodeGenerateReq{
			TableSlug: tableSlug,
		},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}
