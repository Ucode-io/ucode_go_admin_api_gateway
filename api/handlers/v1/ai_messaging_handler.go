package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/services"
)

func (h *HandlerV1) CreateAiChatMessage(c *gin.Context) {
	var (
		userMessage models.NewMessageReq
		chatId      = c.Param("chat-id")
		ctx         = context.Background()
	)

	if err := c.ShouldBindJSON(&userMessage); err != nil {
		h.HandleResponse(c, status_http.BadRequest, "invalid request body: "+err.Error())
		return
	}

	isPendingConfirmation := userMessage.PendingAction != nil

	if !isPendingConfirmation && strings.TrimSpace(userMessage.Content) == "" {
		h.HandleResponse(c, status_http.BadRequest, "content is required")
		return
	}

	service, resourceEnvID, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	chat, err := service.GoObjectBuilderService().AiChat().GetChat(
		ctx, &pbo.ChatPrimaryKey{
			ResourceEnvId: resourceEnvID,
			Id:            chatId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to get chat: %v", err))
		return
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.HandleResponse(c, status_http.Unauthorized, "unauthorized")
		return
	}

	var (
		projectIdObj, _ = c.Get("project_id")
		realProjectID   = projectIdObj.(string)
	)

	processor := newChatProcessor(
		h, service, h.baseConf,
		chatId, chat.GetProjectId(), resourceEnvID, realProjectID,
		authInfo.GetUserIdAuth(), authInfo.GetClientTypeId(), authInfo.GetRoleId(),
		c.GetHeader("Authorization"),
	)
	processor.microFrontendId = userMessage.MicrofrontendID
	processor.microFrontendRepoId = userMessage.MicrofrontendRepoID
	processor.newProject = userMessage.NewProject
	processor.userMessage = userMessage.Content

	if userMessage.UcodeProjectID != "" {
		processor.ucodeProjectId = userMessage.UcodeProjectID
	}

	chatHistory, err := processor.getChatHistory(ctx)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to fetch history: %v", err))
		return
	}

	if isPendingConfirmation {
		h.handlePendingConfirmation(c, ctx, processor, userMessage, chatHistory, service, resourceEnvID, chatId)
		return
	}

	_, err = processor.saveMessage(ctx, "user", userMessage.Content, userMessage.Images)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save user message: %v", err))
		return
	}

	aiResponse, err := processor.routeAndProcess(ctx, userMessage, chatHistory)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("ai processing failed: %v", err))
		return
	}

	if strings.TrimSpace(aiResponse.Description) == "" {
		if aiResponse.PendingAction != nil {
			aiResponse.Description = aiResponse.PendingAction.ConfirmationPrompt
		} else if len(aiResponse.Questions) > 0 {
			aiResponse.Description = aiResponse.Questions[0].Title
		} else if aiResponse.Plan != nil {
			aiResponse.Description = "Here are the diagrams for your project. Review them and let me know when you're ready to build."
		} else {
			aiResponse.Description = "Project has been updated."
		}
	}

	var (
		savedContent   = aiResponse.Description
		updatedProject *pbo.McpProject
	)

	if len(aiResponse.Questions) > 0 {
		savedContent = "[QUESTIONS_ASKED] " + aiResponse.Description
	} else if aiResponse.Plan != nil {
		planJSON, _ := json.Marshal(aiResponse.Plan)
		savedContent = "[DIAGRAMS_GENERATED] " + aiResponse.Description + "\n" + string(planJSON)
	}

	message, err := processor.saveMessage(ctx, "assistant", savedContent, nil)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save ai message: %v", err))
		return
	}

	if aiResponse.Project != nil {
		updatedProject, err = processor.saveProject(ctx, aiResponse)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save project: %v", err))
			return
		}
	}

	if len(chatHistory) == 0 {
		_, _ = service.GoObjectBuilderService().AiChat().UpdateChat(
			ctx, &pbo.UpdateChatRequest{
				ResourceEnvId: resourceEnvID,
				Id:            chatId,
				Title:         truncateString(userMessage.Content, 100),
				Description:   truncateString(aiResponse.Description, 255),
				ProjectId:     processor.mcpProjectId,
			},
		)
	}

	em := models.EnrichedMessage{
		ID:         message.GetId(),
		ChatID:     message.GetChatId(),
		Role:       message.GetRole(),
		Content:    message.GetContent(),
		Images:     message.GetImages(),
		HasFiles:   message.GetHasFiles(),
		TokensUsed: message.GetTokensUsed(),
		CreatedAt:  message.GetCreatedAt(),
	}

	if strings.HasPrefix(em.Content, "[DIAGRAMS_GENERATED] ") {
		body := strings.TrimPrefix(em.Content, "[DIAGRAMS_GENERATED] ")
		if idx := strings.Index(body, "\n"); idx != -1 {
			em.Content = body[:idx]
		} else {
			em.Content = body
		}
	} else if strings.HasPrefix(em.Content, "[QUESTIONS_ASKED] ") {
		em.Content = strings.TrimPrefix(em.Content, "[QUESTIONS_ASKED] ")
	}

	h.HandleResponse(c, status_http.Created, map[string]any{
		"message":               em,
		"project":               updatedProject,
		"mcp_project_id":        processor.mcpProjectId,
		"microfrontend_id":      processor.microFrontendId,
		"microfrontend_repo_id": processor.microFrontendRepoId,
		"ucode_project_id":      processor.mcpUcodeProjectId,
		"pending_action":        aiResponse.PendingAction,
		"questions":             aiResponse.Questions,
		"plan":                  aiResponse.Plan,
	})
}

