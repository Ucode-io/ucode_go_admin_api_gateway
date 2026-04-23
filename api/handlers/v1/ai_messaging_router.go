package v1

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
)

func (p *ChatProcessor) routeAndProcess(ctx context.Context, req models.NewMessageReq, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
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

func (p *ChatProcessor) runGeneratePlan(ctx context.Context, userRequest string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {
	content := helper.BuildPlanGeneratorMessage(userRequest)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	plan, err := callWithTool[models.HaikuPlan](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.ClaudeModel,
			MaxTokens:  p.baseConf.PlannerMaxTokens,
			System:     helper.PromptPlanGenerator,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitDiagrams},
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

func (p *ChatProcessor) runVisualEdit(ctx context.Context, instruction string, contexts []models.VisualContext, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	log.Printf("[VISUAL EDIT] starting: count=%d", len(contexts))

	existingFiles, err := p.fetchMicrofrontendFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("visual edit: failed to fetch microFrontend files: %w", err)
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
		log.Printf("[VISUAL EDIT] no specific files found for contexts, falling back to microFrontend edit flow")
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
