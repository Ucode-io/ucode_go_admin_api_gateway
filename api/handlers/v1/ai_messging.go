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
	timeoutHaiku     = 90 * time.Second
	timeoutArchitect = 720 * time.Second
	timeoutInspector = 180 * time.Second
	timeoutPlanner   = 180 * time.Second
	timeoutCoder     = 720 * time.Second
)

type ChatProcessor struct {
	h              *HandlerV1
	service        services.ServiceManagerI
	baseConf       config.BaseConfig
	chatId         string
	mcpProjectID   string
	resourceEnvID  string
	ucodeProjectID string
	builderResourceID string

	userID       string
	clientTypeID string
	roleID       string

	// Schema cache for DB assistant — avoids N+1 gRPC calls per message
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
// HTTP HANDLER
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

	if strings.TrimSpace(userMessage.Content) == "" {
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
		h, service, h.baseConf, chatId, chat.GetProjectId(), resourceEnvID, realProjectID,
		authInfo.GetUserIdAuth(), authInfo.GetClientTypeId(), authInfo.GetRoleId(),
	)

	chatHistory, err := processor.getChatHistory(ctx)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, fmt.Sprintf("failed to fetch history: %v", err))
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
		aiResponse.Description = "Project has been updated."
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
		updateReq := &pbo.UpdateChatRequest{
			ResourceEnvId: resourceEnvID,
			Id:            chatId,
			Title:         truncateString(userMessage.Content, 100),
			Description:   truncateString(aiResponse.Description, 255),
			ProjectId:     processor.mcpProjectID,
		}
		_, _ = service.GoObjectBuilderService().AiChat().UpdateChat(ctx, updateReq)
	}

	h.HandleResponse(c, status_http.Created, map[string]any{
		"message":        message,
		"project":        updateProject,
		"mcp_project_id": processor.mcpProjectID,
		"pending_action": aiResponse.PendingAction,
	})
}

// ============================================================================
// CORE BUSINESS LOGIC (ROUTING & FLOWS)
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
			log.Printf("[ROUTER] Warning: failed to get project files (mcpProjectID=%s): %v", p.mcpProjectID, err)
			projectData = nil
		}

		if projectData != nil {
			fileGraphJSON, _ = p.buildFileGraphJSON(projectData)
		}
	}

	haikuResult, err := p.callHaikuRouter(req.Content, fileGraphJSON, chatHistory, hasImages)
	if err != nil {
		return nil, err
	}

	log.Printf(
		"[ROUTER] next_step=%v intent=%s has_images=%v files_needed=%v",
		haikuResult.NextStep, haikuResult.Intent, haikuResult.HasImages, len(haikuResult.FilesNeeded),
	)

	if !haikuResult.NextStep {
		return &models.ParsedClaudeResponse{Description: haikuResult.Reply}, nil
	}

	switch haikuResult.Intent {
	case "project_question":
		return &models.ParsedClaudeResponse{Description: haikuResult.Reply}, nil

	case "project_inspect":
		if projectData == nil {
			return &models.ParsedClaudeResponse{Description: "No project exists yet. Please create a project first by describing what you want to build."}, nil
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
	var hasImages = len(imageURLs) > 0

	if projectData == nil || len(projectData.GetProjectFiles()) == 0 {
		log.Println("[CODER] New Project Detected: Routing to Architect & SystemPromptAiChat")
		return p.handleNewProjectPhase(ctx, clarified, chatHistory, imageURLs, projectName)
	}

	plan, err := p.callSonnetPlanner(ctx, clarified, fileGraphJSON, chatHistory, hasImages)
	if err != nil {
		return nil, err
	}

	log.Printf("[PLANNER] Changes planned: modify=%d, create=%d", len(plan.FilesToChange), len(plan.FilesToCreate))

	var neededPaths = make([]string, 0, len(plan.FilesToChange))

	for _, f := range plan.FilesToChange {
		neededPaths = append(neededPaths, f.Path)
	}

	filesContext := p.buildFilesContext(projectData, neededPaths)

	return p.callSonnetCoder(ctx, clarified, plan, filesContext, chatHistory, imageURLs)
}

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
		if buildErr := createBackendFromPlan(context.Background(), bPlan, resourceEnvId, envId, p.service); buildErr != nil {
			log.Printf("[CRITICAL ERROR] Failed to build backend from plan for resourceEnv %s: %v", resourceEnvId, buildErr)
		}
	}(plan, projectData.ResourceEnvId, projectData.EnvironmentId)

	if plan.ProjectType == "admin_panel" {
		log.Printf("[CODER] Admin panel detected — using template-based generation")
		return p.callSonnetCoderWithTemplate(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
	}

	return p.callSonnetCoderNewProject(ctx, clarified, imageURLs, plan, projectData.ApiKey, projectData.EnvironmentId)
}

