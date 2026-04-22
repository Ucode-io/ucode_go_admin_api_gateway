package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
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

	// uGenBranch is the branch where all AI-generated code is pushed.
	// master is only for pipeline triggers (created when the microfrontend is first forked).
	uGenBranch = "u-gen"

	timeoutPublishMicrofrontend = 5 * time.Minute
)

type ChatProcessor struct {
	h                 *HandlerV1
	service           services.ServiceManagerI
	baseConf          config.BaseConfig
	chatId            string
	mcpProjectID      string
	resourceEnvID     string
	ucodeProjectID    string
	mcpUcodeProjectID string

	userID       string
	clientTypeID string
	roleID       string
	authToken    string // forwarded to the function service for microfrontend creation

	microfrontendID     string // populated after PublishAiGeneratedMicroFrontend succeeds, or from request
	microfrontendRepoID string // GitLab numeric project ID — stored from publish response or from request
	newProject          bool   // true → provision a new ucode project; false → create microfrontend in current project

	schemaCache    []models.TableSchema
	schemaCachedAt time.Time
}

func newChatProcessor(h *HandlerV1, service services.ServiceManagerI, baseConf config.BaseConfig, chatId, mcpProjectID, resourceEnvID, ucodeProjectID string, userID, clientTypeID, roleID, authToken string) *ChatProcessor {
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
		authToken:      authToken,
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
		c.GetHeader("Authorization"),
	)
	processor.microfrontendID = userMessage.MicrofrontendID
	processor.microfrontendRepoID = userMessage.MicrofrontendRepoID
	processor.newProject = userMessage.NewProject
	if userMessage.UcodeProjectID != "" {
		processor.ucodeProjectID = userMessage.UcodeProjectID
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

	// Build an EnrichedMessage so the frontend never sees embedded plan JSON or
	// state markers ([DIAGRAMS_GENERATED], [QUESTIONS_ASKED]) in the content field.
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
		"mcp_project_id":        processor.mcpProjectID,
		"microfrontend_id":      processor.microfrontendID,
		"microfrontend_repo_id": processor.microfrontendRepoID,
		"ucode_project_id":      processor.mcpUcodeProjectID,
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

	if len(req.Context) > 0 {
		if p.mcpProjectID == "" && p.microfrontendID == "" {
			return &models.ParsedClaudeResponse{
				Description: "No project found. Please create a project first before using visual editing.",
			}, nil
		}
		return p.runVisualEdit(ctx, req.Content, req.Context, chatHistory, req.Images)
	}

	var (
		fileGraphJSON   = "{}"
		microfrontFiles []models.GitlabFileChange
		err             error
	)

	if p.microfrontendID != "" && p.microfrontendRepoID != "" {
		log.Printf("[ROUTER] microfrontend edit mode — fetching files for repo_id=%s", p.microfrontendRepoID)
		microfrontFiles, err = p.fetchMicrofrontendFiles(ctx)
		if err != nil {
			log.Printf("[ROUTER] failed to fetch microfrontend files: %v", err)
		} else {
			fileGraphJSON = p.buildMicrofrontendFileGraphJSON(microfrontFiles)
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
		if p.microfrontendID != "" {
			return p.runMicrofrontendInspect(ctx, req.Content, routeResult.FilesNeeded, chatHistory, req.Images, microfrontFiles)
		}
		return &models.ParsedClaudeResponse{
			Description: "No project exists yet. Please create a project first by describing what you want to build.",
		}, nil

	case "code_change":
		return p.runCodeChange(ctx, routeResult.Clarified, fileGraphJSON, chatHistory, req.Images, routeResult.ProjectName, microfrontFiles)

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

	plan, err := callWithTool[models.HaikuPlan](
		p, ctx,
		models.AnthropicToolRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.PromptPlanGenerator,
			Messages:  messages,
			Tools:     []models.ClaudeFunctionTool{helper.ToolEmitDiagrams},
			ToolChoice: helper.ForcedTool(helper.ToolEmitDiagrams.Name),
		},
		timeoutPlanner,
		"Generating architectural plan",
	)
	if err != nil {
		return nil, fmt.Errorf("plan generator: %w", err)
	}

	return &models.ParsedClaudeResponse{
		Description: "Here are the diagrams for your project. Review them and let me know when you're ready to build.",
		Plan:        plan,
	}, nil
}


func (p *ChatProcessor) runCodeChange(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string, projectName string, microfrontFiles []models.GitlabFileChange) (*models.ParsedClaudeResponse, error) {
	if p.microfrontendID != "" {
		return p.runMicrofrontendEdit(ctx, clarified, fileGraphJSON, chatHistory, imageURLs, microfrontFiles)
	}
	if p.newProject {
		log.Printf("[CODE] new_project=true — provisioning new ucode project")
		return p.buildNewProject(ctx, clarified, chatHistory, imageURLs, projectName)
	}
	log.Printf("[CODE] new_project=false — creating microfrontend in current project")
	return p.buildMicrofrontendForCurrentProject(ctx, clarified, chatHistory, imageURLs, projectName)
}

func (p *ChatProcessor) runVisualEdit(ctx context.Context, instruction string, contexts []models.VisualContext, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	log.Printf("[VISUAL EDIT] starting: count=%d", len(contexts))

	existingFiles, err := p.fetchMicrofrontendFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("visual edit: failed to fetch microfrontend files: %w", err)
	}

	// Resolve target files for each visual context
	targetPaths := make(map[string]bool)
	resolvedContexts := make([]models.VisualContext, 0, len(contexts))

	for _, vc := range contexts {
		var foundPath string
		for _, f := range existingFiles {
			if vc.Path != "" && f.FilePath == vc.Path {
				foundPath = f.FilePath
				break
			}
			if vc.Path == "" && vc.ElementName != "" && strings.Contains(f.Content, vc.ElementName) {
				foundPath = f.FilePath
				break
			}
		}
		if foundPath != "" {
			targetPaths[foundPath] = true
			vc.Path = foundPath
			resolvedContexts = append(resolvedContexts, vc)
		} else {
			log.Printf("[VISUAL EDIT] WARNING: could not resolve file for element %q (path: %q)", vc.ElementName, vc.Path)
		}
	}

	if len(targetPaths) == 0 {
		log.Printf("[VISUAL EDIT] no specific files found for contexts, falling back to microfrontend edit flow")
		fileGraphJSON := p.buildMicrofrontendFileGraphJSON(existingFiles)
		return p.runMicrofrontendEdit(ctx, instruction, fileGraphJSON, chatHistory, imageURLs, existingFiles)
	}

	paths := make([]string, 0, len(targetPaths))
	for path := range targetPaths {
		paths = append(paths, path)
	}
	filesContext := p.buildMicrofrontendFilesContext(existingFiles, paths)

	prompt := helper.BuildVisualEditPrompt(instruction, resolvedContexts, filesContext)
	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(prompt, imageURLs))

	edited, err := callWithTool[visualEditOutput](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.ClaudeModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptVisualEdit,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitVisualEdit},
			ToolChoice: helper.ForcedTool(helper.ToolEmitVisualEdit.Name),
		},
		timeoutCoder,
		fmt.Sprintf("Visual edit: %d elements in %d files", len(resolvedContexts), len(targetPaths)),
	)
	if err != nil {
		return nil, fmt.Errorf("visual edit: claude call failed: %w", err)
	}

	if len(edited.Files) > 0 {
		if pushErr := p.pushMicrofrontendChanges(ctx, edited.Files); pushErr != nil {
			return nil, fmt.Errorf("visual edit: push to u-gen failed: %w", pushErr)
		}
	}

	description := edited.ChangeSummary
	if description == "" {
		description = "✅ Visual edit applied successfully."
	}

	log.Printf("[VISUAL EDIT] ✅ done — %d files pushed to u-gen, summary=%s", len(edited.Files), description)
	return &models.ParsedClaudeResponse{Description: description}, nil
}

