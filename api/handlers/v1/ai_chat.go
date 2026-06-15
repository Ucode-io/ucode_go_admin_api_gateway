package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// ==================== Enrichment ====================

// enrichMessages converts raw proto messages into EnrichedMessage slice.
// For messages saved with [DIAGRAMS_GENERATED] marker, it strips the embedded
// plan JSON from content and exposes it as a separate Plan field.
func enrichMessages(msgs []*pbo.Message) []models.EnrichedMessage {
	result := make([]models.EnrichedMessage, 0, len(msgs))
	for _, msg := range msgs {
		em := models.EnrichedMessage{
			ID:                  msg.GetId(),
			ChatID:              msg.GetChatId(),
			Role:                msg.GetRole(),
			Content:             msg.GetContent(),
			Images:              msg.GetImages(),
			HasFiles:            msg.GetHasFiles(),
			TokensUsed:          msg.GetTokensUsed(),
			CreatedAt:           msg.GetCreatedAt(),
			LikeCount:           msg.GetLikeCount(),
			DislikeCount:        msg.GetDislikeCount(),
			CurrentUserReaction: aiChatMessageReactionTypeToResponse(msg.GetCurrentUserReaction()),
		}
		content := msg.GetContent()
		if strings.HasPrefix(content, "[DIAGRAMS_GENERATED] ") {
			body := strings.TrimPrefix(content, "[DIAGRAMS_GENERATED] ")
			if idx := strings.Index(body, "\n"); idx != -1 {
				em.Content = body[:idx]
				var plan models.HaikuPlan
				if err := json.Unmarshal([]byte(body[idx+1:]), &plan); err == nil {
					em.Plan = &plan
				}
			} else {
				em.Content = body
			}
		} else if strings.HasPrefix(content, "[QUESTIONS_ASKED] ") {
			body := strings.TrimPrefix(content, "[QUESTIONS_ASKED] ")
			if idx := strings.Index(body, "\n"); idx != -1 {
				em.Content = body[:idx]
				var questions []models.AiQuestion
				if err := json.Unmarshal([]byte(body[idx+1:]), &questions); err == nil {
					em.Questions = questions
				}
			} else {
				em.Content = body
			}
		}
		result = append(result, em)
	}
	return result
}

// ==================== Helper ====================

func (h *HandlerV1) getBuilderService(ctx context.Context, projectId, environmentId string) (services.ServiceManagerI, string, error) {
	resource, err := h.companyServices.ServiceResource().GetSingle(ctx, &pb.GetSingleServiceResourceReq{
		ProjectId:     projectId,
		EnvironmentId: environmentId,
		ServiceType:   pb.ServiceType_BUILDER_SERVICE,
	})
	if err != nil {
		return nil, "", err
	}
	if resource.ResourceType != pb.ResourceType_POSTGRESQL {
		return nil, "", fmt.Errorf("resource type not supported: %s", resource.ResourceType)
	}
	service, err := h.GetProjectSrvc(ctx, projectId, resource.NodeType)
	if err != nil {
		return nil, "", err
	}
	return service, resource.ResourceEnvironmentId, nil
}

func (h *HandlerV1) getAiChatServices(c *gin.Context) (services.ServiceManagerI, string, error) {
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return nil, "", config.ErrProjectIdValid
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return nil, "", config.ErrEnvironmentIdValid
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return nil, "", err
	}

	if resource.ResourceType != pb.ResourceType_POSTGRESQL {
		h.HandleResponse(c, status_http.InvalidArgument, "resource type not supported")
		return nil, "", config.ErrProjectIdValid
	}

	service, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return nil, "", err
	}

	return service, resource.ResourceEnvironmentId, nil
}

type setAiChatMessageReactionRequest struct {
	ReactionType string `json:"reaction_type" binding:"required"`
}

