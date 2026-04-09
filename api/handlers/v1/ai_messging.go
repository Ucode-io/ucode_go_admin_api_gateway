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
	h              *HandlerV1
	service        services.ServiceManagerI
	baseConf       config.BaseConfig
	chatId         string
	mcpProjectID   string
	resourceEnvID  string
	ucodeProjectID string

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

	isPendingConfirmation := userMessage.PendingAction != nil

	if !isPendingConfirmation && strings.TrimSpace(userMessage.Content) == "" {
		h.HandleResponse(c, status_http.BadRequest, "content is required")
		return
	}

	log.Println("CLAUDE MODEL.....", h.baseConf.ClaudeModel)

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

	projectIdObj, _ := c.Get("project_id")
	realProjectID := projectIdObj.(string)

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

	// Persist a state marker in history for every response type so the router can reliably
	// detect conversation state in future turns without the actual structured data.
	// - Questions: save [QUESTIONS_ASKED] marker — the question options are sent only to the frontend.
	// - Diagrams:  save [DIAGRAMS_GENERATED] marker — the plan JSON is sent only to the frontend.
	// - Everything else: save the plain description.
	savedContent := aiResponse.Description
	if len(aiResponse.Questions) > 0 {
		savedContent = "[QUESTIONS_ASKED] " + aiResponse.Description
	} else if aiResponse.Plan != nil {
		// Embed the plan JSON in the content so it survives page refreshes.
		// Format: "[DIAGRAMS_GENERATED] <description>\n<plan_json>"
		// getChatHistory strips the JSON part; GetAiChatMessages parses it back out.
		planJSON, _ := json.Marshal(aiResponse.Plan)
		savedContent = "[DIAGRAMS_GENERATED] " + aiResponse.Description + "\n" + string(planJSON)
	}

	message, err := processor.saveMessage(ctx, "assistant", savedContent, nil)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save ai message: %v", err))
		return
	}

	var updatedProject *pbo.McpProject
	if aiResponse.Project != nil {
		updatedProject, err = processor.saveProject(ctx, aiResponse)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to save project: %v", err))
			return
		}
	}

	// Set chat title on first message
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
		"project":        updatedProject,
		"mcp_project_id": processor.mcpProjectID,
		"pending_action": aiResponse.PendingAction,
		"questions":      aiResponse.Questions,
		"plan":           aiResponse.Plan,
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
	hasImages := len(req.Images) > 0

	// Load existing project files if a project is linked
	var (
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
			log.Printf("[ROUTER] failed to load project files (mcpProjectID=%s): %v", p.mcpProjectID, err)
			projectData = nil
		}
		if projectData != nil {
			fileGraphJSON, _ = p.buildFileGraphJSON(projectData)
		}
	}

	// Step 1: classify intent using fast Haiku model
	routeResult, err := p.routeRequest(req.Content, fileGraphJSON, hasImages, chatHistory)
	if err != nil {
		return nil, err
	}

	log.Printf("[ROUTER] intent=%s next_step=%v files_needed=%d", routeResult.Intent, routeResult.NextStep, len(routeResult.FilesNeeded))

	// If the router wants to present structured questions to the user, return them immediately.
	if routeResult.Intent == "ask_question" {
		return &models.ParsedClaudeResponse{
			Description: routeResult.Reply,
			Questions:   routeResult.Questions,
		}, nil
	}

	// If the router detected a plan request, generate the full structured plan via a dedicated call.
	if routeResult.Intent == "plan_request" {
		return p.runGeneratePlan(ctx, req.Content, chatHistory)
	}

	// If Haiku said no further processing needed, return its reply directly
	if !routeResult.NextStep {
		return &models.ParsedClaudeResponse{Description: routeResult.Reply}, nil
	}

	// Step 2: route to the appropriate handler based on intent
	switch routeResult.Intent {

	case "clarify", "project_question":
		return &models.ParsedClaudeResponse{Description: routeResult.Reply}, nil

	case "project_inspect":
		if projectData == nil {
			return &models.ParsedClaudeResponse{
				Description: "No project exists yet. Please create a project first by describing what you want to build.",
			}, nil
		}
		return p.runInspect(ctx, req.Content, routeResult.FilesNeeded, chatHistory, req.Images, projectData)

	case "code_change":
		return p.runCodeChange(ctx, routeResult.Clarified, fileGraphJSON, chatHistory, req.Images, routeResult.ProjectName, projectData)

	case "database_query":
		clarified := strings.TrimSpace(routeResult.Clarified)
		if clarified == "" {
			clarified = req.Content
			log.Printf("[ROUTER] database_query: clarified was empty, using raw content")
		}
		return p.runDatabaseFlow(ctx, clarified, chatHistory)
	}

	return &models.ParsedClaudeResponse{Description: routeResult.Reply}, nil
}