// ============================================================================
// TOOL-USE GENERIC CALLER
// ============================================================================

// visualEditOutput is the structured result for visual-edit tool calls.
type visualEditOutput struct {
	Files         []models.ProjectFile `json:"files"`
	ChangeSummary string               `json:"change_summary"`
}

// callWithTool is a package-level generic function (Go generics don't allow
// generic methods) that wraps helper.CallAnthropicWithTool with token-usage
// tracking and user-friendly ErrMaxTokens handling.
//
// It returns (*T, error). On ErrMaxTokens it returns a translated error message
// the caller can forward directly to the user.
func callWithTool[T any](
	p *ChatProcessor,
	ctx context.Context,
	req models.AnthropicToolRequest,
	timeout time.Duration,
	description string,
) (*T, error) {
	log.Printf("[AI] Calling Anthropic (tool use): %s", description)

	result, usage, stopReason, err := helper.CallAnthropicWithTool[T](p.baseConf, req, timeout)

	// Record token usage regardless of error — partial usage still counts.
	go func() {
		if usage.InputTokens == 0 && usage.OutputTokens == 0 {
			return
		}
		projectID := p.ucodeProjectID
		if p.mcpUcodeProjectID != "" {
			projectID = p.mcpUcodeProjectID
		}
		_, recErr := p.service.CompanyService().Billing().RecordAiTokenUsage(
			context.Background(),
			&pb.RecordAiTokenUsageRequest{
				ProjectId:    projectID,
				InputTokens:  int32(usage.InputTokens),
				OutputTokens: int32(usage.OutputTokens),
				Model:        req.Model,
				Description:  description,
			},
		)
		if recErr != nil {
			log.Printf("[TOKEN RECORD] error recording usage for %s: %v", description, recErr)
		}
	}()

	if err != nil {
		if errors.Is(err, helper.ErrMaxTokens) {
			log.Printf("[AI] max_tokens for %s (in=%d out=%d)", description, usage.InputTokens, usage.OutputTokens)
			return nil, fmt.Errorf(
				"❌ Generation stopped: the project is too large to generate in one pass (used %d output tokens). "+
					"Please describe a smaller scope or break the request into parts.",
				usage.OutputTokens,
			)
		}
		log.Printf("[AI] error for %s: %v", description, err)
		return nil, err
	}

	log.Printf("[AI] ✅ %s (stop=%s in=%d out=%d)", description, stopReason, usage.InputTokens, usage.OutputTokens)
	return result, nil
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
		if err := createBackendFromPlan(context.Background(), bPlan, resourceEnvId, p.ucodeProjectID, p.userID, envId, p.service); err != nil {
			log.Printf("[ARCHITECT] backend table creation failed (resourceEnv=%s): %v", resourceEnvId, err)
		} else {
			log.Printf("[ARCHITECT] ✅ Async backend tables created successfully")
		}
	}(plan, projectData.ResourceEnvId, projectData.EnvironmentId)

	log.Printf("[NEW PROJECT] [Step 4/4] Writing Frontend Code (Coder Phase)...")
	var generated *models.ParsedClaudeResponse
	if plan.ProjectType == "admin_panel" {
		log.Printf("[CODE] Using admin panel template system...")
		generated, err = p.generateAdminPanel(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	} else {
		log.Printf("[CODE] Using open project generator...")
		generated, err = p.generateProject(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	}
	if err != nil {
		return nil, err
	}

	// Previously the generated project files were returned here and saved to
	// McpProject JSON in the DB via saveProject(). That approach is now replaced
	// by creating a real microfrontend and pushing the code to the u-gen branch.
	//
	// OLD (kept for reference):
	// return generated, nil  →  caller called saveProject() → McpProject.UpdateMcpProject()

	log.Printf("[NEW PROJECT] [Step 5/5] Creating microfrontend and pushing code to %s branch...", uGenBranch)
	if err = p.publishToMicrofrontend(ctx, plan.ProjectName, "app", generated, projectData); err != nil {
		return nil, fmt.Errorf("microfrontend publish failed: %w", err)
	}

	log.Printf("[NEW PROJECT] ✅ Microfrontend created (id: %s)", p.microfrontendID)
	return &models.ParsedClaudeResponse{Description: generated.Description}, nil
}

// ============================================================================
// CLAUDE API CALLS
// ============================================================================

// routeRequest classifies the user's message and decides the next step using the fast Haiku model.
func (p *ChatProcessor) routeRequest(userPrompt, fileGraphJSON string, hasImages bool, chatHistory []models.ChatMessage) (*models.HaikuRoutingResult, error) {
	historyText := buildHistoryText(chatHistory)
	content := helper.BuildRouterMessage(userPrompt, fileGraphJSON, hasImages, historyText)

	response, err := p.callAnthropicWithTracking(
		context.Background(),
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeHaikuModel,
			MaxTokens: p.baseConf.RouterMaxTokens,
			System:    helper.PromptRouter,
			Messages: []models.ChatMessage{
				{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: content}}},
			},
		},
		timeoutHaiku,
		"Routing user intent",
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

