package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"
)

const (
	timeoutHaiku     = 180 * time.Second
	timeoutArchitect = 900 * time.Second
	timeoutInspector = 300 * time.Second
	timeoutPlanner   = 300 * time.Second
	timeoutCoder     = 900 * time.Second
)

type ChatProcessor struct {
	h                 *HandlerV1
	service           services.ServiceManagerI
	baseConf          config.BaseConfig
	chatId            string
	mcpProjectID      string
	resourceEnvID     string
	ucodeProjectID    string
	builderResourceID string

	userID       string
	clientTypeID string
	roleID       string

	schemaCache    []models.TableSchema
	schemaCachedAt time.Time
}

func newChatProcessor(h *HandlerV1, service services.ServiceManagerI, baseConf config.BaseConfig, chatId, mcpProjectID, resourceEnvID, ucodeProjectID string, userID, clientTypeID, roleID string) *ChatProcessor {
	return &ChatProcessor{
		h:              h,
		service:        service,
		baseConf:       baseConf,
		chatId:         chatId,
		mcpProjectID:   mcpProjectID,
		resourceEnvID:  resourceEnvID,
		ucodeProjectID: ucodeProjectID,
		userID:         userID,
		clientTypeID:   clientTypeID,
		roleID:         roleID,
	}
}

// ============================================================================
// HTTP HANDLER — main entry point
// ============================================================================

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

	var isPendingConfirmation = userMessage.PendingAction != nil

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

		updateProject *pbo.McpProject
	)

	processor := newChatProcessor(
		h, service, h.baseConf,
		chatId, chat.GetProjectId(), resourceEnvID, realProjectID,
		authInfo.GetUserIdAuth(), authInfo.GetClientTypeId(), authInfo.GetRoleId(),
	)

	chatHistory, err := processor.getChatHistory(ctx)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to fetch history: %v", err))
		return
	}

	if isPendingConfirmation {
		h.handlePendingConfirmation(c, ctx, processor, userMessage, chatHistory, service, resourceEnvID, chatId)
		return
	}

	_, err = processor.createMessageRecord(ctx, "user", userMessage.Content, userMessage.Images)
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
		} else {
			aiResponse.Description = "Project has been updated."
		}
	}

	message, err := processor.createMessageRecord(ctx, "assistant", aiResponse.Description, nil)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save ai message: %v", err))
		return
	}

	if aiResponse.Project != nil {
		updateProject, err = processor.saveProject(ctx, aiResponse)
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
				ProjectId:     processor.mcpProjectID,
			},
		)
	}

	h.HandleResponse(c, status_http.Created, map[string]any{
		"message":        message,
		"project":        updateProject,
		"mcp_project_id": processor.mcpProjectID,
		"pending_action": aiResponse.PendingAction,
	})
}

func (h *HandlerV1) handlePendingConfirmation(c *gin.Context, ctx context.Context, processor *ChatProcessor, req models.NewMessageReq, chatHistory []models.ChatMessage, service services.ServiceManagerI, resourceEnvID, chatId string) {
	var (
		action = req.PendingAction

		assistantReply string
		mutationResult any

		userContent = strings.TrimSpace(req.Content)
	)

	if userContent == "" {
		if action.Approved {
			userContent = "Да"
		} else {
			userContent = "Нет"
		}
	}

	_, err := processor.createMessageRecord(ctx, "user", userContent, nil)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save confirmation message: %v", err))
		return
	}

	if !action.Approved {
		// Используем cancel_message от AI, fallback — дефолтный
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
			// fallback если AI не заполнил
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

	message, err := processor.createMessageRecord(ctx, "assistant", assistantReply, nil)
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
				ProjectId:     processor.mcpProjectID,
			},
		)
	}

	h.HandleResponse(c, status_http.Created, map[string]any{
		"message":         message,
		"mcp_project_id":  processor.mcpProjectID,
		"mutation_result": mutationResult,
	})
}

// ============================================================================
// ROUTING & FLOW ORCHESTRATION
// ============================================================================

