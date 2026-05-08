package v1

import (
	"context"
	"fmt"
	"log"
	"strings"
	"ucode/ucode_go_api_gateway/api/handlers/helper/chat_prompts"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
)

func (p *ChatProcessor) routeAndProcess(ctx context.Context, req models.NewMessageReq, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	p.initTokenBudget(ctx)

	if len(req.Context) > 0 {
		if p.mcpProjectId == "" && p.microFrontendId == "" {
			return &models.ParsedClaudeResponse{
				Description: "No project found. Please create a project first before using visual editing.",
			}, nil
		}
		return p.runVisualEdit(ctx, req.Content, req.Context, chatHistory, req.Images)
	}

	var (
		hasImages = len(req.Images) > 0

		fileGraphJSON   = "{}"
		microFrontFiles []models.GitlabFileChange
		err             error
	)

	if p.microFrontendId != "" && p.microFrontendRepoId != "" {
		log.Printf("[ROUTER] microFrontend edit mode — fetching files for repo_id=%s", p.microFrontendRepoId)
		microFrontFiles, err = p.fetchMicrofrontendFiles(ctx)
		if err != nil {
			log.Printf("[ROUTER] failed to fetch microFrontend files: %v", err)
		} else {
			fileGraphJSON = p.buildMicrofrontendFileGraphJSON(microFrontFiles)
		}
	}

	routeResult, err := p.routeRequest(req.Content, fileGraphJSON, hasImages, chatHistory)
	if err != nil {
		return nil, err
	}

	log.Printf("[ROUTER] intent=%s next_step=%v files_needed=%d", routeResult.Intent, routeResult.NextStep, len(routeResult.FilesNeeded))

	if routeResult.Intent == "ask_question" {
		return &models.ParsedClaudeResponse{
			Description: routeResult.Reply,
			Questions:   routeResult.Questions,
		}, nil
	}

	if routeResult.Intent == "plan_request" {
		clarified := routeResult.Clarified
		if clarified == "" {
			clarified = req.Content
		}
		return p.runCodeChange(ctx, clarified, fileGraphJSON, chatHistory, req.Images, routeResult.ProjectName, microFrontFiles)
	}

	if !routeResult.NextStep {
		return &models.ParsedClaudeResponse{Description: routeResult.Reply}, nil
	}

	switch routeResult.Intent {

	case "clarify", "project_question":
		return &models.ParsedClaudeResponse{Description: routeResult.Reply}, nil

	case "project_inspect":
		if p.microFrontendId != "" {
			return p.runMicrofrontendInspect(ctx, req.Content, routeResult.FilesNeeded, chatHistory, req.Images, microFrontFiles)
		}
		return &models.ParsedClaudeResponse{
			Description: "No project exists yet. Please create a project first by describing what you want to build.",
		}, nil

	case "code_change":
		return p.runCodeChange(ctx, routeResult.Clarified, fileGraphJSON, chatHistory, req.Images, routeResult.ProjectName, microFrontFiles)

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

func (p *ChatProcessor) runCodeChange(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string, projectName string, microFrontFiles []models.GitlabFileChange) (*models.ParsedClaudeResponse, error) {
	if p.microFrontendId != "" {
		return p.runMicrofrontendEdit(ctx, clarified, fileGraphJSON, chatHistory, imageURLs, microFrontFiles)
	}

	if p.newProject {
		log.Printf("[CODE] new_project=true — provisioning new ucode project")
		return p.buildNewProject(ctx, clarified, chatHistory, imageURLs, projectName)
	}

	log.Printf("[CODE] new_project=false — creating microFrontend in current project")
	return p.buildMicrofrontendForCurrentProject(ctx, clarified, chatHistory, imageURLs, projectName)
}

func (p *ChatProcessor) routeRequest(userPrompt, fileGraphJSON string, hasImages bool, chatHistory []models.ChatMessage) (*models.HaikuRoutingResult, error) {
	historyText := buildHistoryText(chatHistory)
	content := chat_prompts.BuildRouterMessage(userPrompt, fileGraphJSON, hasImages, historyText)

	response, err := p.callAnthropicWithTracking(
		context.Background(),
		models.AnthropicRequest{
			Model:     p.baseConf.ClaudeHaikuModel,
			MaxTokens: p.baseConf.RouterMaxTokens,
			System:    chat_prompts.PromptRouter,
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