func (p *ChatProcessor) runGeneratePlan(ctx context.Context, userRequest string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	content := helper.BuildPlanGeneratorMessage(userRequest)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.PromptPlanGenerator,
			Messages:  messages,
		},
		timeoutPlanner,
	)
	if err != nil {
		return nil, fmt.Errorf("plan generator: %w", err)
	}

	plan, err := helper.ParsePlanResult(response)
	if err != nil {
		return nil, fmt.Errorf("plan generator: parse failed: %w", err)
	}

	return &models.ParsedClaudeResponse{
		Description: "Here are the diagrams for your project. Review them and let me know when you're ready to build.",
		Plan:        plan,
	}, nil
}

func (p *ChatProcessor) runInspect(ctx context.Context, userQuestion string, filesNeeded []string, chatHistory []models.ChatMessage, imageURLs []string, projectData *pbo.McpProject) (*models.ParsedClaudeResponse, error) {
	filesContext := p.buildFilesContext(projectData, filesNeeded)
	answer, err := p.inspectCode(ctx, userQuestion, filesContext, chatHistory, imageURLs)
	if err != nil {
		return nil, err
	}
	return &models.ParsedClaudeResponse{Description: answer}, nil
}

func (p *ChatProcessor) runCodeChange(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string, projectName string, projectData *pbo.McpProject) (*models.ParsedClaudeResponse, error) {
	// No existing project — run the full new project creation flow
	if projectData == nil || len(projectData.GetProjectFiles()) == 0 {
		log.Printf("[CODE] no existing project — starting new project build")
		return p.buildNewProject(ctx, clarified, chatHistory, imageURLs, projectName)
	}

	// Existing project — plan then edit
	plan, err := p.planChanges(ctx, clarified, fileGraphJSON, chatHistory, len(imageURLs) > 0)
	if err != nil {
		return nil, err
	}
	log.Printf("[PLANNER] files_to_change=%d files_to_create=%d", len(plan.FilesToChange), len(plan.FilesToCreate))

	neededPaths := make([]string, 0, len(plan.FilesToChange))
	for _, f := range plan.FilesToChange {
		neededPaths = append(neededPaths, f.Path)
	}

	return p.editCode(ctx, clarified, plan, p.buildFilesContext(projectData, neededPaths), chatHistory, imageURLs)
}

// ============================================================================
// NEW PROJECT BUILD
// ============================================================================

func (p *ChatProcessor) buildNewProject(ctx context.Context, clarified string, chatHistory []models.ChatMessage, imageURLs []string, estimatedName string) (*models.ParsedClaudeResponse, error) {
	log.Printf("\n=======================================================")
	log.Printf("[NEW PROJECT] 🚀 STARTING FULL-STACK GENERATION 🚀")
	log.Printf("=======================================================")

	log.Printf("[NEW PROJECT] [Step 1/4] Calling Architect (Planning & Design Phase)...")
	plan, err := p.callArchitect(ctx, clarified, imageURLs)
	if err != nil {
		return nil, fmt.Errorf("architect phase failed: %w", err)
	}
	log.Printf("[NEW PROJECT] ✅ Architect generation successful. Name: %q, Type: %q", plan.ProjectName, plan.ProjectType)

	if plan.ProjectName == "" {
		plan.ProjectName = "AI Project"
		if estimatedName != "" {
			plan.ProjectName = estimatedName
		}
	}

	log.Printf("[NEW PROJECT] [Step 2/4] Provisioning Backend (Ucode Project & Env)...")
	projectData, err := p.provisionBackend(ctx, plan.ProjectName, p.mcpProjectID)
	if err != nil {
		return nil, fmt.Errorf("backend provisioning failed: %w", err)
	}
	p.mcpProjectID = projectData.McpProjectId
	log.Printf("[NEW PROJECT] ✅ Backend provisioned successfully (MCP ID: %s)", p.mcpProjectID)

	log.Printf("[NEW PROJECT] [Step 3/4] Creating Tables (Async in background)...")
	go func(bPlan *models.ArchitectPlan, resourceEnvId, envId string) {
		if err := createBackendFromPlan(context.Background(), bPlan, resourceEnvId, envId, p.service); err != nil {
			log.Printf("[ARCHITECT] backend table creation failed (resourceEnv=%s): %v", resourceEnvId, err)
		} else {
			log.Printf("[ARCHITECT] ✅ Async backend tables created successfully")
		}
	}(plan, projectData.ResourceEnvId, projectData.EnvironmentId)

	log.Printf("[NEW PROJECT] [Step 4/4] Writing Frontend Code (Coder Phase)...")
	if plan.ProjectType == "admin_panel" {
		log.Printf("[CODE] Using admin panel template system...")
		return p.generateAdminPanel(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	}

	log.Printf("[CODE] Using open project generator...")
	return p.generateProject(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
}

// ============================================================================
// CLAUDE API CALLS
// ============================================================================

// routeRequest classifies the user's message and decides the next step using the fast Haiku model.
func (p *ChatProcessor) routeRequest(userPrompt, fileGraphJSON string, hasImages bool, chatHistory []models.ChatMessage) (*models.HaikuRoutingResult, error) {
	historyText := buildHistoryText(chatHistory)
	content := helper.BuildRouterMessage(userPrompt, fileGraphJSON, hasImages, historyText)

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeHaikuModel,
			MaxTokens: p.baseConf.RouterMaxTokens,
			System:    helper.PromptRouter,
			Messages: []models.ChatMessage{
				{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: content}}},
			},
		},
		timeoutHaiku,
	)
	if err != nil {
		return nil, fmt.Errorf("router (haiku): %w", err)
	}

	result, err := helper.ParseHaikuRoutingResult(response)
	if err != nil {
		return nil, fmt.Errorf("router: parse failed: %w", err)
	}

	result.HasImages = hasImages
	return result, nil
}