// ============================================================================
// ANTHROPIC AI INTEGRATIONS
// ============================================================================

func (p *ChatProcessor) callHaikuRouter(userPrompt, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.HaikuRoutingResult, error) {
	var (
		content  = helper.ProcessHaikuPrompt(userPrompt, fileGraphJSON, hasImages)
		messages = buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})
	)

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
		return nil, fmt.Errorf("haiku router api call failed: %w", err)
	}

	result, err := helper.ParseHaikuRoutingResult(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse haiku result: %w", err)
	}

	result.HasImages = hasImages
	return result, nil
}

func (p *ChatProcessor) callSonnetInspector(ctx context.Context, userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	var (
		content  = helper.ProcessSonnetInspectorPrompt(userQuestion, filesContext)
		messages = buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(content, imageURLs))
	)

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
		return "", fmt.Errorf("sonnet inspector api call failed: %w", err)
	}

	answer, err := helper.ExtractPlainText(response)
	if err != nil {
		return "", fmt.Errorf("failed to extract plain text from inspector: %w", err)
	}
	return answer, nil
}

func (p *ChatProcessor) callSonnetPlanner(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.SonnetPlanResult, error) {
	var (
		content  = helper.ProcessSonnetPlanPrompt(clarified, fileGraphJSON, hasImages)
		messages = buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})
	)

	response, err := helper.CallAnthropicAPI(
		p.baseConf, models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.SystemPromptSonnetPlanner,
			Messages:  messages,
		},
		timeoutPlanner,
	)
	if err != nil {
		return nil, fmt.Errorf("sonnet planner api call failed: %w", err)
	}

	result, err := helper.ParseSonnetPlanResult(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sonnet plan: %w", err)
	}

	return result, nil
}

func (p *ChatProcessor) callSonnetCoder(ctx context.Context, clarified string, plan *models.SonnetPlanResult, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	var (
		hasFiles      = filesContext != "No existing files to modify." && filesContext != "No matching files found."
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

	messages := buildMessagesWithHistory(chatHistory, contentBlocks)

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    systemPrompt,
			Messages:  messages,
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("sonnet coder api call failed: %w", err)
	}

	parsedProject, err := helper.ParseClaudeResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse claude coder response: %w", err)
	}

	return parsedProject, nil
}

func (p *ChatProcessor) callArchitect(ctx context.Context, clarified string, imageURLs []string) (*models.ArchitectPlan, error) {
	var (
		messages = []models.ChatMessage{
			{
				Role:    "user",
				Content: buildContentBlocksWithImages(clarified, imageURLs),
			},
		}
		plan models.ArchitectPlan
	)

	response, err := helper.CallAnthropicAPI(
		p.baseConf,
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.SystemPromptArchitect,
			Messages:  messages,
		},
		timeoutArchitect,
	)
	if err != nil {
		return nil, fmt.Errorf("architect api call failed: %w", err)
	}

	text, err := helper.ExtractPlainText(response)
	if err != nil {
		return nil, fmt.Errorf("architect text extraction failed: %w", err)
	}

	text = helper.CleanJSONResponse(text)

	if err = json.Unmarshal([]byte(text), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal architect plan: %w", err)
	}

	return &plan, nil
}