func (p *ChatProcessor) routeAndProcess(ctx context.Context, req models.NewMessageReq, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	var (
		hasImages     = len(req.Images) > 0
		projectData   *pbo.McpProject
		fileGraphJSON = "{}"
		err           error
	)

	if p.mcpProjectID != "" {
		projectData, err = p.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
			ctx, &pbo.McpProjectId{
				ResourceEnvId: p.resourceEnvID,
				Id:            p.mcpProjectID,
			},
		)
		if err != nil {
			log.Printf("[ROUTER] failed to get project files (mcpProjectID=%s): %v", p.mcpProjectID, err)
			projectData = nil
		}

		if projectData != nil {
			fileGraphJSON, _ = p.buildFileGraphJSON(projectData)
		}
	}

	haikuResult, err := p.callHaikuRouter(req.Content, fileGraphJSON, hasImages)
	if err != nil {
		return nil, err
	}

	log.Printf("[ROUTER] next_step=%v intent=%s has_images=%v files_needed=%d",
		haikuResult.NextStep, haikuResult.Intent, haikuResult.HasImages, len(haikuResult.FilesNeeded))

	if !haikuResult.NextStep {
		return &models.ParsedClaudeResponse{Description: haikuResult.Reply}, nil
	}

	switch haikuResult.Intent {

	case "clarify":
		return &models.ParsedClaudeResponse{Description: haikuResult.Reply}, nil

	case "project_question":
		return &models.ParsedClaudeResponse{Description: haikuResult.Reply}, nil

	case "project_inspect":
		if projectData == nil {
			return &models.ParsedClaudeResponse{
				Description: "No project exists yet. Please create a project first by describing what you want to build.",
			}, nil
		}
		return p.runInspectFlow(ctx, req.Content, haikuResult.FilesNeeded, chatHistory, req.Images, projectData)

	case "code_change":
		return p.runCodeFlow(ctx, haikuResult.Clarified, fileGraphJSON, chatHistory, req.Images, haikuResult.ProjectName, projectData)

	case "database_query":
		return p.runDatabaseFlow(ctx, haikuResult.Clarified, chatHistory)
	}

	return &models.ParsedClaudeResponse{Description: haikuResult.Reply}, nil
}

func (p *ChatProcessor) runInspectFlow(ctx context.Context, userQuestion string, filesNeeded []string, chatHistory []models.ChatMessage, imageURLs []string, projectData *pbo.McpProject) (*models.ParsedClaudeResponse, error) {
	filesContext := p.buildFilesContext(projectData, filesNeeded)

	answer, err := p.callSonnetInspector(ctx, userQuestion, filesContext, chatHistory, imageURLs)
	if err != nil {
		return nil, err
	}

	return &models.ParsedClaudeResponse{Description: answer}, nil
}

func (p *ChatProcessor) runCodeFlow(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string, projectName string, projectData *pbo.McpProject) (*models.ParsedClaudeResponse, error) {

	if projectData == nil || len(projectData.GetProjectFiles()) == 0 {
		log.Println("[CODER] New project — routing to Architect")
		return p.handleNewProjectPhase(ctx, clarified, chatHistory, imageURLs, projectName)
	}

	plan, err := p.callSonnetPlanner(ctx, clarified, fileGraphJSON, chatHistory, len(imageURLs) > 0)
	if err != nil {
		return nil, err
	}

	log.Printf("[PLANNER] modify=%d create=%d", len(plan.FilesToChange), len(plan.FilesToCreate))

	var neededPaths = make([]string, 0, len(plan.FilesToChange))

	for _, f := range plan.FilesToChange {
		neededPaths = append(neededPaths, f.Path)
	}

	return p.callSonnetCoder(ctx, clarified, plan, p.buildFilesContext(projectData, neededPaths), chatHistory, imageURLs)
}

// ============================================================================
// NEW PROJECT PHASE
// ============================================================================

