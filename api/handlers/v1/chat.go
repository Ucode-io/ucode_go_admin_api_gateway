package v1

import (
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/chat_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateChat godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) CreatChat(c *gin.Context) {

	var (
		body models.CreateChatRequest
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("ShouldBindJSON", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentID, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentID.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.ChatService().Chat().CreateChat(c.Request.Context(), &chat_service.CreateChatRequest{
		UserId: body.UserId,
		Chat: &chat_service.Chat{
			SenderName:    body.Chat.Sender_name,
			PhoneNumber:   body.Chat.PhoneNumber,
			PlatformType:  body.Chat.PlatformType,
			EnvironmentId: environmentID.(string),
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
func (h *HandlerV1) GetChatList(c *gin.Context) {

	environmentID, ok := c.Get("environment_id")
	if !ok {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentID.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	Search := c.Query("search")
	resp, err := services.ChatService().Chat().GetChatList(c.Request.Context(), &chat_service.GetChatListRequest{
		EnvironmentId: environmentID.(string),
		Search:        Search,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetChatByChatID godoc
// @Security ApiKeyAuth
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
func (h *HandlerV1) GetChatByChatID(c *gin.Context) {

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "id not found")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	idstr := c.Param("id")
	resp, err := services.ChatService().Chat().GetChatByChatId(c.Request.Context(), &chat_service.GetChatByChatIdRequest{
		ChatId: idstr,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// CreateBot godoc
// @Security ApiKeyAuth
// @ID CreateBot
// @Router /v3/bot [POST]
// @Summary Create bot
// @Description Create bot
// @Tags Chat
// @Accept json
// @Produce json
// @Param bot_token body models.CreateBotToken true "body"
// @Success 200 {object} status_http.Response{data=string} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateBot(c *gin.Context) {

	var (
		body models.CreateBotToken
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("ShouldBindJSON", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	environmentId, ok := c.Get("environment_id")

	if !ok {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}
	resp, err := services.ChatService().Chat().CreateBot(c.Request.Context(), &chat_service.CreateBotRequest{
		BotToken:      body.BotToken,
		EnvironmentId: environmentId.(string),
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetBotTokenlist godoc
// @Security ApiKeyAuth
// @ID GetBotTokenlist
// @Router /v3/bot [GET]
// @Summary GetBotTokenlist
// @Description GetBotTokenlist
// @Tags Chat
// @Accept json
// @Produce json
// @Param project-id query string false "project-id"
// @Success 200 {object} status_http.Response{data=chat_service.GetBotTokenListResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetBotTokenList(c *gin.Context) {

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err := errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"+err.Error()))
		return
	}
	if !util.IsValidUUID(environmentId.(string)) {
		h.handleResponse(c, status_http.BadRequest, "environment_id not found")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.ChatService().Chat().GetBotTokenList(c.Request.Context(), &chat_service.GetBotTokenListRequest{
		EnvironmentId: environmentId.(string),
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateBotToken godoc
// @Security ApiKeyAuth
// @ID UpdateBotToken
// @Router /v3/bot [PUT]
// @Summary UpdateBotToken
// @Description UpdateBotToken
// @Tags Chat
// @Accept json
// @Produce json
// @Param project-id query string false "project-id"
// @Param bot_token body models.UpdateBotToken true "body"
// @Success 200 {object} status_http.Response{data=string} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateBotToken(c *gin.Context) {

	var (
		body models.UpdateBotToken
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("ShouldBindJSON", logger.Error(err))
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resp, err := services.ChatService().Chat().UpdateBotToken(c.Request.Context(), &chat_service.UpdateBotTokenRequest{
		BotId:    body.BotId,
		BotToken: body.BotToken,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteBotToken godoc
// @Security ApiKeyAuth
// @ID DeleteBotToken
// @Router /v3/bot/{id} [DELETE]
// @Summary DeleteBotToken
// @Description DeleteBotToken
// @Tags Chat
// @Accept json
// @Produce json
// @Param id path string true "bot-id"
// @Success 200 {object} status_http.Response{data=string} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteBotToken(c *gin.Context) {

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "id not found")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	idstr := c.Param("id")
	resp, err := services.ChatService().Chat().DeleteBotToken(c.Request.Context(), &chat_service.DeleteBotTokenRequest{
		BotId: idstr,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetBotTokenByBotID godoc
// @Security ApiKeyAuth
// @ID GetBotTokenByBotID
// @Router /v3/bot/{id} [GET]
// @Summary GetBotTokenByBotID
// @Description GetBotTokenByBotID
// @Tags Chat
// @Accept json
// @Produce json
// @Param id path string true "bot-id"
// @Success 200 {object} status_http.Response{data=chat_service.GetBotByBotIdResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetBotTokenByBotID(c *gin.Context) {

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "id not found")
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	idstr := c.Param("id")
	resp, err := services.ChatService().Chat().GetBotTokenByBotId(c.Request.Context(), &chat_service.GetBotByBotIdRequest{
		BotId: idstr,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