func (p *ChatProcessor) callSonnetCoderNewProject(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	var apiContext strings.Builder

	apiContext.WriteString(
		fmt.Sprintf("\n====================================\nAPI CONFIGURATION FOR FRONTEND\n====================================\nVITE_API_BASE_URL: %s\nVITE_X_API_KEY: %s\n\nTables to use:\n",
			p.baseConf.UcodeBaseUrl, apiKey,
		),
	)

	for _, t := range plan.Tables {
		apiContext.WriteString(fmt.Sprintf("- Table: %s, slug: %s\n", t.Label, t.Slug))
		for _, f := range t.Fields {
			apiContext.WriteString(fmt.Sprintf("  * field: %s, type: %s\n", f.Slug, f.Type))
		}
	}
	apiContext.WriteString("\nUse this UI Structure provided by the Architect:\n" + plan.UIStructure + "\n")

	var (
		fullPrompt = clarified + "\n\n" + apiContext.String()
		messages   = []models.ChatMessage{
			{
				Role:    "user",
				Content: buildContentBlocksWithImages(fullPrompt, imageURLs),
			},
		}
	)

	response, err := helper.CallAnthropicAPI(
		p.baseConf, models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.SystemPromptAiChat,
			Messages:  messages,
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("new project coder api call failed: %w", err)
	}

	parsedProject, err := helper.ParseClaudeResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new project response: %w", err)
	}

	if parsedProject.Project == nil {
		return nil, fmt.Errorf("claude returned empty project data")
	}

	log.Printf("[NEW PROJECT GENERATED] Files created: %d", len(parsedProject.Project.Files))
	return parsedProject, nil
}

func (p *ChatProcessor) callSonnetCoderWithTemplate(ctx context.Context, clarified string, imageURLs []string, plan *models.ArchitectPlan, apiKey, envId string) (*models.ParsedClaudeResponse, error) {
	var apiContext strings.Builder

	apiContext.WriteString(
		fmt.Sprintf("\n====================================\nAPI CONFIGURATION FOR FRONTEND\n====================================\nVITE_API_BASE_URL: %s\nVITE_X_API_KEY: %s\n\nTables to use:\n",
			p.baseConf.UcodeBaseUrl, apiKey,
		),
	)

	for _, t := range plan.Tables {
		apiContext.WriteString(fmt.Sprintf("- Table: %s, slug: %s\n", t.Label, t.Slug))
		for _, f := range t.Fields {
			apiContext.WriteString(fmt.Sprintf("  * field: %s, type: %s\n", f.Slug, f.Type))
		}
	}
	apiContext.WriteString("\nUse this UI Structure provided by the Architect:\n" + plan.UIStructure + "\n")

	var (
		fullPrompt = clarified + "\n\n" + apiContext.String()
		messages   = []models.ChatMessage{
			{
				Role:    "user",
				Content: buildContentBlocksWithImages(fullPrompt, imageURLs),
			},
		}
	)

	response, err := helper.CallAnthropicAPI(
		p.baseConf, models.AnthropicRequest{
			Model:     p.baseConf.ClaudeModel,
			MaxTokens: p.baseConf.CoderMaxTokens,
			System:    helper.SystemPromptAiChatTemplate,
			Messages:  messages,
		},
		timeoutCoder,
	)
	if err != nil {
		return nil, fmt.Errorf("template coder api call failed: %w", err)
	}

	parsedProject, err := helper.ParseClaudeResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template coder response: %w", err)
	}

	if parsedProject.Project == nil {
		return nil, fmt.Errorf("claude returned empty project data")
	}

	templateFiles := GetTemplate("admin_panel")
	if templateFiles != nil {
		parsedProject.Project.Files = MergeTemplateWithAIFiles(templateFiles, parsedProject.Project.Files)
		log.Printf("[TEMPLATE MERGE] Merged %d template + %d AI files", len(templateFiles), len(parsedProject.Project.Files)-len(templateFiles))
	}

	log.Printf("[NEW PROJECT GENERATED WITH TEMPLATE] Total files: %d", len(parsedProject.Project.Files))
	return parsedProject, nil
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
		return nil, fmt.Errorf("database error getting chat history: %w", err)
	}

	var (
		msgList = messages.GetMessages()
		result  = make([]models.ChatMessage, 0, len(msgList))
	)

	if len(msgList) > 10 {
		msgList = msgList[len(msgList)-10:]
	}

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

	var graph = make(map[string]models.GraphNode, len(project.GetProjectFiles()))

	for _, f := range project.GetProjectFiles() {
		graph[f.GetPath()] = models.GraphNode{
			Path:      f.GetPath(),
			FileGraph: f.GetFileGraph(),
		}
	}

	jsonBytes, err := json.Marshal(graph)
	if err != nil {
		return "{}", fmt.Errorf("failed to marshal file graph: %w", err)
	}

	return string(jsonBytes), nil
}