func (p *ChatProcessor) handleNewProjectPhase(ctx context.Context, clarified string, chatHistory []models.ChatMessage, imageURLs []string, estimatedName string) (*models.ParsedClaudeResponse, error) {
	plan, err := p.callArchitect(ctx, clarified, imageURLs)
	if err != nil {
		return nil, fmt.Errorf("architect phase failed: %w", err)
	}

	if plan.ProjectName == "" {
		plan.ProjectName = "AI Project"
		if estimatedName != "" {
			plan.ProjectName = estimatedName
		}
	}

	projectData, err := p.createUcodeProject(ctx, plan.ProjectName, p.mcpProjectID)
	if err != nil {
		return nil, fmt.Errorf("backend provisioning failed: %w", err)
	}

	p.mcpProjectID = projectData.McpProjectId

	go func(bPlan *models.ArchitectPlan, resourceEnvId, envId string) {
		if err := createBackendFromPlan(context.Background(), bPlan, resourceEnvId, envId, p.service); err != nil {
			log.Printf("[ARCHITECT] backend build failed (resourceEnv=%s): %v", resourceEnvId, err)
		}
	}(plan, projectData.ResourceEnvId, projectData.EnvironmentId)

	if plan.ProjectType == "admin_panel" {
		log.Printf("[CODER] admin panel — using template generation")
		return p.callSonnetCoderWithTemplate(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	}

	return p.callSonnetCoderNewProject(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
}

// ============================================================================
// CLAUDE API INTEGRATIONS
// ============================================================================

func (p *ChatProcessor) callHaikuRouter(userPrompt, fileGraphJSON string, hasImages bool) (*models.HaikuRoutingResult, error) {
	content := helper.ProcessHaikuPrompt(userPrompt, fileGraphJSON, hasImages)
	messages := []models.ChatMessage{
		{
			Role: "user",
			Content: []models.ContentBlock{
				{
					Type: "text", Text: content,
				},
			},
		},
	}
	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeHaikuModel,
			MaxTokens: p.baseConf.RouterMaxTokens,
			System:    helper.SystemPromptHaikuRouter,
			Messages:  messages,
		},
		timeoutHaiku,
	)
	if err != nil {
		return nil, fmt.Errorf("haiku router: %w", err)
	}

	result, err := helper.ParseHaikuRoutingResult(response)
	if err != nil {
		return nil, fmt.Errorf("parse haiku result: %w", err)
	}

	result.HasImages = hasImages
	return result, nil
}

func (p *ChatProcessor) callSonnetInspector(ctx context.Context, userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	content := helper.ProcessSonnetInspectorPrompt(userQuestion, filesContext)
	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(content, imageURLs))

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.InspectorMaxTokens,
			System:    helper.SystemPromptSonnetInspector,
			Messages:  messages,
		},
		timeoutInspector,
	)
	if err != nil {
		return "", fmt.Errorf("sonnet inspector: %w", err)
	}

	answer, err := helper.ExtractPlainText(response)
	if err != nil {
		return "", fmt.Errorf("extract inspector text: %w", err)
	}

	return answer, nil
}

func (p *ChatProcessor) callSonnetPlanner(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.SonnetPlanResult, error) {
	content := helper.ProcessSonnetPlanPrompt(clarified, fileGraphJSON, hasImages)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.SystemPromptSonnetPlanner,
			Messages:  messages,
		},
		timeoutPlanner,
	)
	if err != nil {
		return nil, fmt.Errorf("sonnet planner: %w", err)
	}

	result, err := helper.ParseSonnetPlanResult(response)
	if err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}

	return result, nil
}

func (p *ChatProcessor) callSonnetCoder(ctx context.Context, clarified string, plan *models.SonnetPlanResult, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	hasFiles := filesContext != "No existing files to modify." && filesContext != "No matching files found."

	var (
		systemPrompt  string
		contentBlocks []models.ContentBlock
	)

	if !hasFiles {
		systemPrompt = helper.SystemPromptAiChat
		contentBlocks = buildContentBlocksWithImages(clarified, imageURLs)
	} else {
		systemPrompt = helper.SystemPromptSonnetCoder
		planJSON, _ := json.Marshal(plan)
		content := helper.ProcessSonnetCoderPrompt(clarified, string(planJSON), filesContext, len(imageURLs) > 0)
		contentBlocks = buildContentBlocksWithImages(content, imageURLs)
	}

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    systemPrompt,
			Messages:  buildMessagesWithHistory(chatHistory, contentBlocks),
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("sonnet coder: %w", err)
	}

	parsed, parseErr := helper.ParseClaudeResponse(response)
	if parseErr != nil {
		log.Printf("[CODER] parse failed, attempting Claude-side JSON repair: %v", parseErr)
		return p.retryWithJSONRepair(ctx, response, clarified, systemPrompt)
	}
	return parsed, nil
}