// inspectCode answers questions about existing project file contents.
func (p *ChatProcessor) inspectCode(ctx context.Context, userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	content := helper.BuildInspectorMessage(userQuestion, filesContext)
	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(content, imageURLs))

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.InspectorMaxTokens,
			System:    helper.PromptInspector,
			Messages:  messages,
		},
		timeoutInspector,
	)
	if err != nil {
		return "", fmt.Errorf("inspector: %w", err)
	}

	answer, err := helper.ExtractPlainText(response)
	if err != nil {
		return "", fmt.Errorf("inspector: extract text: %w", err)
	}
	return answer, nil
}

// planChanges analyzes the file graph and returns a list of files to create/modify.
func (p *ChatProcessor) planChanges(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.SonnetPlanResult, error) {
	content := helper.BuildPlannerMessage(clarified, fileGraphJSON, hasImages)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.PromptPlanner,
			Messages:  messages,
		},
		timeoutPlanner,
	)
	if err != nil {
		return nil, fmt.Errorf("planner: %w", err)
	}

	result, err := helper.ParseSonnetPlanResult(response)
	if err != nil {
		return nil, fmt.Errorf("planner: parse failed: %w", err)
	}
	return result, nil
}

// editCode applies changes to specific files in an existing project.
// If the planned files are not found in the project, falls back to free generation.
func (p *ChatProcessor) editCode(ctx context.Context, clarified string, plan *models.SonnetPlanResult, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	hasMatchingFiles := filesContext != "No existing files to modify." && filesContext != "No matching files found."

	var systemPrompt string
	var contentBlocks []models.ContentBlock

	if hasMatchingFiles {
		systemPrompt = helper.PromptCodeEditor
		planJSON, _ := json.Marshal(plan)
		content := helper.BuildCodeEditorMessage(clarified, string(planJSON), filesContext, len(imageURLs) > 0)
		contentBlocks = buildContentBlocksWithImages(content, imageURLs)
	} else {
		// Planner found files to change but none exist in the project yet —
		// fall back to free generation using the full code generator prompt.
		log.Printf("[CODE] planned files not found in project, falling back to free generation")
		systemPrompt = helper.PromptCodeGenerator
		contentBlocks = buildContentBlocksWithImages(clarified, imageURLs)
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
		return nil, fmt.Errorf("code editor: %w", err)
	}

	parsed, parseErr := helper.ParseClaudeResponse(response)
	if parseErr != nil {
		log.Printf("[CODE] parse failed, attempting JSON repair: %v", parseErr)
		return p.repairJSON(ctx, response, clarified, systemPrompt)
	}
	return parsed, nil
}

// repairJSON asks Claude to fix a broken JSON response it previously generated.
func (p *ChatProcessor) repairJSON(ctx context.Context, brokenRawResponse, originalTask, systemPrompt string) (*models.ParsedClaudeResponse, error) {
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
		return nil, fmt.Errorf("JSON repair: API call failed: %w", err)
	}

	parsed, err := helper.ParseClaudeResponse(retryResponse)
	if err != nil {
		return nil, fmt.Errorf("JSON repair: parse still failed: %w", err)
	}

	fileCount := 0
	if parsed.Project != nil {
		fileCount = len(parsed.Project.Files)
	}
	log.Printf("[CODE] JSON repair succeeded, files=%d", fileCount)
	return parsed, nil
}

