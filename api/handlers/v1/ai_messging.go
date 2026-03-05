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
			Content:       userMessage.Prompt,
			Images:        userMessage.Images,
			ResourceEnvId: resourceEnvId,
		})
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	log.Println("CHAT HISOTRY:", chatHistory)

	aiResponse, err := messaging.routeAndProcess(userMessage, chatHistory)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	// В историю чата сохраняем только description — без кода
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

	var updateProject *pbo.McpProject
	if aiResponse.Project != nil {
		updateProject, err = messaging.saveProject(aiResponse)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.HandleResponse(c, status_http.Created,
		map[string]any{
			"message": message,
			"project": updateProject,
		},
	)
}

func (m *messagingStc) routeAndProcess(req models.NewMessageReq, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	fileGraphJSON, err := m.getFileGraphJSON()
	if err != nil {
		return nil, err
	}

	haikuResult, err := m.callHaikuRouter(req.Prompt, fileGraphJSON, chatHistory)
	if err != nil {
		return nil, err
	}

	log.Printf("HAIKU ROUTING: next_step=%v intent=%s files_needed=%v", haikuResult.NextStep, haikuResult.Intent, haikuResult.FilesNeeded)

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
		return m.runInspectFlow(req.Prompt, haikuResult.FilesNeeded, chatHistory)

	case "code_change":
		return m.runCodeFlow(haikuResult.Clarified, fileGraphJSON, chatHistory)
	}

	// fallback
	return &models.ParsedClaudeResponse{
		Description: haikuResult.Reply,
	}, nil
}

func (m *messagingStc) runInspectFlow(userQuestion string, filesNeeded []string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	filesContext, err := m.getFilesContext(filesNeeded)
	if err != nil {
		return nil, err
	}

	answer, err := m.callSonnetInspector(userQuestion, filesContext, chatHistory)
	if err != nil {
		return nil, err
	}

	return &models.ParsedClaudeResponse{
		Description: answer,
	}, nil
}

func (m *messagingStc) runCodeFlow(clarified, fileGraphJSON string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	plan, err := m.callSonnetPlanner(clarified, fileGraphJSON)
	if err != nil {
		return nil, err
	}

	log.Printf("SONNET PLAN: change=%d create=%d", len(plan.FilesToChange), len(plan.FilesToCreate))

	neededPaths := make([]string, 0, len(plan.FilesToChange))
	for _, f := range plan.FilesToChange {
		neededPaths = append(neededPaths, f.Path)
	}

	filesContext, err := m.getFilesContext(neededPaths)
	if err != nil {
		return nil, err
	}

	return m.callSonnetCoder(clarified, plan, filesContext, chatHistory)
}

// --- Вызовы к Anthropic API ---

func (m *messagingStc) callHaikuRouter(userPrompt, fileGraphJSON string, chatHistory []models.ChatMessage) (*models.HaikuRoutingResult, error) {
	log.Println("USER PROMPT:", userPrompt)

	var (
		content  = helper.ProcessHaikuPrompt(userPrompt, fileGraphJSON)
		messages = buildMessagesWithHistory(chatHistory, content)
	)

	log.Println("MESSAGESaaaaaaa", messages)

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     "claude-haiku-4-5", // TODO m.baseConf.ClaudeHaikuModel
			MaxTokens: 3500,
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

	return result, nil
}

func (m *messagingStc) callSonnetInspector(userQuestion, filesContext string, chatHistory []models.ChatMessage) (string, error) {
	var (
		content  = helper.ProcessSonnetInspectorPrompt(userQuestion, filesContext)
		messages = buildMessagesWithHistory(chatHistory, content)
	)

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeModel,
			MaxTokens: 3500,
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

func (m *messagingStc) callSonnetPlanner(clarified, fileGraphJSON string) (*models.SonnetPlanResult, error) {
	var content = helper.ProcessSonnetPlanPrompt(clarified, fileGraphJSON)

	rawResp, err := helper.CallAnthropicAPI(
		m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeModel,
			MaxTokens: 10000,
			System:    helper.SystemPromptSonnetPlanner,
			Messages: []models.ChatMessage{
				{
					Role:    "user",
					Content: []models.ContentBlock{{Type: "text", Text: content}},
				},
			},
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

func (m *messagingStc) callSonnetCoder(clarified string, plan *models.SonnetPlanResult, filesContext string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	var (
		planJSON, _ = json.Marshal(plan)
		content     = helper.ProcessSonnetCoderPrompt(clarified, string(planJSON), filesContext)
		messages    = buildMessagesWithHistory(chatHistory, content)
	)

	systemPrompt := helper.SystemPromptSonnetCoder
	if filesContext == "No existing files to modify." {
		systemPrompt = helper.SystemPromptAiChat
	}

	rawResp, err := helper.CallAnthropicAPI(m.baseConf,
		models.AnthropicRequest{
			Model:     m.baseConf.ClaudeModel,
			MaxTokens: m.baseConf.MaxTokens,
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

func (m *messagingStc) getFileGraphJSON() (string, error) {
	project, err := m.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		context.Background(),
		&pbo.McpProjectId{
			ResourceEnvId: m.resourceEnvId,
			Id:            m.mcpProjectId,
		},
	)
	if err != nil {
		return "{}", fmt.Errorf("error getting project files: %w", err)
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

func (m *messagingStc) getFilesContext(paths []string) (string, error) {
	if len(paths) == 0 {
		return "No existing files to modify.", nil
	}

	project, err := m.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(
		context.Background(),
		&pbo.McpProjectId{
			ResourceEnvId: m.resourceEnvId,
			Id:            m.mcpProjectId,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error getting project files: %w", err)
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

func (m *messagingStc) saveProject(req *models.ParsedClaudeResponse) (*pbo.McpProject, error) {
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
		context.Background(),
		&pbo.McpProject{
			Id:            m.mcpProjectId,
			ResourceEnvId: m.resourceEnvId,
			Title:         req.Project.ProjectName,
			Description:   "Generated with claude",
			ProjectFiles:  projectFiles,
			ProjectEnv:    projectEnv,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error in saving project: %v", err)
	}

	return createdProject, nil
}

func buildMessagesWithHistory(history []models.ChatMessage, currentContent string) []models.ChatMessage {
	var messages = make([]models.ChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, models.ChatMessage{
		Role:    "user",
		Content: []models.ContentBlock{{Type: "text", Text: currentContent}},
	})
	return messages
}