func (p *ChatProcessor) retryWithJSONRepair(ctx context.Context, brokenRawResponse, originalTask, systemPrompt string) (*models.ParsedClaudeResponse, error) {
	brokenText, _ := helper.ExtractPlainText(brokenRawResponse)

	fixPrompt := fmt.Sprintf(
		"Your previous response contained a JSON object that could not be parsed because "+
			"some string values had improperly escaped characters "+
			"(e.g. raw newlines, unescaped backslashes, or invalid control characters inside JSON strings).\n\n"+
			"Original task: %s\n\n"+
			"Return the SAME project JSON but with ALL string values correctly escaped:\n"+
			"  - Newlines inside strings  → \\n\n"+
			"  - Backslashes inside strings → \\\\\n"+
			"  - Double quotes inside strings → \\\"\n"+
			"  - No raw control characters (ASCII < 0x20) anywhere\n\n"+
			"Output ONLY the corrected raw JSON object starting with { and ending with }. "+
			"No markdown, no explanation, no backticks.\n\n"+
			"Broken response to fix (truncated for context):\n%.600s",
		originalTask, brokenText,
	)

	retryResponse, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    systemPrompt,
			Messages: []models.ChatMessage{
				{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: fixPrompt}}},
			},
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("JSON repair retry API call failed: %w", err)
	}

	parsed, err := helper.ParseClaudeResponse(retryResponse)
	if err != nil {
		return nil, fmt.Errorf("JSON repair retry parse also failed: %w", err)
	}

	log.Printf("[CODER] JSON repair retry succeeded, files=%d", func() int {
		if parsed.Project != nil {
			return len(parsed.Project.Files)
		}
		return 0
	}())
	return parsed, nil
}

func (p *ChatProcessor) callArchitect(ctx context.Context, clarified string, imageURLs []string) (*models.ArchitectPlan, error) {
	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.SystemPromptArchitect,
			Messages: []models.ChatMessage{
				{
					Role:    "user",
					Content: buildContentBlocksWithImages(clarified, imageURLs),
				},
			},
		},
		timeoutArchitect,
	)
	if err != nil {
		return nil, fmt.Errorf("architect: %w", err)
	}

	text, err := helper.ExtractPlainText(response)
	if err != nil {
		return nil, fmt.Errorf("architect extract text: %w", err)
	}

	cleaned := helper.CleanJSONResponse(text)

	var plan models.ArchitectPlan
	if err = json.Unmarshal([]byte(cleaned), &plan); err != nil {
		log.Printf("[ARCHITECT] first parse failed (%v), attempting repair...", err)

		repairPrompt := fmt.Sprintf(
			"Your previous response was a JSON object that could not be parsed due to invalid escaping or truncation.\n\n"+
				"Original task: %s\n\n"+
				"Broken response (truncated for context):\n%.800s\n\n"+
				"Return ONLY the corrected, complete JSON object. No markdown, no backticks, no explanation.\n"+
				"Ensure ALL string values are properly escaped (newlines → \\n, quotes → \\\", backslashes → \\\\).",
			clarified, cleaned,
		)

		retryResponse, retryErr := helper.CallAnthropicAPI(
			p.baseConf,
			models.AnthropicRequest{
				Model:     p.baseConf.ClaudeModel,
				MaxTokens: p.baseConf.PlannerMaxTokens,
				System:    helper.SystemPromptArchitect,
				Messages: []models.ChatMessage{
					{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: repairPrompt}}},
				},
			},
			timeoutArchitect,
		)
		if retryErr != nil {
			return nil, fmt.Errorf("unmarshal architect plan: %w (repair call also failed: %v)", err, retryErr)
		}

		retryText, _ := helper.ExtractPlainText(retryResponse)
		if retryErr = json.Unmarshal([]byte(helper.CleanJSONResponse(retryText)), &plan); retryErr != nil {
			return nil, fmt.Errorf("unmarshal architect plan: %w (repair also failed: %v)", err, retryErr)
		}

		log.Printf("[ARCHITECT] repair retry succeeded")
	}

	return &plan, nil
}

