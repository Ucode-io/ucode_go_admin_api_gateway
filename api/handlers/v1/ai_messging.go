package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
)

type messagingStc struct {
	service       services.ServiceManagerI
	baseConf      config.BaseConfig
	chatId        string
	mcpProjectId  string
	resourceEnvId string

	cachedProject *pbo.McpProject
}

func newMessaging(service services.ServiceManagerI, baseConf config.BaseConfig, chatId, mcpProjectId, resourceEnvId string) *messagingStc {
	return &messagingStc{
		service:       service,
		baseConf:      baseConf,
		chatId:        chatId,
		mcpProjectId:  mcpProjectId,
		resourceEnvId: resourceEnvId,
	}
}

func (h *HandlerV1) CreateAiChatMessage(c *gin.Context) {
	var (
		userMessage models.NewMessageReq
		chatId      = c.Param("chat-id")
	)

	if err := c.ShouldBindJSON(&userMessage); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if strings.TrimSpace(userMessage.Content) == "" {
		h.HandleResponse(c, status_http.BadRequest, "content is required")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	chat, err := service.GoObjectBuilderService().AiChat().GetChat(
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

	if chat.GetProjectId() == "" {
		h.HandleResponse(c, status_http.GRPCError, "project not found for this chat")
		return
	}

	messaging := newMessaging(service, h.baseConf, chatId, chat.GetProjectId(), resourceEnvId)

	chatHistory, err := messaging.getChatHistory(c.Request.Context())
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = service.GoObjectBuilderService().AiChat().CreateMessage(
		c.Request.Context(),
		&pbo.CreateMessageRequest{
			ChatId:        chatId,
			Role:          "user",
			Content:       userMessage.Content,
			Images:        userMessage.Images,
			ResourceEnvId: resourceEnvId,
		})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	aiResponse, err := messaging.routeAndProcess(c.Request.Context(), userMessage, chatHistory)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	message, err := service.GoObjectBuilderService().AiChat().CreateMessage(
		c.Request.Context(),
		&pbo.CreateMessageRequest{
			ChatId:        chatId,
			Role:          "assistant",
			Content:       aiResponse.Description,
			Images:        userMessage.Images,
			ResourceEnvId: resourceEnvId,
		})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		updateProject  *pbo.McpProject
		isFirstMessage = len(chatHistory) == 0
	)

	if aiResponse.Project != nil {
		updateProject, err = messaging.saveProject(c.Request.Context(), aiResponse)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	if isFirstMessage {
		chatTitle := truncateString(userMessage.Content, 80)
		chatDescription := truncateString(aiResponse.Description, 200)

		_, _ = service.GoObjectBuilderService().AiChat().UpdateChat(
			c.Request.Context(),
			&pbo.UpdateChatRequest{
				ResourceEnvId: resourceEnvId,
				Id:            chatId,
				Title:         chatTitle,
				Description:   chatDescription,
			},
		)
	}

	h.HandleResponse(c, status_http.Created,
		map[string]any{
			"message": message,
			"project": updateProject,
		},
	)
}

func (m *messagingStc) routeAndProcess(ctx context.Context, req models.NewMessageReq, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	hasImages := len(req.Images) > 0

	fileGraphJSON, err := m.getFileGraphJSON(ctx)
	if err != nil {
		return nil, err
	}

	haikuResult, err := m.callHaikuRouter(req.Content, fileGraphJSON, chatHistory, hasImages)
	if err != nil {
		return nil, err
	}

	log.Printf("HAIKU ROUTING: next_step=%v intent=%s has_images=%v files_needed=%v", haikuResult.NextStep, haikuResult.Intent, haikuResult.HasImages, haikuResult.FilesNeeded)

	if !haikuResult.NextStep {
		return &models.ParsedClaudeResponse{
			Description: haikuResult.Reply,
		}, nil
	}

	switch haikuResult.Intent {

	case "project_question":
		return &models.ParsedClaudeResponse{
			Description: haikuResult.Reply,
		}, nil

	case "project_inspect":
		return m.runInspectFlow(ctx, req.Content, haikuResult.FilesNeeded, chatHistory, req.Images)

	case "code_change":
		return m.runCodeFlow(ctx, haikuResult.Clarified, fileGraphJSON, chatHistory, req.Images)
	}

	return &models.ParsedClaudeResponse{
		Description: haikuResult.Reply,
	}, nil
}

func (m *messagingStc) runInspectFlow(ctx context.Context, userQuestion string, filesNeeded []string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	filesContext, err := m.getFilesContext(ctx, filesNeeded)
	if err != nil {
		return nil, err
	}

	answer, err := m.callSonnetInspector(userQuestion, filesContext, chatHistory, imageURLs)
	if err != nil {
		return nil, err
	}

	return &models.ParsedClaudeResponse{
		Description: answer,
	}, nil
}

func (m *messagingStc) runCodeFlow(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	hasImages := len(imageURLs) > 0

	plan, err := m.callSonnetPlanner(clarified, fileGraphJSON, chatHistory, hasImages)
	if err != nil {
		return nil, err
	}

	log.Printf("SONNET PLAN: change=%d create=%d", len(plan.FilesToChange), len(plan.FilesToCreate))

	neededPaths := make([]string, 0, len(plan.FilesToChange))
	for _, f := range plan.FilesToChange {
		neededPaths = append(neededPaths, f.Path)
	}

	filesContext, err := m.getFilesContext(ctx, neededPaths)
	if err != nil {
		return nil, err
	}

	return m.callSonnetCoder(clarified, plan, filesContext, chatHistory, imageURLs)
}

// --- Вызовы к Anthropic API ---

func (m *messagingStc) callHaikuRouter(userPrompt, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.HaikuRoutingResult, error) {
	log.Println("USER PROMPT:", userPrompt)

	var (
		content       = helper.ProcessHaikuPrompt(userPrompt, fileGraphJSON, hasImages)
		contentBlocks = []models.ContentBlock{{Type: "text", Text: content}}
		messages      = buildMessagesWithHistory(chatHistory, contentBlocks)
	)

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeHaikuModel,
			MaxTokens: m.baseConf.RouterMaxTokens,
			System:    helper.SystemPromptHaikuRouter,
			Messages:  messages,
		},
		60*time.Second,
	)
	if err != nil {
		log.Println("ERROR IN HAIKU ROUTER:", err)
		return nil, err
	}

	result, err := helper.ParseHaikuRoutingResult(rawResp)
	if err != nil {
		log.Println("ERROR PARSING HAIKU RESULT:", err)
		return nil, err
	}

	if hasImages {
		result.HasImages = true
	}

	return result, nil
}