// buildMicrofrontendForCurrentProject generates and publishes a microfrontend
// inside the caller's existing ucode project (no new project/environment provisioned).
func (p *ChatProcessor) buildMicrofrontendForCurrentProject(ctx context.Context, clarified string, chatHistory []models.ChatMessage, imageURLs []string, estimatedName string) (*models.ParsedClaudeResponse, error) {
	log.Printf("[MFE CURRENT PROJECT] 🚀 generating microfrontend for existing project")

	log.Printf("[MFE CURRENT PROJECT] [Step 1/3] Calling Architect...")
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
	log.Printf("[MFE CURRENT PROJECT] ✅ Architect done. Name: %q, Type: %q", plan.ProjectName, plan.ProjectType)

	log.Printf("[MFE CURRENT PROJECT] [Step 2/3] Fetching existing project data...")
	projectData, err := p.getExistingProjectData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing project data: %w", err)
	}

	log.Printf("[MFE CURRENT PROJECT] [Step 3/3] Generating frontend code...")
	var generated *models.ParsedClaudeResponse
	if plan.ProjectType == "admin_panel" {
		generated, err = p.generateAdminPanel(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	} else {
		generated, err = p.generateProject(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	}
	if err != nil {
		return nil, err
	}

	// Derive a unique path from the architect's plan name so that multiple
	// microfrontends inside the same project get distinct GitLab paths.
	mfePath := slugify(plan.ProjectName)
	if len(mfePath) > 10 {
		mfePath = strings.TrimRight(mfePath[:10], "-")
	}

	log.Printf("[MFE CURRENT PROJECT] Publishing microfrontend to u-gen branch (path=%s)...", mfePath)
	if err = p.publishToMicrofrontend(ctx, plan.ProjectName, mfePath, generated, projectData); err != nil {
		return nil, fmt.Errorf("microfrontend publish failed: %w", err)
	}

	log.Printf("[MFE CURRENT PROJECT] ✅ Done (id: %s)", p.microfrontendID)
	return &models.ParsedClaudeResponse{Description: generated.Description}, nil
}

// getExistingProjectData fetches the environment and API key for the target
// ucode project without creating anything new.
// Priority: MCP project's ucode_project_id (set when a project was provisioned
// in a previous request) > middleware project ID.
func (p *ChatProcessor) getExistingProjectData(ctx context.Context) (*models.ProjectData, error) {
	ucodeProjectID := p.ucodeProjectID

	if p.mcpProjectID != "" {
		mcpProject, err := p.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(ctx, &pbo.McpProjectId{
			ResourceEnvId: p.resourceEnvID,
			Id:            p.mcpProjectID,
		})
		if err == nil && mcpProject != nil && mcpProject.GetUcodeProjectId() != "" {
			ucodeProjectID = mcpProject.GetUcodeProjectId()
			log.Printf("[GET EXISTING PROJECT] using ucode_project_id=%s from MCP project", ucodeProjectID)
		}
	}

	envList, err := p.h.companyServices.Environment().GetList(ctx, &pb.GetEnvironmentListRequest{
		ProjectId: ucodeProjectID,
		Limit:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("get environment list: %w", err)
	}

	envs := envList.GetEnvironments()
	if len(envs) == 0 {
		return nil, fmt.Errorf("no environment found for project %s", ucodeProjectID)
	}
	env := envs[0]

	apiKeys, err := p.h.authService.ApiKey().GetList(ctx, &as.GetListReq{
		EnvironmentId: env.GetId(),
		ProjectId:     ucodeProjectID,
		Limit:         1,
	})
	if err != nil {
		return nil, fmt.Errorf("get api keys: %w", err)
	}

	var apiKey string
	if keys := apiKeys.GetData(); len(keys) > 0 {
		apiKey = keys[0].GetAppId()
	}

	return &models.ProjectData{
		UcodeProjectId: ucodeProjectID,
		EnvironmentId:  env.GetId(),
		ResourceEnvId:  env.GetResourceEnvironmentId(),
		ApiKey:         apiKey,
	}, nil
}

// inspectCode answers questions about existing project file contents.
func (p *ChatProcessor) inspectCode(ctx context.Context, userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	content := helper.BuildInspectorMessage(userQuestion, filesContext)
	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(content, imageURLs))

	response, err := p.callAnthropicWithTracking(
		ctx,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.InspectorMaxTokens,
			System:    helper.PromptInspector,
			Messages:  messages,
		},
		timeoutInspector,
		"Inspecting code context",
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

	result, err := callWithTool[models.SonnetPlanResult](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.ClaudeModel,
			MaxTokens:  p.baseConf.PlannerMaxTokens,
			System:     helper.PromptPlanner,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolPlanChanges},
			ToolChoice: helper.ForcedTool(helper.ToolPlanChanges.Name),
		},
		timeoutPlanner,
		"Planning code changes",
	)
	if err != nil {
		return nil, fmt.Errorf("planner: %w", err)
	}
	return result, nil
}