func (p *ChatProcessor) callSonnetCoderNewProject(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	var apiCtx strings.Builder

	apiCtx.WriteString(fmt.Sprintf(
		"\n====================================\nAPI CONFIGURATION FOR FRONTEND\n====================================\nVITE_API_BASE_URL: %s\nVITE_X_API_KEY: %s\n\nTables to use:\n",
		p.baseConf.UcodeBaseUrl, apiKey,
	))
	for _, t := range plan.Tables {
		apiCtx.WriteString(fmt.Sprintf("- Table: %s, slug: %s", t.Label, t.Slug))
		apiCtx.WriteString("\n")
		for _, f := range t.Fields {
			apiCtx.WriteString(fmt.Sprintf("  * field: %s, type: %s\n", f.Slug, f.Type))
		}
	}

	apiCtx.WriteString("\nUse this UI Structure provided by the Architect:\n" + plan.UIStructure + "\n")

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.SystemPromptAiChat,
			Messages: []models.ChatMessage{
				{
					Role:    "user",
					Content: buildContentBlocksWithImages(clarified+"\n\n"+apiCtx.String(), imageURLs),
				},
			},
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("new project coder: %w", err)
	}

	parsed, parseErr := helper.ParseClaudeResponse(response)
	if parseErr != nil {
		log.Printf("[NEW PROJECT] parse failed, attempting Claude-side JSON repair: %v", parseErr)
		parsed, parseErr = p.retryWithJSONRepair(ctx, response, clarified, helper.SystemPromptAiChat)
		if parseErr != nil {
			return nil, fmt.Errorf("new project coder failed after repair retry: %w", parseErr)
		}
	}

	if parsed.Project == nil {
		return nil, fmt.Errorf("claude returned empty project data")
	}

	log.Printf("[NEW PROJECT] files created: %d", len(parsed.Project.Files))
	return parsed, nil
}

func (p *ChatProcessor) callSonnetCoderWithTemplate(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	var apiCtx strings.Builder

	apiCtx.WriteString(fmt.Sprintf(
		"\n====================================\nAPI CONFIGURATION FOR FRONTEND\n====================================\nVITE_API_BASE_URL: %s\nVITE_X_API_KEY: %s\n\nTables to use:\n",
		p.baseConf.UcodeBaseUrl, apiKey,
	))

	for _, t := range plan.Tables {
		apiCtx.WriteString(fmt.Sprintf("- Table: %s, slug: %s", t.Label, t.Slug))
		apiCtx.WriteString("\n")
		for _, f := range t.Fields {
			apiCtx.WriteString(fmt.Sprintf("  * field: %s, type: %s\n", f.Slug, f.Type))
		}
	}
	apiCtx.WriteString("\nUse this UI Structure provided by the Architect:\n" + plan.UIStructure + "\n")

	templateFiles := GetTemplate("admin_panel")
	var templateCtx strings.Builder

	if len(templateFiles) > 0 {
		templateCtx.WriteString("\n====================================\nBASE TEMPLATE FILES\n====================================\n")
		templateCtx.WriteString("Below are the existing files from the 'admin_panel' template.\n")
		templateCtx.WriteString("These are BASE TEMPLATE files for API/config/hooks only.\n")
		templateCtx.WriteString("USE only: hooks, axios config, queryClient, utility functions, types.\n")
		templateCtx.WriteString("IGNORE AND REGENERATE: src/index.css and src/App.tsx — output them fully with your own unique design.\n")
		templateCtx.WriteString("NEVER copy colors, layout or structure from template files — only copy API/utility logic.\n")

		for _, f := range templateFiles {
			templateCtx.WriteString(fmt.Sprintf("\n### FILE: %s\n```\n%s\n```\n", f.Path, f.Content))
		}
	}

	designForce := buildDesignForceBlock(clarified, len(imageURLs) > 0)

	finalPrompt := designForce + clarified + "\n\n" + apiCtx.String() + "\n\n" + templateCtx.String()
	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.SystemPromptAiChatTemplate,
			Messages: []models.ChatMessage{
				{
					Role:    "user",
					Content: buildContentBlocksWithImages(finalPrompt, imageURLs),
				},
			},
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("template coder: %w", err)
	}

	parsed, parseErr := helper.ParseClaudeResponse(response)
	if parseErr != nil {
		log.Printf("[TEMPLATE CODER] parse failed, attempting Claude-side JSON repair: %v", parseErr)
		parsed, parseErr = p.retryWithJSONRepair(ctx, response, clarified, helper.SystemPromptAiChatTemplate)
		if parseErr != nil {
			return nil, fmt.Errorf("template coder failed after repair retry: %w", parseErr)
		}
	}

	if parsed.Project == nil {
		return nil, fmt.Errorf("claude returned empty project data")
	}

	log.Printf("[NEW PROJECT WITH TEMPLATE] total files: %d", len(parsed.Project.Files))
	return parsed, nil
}