func (m *messagingStc) callSonnetInspector(userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	var (
		content       = helper.ProcessSonnetInspectorPrompt(userQuestion, filesContext)
		contentBlocks = buildContentBlocksWithImages(content, imageURLs)
		messages      = buildMessagesWithHistory(chatHistory, contentBlocks)
	)

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeModel,
			MaxTokens: m.baseConf.InspectorMaxTokens,
			System:    helper.SystemPromptSonnetInspector,
			Messages:  messages,
		},
		120*time.Second,
	)
	if err != nil {
		log.Println("ERROR IN SONNET INSPECTOR:", err)
		return "", err
	}

	answer, err := helper.ExtractPlainText(rawResp)
	if err != nil {
		log.Println("ERROR EXTRACTING SONNET INSPECTOR TEXT:", err)
		return "", err
	}

	return answer, nil
}

func (m *messagingStc) callSonnetPlanner(clarified, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.SonnetPlanResult, error) {
	var (
		content       = helper.ProcessSonnetPlanPrompt(clarified, fileGraphJSON, hasImages)
		contentBlocks = []models.ContentBlock{{Type: "text", Text: content}}
		messages      = buildMessagesWithHistory(chatHistory, contentBlocks)
	)

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeModel,
			MaxTokens: m.baseConf.PlannerMaxTokens,
			System:    helper.SystemPromptSonnetPlanner,
			Messages:  messages,
		},
		120*time.Second,
	)
	if err != nil {
		log.Println("ERROR IN SONNET PLANNER:", err)
		return nil, err
	}

	result, err := helper.ParseSonnetPlanResult(rawResp)
	if err != nil {
		log.Println("ERROR PARSING SONNET PLAN:", err)
		return nil, err
	}

	return result, nil
}