// editCode applies changes to specific files in an existing project.
// If the planned files are not found in the project yet, falls back to free generation.
// Uses tool use — no JSON parsing or repair needed.
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
		log.Printf("[CODE] planned files not found in project, falling back to free generation")
		systemPrompt = helper.PromptAdminPanelGenerator
		contentBlocks = buildContentBlocksWithImages(clarified, imageURLs)
	}

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.ClaudeModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     systemPrompt,
			Messages:   buildMessagesWithHistory(chatHistory, contentBlocks),
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Applying/generating code changes",
	)
	if err != nil {
		return nil, fmt.Errorf("code editor: %w", err)
	}

	return &models.ParsedClaudeResponse{
		Project:     project,
		Description: "Changes applied successfully.",
	}, nil
}

// callArchitect plans the full project structure — tables, fields, UI layout.
// Uses Anthropic tool use so the response is a validated JSON object — no parsing or repair needed.
func (p *ChatProcessor) callArchitect(ctx context.Context, clarified string, imageURLs []string) (*models.ArchitectPlan, error) {
	plan, err := callWithTool[models.ArchitectPlan](
		p, ctx,
		models.AnthropicToolRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.PromptArchitect,
			Messages: []models.ChatMessage{
				{Role: "user", Content: buildContentBlocksWithImages(clarified, imageURLs)},
			},
			Tools:      []models.ClaudeFunctionTool{helper.ToolArchitectPlan},
			ToolChoice: helper.ForcedTool(helper.ToolArchitectPlan.Name),
		},
		timeoutArchitect,
		"Architecting project structure",
	)
	if err != nil {
		return nil, fmt.Errorf("architect: %w", err)
	}
	return plan, nil
}