// ============================================================================
// DATA ACCESS & DB HELPERS
// ============================================================================

func (p *ChatProcessor) createMessageRecord(ctx context.Context, role, content string, images []string) (*pbo.Message, error) {
	return p.service.GoObjectBuilderService().AiChat().CreateMessage(ctx, &pbo.CreateMessageRequest{
		ChatId:        p.chatId,
		Role:          role,
		Content:       content,
		Images:        images,
		ResourceEnvId: p.resourceEnvID,
	})
}

func (p *ChatProcessor) getChatHistory(ctx context.Context) ([]models.ChatMessage, error) {
	messages, err := p.service.GoObjectBuilderService().AiChat().GetMessages(ctx, &pbo.GetMessagesRequest{
		ResourceEnvId: p.resourceEnvID,
		ChatId:        p.chatId,
	})
	if err != nil {
		return nil, fmt.Errorf("get chat history: %w", err)
	}

	msgList := messages.GetMessages()

	if len(msgList) > 10 {
		msgList = msgList[len(msgList)-10:]
	}

	result := make([]models.ChatMessage, 0, len(msgList))
	for _, msg := range msgList {
		result = append(result, models.ChatMessage{
			Role:    msg.GetRole(),
			Content: []models.ContentBlock{{Type: "text", Text: msg.GetContent()}},
		})
	}
	return result, nil
}

func (p *ChatProcessor) buildFileGraphJSON(project *pbo.McpProject) (string, error) {
	if project == nil || len(project.GetProjectFiles()) == 0 {
		return "{}", nil
	}

	graph := make(map[string]models.GraphNode, len(project.GetProjectFiles()))
	for _, f := range project.GetProjectFiles() {
		graph[f.GetPath()] = models.GraphNode{
			Path:      f.GetPath(),
			FileGraph: f.GetFileGraph(),
		}
	}

	jsonBytes, err := json.Marshal(graph)
	if err != nil {
		return "{}", fmt.Errorf("marshal file graph: %w", err)
	}

	return string(jsonBytes), nil
}

func (p *ChatProcessor) buildFilesContext(project *pbo.McpProject, paths []string) string {
	if len(paths) == 0 || project == nil {
		return "No existing files to modify."
	}

	pathSet := make(map[string]bool, len(paths))
	for _, path := range paths {
		pathSet[path] = true
	}

	var sb strings.Builder
	for _, file := range project.GetProjectFiles() {
		if pathSet[file.GetPath()] {
			sb.WriteString(fmt.Sprintf("\n\n### FILE: %s\n```\n%s\n```", file.GetPath(), file.GetContent()))
		}
	}

	if sb.Len() == 0 {
		return "No matching files found."
	}
	return sb.String()
}

