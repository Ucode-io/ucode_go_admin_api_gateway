package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

type visualEditOutput struct {
	Files         []models.ProjectFile `json:"files"`
	ChangeSummary string               `json:"change_summary"`
}

// recordTokenUsage ships token counts to the billing service asynchronously.
func (p *ChatProcessor) recordTokenUsage(usage models.ClaudeUsage, model, description string) {
	if usage.InputTokens == 0 && usage.OutputTokens == 0 {
		return
	}
	projectId := p.ucodeProjectId
	if p.mcpUcodeProjectId != "" {
		projectId = p.mcpUcodeProjectId
	}
	go func() {
		_, recErr := p.service.CompanyService().Billing().RecordAiTokenUsage(
			context.Background(),
			&pb.RecordAiTokenUsageRequest{
				ProjectId:    projectId,
				InputTokens:  int32(usage.InputTokens),
				OutputTokens: int32(usage.OutputTokens),
				Model:        model,
				Description:  description,
			},
		)
		if recErr != nil {
			log.Printf("[TOKEN RECORD] error recording usage for %s: %v", description, recErr)
		}
	}()
}

func callWithTool[T any](p *ChatProcessor, ctx context.Context, req models.AnthropicToolRequest, timeout time.Duration, description string) (*T, error) {
	log.Printf("[AI] Calling Anthropic (tool use): %s", description)

	result, usage, stopReason, err := helper.CallAnthropicWithTool[T](p.baseConf, req, timeout)

	// Record token usage regardless of error — partial usage still counts.
	p.recordTokenUsage(usage, req.Model, description)

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

// callArchitect asks the architect agent to plan the project structure and design system.
// existingSchemaCtx (optional): JSON list of existing tables — pass when adding a new microfrontend
// to an existing project so the architect knows which APIs are already available.
func (p *ChatProcessor) callArchitect(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, existingSchemaCtx string) (*models.ArchitectPlan, error) {
	userMsg := clarified
	if existingSchemaCtx != "" {
		userMsg += "\n\n====================================\nEXISTING PROJECT TABLES (already provisioned — use these slugs for API calls, do NOT recreate them)\n====================================\n" + existingSchemaCtx
	}

	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(userMsg, imageURLs))

	plan, err := callWithTool[models.ArchitectPlan](
		p, ctx,
		models.AnthropicToolRequest{
			Model:     p.baseConf.ArchitectModel,
			MaxTokens: p.baseConf.PlannerMaxTokens,
			System:    helper.PromptArchitect,
			Messages:  messages,
			Tools:     []models.ClaudeFunctionTool{helper.ToolArchitectPlan},
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

// generateCode is the unified code generation agent. It receives the architect's plan
// (including design tokens) and produces all frontend files.
// For admin_panel projects it injects template context files, runs two-phase generation
// to avoid max_tokens errors on large projects, and silently merges scaffold files.
func (p *ChatProcessor) generateCode(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	prompt := clarified + "\n\n" + buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)

	// For admin panels: inject importable template context files.
	// Scaffold files (package.json, vite.config.ts, etc.) are merged silently after generation.
	var scaffoldFiles []models.ProjectFile
	if plan.ProjectType == "admin_panel" {
		contextFiles := GetTemplateContext("admin_panel")
		scaffoldFiles = GetTemplateScaffold("admin_panel")

		if len(contextFiles) > 0 {
			var templateCtx strings.Builder
			templateCtx.WriteString("\n====================================\n")
			templateCtx.WriteString("PRE-BUILT UTILITIES — MANDATORY USAGE\n")
			templateCtx.WriteString("====================================\n")
			templateCtx.WriteString("The following files ALREADY EXIST in the project. You MUST import from them.\n")
			templateCtx.WriteString("NEVER re-implement these utilities. NEVER output these files in your response.\n\n")
			templateCtx.WriteString("REQUIRED IMPORTS (use exactly these paths):\n")
			templateCtx.WriteString("  import { useApiQuery, useApiMutation } from '@/hooks/useApi'\n")
			templateCtx.WriteString("  import { extractList, extractSingle, extractCount } from '@/lib/apiUtils'\n")
			templateCtx.WriteString("  import { cn, formatDate, formatCurrency, getInitials } from '@/lib/utils'\n")
			templateCtx.WriteString("  import { AppProviders } from '@/components/shared/AppProviders' (wrap root in App.tsx)\n\n")
			templateCtx.WriteString("FILE CONTENTS FOR REFERENCE:\n")
			for _, f := range contextFiles {
				fmt.Fprintf(&templateCtx, "\n### %s\n```typescript\n%s\n```\n", f.Path, f.Content)
			}
			prompt += templateCtx.String()
		}

		// Two-phase generation for admin_panel prevents max_tokens errors on large projects.
		return p.generateCodeTwoPhase(ctx, prompt, imageURLs, chatHistory, plan, scaffoldFiles)
	}

	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(prompt, imageURLs))

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptAdminPanelGenerator,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating project code",
	)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	if len(project.Files) == 0 {
		return nil, fmt.Errorf("generate code: claude returned empty project")
	}

	log.Printf("[generate] done: %d files (type=%s)", len(project.Files), plan.ProjectType)
	return &models.ParsedClaudeResponse{Project: project}, nil
}