func (p *ChatProcessor) generateProject(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	apiConfig := buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.PromptAdminPanelGenerator,
			Messages: []models.ChatMessage{
				{Role: "user", Content: buildContentBlocksWithImages(clarified+"\n\n"+apiConfig, imageURLs)},
			},
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating full project code",
	)
	if err != nil {
		return nil, fmt.Errorf("generate project: %w", err)
	}

	if len(project.Files) == 0 {
		return nil, fmt.Errorf("generate project: claude returned empty project")
	}

	log.Printf("[CODE] ✅ Generate project completed. Built %d files:", len(project.Files))
	for _, f := range project.Files {
		log.Printf("  - %s (%d bytes)", f.Path, len(f.Content))
	}

	return &models.ParsedClaudeResponse{Project: project}, nil
}

// generateAdminPanel generates an admin panel using the template system with pre-built hooks and utilities.
// Uses tool use — the model must populate emit_project with the exact schema.
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
			fmt.Fprintf(&templateCtx, "\n### FILE: %s\n```\n%s\n```\n", f.Path, f.Content)
		}
	}

	finalPrompt := clarified + "\n\n" + apiConfig + "\n" + templateCtx.String()

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.PromptAdminPanelGenerator,
			Messages: []models.ChatMessage{
				{Role: "user", Content: buildContentBlocksWithImages(finalPrompt, imageURLs)},
			},
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating admin panel code",
	)
	if err != nil {
		return nil, fmt.Errorf("generate admin panel: %w", err)
	}

	if len(project.Files) == 0 {
		return nil, fmt.Errorf("generate admin panel: claude returned empty project")
	}

	// Merge any template files the AI didn't regenerate (safety net)
	if len(templateFiles) > 0 {
		generatedPaths := make(map[string]struct{}, len(project.Files))
		for _, f := range project.Files {
			generatedPaths[f.Path] = struct{}{}
		}
		for _, tf := range templateFiles {
			if _, exists := generatedPaths[tf.Path]; !exists {
				project.Files = append(project.Files, models.ProjectFile{
					Path:    tf.Path,
					Content: tf.Content,
				})
				log.Printf("[CODE] merged missing template file: %s", tf.Path)
			}
		}
	}

	log.Printf("[CODE] ✅ Admin panel generation completed. Total %d files:", len(project.Files))
	for _, f := range project.Files {
		log.Printf("  - %s (%d bytes)", f.Path, len(f.Content))
	}

	return &models.ParsedClaudeResponse{Project: project}, nil
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


