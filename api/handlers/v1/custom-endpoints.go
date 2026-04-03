package v1

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"

	"github.com/gin-gonic/gin"
)

func (h *HandlerV1) ExecQuery(c *gin.Context) {
	var request pbo.ExecuteSQLRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	request.ResourceEnvId = resourceEnvId

	response, err := service.GoObjectBuilderService().ObjectBuilder().ExecuteSQL(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.Created, response)
}