// callArchitect plans the full project structure — tables, fields, UI layout.
func (p *ChatProcessor) callArchitect(ctx context.Context, clarified string, imageURLs []string) (*models.ArchitectPlan, error) {
	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.PromptArchitect,
			Messages: []models.ChatMessage{
				{Role: "user", Content: buildContentBlocksWithImages(clarified, imageURLs)},
			},
		},
		timeoutArchitect,
	)
	if err != nil {
		return nil, fmt.Errorf("architect: %w", err)
	}

	text, err := helper.ExtractPlainText(response)
	if err != nil {
		return nil, fmt.Errorf("architect: extract text: %w", err)
	}

	log.Printf("\n--- [ARCHITECT RAW RESPONSE] ---\n%s\n--------------------------------\n", text)

	var plan models.ArchitectPlan
	if err = json.Unmarshal([]byte(helper.CleanJSONResponse(text)), &plan); err != nil {
		log.Printf("[ARCHITECT] parse failed (%v), attempting JSON repair...", err)

		repairPrompt := fmt.Sprintf(
			"Your previous response was a JSON object that could not be parsed due to invalid escaping or truncation.\n\n"+
				"Original task: %s\n\n"+
				"Broken response (truncated for context):\n%.800s\n\n"+
				"Return ONLY the corrected, complete JSON object. No markdown, no backticks, no explanation.\n"+
				"Ensure ALL string values are properly escaped (newlines → \\n, quotes → \\\", backslashes → \\\\).",
			clarified, text,
		)

		retryResponse, retryErr := helper.CallAnthropicAPI(
			p.baseConf,
			models.AnthropicRequest{
				Model:     p.baseConf.ClaudeModel,
				MaxTokens: p.baseConf.PlannerMaxTokens,
				System:    helper.PromptArchitect,
				Messages: []models.ChatMessage{
					{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: repairPrompt}}},
				},
			},
			timeoutArchitect,
		)
		if retryErr != nil {
			return nil, fmt.Errorf("architect: parse failed: %w (repair call also failed: %v)", err, retryErr)
		}

		retryText, _ := helper.ExtractPlainText(retryResponse)
		if retryErr = json.Unmarshal([]byte(helper.CleanJSONResponse(retryText)), &plan); retryErr != nil {
			return nil, fmt.Errorf("architect: parse failed: %w (repair also failed: %v)", err, retryErr)
		}
		log.Printf("[ARCHITECT] JSON repair succeeded")
	}

	return &plan, nil
}

// generateProject generates frontend code for non-admin-panel project types (landing, web, other).
func (p *ChatProcessor) generateProject(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	apiConfig := buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.PromptCodeGenerator,
			Messages: []models.ChatMessage{
				{
					Role:    "user",
					Content: buildContentBlocksWithImages(clarified+"\n\n"+apiConfig, imageURLs),
				},
			},
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("generate project: %w", err)
	}

	parsed, parseErr := helper.ParseClaudeResponse(response)
	if parseErr != nil {
		log.Printf("[CODE] generate project: parse failed, attempting JSON repair: %v", parseErr)
		parsed, parseErr = p.repairJSON(ctx, response, clarified, helper.PromptCodeGenerator)
		if parseErr != nil {
			return nil, fmt.Errorf("generate project: failed after repair: %w", parseErr)
		}
	}

	if parsed.Project == nil {
		return nil, fmt.Errorf("generate project: claude returned empty project")
	}

	log.Printf("[CODE] ✅ Generate project completed. Built %d files:", len(parsed.Project.Files))
	for _, f := range parsed.Project.Files {
		log.Printf("  - %s (%d bytes)", f.Path, len(f.Content))
	}

	return parsed, nil
}