func (p *ChatProcessor) callAnthropicWithTracking(ctx context.Context, req models.AnthropicRequest, timeout time.Duration, description string) (string, error) {
	log.Printf("[AI] Calling Anthropic: %s", description)
	response, err := helper.CallAnthropicAPI(p.baseConf, req, timeout)
	if err != nil {
		log.Printf("[AI] Anthropic error for %s: %v", description, err)
		return "", err
	}

	// Record token usage in background
	go func() {
		var parsed struct {
			Usage models.ClaudeUsage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(response), &parsed); err == nil && (parsed.Usage.InputTokens > 0 || parsed.Usage.OutputTokens > 0) {
			projectID := p.ucodeProjectID
			if p.mcpUcodeProjectID != "" {
				projectID = p.mcpUcodeProjectID
			}

			_, recErr := p.service.CompanyService().Billing().RecordAiTokenUsage(context.Background(), &pb.RecordAiTokenUsageRequest{
				ProjectId:    projectID,
				InputTokens:  int32(parsed.Usage.InputTokens),
				OutputTokens: int32(parsed.Usage.OutputTokens),
				Model:        req.Model,
				Description:  description,
			})
			if recErr != nil {
				log.Printf("[TOKEN RECORD] error recording usage for %s: %v", description, recErr)
			}
		}
	}()

	return response, nil
}

// ============================================================================
// MICROFRONTEND EDIT HELPERS
// ============================================================================