func (p *ChatProcessor) saveProject(ctx context.Context, req *models.ParsedClaudeResponse) (*pbo.McpProject, error) {
	if req == nil || req.Project == nil {
		return nil, fmt.Errorf("invalid project data")
	}

	projectEnv, err := helperFunc.ConvertMapToStruct(req.Project.Env)
	if err != nil {
		return nil, fmt.Errorf("convert project env: %w", err)
	}

	var projectFiles []*pbo.McpProjectFiles

	for _, file := range req.Project.Files {
		var fileGraph map[string]any
		if val, ok := req.Project.FileGraph[file.Path].(map[string]any); ok {
			fileGraph = val
		}
		fileGraphStruct, _ := helperFunc.ConvertMapToStruct(fileGraph)
		projectFiles = append(projectFiles, &pbo.McpProjectFiles{
			Path:      file.Path,
			Content:   file.Content,
			FileGraph: fileGraphStruct,
		})
	}

	return p.service.GoObjectBuilderService().McpProject().UpdateMcpProject(ctx, &pbo.McpProject{
		Id:            p.mcpProjectID,
		ResourceEnvId: p.resourceEnvID,
		Title:         truncateString(req.Project.ProjectName, 255),
		Description:   truncateString(req.Description, 255),
		ProjectFiles:  projectFiles,
		ProjectEnv:    projectEnv,
	})
}

