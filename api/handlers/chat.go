package handlers

import (
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/chat_service"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
)

// CreateChat godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string false "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID Create_Chat
// @Router /v3/chat [POST]
// @Summary Create Chat
// @Description Create Chat
// @Tags Chat
// @Accept json
// @Produce json
// @Param chat_service body models.CreateChatRequest true "Chat body"
// @Param body body models.CreateChatRequest  true "Request body"
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
	resp, err := h.companyServices.ChatService().Chat().CreateChat(c.Request.Context(), &chat_service.CreateChatRequest{
		UserId: body.UserId,
		Chat: &chat_service.Chat{
			SenderName:    body.Chat.Sender_name,
			Message:       body.Chat.Message,
			Types:         body.Chat.Types,
			EnvironmentId: body.Chat.Environment_id,
		},
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// // GetListByChatId godoc
// //
// //	@Summary		List chats by chat id
// //	@Description	GetListByChatId
// //	@Tags			chat_service
// //	@Accept			json
// //	@Produce		json
// //	@Param			offset	query		int		false	"0"
// //	@Param			limit	query		int		false	"100"
// //	@Param			chatid	path		string	true	"Chat id"
// //	@Success		200		{object}	models.JSONResult{data=[]models.Chat}
// //	@Router			/v3/chat_service/chatid/{chatid} [get]
// func (h *Handler) GetListByChatId(c *gin.Context) {

// 	offsetStr := c.DefaultQuery("offset", "0")
// 	limitStr := c.DefaultQuery("limit", "10")

// 	chatid := c.Param("chatid")

// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
// 			Error: err.Error(),
// 		})
// 		return
// 	}

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
// 			Error: err.Error(),
// 		})
// 		return
// 	}

// 	chatList, err := h.companyServices.ChatService().Chat().GetChatList(c.Request.Context(), &chat_service.GetChatListRequest{
// 		Offset: int32(offset),
// 		Limit:  int32(limit),
// 		Chatid: chatid,
// 	})

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, models.JSONErrorResponse{
// 			Error: err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, models.JSONResult{
// 		Message: "GetListByChatId	 OK",
// 		Data:    chatList,
// 	})
// }

// // GetListByProjectId godoc
// //
// //	@Summary		List chats by project id
// //	@Description	GetListByProjectId
// //	@Tags			chat_service
// //	@Accept			json
// //	@Produce		json
// //	@Param			offset		query		int		false	"0"
// //	@Param			limit		query		int		false	"100"
// //	@Param			projectid	path		string	true	"Project id"
// //	@Success		200			{object}	models.JSONResult{data=[]models.Chat}
// //	@Router			/v3/chat_service/projectid/{projectid} [get]
// func (h *Handler) GetListByProjectId(c *gin.Context) {

// 	offsetStr := c.DefaultQuery("offset", "0")
// 	limitStr := c.DefaultQuery("limit", "10")
// 	//projectid := c.DefaultQuery("projectid", "")
// 	projectid := c.Param("projectid")

// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
// 			Error: err.Error(),
// 		})
// 		return
// 	}

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
// 			Error: err.Error(),
// 		})
// 		return
// 	}

// 	chatList, err := h.companyServices.ChatService().Chat().GetListByProjectId(c.Request.Context(), &chat_service.GetListByProjectIdRequest{
// 		Offset:  int32(offset),
// 		Limit:   int32(limit),
// 		Project: projectid,
// 	})

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, models.JSONErrorResponse{
// 			Error: err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, models.JSONResult{
// 		Message: "GetListByProjectId	 OK",
// 		Data:    chatList,
// 	})
// }
