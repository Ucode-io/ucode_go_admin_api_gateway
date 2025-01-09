package v1

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateFare godoc
// @Security ApiKeyAuth
// @ID create-fare
// @Router /v1/fare [POST]
// @Summary Create fare
// @Description Create fare
// @Tags Billing
// @Accept json
// @Produce json
// @Param billing body pb.CreateFareRequest true "FareCreateRequest"
// @Success 201 {object} status_http.Response{data=pb.Fare} "Fare data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateFare(c *gin.Context) {
	var request pb.CreateFareRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	response, err := h.companyServices.Billing().CreateFare(c, &request)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

// ListFares godoc
// @Security ApiKeyAuth
// @ID list-fares
// @Router /v1/fare [GET]
// @Summary List fares
// @Description List fares
// @Tags Billing
// @Accept json
// @Produce json
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 201 {object} status_http.Response{data=pb.Fare} "Fares data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllFares(c *gin.Context) {
	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	response, err := h.companyServices.Billing().ListFares(c, &pb.ListRequest{
		Offset: int32(offset),
		Limit:  int32(limit),
	})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

// GetFare godoc
// @Security ApiKeyAuth
// @ID get-fare
// @Router /v1/fare/{id} [GET]
// @Summary Get fare
// @Description Get fare
// @Tags Billing
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 201 {object} status_http.Response{data=pb.Fare} "Fare data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetFare(c *gin.Context) {
	var id = c.Param("id")

	if util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "invalid id")
		return
	}

	response, err := h.companyServices.Billing().GetFare(c, &pb.PrimaryKey{Id: id})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

// UpdateFare godoc
// @Security ApiKeyAuth
// @ID update-fare
// @Router /v1/fare [PUT]
// @Summary Update fare
// @Description Update fare
// @Tags Billing
// @Accept json
// @Produce json
// @Param billing body pb.Fare true "Fare"
// @Success 201 {object} status_http.Response{data=pb.Fare} "Fare data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateFare(c *gin.Context) {
	var request pb.Fare

	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	response, err := h.companyServices.Billing().UpdateFare(c, &request)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

// DeleteFare godoc
// @Security ApiKeyAuth
// @ID delete-fare
// @Router /v1/fare/{id} [DELETE]
// @Summary Delete fare
// @Description Delete fare
// @Tags Billing
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 201 {object} status_http.Response{data=string} "Fare deleted"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteFare(c *gin.Context) {
	var id = c.Param("id")

	if util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "invalid id")
		return
	}

	response, err := h.companyServices.Billing().DeleteFare(c, &pb.PrimaryKey{Id: id})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}