type aiChatMessageReactionResponse struct {
	Id           string `json:"id"`
	MessageId    string `json:"message_id"`
	UserId       string `json:"user_id"`
	ReactionType string `json:"reaction_type"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	DeletedAt    int64  `json:"deleted_at"`
}

func (h *HandlerV1) getAiChatUserID(c *gin.Context) (string, error) {
	authInfo, err := h.adminAuthInfo(c)
	if err == nil {
		userID := authInfo.GetUserIdAuth()
		if userID == "" {
			userID = authInfo.GetUserId()
		}
		if userID != "" {
			return userID, nil
		}
	}

	authDataRaw, ok := c.Get("auth")
	if !ok {
		return "", err
	}
	authData, ok := authDataRaw.(models.AuthData)
	if !ok || authData.Type != "API-KEY" {
		return "", err
	}

	for _, key := range []string{"id", "app_id", "client_id"} {
		if userID := strings.TrimSpace(fmt.Sprintf("%v", authData.Data[key])); userID != "" && userID != "<nil>" {
			return userID, nil
		}
	}

	return "", fmt.Errorf("user_id is required")
}

func aiChatMessageReactionTypeFromRequest(reactionType string) (pbo.MessageReactionType, error) {
	switch strings.ToLower(strings.TrimSpace(reactionType)) {
	case "like", "message_reaction_type_like":
		return pbo.MessageReactionType_MESSAGE_REACTION_TYPE_LIKE, nil
	case "dislike", "message_reaction_type_dislike":
		return pbo.MessageReactionType_MESSAGE_REACTION_TYPE_DISLIKE, nil
	default:
		return pbo.MessageReactionType_MESSAGE_REACTION_TYPE_UNSPECIFIED, fmt.Errorf("reaction_type must be like or dislike")
	}
}

func aiChatMessageReactionTypeToResponse(reactionType pbo.MessageReactionType) string {
	switch reactionType {
	case pbo.MessageReactionType_MESSAGE_REACTION_TYPE_LIKE:
		return "like"
	case pbo.MessageReactionType_MESSAGE_REACTION_TYPE_DISLIKE:
		return "dislike"
	default:
		return ""
	}
}

func newAiChatMessageReactionResponse(reaction *pbo.MessageReaction) *aiChatMessageReactionResponse {
	if reaction == nil {
		return nil
	}
	return &aiChatMessageReactionResponse{
		Id:           reaction.GetId(),
		MessageId:    reaction.GetMessageId(),
		UserId:       reaction.GetUserId(),
		ReactionType: aiChatMessageReactionTypeToResponse(reaction.GetReactionType()),
		CreatedAt:    reaction.GetCreatedAt(),
		UpdatedAt:    reaction.GetUpdatedAt(),
		DeletedAt:    reaction.GetDeletedAt(),
	}
}

// ==================== Chat Endpoints ====================

func (h *HandlerV1) CreateAiChat(c *gin.Context) {
	var request pbo.CreateChatRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	if request.GetTitle() == "" && request.GetDescription() != "" {
		request.Title = request.GetDescription()
	}

	request.Model = string(config.ParseAIProvider(request.Model))

	request.ResourceEnvId = resourceEnvId

	if request.GetProjectId() == "" {
		mcpProject, err := service.GoObjectBuilderService().McpProject().CreateMcpProject(
			c.Request.Context(),
			&pbo.CreateMcpProjectReqeust{
				ResourceEnvId: resourceEnvId,
				Title:         "Draft Project",
				Description:   "Provisioning...",
			},
		)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to pre-create mcp project: %v", err))
			return
		}
		request.ProjectId = mcpProject.GetId()
	}

	response, err := service.GoObjectBuilderService().AiChat().CreateChat(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.Created, response)
}

func (h *HandlerV1) GetAllChats(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		title          = c.Query("title")
		model          = c.Query("model")
		projectId      = c.Query("project_id")
		orderBy        = c.Query("order_by")
		orderDirection = c.Query("order_direction")
		limit          = cast.ToInt32(c.DefaultQuery("limit", "20"))
		offset         = cast.ToInt32(c.DefaultQuery("offset", "0"))
	)

	response, err := service.GoObjectBuilderService().AiChat().GetAllChats(
		c.Request.Context(),
		&pbo.GetAllChatsRequest{
			ResourceEnvId:  resourceEnvId,
			Title:          title,
			Model:          model,
			ProjectId:      projectId,
			OrderBy:        orderBy,
			OrderDirection: orderDirection,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) GetProjectChat(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		withMessages = c.DefaultQuery("with_messages", "true") == "true"
		projectId    = c.Param("project-id")
	)
	userID, err := h.getAiChatUserID(c)
	if err != nil {
		h.HandleResponse(c, status_http.Unauthorized, err.Error())
		return
	}

	chat, err := service.GoObjectBuilderService().AiChat().GetChatByProjectId(
		c.Request.Context(),
		&pbo.ChatByProjectIdRequest{
			ResourceEnvId: resourceEnvId,
			ProjectId:     projectId,
			WithMessages:  withMessages,
			UserId:        userID,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, map[string]any{
		"id":           chat.GetId(),
		"project_id":   chat.GetProjectId(),
		"title":        chat.GetTitle(),
		"description":  chat.GetDescription(),
		"model":        string(config.ParseAIProvider(chat.GetModel())),
		"total_tokens": chat.GetTotalTokens(),
		"created_at":   chat.GetCreatedAt(),
		"updated_at":   chat.GetUpdatedAt(),
		"messages":     enrichMessages(chat.GetMessages()),
	})
}

func (h *HandlerV1) UpdateAiChat(c *gin.Context) {
	var (
		request pbo.UpdateChatRequest
		chatId  = c.Param("chat-id")
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	request.ResourceEnvId = resourceEnvId
	request.Id = chatId

	// PATCH semantics: empty Model means "leave current provider unchanged".
	if request.GetModel() != "" {
		request.Model = string(config.ParseAIProvider(request.Model))
	}

	response, err := service.GoObjectBuilderService().AiChat().UpdateChat(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) DeleteAiChat(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var chatId = c.Param("chat-id")

	response, err := service.GoObjectBuilderService().AiChat().DeleteChat(
		c.Request.Context(),
		&pbo.ChatPrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            chatId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

// ==================== Message Endpoints ====================

func (h *HandlerV1) GetAiChatMessages(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		chatId = c.Param("chat-id")
		limit  = cast.ToInt32(c.DefaultQuery("limit", "50"))
		offset = cast.ToInt32(c.DefaultQuery("offset", "0"))
	)
	userID, err := h.getAiChatUserID(c)
	if err != nil {
		h.HandleResponse(c, status_http.Unauthorized, err.Error())
		return
	}

	response, err := service.GoObjectBuilderService().AiChat().GetMessages(
		c.Request.Context(),
		&pbo.GetMessagesRequest{
			ResourceEnvId: resourceEnvId,
			ChatId:        chatId,
			Limit:         limit,
			Offset:        offset,
			UserId:        userID,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, map[string]any{
		"messages": enrichMessages(response.GetMessages()),
		"count":    response.GetCount(),
	})
}

func (h *HandlerV1) SetAiChatMessageReaction(c *gin.Context) {
	messageID := c.Param("message_id")
	if !util.IsValidUUID(messageID) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid message_id")
		return
	}

	var req setAiChatMessageReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	reactionType, err := aiChatMessageReactionTypeFromRequest(req.ReactionType)
	if err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	userID, err := h.getAiChatUserID(c)
	if err != nil {
		h.HandleResponse(c, status_http.Unauthorized, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().AiChat().SetMessageReaction(
		c.Request.Context(),
		&pbo.SetMessageReactionRequest{
			ResourceEnvId: resourceEnvId,
			MessageId:     messageID,
			UserId:        userID,
			ReactionType:  reactionType,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, newAiChatMessageReactionResponse(response))
}

func (h *HandlerV1) DeleteAiChatMessageReaction(c *gin.Context) {
	messageID := c.Param("message_id")
	if !util.IsValidUUID(messageID) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid message_id")
		return
	}

	userID, err := h.getAiChatUserID(c)
	if err != nil {
		h.HandleResponse(c, status_http.Unauthorized, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	_, err = service.GoObjectBuilderService().AiChat().DeleteMessageReaction(
		c.Request.Context(),
		&pbo.DeleteMessageReactionRequest{
			ResourceEnvId: resourceEnvId,
			MessageId:     messageID,
			UserId:        userID,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, gin.H{"message": "deleted"})
}

func (h *HandlerV1) DeleteAiChatMessage(c *gin.Context) {
	var messageId = c.Param("message_id")
	if !util.IsValidUUID(messageId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid message_id")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().AiChat().DeleteMessage(
		c.Request.Context(),
		&pbo.MessagePrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            messageId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}