func (p *ChatProcessor) createUcodeProject(ctx context.Context, projectName string, existingMcpID string) (*models.ProjectData, error) {
	currentProject, err := p.h.companyServices.Project().GetById(
		ctx, &pb.GetProjectByIdRequest{
			ProjectId: p.ucodeProjectID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get current project info: %w", err)
	}

	backendProject, err := p.h.companyServices.Project().Create(
		ctx, &pb.CreateProjectRequest{
			Title:        projectName,
			CompanyId:    currentProject.GetCompanyId(),
			K8SNamespace: currentProject.GetK8SNamespace(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create backend project: %w", err)
	}

	env, err := p.h.companyServices.Environment().CreateV2(
		ctx, &pb.CreateEnvironmentRequest{
			CompanyId:    currentProject.GetCompanyId(),
			ProjectId:    backendProject.GetProjectId(),
			UserId:       p.userID,
			ClientTypeId: p.clientTypeID,
			RoleId:       p.roleID,
			Name:         "Production",
			DisplayColor: "#00FF00",
			Description:  "Production Environment",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}

	resource, err := p.h.companyServices.ServiceResource().GetSingle(
		ctx, &pb.GetSingleServiceResourceReq{
			ProjectId:     backendProject.GetProjectId(),
			EnvironmentId: env.GetId(),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fetch service resource: %w", err)
	}

	apiKeys, err := p.h.authService.ApiKey().GetList(
		ctx, &as.GetListReq{
			EnvironmentId: env.GetId(),
			ProjectId:     backendProject.GetProjectId(),
			Limit:         1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fetch api keys: %w", err)
	}

	var apiKey string

	if len(apiKeys.GetData()) > 0 {
		apiKey = apiKeys.GetData()[0].GetAppId()
	}

	mcpProjectID := existingMcpID
	if mcpProjectID != "" {
		_, err = p.service.GoObjectBuilderService().McpProject().UpdateMcpProject(
			ctx, &pbo.McpProject{
				ResourceEnvId:  p.resourceEnvID,
				Id:             mcpProjectID,
				Title:          projectName,
				Description:    "Provisioned by AI architect",
				UcodeProjectId: backendProject.GetProjectId(),
				ApiKey:         apiKey,
				EnvironmentId:  env.GetId(),
				Status:         "ready",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("update MCP project: %w", err)
		}
	} else {
		project, err := p.service.GoObjectBuilderService().McpProject().CreateMcpProject(
			ctx, &pbo.CreateMcpProjectReqeust{
				ResourceEnvId:  p.resourceEnvID,
				Title:          projectName,
				Description:    "Generated by AI Architect",
				UcodeProjectId: backendProject.GetProjectId(),
				ApiKey:         apiKey,
				EnvironmentId:  env.GetId(),
				Status:         "ready",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("create MCP project link: %w", err)
		}
		mcpProjectID = project.GetId()
	}

	return &models.ProjectData{
		UcodeProjectId: backendProject.GetProjectId(),
		McpProjectId:   mcpProjectID,
		ApiKey:         apiKey,
		EnvironmentId:  env.GetId(),
		ResourceEnvId:  resource.GetResourceEnvironmentId(),
	}, nil
}

// ============================================================================
// UTILITIES
// ============================================================================

func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)

	if len(runes) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-3]) + "..."
}

func buildContentBlocksWithImages(textContent string, imageURLs []string) []models.ContentBlock {
	var blocks = make([]models.ContentBlock, 0, len(imageURLs)+1)
	for _, imageURL := range imageURLs {
		if strings.TrimSpace(imageURL) != "" {
			blocks = append(blocks, models.ContentBlock{
				Type:   "image",
				Source: &models.ImageSource{Type: "url", URL: imageURL},
			})
		}
	}

	blocks = append(blocks, models.ContentBlock{Type: "text", Text: textContent})

	return blocks
}

func buildMessagesWithHistory(history []models.ChatMessage, contentBlocks []models.ContentBlock) []models.ChatMessage {
	var messages = make([]models.ChatMessage, 0, len(history)+1)

	messages = append(messages, history...)
	messages = append(messages, models.ChatMessage{
		Role:    "user",
		Content: contentBlocks,
	})
	return messages
}

func buildDesignForceBlock(clarified string, hasImages bool) string {
	var (
		lower     = strings.ToLower(clarified)
		reference string
	)

	platforms := []string{
		"planfact", "linear", "notion", "stripe", "salesforce", "shopify",
		"amocrm", "amo crm", "jira", "figma", "asana", "trello", "monday",
		"hubspot", "pipedrive", "clickup",
	}
	for _, p := range platforms {
		if strings.Contains(lower, p) {
			reference = p
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("====================================\n")
	sb.WriteString("DESIGN DIRECTIVE — HIGHEST PRIORITY (READ BEFORE ANYTHING ELSE)\n")
	sb.WriteString("====================================\n")

	// НОВОЕ: image блок — самый высокий приоритет
	if hasImages {
		sb.WriteString("⚠️ AN IMAGE HAS BEEN PROVIDED IN THIS REQUEST.\n")
		sb.WriteString("You MUST extract colors directly from the image:\n")
		sb.WriteString("- Identify: background color, sidebar color, primary/accent color, text color\n")
		sb.WriteString("- Convert each to HSL format\n")
		sb.WriteString("- Use THESE HSL values in src/index.css — nothing else takes priority\n")
		sb.WriteString("- Do NOT use domain color map or reference platform colors\n")
		sb.WriteString("- The UI must visually match the image palette\n")
		sb.WriteString("This overrides domain map, reference platforms, and everything below.\n\n")
	} else if reference != "" {
		sb.WriteString(fmt.Sprintf(
			"The user explicitly requested UI style: \"%s\"\n"+
				"You MUST replicate its exact visual design:\n"+
				"- Color scheme, typography, spacing, component styles\n"+
				"- Layout structure (sidebar style, header, content areas)\n"+
				"- Navigation patterns and interaction patterns\n"+
				"This is NON-NEGOTIABLE. Generic white/gray default UI is FAILURE.\n\n",
			reference,
		))
	} else {
		sb.WriteString(
			"Choose a UNIQUE, domain-appropriate visual style. Generic white/gray default UI is FAILURE.\n" +
				"- Pick a brand color that fits the domain (NOT default blue #3b82f6)\n" +
				"- Make the sidebar visually distinct from content\n" +
				"- The result must look like a real SaaS product\n\n",
		)
	}

	sb.WriteString("MANDATORY: src/index.css MUST be in your output with ALL [AI: Generate HSL] placeholders replaced with real HSL values.\n")
	sb.WriteString("The --primary color MUST NOT be 221 83% 53% or 243 75% 59% (both are forbidden defaults).\n")
	sb.WriteString("--background MUST NOT be 0 0% 100% pure white UNLESS image explicitly shows white background.\n")
	sb.WriteString("====================================\n\n")

	return sb.String()
}
