package v1

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) GetVerifyCode(c *gin.Context) {
	var request pb.GetVerifyCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	request.ProjectId = projectId.(string)
	resp, err := h.companyServices.Billing().GetVerifyCode(c, &request)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

func (h *HandlerV1) Verify(c *gin.Context) {
	var request pb.VerifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	request.ProjectId = projectId.(string)
	resp, err := h.companyServices.Billing().Verify(c, &request)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

func (h *HandlerV1) GetAllProjectCards(c *gin.Context) {
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

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

	response, err := h.companyServices.Billing().ListProjectCards(c, &pb.ListRequest{
		Offset:    int32(offset),
		Limit:     int32(limit),
		ProjectId: projectId.(string),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) ReceiptPay(c *gin.Context) {
	var request pb.ReceiptPayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	userId, _ := c.Get("user_id")

	request.ProjectId = projectId.(string)
	request.UserId = userId.(string)
	response, err := h.companyServices.Billing().ReceiptPay(c, &request)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) DeleteProjectCard(c *gin.Context) {
	var id = c.Param("id")

	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.BadRequest, "invalid id")
		return
	}

	response, err := h.companyServices.Billing().DeleteProjectCard(c, &pb.PrimaryKey{Id: id})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, response)
}
