package v1

import (
	"context"
	"log"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
)

const (
	intentAskQuestion     = "ask_question"
	intentPlanRequest     = "plan_request"
	intentClarify         = "clarify"
	intentProjectQuestion = "project_question"
	intentProjectInspect  = "project_inspect"
	intentCodeChange      = "code_change"
	intentDatabaseQuery   = "database_query"
)

func (p *ChatProcessor) routeAndProcess(ctx context.Context, req models.NewMessageReq, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	p.initTokenBudget(ctx)
	if err := p.Check(); err != nil {
		return nil, err
	}

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

	routeResult, err := p.agent.RouteRequest(ctx, models.RouterInput{
		UserMessage:   req.Content,
		FileGraphJSON: fileGraphJSON,
		HasImages:     hasImages,
		History:       chatHistory,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("[ROUTER] intent=%s next_step=%v files_needed=%d", routeResult.Intent, routeResult.NextStep, len(routeResult.FilesNeeded))

	if routeResult.Intent == intentAskQuestion {
		return &models.ParsedClaudeResponse{
			Description: routeResult.Reply,
			Questions:   routeResult.Questions,
		}, nil
	}

	if routeResult.Intent == intentPlanRequest {
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

	case intentClarify, intentProjectQuestion:
		return &models.ParsedClaudeResponse{Description: routeResult.Reply}, nil

	case intentProjectInspect:
		if p.microFrontendId != "" {
			return p.runMicrofrontendInspect(ctx, req.Content, routeResult.FilesNeeded, chatHistory, req.Images, microFrontFiles)
		}
		return &models.ParsedClaudeResponse{
			Description: "No project exists yet. Please create a project first by describing what you want to build.",
		}, nil

	case intentCodeChange:
		return p.runCodeChange(ctx, routeResult.Clarified, fileGraphJSON, chatHistory, req.Images, routeResult.ProjectName, microFrontFiles)

	case intentDatabaseQuery:
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