// generateCodeTwoPhase splits admin_panel generation into two sequential API calls:
//   - Phase 1: core structural files (App.tsx with all routes, layout, types)
//   - Phase 2: all remaining feature files using Phase 1 as context
//
// This prevents max_tokens failures on large admin panels (25-35 files).
func (p *ChatProcessor) generateCodeTwoPhase(ctx context.Context, basePrompt string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, scaffoldFiles []models.ProjectFile) (*models.ParsedClaudeResponse, error) {
	const phase1Instruction = `
====================================
PHASE 1 — CORE FILES ONLY
====================================
In this phase, generate ONLY these 6 files (no more, no less):
  1. src/App.tsx           — declare ALL routes for every page you plan to build, including pages from Phase 2
  2. src/layouts/MainLayout.tsx
  3. src/components/layout/Sidebar.tsx — full navigation menu listing all planned pages
  4. src/components/layout/Header.tsx
  5. src/pages/Dashboard.tsx           — complete implementation
  6. src/types/index.ts                — ALL TypeScript interfaces for the entire project

DO NOT generate any other files. Phase 2 will generate all remaining pages, components, forms, and tables.
`

	phase1Messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(basePrompt+phase1Instruction, imageURLs))

	phase1Project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptAdminPanelGenerator,
			Messages:   phase1Messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating project code (phase 1 — core files)",
	)
	if err != nil {
		return nil, fmt.Errorf("generate code phase 1: %w", err)
	}
	if len(phase1Project.Files) == 0 {
		return nil, fmt.Errorf("generate code phase 1: claude returned empty project")
	}
	log.Printf("[generate] phase 1 done: %d files", len(phase1Project.Files))

	// Phase 2: generate all remaining feature files with Phase 1 as context.
	phase1FilePaths := make(map[string]struct{}, len(phase1Project.Files))
	for _, f := range phase1Project.Files {
		phase1FilePaths[f.Path] = struct{}{}
	}

	phase2Instruction := buildPhase1Context(phase1Project.Files) + `
====================================
PHASE 2 — FEATURE FILES ONLY
====================================
Generate ALL remaining pages, components, forms, and tables for this project.
RULES:
  - DO NOT include any Phase 1 file paths listed above in your output
  - Use EXACT import paths from Phase 1 files (App.tsx routes, types/index.ts interfaces, layout components)
  - Every page must match a route already declared in the Phase 1 src/App.tsx
  - Complete every feature with full CRUD, data tables, forms, and API integration
`

	phase2Messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(basePrompt+phase2Instruction, imageURLs))

	phase2Project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptAdminPanelGenerator,
			Messages:   phase2Messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating project code (phase 2 — feature files)",
	)
	if err != nil {
		return nil, fmt.Errorf("generate code phase 2: %w", err)
	}
	log.Printf("[generate] phase 2 done: %d files", len(phase2Project.Files))

	// Merge Phase 1 + Phase 2; Phase 1 wins on path conflicts.
	allFiles := make([]models.ProjectFile, 0, len(phase1Project.Files)+len(phase2Project.Files))
	allFiles = append(allFiles, phase1Project.Files...)
	for _, f := range phase2Project.Files {
		if _, exists := phase1FilePaths[f.Path]; !exists {
			allFiles = append(allFiles, f)
		}
	}

	// Silently merge scaffold files that AI must not re-emit (prevents template drift).
	if len(scaffoldFiles) > 0 {
		generatedPaths := make(map[string]struct{}, len(allFiles))
		for _, f := range allFiles {
			generatedPaths[f.Path] = struct{}{}
		}
		for _, sf := range scaffoldFiles {
			if _, exists := generatedPaths[sf.Path]; !exists {
				allFiles = append(allFiles, sf)
			}
		}
	}

	projectName := phase1Project.ProjectName
	if projectName == "" {
		projectName = phase2Project.ProjectName
	}
	merged := &models.GeneratedProject{
		ProjectName: projectName,
		Files:       allFiles,
		FileGraph:   phase1Project.FileGraph,
		Env:         phase1Project.Env,
	}

	log.Printf("[generate] two-phase done: %d files total (type=%s)", len(allFiles), plan.ProjectType)
	return &models.ParsedClaudeResponse{Project: merged}, nil
}