// generateAdminPanel generates an admin panel using the template system with pre-built hooks and utilities.
func (p *ChatProcessor) generateAdminPanel(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	apiConfig := buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)

	// Inject read-only template files for the AI to import from (not regenerate)
	var templateCtx strings.Builder
	templateFiles := GetTemplate("admin_panel")
	if len(templateFiles) > 0 {
		templateCtx.WriteString("\n====================================\n")
		templateCtx.WriteString("BASE TEMPLATE FILES (read-only — DO NOT output these files)\n")
		templateCtx.WriteString("====================================\n")
		templateCtx.WriteString("Import hooks, utils, and config from these paths. Do not re-implement them.\n")
		templateCtx.WriteString("Do NOT copy colors or layout from these files.\n")
		templateCtx.WriteString("src/index.css and src/App.tsx MUST be fully regenerated by you.\n")
		for _, f := range templateFiles {
			templateCtx.WriteString(fmt.Sprintf("\n### FILE: %s\n```\n%s\n```\n", f.Path, f.Content))
		}
	}

	finalPrompt := clarified + "\n\n" + apiConfig + "\n" + templateCtx.String()

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.PromptAdminPanelGenerator,
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
		return nil, fmt.Errorf("generate admin panel: %w", err)
	}

	parsed, parseErr := helper.ParseClaudeResponse(response)
	if parseErr != nil {
		log.Printf("[CODE] admin panel: parse failed, attempting JSON repair: %v", parseErr)
		parsed, parseErr = p.repairJSON(ctx, response, clarified, helper.PromptAdminPanelGenerator)
		if parseErr != nil {
			return nil, fmt.Errorf("generate admin panel: failed after repair: %w", parseErr)
		}
	}

	if parsed.Project == nil {
		return nil, fmt.Errorf("generate admin panel: claude returned empty project")
	}

	// Merge any template files the AI didn't regenerate (safety fallback)
	if len(templateFiles) > 0 {
		generatedPaths := make(map[string]struct{}, len(parsed.Project.Files))
		for _, f := range parsed.Project.Files {
			generatedPaths[f.Path] = struct{}{}
		}
		for _, tf := range templateFiles {
			if _, exists := generatedPaths[tf.Path]; !exists {
				parsed.Project.Files = append(parsed.Project.Files, models.ProjectFile{
					Path:    tf.Path,
					Content: tf.Content,
				})
				log.Printf("[CODE] merged missing template file: %s", tf.Path)
			}
		}
	}

	log.Printf("[CODE] ✅ Admin panel generation completed. Total %d files:", len(parsed.Project.Files))
	for _, f := range parsed.Project.Files {
		log.Printf("  - %s (%d bytes)", f.Path, len(f.Content))
	}

	return parsed, nil
}

// ============================================================================
// DATA ACCESS HELPERS
// ============================================================================

func (p *ChatProcessor) saveMessage(ctx context.Context, role, content string, images []string) (*pbo.Message, error) {
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
		text := msg.GetContent()
		// Strip embedded plan JSON — the AI only needs the marker + description for state detection.
		if strings.HasPrefix(text, "[DIAGRAMS_GENERATED] ") {
			if idx := strings.Index(text, "\n"); idx != -1 {
				text = text[:idx]
			}
		}
		result = append(result, models.ChatMessage{
			Role:    msg.GetRole(),
			Content: []models.ContentBlock{{Type: "text", Text: text}},
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

// provisionBackend creates (or links) the ucode backend project, environment, and API key.
func (p *ChatProcessor) provisionBackend(ctx context.Context, projectName string, existingMcpID string) (*models.ProjectData, error) {
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

// buildAPIConfigBlock generates the API configuration section injected into the coder prompt.
func buildAPIConfigBlock(baseURL, apiKey string, plan *models.ArchitectPlan) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"\n====================================\nAPI CONFIGURATION FOR FRONTEND\n====================================\nVITE_API_BASE_URL: %s\nVITE_X_API_KEY: %s\n\nTables to use:\n",
		baseURL, apiKey,
	))
	for _, t := range plan.Tables {
		sb.WriteString(fmt.Sprintf("- Table: %s, slug: %s\n", t.Label, t.Slug))
		for _, f := range t.Fields {
			sb.WriteString(fmt.Sprintf("  * field: %s, type: %s\n", f.Slug, f.Type))
		}
	}
	sb.WriteString("\nUse this UI Structure provided by the Architect:\n" + plan.UIStructure + "\n")
	return sb.String()
}

func buildContentBlocksWithImages(textContent string, imageURLs []string) []models.ContentBlock {
	blocks := make([]models.ContentBlock, 0, len(imageURLs)+1)
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
	messages := make([]models.ChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, models.ChatMessage{
		Role:    "user",
		Content: contentBlocks,
	})
	return messages
}

func buildHistoryText(history []models.ChatMessage) string {
	if len(history) == 0 {
		return ""
	}
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	var sb strings.Builder
	for _, msg := range history[start:] {
		var text string
		for _, block := range msg.Content {
			if block.Type == "text" {
				text += block.Text
			}
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", strings.ToUpper(msg.Role), text))
	}
	return sb.String()
}
