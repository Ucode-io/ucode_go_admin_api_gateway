package handlers

import (
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/chat_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateChat godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID CreateChat
// @Router /v3/chat [POST]
// @Summary Create Chat
// @Description Create Chat
// @Tags Chat
// @Accept json
// @Produce json
// @Param chat_service body models.CreateChatRequest true "Chat body"
// @Success 200 {object} status_http.Response{data=models.ChatResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreatChat(c *gin.Context) {

	var (
		body models.CreateChatRequest
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("ShouldBindJSON", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	EnvironmentId := c.GetHeader("Environment-Id")

	if !util.IsValidUUID(EnvironmentId) {
		h.handleResponse(c, status_http.BadRequest, "Environment-Id not found")
		return
	}

	resp, err := h.companyServices.ChatService().Chat().CreateChat(c.Request.Context(), &chat_service.CreateChatRequest{
		UserId: body.UserId,
		Chat: &chat_service.Chat{
			SenderName:    body.Chat.Sender_name,
			PhoneNumber:   body.Chat.PhoneNumber,
			PlatformType:  body.Chat.PlatformType,
			EnvironmentId: EnvironmentId,
			Check:         false,
		},
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetChatlist godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID GetChatlist
// @Router /v3/chat [GET]
// @Summary GetChatlist
// @Description GetChatlist
// @Tags Chat
// @Accept json
// @Produce json
// @Param project-id query string false "project-id"
// @Param search query string false "search"
// @Success 200 {object} status_http.Response{data=chat_service.GetChatListResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetChatList(c *gin.Context) {

	EnvironmentId := c.GetHeader("Environment-Id")

	if !util.IsValidUUID(EnvironmentId) {
		h.handleResponse(c, status_http.BadRequest, "Environment-Id not found")
		return
	}

	Search := c.Query("search")
	resp, err := h.companyServices.ChatService().Chat().GetChatList(c.Request.Context(), &chat_service.GetChatListRequest{
		EnvironmentId: EnvironmentId,
		Search:        Search,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	fmt.Println(resp, "::::::::::::::::::::::::")
	h.handleResponse(c, status_http.OK, resp)
}

// GetChatByChatID godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID GetChatByChatID
// @Router /v3/chat/{id} [GET]
// @Summary GetChatByChatID
// @Description GetChatByChatID
// @Tags Chat
// @Accept json
// @Produce json
// @Param id path string true "chat-id"
// @Success 200 {object} status_http.Response{data=chat_service.GetChatByChatIdResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetChatByChatID(c *gin.Context) {

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "id not found")
		return
	}
	idstr := c.Param("id")
	resp, err := h.companyServices.ChatService().Chat().GetChatByChatId(c.Request.Context(), &chat_service.GetChatByChatIdRequest{
		ChatId: idstr,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
