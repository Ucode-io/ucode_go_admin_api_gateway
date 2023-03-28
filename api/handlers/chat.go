package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/chat_ucode"

	"github.com/gin-gonic/gin"
)

// CreatChat godoc
//
//	@Summary		Creat Chat
//	@Description	Creat a new chat_service
//	@Tags			chat_service
//	@Accept			json
//	@Produce		json
//	@Param			chat_service	body		models.CreateChatModul	true	"Chat body"
//	@Success		201				{object}	models.JSONResult{data=models.Chat}
//	@Failure		400				{object}	models.JSONErrorResponse
//	@Router			/v3/chat_service [post]
func (h *Handler) CreatChat(c *gin.Context) {
	var body models.CreateChatModul
	fmt.Println("create chat")
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{Error: err.Error()})
		return
	}

	chat_service, err := h.companyServices.ChatService().Chat().CreateChat(c.Request.Context(), &chat_ucode.CreateChatRequest{
		Chatid:            body.Chatid,
		MessageSenderName: body.Message_sender_name,
		Messages:          body.Messages,
		Types:             body.Types,
		ProjectId:         body.Project_id,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.JSONResult{
		Message: "CreatChat success",
		Data:    chat_service,
	})
}

// GetListByChatId godoc
//
//	@Summary		List chats by chat id
//	@Description	GetListByChatId
//	@Tags			chat_service
//	@Accept			json
//	@Produce		json
//	@Param			offset	query		int		false	"0"
//	@Param			limit	query		int		false	"100"
//	@Param			chatid	path		string	true	"Chat id"
//	@Success		200		{object}	models.JSONResult{data=[]models.Chat}
//	@Router			/v3/chat_service/chatid/{chatid} [get]
func (h *Handler) GetListByChatId(c *gin.Context) {

	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "10")

	chatid := c.Param("chatid")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	chatList, err := h.companyServices.ChatService().Chat().GetListByChatId(c.Request.Context(), &chat_ucode.GetListByChatIdRequest{
		Offset: int32(offset),
		Limit:  int32(limit),
		Chatid: chatid,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.JSONResult{
		Message: "GetListByChatId	 OK",
		Data:    chatList,
	})
}

// GetListByProjectId godoc
//
//	@Summary		List chats by project id
//	@Description	GetListByProjectId
//	@Tags			chat_service
//	@Accept			json
//	@Produce		json
//	@Param			offset		query		int		false	"0"
//	@Param			limit		query		int		false	"100"
//	@Param			projectid	path		string	true	"Project id"
//	@Success		200			{object}	models.JSONResult{data=[]models.Chat}
//	@Router			/v3/chat_service/projectid/{projectid} [get]
func (h *Handler) GetListByProjectId(c *gin.Context) {

	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "10")
	//projectid := c.DefaultQuery("projectid", "")
	projectid := c.Param("projectid")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	chatList, err := h.companyServices.ChatService().Chat().GetListByProjectId(c.Request.Context(), &chat_ucode.GetListByProjectIdRequest{
		Offset:  int32(offset),
		Limit:   int32(limit),
		Project: projectid,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.JSONErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.JSONResult{
		Message: "GetListByProjectId	 OK",
		Data:    chatList,
	})
}