func (m *messagingStc) callSonnetCoder(clarified string, plan *models.SonnetPlanResult, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	hasImages := len(imageURLs) > 0

	var (
		planJSON, _   = json.Marshal(plan)
		content       = helper.ProcessSonnetCoderPrompt(clarified, string(planJSON), filesContext, hasImages)
		contentBlocks = buildContentBlocksWithImages(content, imageURLs)
		messages      = buildMessagesWithHistory(chatHistory, contentBlocks)
	)

	systemPrompt := helper.SystemPromptSonnetCoder
	if filesContext == "No existing files to modify." {
		systemPrompt = helper.SystemPromptAiChat
	}

	rawResp, err := helper.CallAnthropicAPI(m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeModel,
			MaxTokens: m.baseConf.CoderMaxTokens,
			System:    systemPrompt,
			Messages:  messages,
		},
		720*time.Second,
	)
	if err != nil {
		log.Println("ERROR IN SONNET CODER:", err)
		return nil, err
	}

	parsedProject, err := helper.ParseClaudeResponse(rawResp)
	if err != nil {
		log.Println("ERROR PARSING SONNET CODER RESPONSE:", err)
		return nil, fmt.Errorf("error in parse claude response: %v", err)
	}

	return parsedProject, nil
}

// --- Хелперы ---

func (m *messagingStc) getChatHistory(ctx context.Context) ([]models.ChatMessage, error) {
	messages, err := m.service.GoObjectBuilderService().AiChat().GetMessages(ctx,
		&pbo.GetMessagesRequest{
			ResourceEnvId: m.resourceEnvId,
			ChatId:        m.chatId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting chat history: %w", err)
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

func (m *messagingStc) getProjectData(ctx context.Context) (*pbo.McpProject, error) {
	if m.cachedProject != nil {
		return m.cachedProject, nil
	}

	project, err := m.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		ctx,
		&pbo.McpProjectId{
			ResourceEnvId: m.resourceEnvId,
			Id:            m.mcpProjectId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting project files: %w", err)
	}

	m.cachedProject = project
	return project, nil
}

func (m *messagingStc) getFileGraphJSON(ctx context.Context) (string, error) {
	project, err := m.getProjectData(ctx)
	if err != nil {
		return "{}", err
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
		return "{}", err
	}

	return string(jsonBytes), nil
}

func (m *messagingStc) getFilesContext(ctx context.Context, paths []string) (string, error) {
	if len(paths) == 0 {
		return "No existing files to modify.", nil
	}

	project, err := m.getProjectData(ctx)
	if err != nil {
		return "", err
	}

	var (
		pathSet = make(map[string]bool, len(paths))
		sb      strings.Builder
	)

	for _, p := range paths {
		pathSet[p] = true
	}

	for _, f := range project.GetProjectFiles() {
		if pathSet[f.GetPath()] {
			sb.WriteString(fmt.Sprintf("\n\n### FILE: %s\n```\n%s\n```", f.GetPath(), f.GetContent()))
		}
	}

	if sb.Len() == 0 {
		return "No matching files found.", nil
	}

	return sb.String(), nil
}

func (m *messagingStc) saveProject(ctx context.Context, req *models.ParsedClaudeResponse) (*pbo.McpProject, error) {
	projectEnv, err := helperFunc.ConvertMapToStruct(req.Project.Env)
	if err != nil {
		return nil, err
	}

	var (
		projectFiles []*pbo.McpProjectFiles
		fileGraph    map[string]any
	)

	for _, file := range req.Project.Files {
		if val, ok := req.Project.FileGraph[file.Path].(map[string]any); ok {
			fileGraph = val
		}

		fileGraphStruct, _ := helperFunc.ConvertMapToStruct(fileGraph)

		projectFiles = append(projectFiles, &pbo.McpProjectFiles{
			Path:      file.Path,
			Content:   file.Content,
			FileGraph: fileGraphStruct,
		})

		fileGraph = make(map[string]any)
	}

	createdProject, err := m.service.GoObjectBuilderService().McpProject().UpdateMcpProject(
		ctx,
		&pbo.McpProject{
			Id:            m.mcpProjectId,
			ResourceEnvId: m.resourceEnvId,
			Title:         req.Project.ProjectName,
			Description:   truncateString(req.Description, 300),
			ProjectFiles:  projectFiles,
			ProjectEnv:    projectEnv,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error in saving project: %v", err)
	}

	return createdProject, nil
}

func buildContentBlocksWithImages(textContent string, imageURLs []string) []models.ContentBlock {
	var blocks []models.ContentBlock

	for _, imageURL := range imageURLs {
		if imageURL != "" {
			blocks = append(blocks, models.ContentBlock{
				Type: "image",
				Source: &models.ImageSource{
					Type: "url",
					URL:  imageURL,
				},
			})
		}
	}

	// Then add text
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

func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