// buildPhase1Context formats Phase 1 generated files as a prompt section for Phase 2.
func buildPhase1Context(files []models.ProjectFile) string {
	var sb strings.Builder
	sb.WriteString("\n====================================\n")
	sb.WriteString("PHASE 1 FILES — ALREADY GENERATED (DO NOT REGENERATE)\n")
	sb.WriteString("====================================\n")
	sb.WriteString("The following files were generated in Phase 1. You MUST:\n")
	sb.WriteString("  1. Use EXACT import paths from these files\n")
	sb.WriteString("  2. NOT include any of these file paths in your output\n")
	sb.WriteString("  3. Ensure your new files import from these paths correctly\n\n")
	for _, f := range files {
		fmt.Fprintf(&sb, "### %s\n```typescript\n%s\n```\n\n", f.Path, f.Content)
	}
	return sb.String()
}

func (p *ChatProcessor) inspectCode(ctx context.Context, userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	content := helper.BuildInspectorMessage(userQuestion, filesContext)
	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(content, imageURLs))

	response, err := p.callAnthropicWithTracking(
		ctx,
		models.AnthropicRequest{
			Model:     p.baseConf.InspectorModel,
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

func (p *ChatProcessor) planChanges(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, hasImages bool) (*models.SonnetPlanResult, error) {
	content := helper.BuildPlannerMessage(clarified, fileGraphJSON, hasImages)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	result, err := callWithTool[models.SonnetPlanResult](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.PlannerModel,
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

func (p *ChatProcessor) editCode(ctx context.Context, clarified string, plan *models.SonnetPlanResult, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (*models.ParsedClaudeResponse, error) {
	hasMatchingFiles := filesContext != "No existing files to modify." && filesContext != "No matching files found."

	var (
		systemPrompt  string
		contentBlocks []models.ContentBlock
	)

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
			Model:      p.baseConf.CoderModel,
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

func (p *ChatProcessor) callAnthropicWithTracking(ctx context.Context, req models.AnthropicRequest, timeout time.Duration, description string) (string, error) {
	log.Printf("[AI] Calling Anthropic: %s", description)
	response, err := helper.CallAnthropicAPI(p.baseConf, req, timeout)
	if err != nil {
		log.Printf("[AI] Anthropic error for %s: %v", description, err)
		return "", err
	}

	var parsed struct {
		Usage models.ClaudeUsage `json:"usage"`
	}
	if jsonErr := json.Unmarshal([]byte(response), &parsed); jsonErr == nil {
		p.recordTokenUsage(parsed.Usage, req.Model, description)
	}

	return response, nil
}