// runMicrofrontendEdit fetches the current files from u-gen, asks the AI to edit
// them, then pushes the result back to u-gen. No McpProject is touched.
func (p *ChatProcessor) runMicrofrontendEdit(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string, existingFiles []models.GitlabFileChange) (*models.ParsedClaudeResponse, error) {
	log.Printf("[MICROFE EDIT] planning changes for microfrontend id=%s", p.microfrontendID)

	plan, err := p.planChanges(ctx, clarified, fileGraphJSON, chatHistory, len(imageURLs) > 0)
	if err != nil {
		return nil, err
	}
	log.Printf("[MICROFE EDIT] planner: files_to_change=%d files_to_create=%d", len(plan.FilesToChange), len(plan.FilesToCreate))

	neededPaths := make([]string, 0, len(plan.FilesToChange))
	for _, f := range plan.FilesToChange {
		neededPaths = append(neededPaths, f.Path)
	}

	filesContext := p.buildMicrofrontendFilesContext(existingFiles, neededPaths)

	edited, err := p.editCode(ctx, clarified, plan, filesContext, chatHistory, imageURLs)
	if err != nil {
		return nil, err
	}

	// With tool use, edited.Project is always populated (the tool schema requires files[]).
	// An empty files list means the model has nothing to change — return description only.
	if edited.Project == nil || len(edited.Project.Files) == 0 {
		log.Printf("[MICROFE EDIT] editor returned no files — nothing to push")
		return &models.ParsedClaudeResponse{Description: edited.Description}, nil
	}

	log.Printf("[MICROFE EDIT] pushing %d file(s) to u-gen branch", len(edited.Project.Files))
	if err = p.pushMicrofrontendChanges(ctx, edited.Project.Files); err != nil {
		return nil, fmt.Errorf("failed to push microfrontend changes: %w", err)
	}

	return &models.ParsedClaudeResponse{Description: edited.Description}, nil
}

// runMicrofrontendInspect answers questions about the microfrontend's current code
// by loading the requested files from the u-gen branch.
func (p *ChatProcessor) runMicrofrontendInspect(ctx context.Context, userQuestion string, filesNeeded []string, chatHistory []models.ChatMessage, imageURLs []string, existingFiles []models.GitlabFileChange) (*models.ParsedClaudeResponse, error) {
	filesContext := p.buildMicrofrontendFilesContext(existingFiles, filesNeeded)
	answer, err := p.inspectCode(ctx, userQuestion, filesContext, chatHistory, imageURLs)
	if err != nil {
		return nil, err
	}
	return &models.ParsedClaudeResponse{Description: answer}, nil
}

// fetchMicrofrontendFiles calls the function service to get all files from the
// microfrontend's u-gen branch. Returns a flat list of {FilePath, Content}.
func (p *ChatProcessor) fetchMicrofrontendFiles(ctx context.Context) ([]models.GitlabFileChange, error) {
	url := p.baseConf.GoFunctionServiceHost + p.baseConf.GoFunctionServiceHTTPPort +
		"/v2/functions/micro-frontend/files?repo_id=" + p.microfrontendRepoID

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", p.authToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("function service call: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("function service returned %d: %s", resp.StatusCode, string(respBytes))
	}

	// Response shape: {"status":"...","data":{"files":[{"path":"...","content":"..."}]}}
	var result struct {
		Data struct {
			Files []struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			} `json:"files"`
		} `json:"data"`
	}
	if err = json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	files := make([]models.GitlabFileChange, 0, len(result.Data.Files))
	for _, f := range result.Data.Files {
		files = append(files, models.GitlabFileChange{
			FilePath: f.Path,
			Content:  f.Content,
		})
	}
	log.Printf("[MICROFE EDIT] fetched %d files from u-gen", len(files))
	return files, nil
}

// pushMicrofrontendChanges sends AI-generated files to the function service which
// commits them to the u-gen branch of the microfrontend's repo.
func (p *ChatProcessor) pushMicrofrontendChanges(ctx context.Context, generatedFiles []models.ProjectFile) error {
	repoIDInt := 0
	fmt.Sscanf(p.microfrontendRepoID, "%d", &repoIDInt)
	if repoIDInt == 0 {
		return fmt.Errorf("invalid microfrontend_repo_id: %q", p.microfrontendRepoID)
	}

	files := make([]models.GitlabFileChange, 0, len(generatedFiles))
	for _, f := range generatedFiles {
		files = append(files, models.GitlabFileChange{
			FilePath: f.Path,
			Content:  f.Content,
		})
	}

	type pushReq struct {
		RepoID int                       `json:"repo_id"`
		Files  []models.GitlabFileChange `json:"files"`
	}

	bodyBytes, err := json.Marshal(pushReq{RepoID: repoIDInt, Files: files})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseConf.GoFunctionServiceHost + p.baseConf.GoFunctionServiceHTTPPort +
		"/v2/functions/micro-frontend/push-changes"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", p.authToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("function service call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("function service returned %d: %s", resp.StatusCode, string(respBytes))
	}
	return nil
}