func (p *ChatProcessor) buildFilesContext(project *pbo.McpProject, paths []string) string {
	if len(paths) == 0 || project == nil {
		return "No existing files to modify."
	}

	var (
		pathSet = make(map[string]bool, len(paths))
		sb      strings.Builder
	)

	for _, path := range paths {
		pathSet[path] = true
	}

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
		return nil, fmt.Errorf("invalid project data received for saving")
	}

	projectEnv, err := helperFunc.ConvertMapToStruct(req.Project.Env)
	if err != nil {
		return nil, fmt.Errorf("failed to convert project env: %w", err)
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

	createdProject, err := p.service.GoObjectBuilderService().McpProject().UpdateMcpProject(ctx, &pbo.McpProject{
		Id:            p.mcpProjectID,
		ResourceEnvId: p.resourceEnvID,
		Title:         truncateString(req.Project.ProjectName, 255),
		Description:   truncateString(req.Description, 255),
		ProjectFiles:  projectFiles,
		ProjectEnv:    projectEnv,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update MCP project in DB: %w", err)
	}

	return createdProject, nil
}

func (p *ChatProcessor) createUcodeProject(ctx context.Context, projectName string, existingMcpID string) (*models.ProjectData, error) {
	currentProject, err := p.h.companyServices.Project().GetById(
		ctx, &pb.GetProjectByIdRequest{
			ProjectId: p.ucodeProjectID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get current project info: %w", err)
	}

	backendProject, err := p.h.companyServices.Project().Create(
		ctx, &pb.CreateProjectRequest{
			Title:        projectName,
			CompanyId:    currentProject.GetCompanyId(),
			K8SNamespace: currentProject.GetK8SNamespace(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend project: %w", err)
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
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	// Fetch service resource to get the correct resource_environment_id for the builder service
	resource, err := p.h.companyServices.ServiceResource().GetSingle(
		ctx,
		&pb.GetSingleServiceResourceReq{
			ProjectId:     backendProject.GetProjectId(),
			EnvironmentId: env.GetId(),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch service resource details: %w", err)
	}

	apiKeys, err := p.h.authService.ApiKey().GetList(
		ctx, &as.GetListReq{
			EnvironmentId: env.GetId(),
			ProjectId:     backendProject.GetProjectId(),
			Limit:         1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch api keys: %w", err)
	}

	var apiKey string

	if len(apiKeys.GetData()) > 0 {
		apiKey = apiKeys.GetData()[0].GetAppId()
	}

	var (
		mcpProjectID = existingMcpID
		status       = "ready"
	)

	if mcpProjectID != "" {
		// Update existing skeleton project
		_, err = p.service.GoObjectBuilderService().McpProject().UpdateMcpProject(
			ctx, &pbo.McpProject{
				ResourceEnvId:  p.resourceEnvID,
				Id:             mcpProjectID,
				Title:          projectName,
				Description:    "Provisioned by AI architect",
				UcodeProjectId: backendProject.GetProjectId(),
				ApiKey:         apiKey,
				EnvironmentId:  env.GetId(),
				Status:         status,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update MCP project: %w", err)
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
				Status:         status,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCP project link: %w", err)
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
// UTILITIES & FORMATTING
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
	var blocks []models.ContentBlock

	for _, imageURL := range imageURLs {
		if strings.TrimSpace(imageURL) != "" {
			blocks = append(blocks, models.ContentBlock{
				Type: "image",
				Source: &models.ImageSource{
					Type: "url",
					URL:  imageURL,
				},
			})
		}
	}

	blocks = append(blocks, models.ContentBlock{
		Type: "text",
		Text: textContent,
	})

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