func (h *HandlerV1) handlePendingConfirmation(
	c *gin.Context,
	ctx context.Context,
	processor *ChatProcessor,
	req models.NewMessageReq,
	chatHistory []models.ChatMessage,
	service services.ServiceManagerI,
	resourceEnvID, chatId string,
) {
	action := req.PendingAction

	// Use explicit content if provided, otherwise derive from approval status
	userContent := strings.TrimSpace(req.Content)
	if userContent == "" {
		if action.Approved {
			userContent = "Да"
		} else {
			userContent = "Нет"
		}
	}

	_, err := processor.saveMessage(ctx, "user", userContent, nil)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save confirmation message: %v", err))
		return
	}

	var (
		assistantReply string
		mutationResult any
	)

	if !action.Approved {
		assistantReply = action.CancelMessage
		if strings.TrimSpace(assistantReply) == "" {
			assistantReply = "Окей, действие отменено. Ничего не изменено."
		}
	} else {
		mutationResult, err = executeMutation(ctx, action, service)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("mutation failed: %v", err))
			return
		}

		assistantReply = action.SuccessMessage
		if strings.TrimSpace(assistantReply) == "" {
			switch action.Action {
			case "create":
				assistantReply = fmt.Sprintf("✅ Запись успешно создана в `%s`.", action.TableSlug)
			case "update":
				assistantReply = fmt.Sprintf("✅ Обновлено **%d** запис(ей) в `%s`.", action.AffectedCount, action.TableSlug)
			case "delete":
				assistantReply = fmt.Sprintf("✅ Удалено **%d** запис(ей) из `%s`.", action.AffectedCount, action.TableSlug)
			default:
				assistantReply = "✅ Готово."
			}
		}
	}

	message, err := processor.saveMessage(ctx, "assistant", assistantReply, nil)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save assistant message: %v", err))
		return
	}

	if len(chatHistory) == 0 {
		_, _ = service.GoObjectBuilderService().AiChat().UpdateChat(
			ctx, &pbo.UpdateChatRequest{
				ResourceEnvId: resourceEnvID,
				Id:            chatId,
				Title:         truncateString(userContent, 100),
				Description:   truncateString(assistantReply, 255),
				ProjectId:     processor.mcpProjectId,
			},
		)
	}

	h.HandleResponse(c, status_http.Created, map[string]any{
		"message":         message,
		"mcp_project_id":  processor.mcpProjectId,
		"mutation_result": mutationResult,
	})
}