// buildMicrofrontendFileGraphJSON builds the same file graph JSON that the router
// uses, from a flat list of GitlabFileChange entries (no per-file graph data).
func (p *ChatProcessor) buildMicrofrontendFileGraphJSON(files []models.GitlabFileChange) string {
	if len(files) == 0 {
		return "{}"
	}
	graph := make(map[string]models.GraphNode, len(files))
	for _, f := range files {
		graph[f.FilePath] = models.GraphNode{Path: f.FilePath}
	}
	jsonBytes, err := json.Marshal(graph)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

// buildMicrofrontendFilesContext returns the file contents for the paths the
// planner requested, formatted for the code-editor prompt.
func (p *ChatProcessor) buildMicrofrontendFilesContext(files []models.GitlabFileChange, paths []string) string {
	if len(paths) == 0 || len(files) == 0 {
		return "No existing files to modify."
	}
	pathSet := make(map[string]bool, len(paths))
	for _, path := range paths {
		pathSet[path] = true
	}
	var sb strings.Builder
	for _, f := range files {
		if pathSet[f.FilePath] {
			sb.WriteString(fmt.Sprintf("\n\n### FILE: %s\n```\n%s\n```", f.FilePath, f.Content))
		}
	}
	if sb.Len() == 0 {
		return "No matching files found."
	}
	return sb.String()
}

// publishToMicrofrontend calls the function service to create a microfrontend
// and push all AI-generated files to the u-gen branch.
// It stores the resulting microfrontend ID on the processor for the response.
func (p *ChatProcessor) publishToMicrofrontend(ctx context.Context, projectName, path string, generated *models.ParsedClaudeResponse, projectData *models.ProjectData) error {
	if generated == nil || generated.Project == nil || len(generated.Project.Files) == 0 {
		return fmt.Errorf("no generated files to publish")
	}

	// Convert ProjectFile list to the format the function service expects
	files := make([]models.GitlabFileChange, 0, len(generated.Project.Files))
	for _, f := range generated.Project.Files {
		files = append(files, models.GitlabFileChange{
			FilePath: f.Path,
			Content:  f.Content,
		})
	}

	reqBody := models.PublishAiMicroFrontendRequest{
		ProjectId:     projectData.UcodeProjectId,
		EnvironmentId: projectData.EnvironmentId,
		Name:          projectName,
		Path:          path,
		FrameworkType: "REACT",
		Files:         files,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseConf.GoFunctionServiceHost + p.baseConf.GoFunctionServiceHTTPPort + "/v2/functions/micro-frontend/publish-ai"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", p.authToken)

	client := &http.Client{Timeout: timeoutPublishMicrofrontend}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("function service call failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read function service response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("function service returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var result models.PublishAiMicroFrontendResponse
	if err = json.Unmarshal(respBytes, &result); err != nil {
		return fmt.Errorf("parse function service response: %w", err)
	}

	p.microfrontendID = result.Data.ID
	p.microfrontendRepoID = result.Data.RepoId
	log.Printf("[MICROFRONTEND] created id=%s repo_id=%s url=%s", result.Data.ID, result.Data.RepoId, result.Data.Url)
	return nil
}

// slugify converts a project name to a lowercase hyphen-separated slug
// valid for use as a GitLab path (only [a-z0-9-]).
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) == 0 {
		s = "ai-project"
	}
	return s
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

	p.mcpUcodeProjectID = backendProject.GetProjectId()

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
