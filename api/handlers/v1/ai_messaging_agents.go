package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

// chunkResult carries the result of one parallel feature chunk generation.
type chunkResult struct {
	group   models.ManifestGroup
	project *models.GeneratedProject
	err     error
}

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

func callWithTool[T any](p *ChatProcessor, _ context.Context, req models.AnthropicToolRequest, timeout time.Duration, description string) (*T, error) {
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
			Model:      p.baseConf.ArchitectModel,
			MaxTokens:  p.baseConf.PlannerMaxTokens,
			System:     helper.PromptArchitect,
			Messages:   messages,
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

// generateCode routes to chunked or single-call generation based on project type.
// Admin panels always use chunked generation (250K+ output exceeds single-call 64K limit).
func (p *ChatProcessor) generateCode(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	if plan.ProjectType == "admin_panel" {
		log.Println("GENERATION CODE: generateCodeChunked ")
		return p.generateCodeChunked(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}
	log.Println("GENERATION CODE: generateCodeSingle ")
	return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
}

// generateCodeSingle is the original single-call path for landing pages and websites.
func (p *ChatProcessor) generateCodeSingle(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	prompt := clarified + "\n\n" + buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)

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

	if len(scaffoldFiles) > 0 {
		generatedPaths := make(map[string]struct{}, len(project.Files))
		for _, f := range project.Files {
			generatedPaths[f.Path] = struct{}{}
		}
		for _, sf := range scaffoldFiles {
			if _, exists := generatedPaths[sf.Path]; !exists {
				project.Files = append(project.Files, sf)
			}
		}
	}

	log.Printf("[generate] done: %d files (type=%s)", len(project.Files), plan.ProjectType)
	return &models.ParsedClaudeResponse{Project: project}, nil
}

// generateCodeChunked orchestrates the 3-phase chunked generation for admin panels:
// manifest → foundation (sequential) → feature groups (parallel).
func (p *ChatProcessor) generateCodeChunked(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	log.Printf("[chunked] starting chunked generation for admin_panel: %s", plan.ProjectName)

	// Phase 1: manifest — determine file structure and groups.
	manifest, err := p.generateManifest(ctx, plan, chatHistory)
	if err != nil || len(manifest.Groups) < 2 {
		log.Printf("[chunked] manifest failed or insufficient groups (%v) — falling back to single call", err)
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	// Split into foundation (group 0), UI kit (group 1), and feature groups (2..N).
	var foundationGroup models.ManifestGroup
	var uiKitGroup models.ManifestGroup
	var featureGroups []models.ManifestGroup

	for _, g := range manifest.Groups {
		if g.ID == 0 {
			foundationGroup = g
		} else if g.ID == 1 {
			uiKitGroup = g
		} else {
			featureGroups = append(featureGroups, g)
		}
	}
	if len(foundationGroup.Files) == 0 || len(featureGroups) == 0 {
		log.Printf("[chunked] manifest missing foundation or features — falling back to single call")
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	log.Printf("[chunked] manifest: foundation=%d files, uikit=%d files, feature_groups=%d", len(foundationGroup.Files), len(uiKitGroup.Files), len(featureGroups))

	// Phase 2: foundation — generate shared files first (sequential).
	apiConfig := buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)
	foundation, err := p.generateFoundation(ctx, clarified, imageURLs, chatHistory, apiConfig, foundationGroup, manifest)
	if err != nil {
		log.Printf("[chunked] foundation failed (%v) — falling back to single call", err)
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	// Filter foundation output to ONLY the files it was asked to generate.
	foundation.Files = filterToGroup(foundation.Files, foundationGroup)
	log.Printf("[chunked] foundation done: %d files (after filter)", len(foundation.Files))

	// Phase 2b: UI Kit — generate after foundation, before feature groups (sequential).
	// Feature groups import from ui/*, so UI Kit must be fully ready before they start.
	var uiKit *models.GeneratedProject

	if len(uiKitGroup.Files) > 0 {
		foundationCtxForUIKit := buildFoundationContext(foundation.Files)
		uiKit, err = p.generateUIKit(ctx, uiKitGroup, foundationCtxForUIKit, apiConfig)
		if err != nil {
			log.Printf("[chunked] UI kit failed (%v) — continuing without it", err)
		} else {
			uiKit.Files = filterToGroup(uiKit.Files, uiKitGroup)
			log.Printf("[chunked] UI kit done: %d files (after filter)", len(uiKit.Files))
		}
	}

	// Phase 3: feature groups — parallel goroutines, each gets foundation + UI kit context.
	// Build combined context: foundation files + ui kit files.
	allSharedFiles := make([]models.ProjectFile, 0, len(foundation.Files)+len(uiKitGroup.Files))
	allSharedFiles = append(allSharedFiles, foundation.Files...)
	if uiKit != nil {
		allSharedFiles = append(allSharedFiles, uiKit.Files...)
	}
	foundationCtx := buildFoundationContext(allSharedFiles)
	manifestSummary := buildManifestSummary(manifest)

	results := make(chan chunkResult, len(featureGroups))
	for _, group := range featureGroups {
		g := group
		go func() {
			proj, chunkErr := p.generateChunk(ctx, g, foundationCtx, manifestSummary, apiConfig)
			results <- chunkResult{group: g, project: proj, err: chunkErr}
		}()
	}

	var successChunks []*models.GeneratedProject
	var failedCount int
	for range featureGroups {
		res := <-results
		if res.err != nil {
			log.Printf("[chunked] feature chunk %q error: %v", res.group.Name, res.err)
			failedCount++
		} else {
			// Filter each chunk to ONLY its assigned files — prevents cross-group pollution.
			res.project.Files = filterToGroup(res.project.Files, res.group)
			log.Printf("[chunked] ✅ chunk %q: %d files", res.group.Name, len(res.project.Files))
			successChunks = append(successChunks, res.project)
		}
	}

	if failedCount == len(featureGroups) {
		return nil, fmt.Errorf("chunked: all %d feature chunks failed", len(featureGroups))
	}
	if failedCount > 0 {
		log.Printf("[chunked] WARNING: %d/%d feature chunks failed — deploying partial project", failedCount, len(featureGroups))
	}

	// Merge foundation + UI kit + successful feature chunks.
	var allChunks []*models.GeneratedProject
	if uiKit != nil {
		allChunks = append(allChunks, uiKit)
	}
	allChunks = append(allChunks, successChunks...)
	merged := mergeChunks(foundation, allChunks)

	// Add scaffold files (package.json, vite.config.ts, etc.).
	scaffoldFiles := GetTemplateScaffold("admin_panel")
	if len(scaffoldFiles) > 0 {
		generatedPaths := make(map[string]struct{}, len(merged.Files))
		for _, f := range merged.Files {
			generatedPaths[f.Path] = struct{}{}
		}
		for _, sf := range scaffoldFiles {
			if _, exists := generatedPaths[sf.Path]; !exists {
				merged.Files = append(merged.Files, sf)
			}
		}
	}

	log.Printf("[chunked] done: %d total files (%d feature groups, %d failed)", len(merged.Files), len(featureGroups), failedCount)
	return &models.ParsedClaudeResponse{Project: merged}, nil
}

// generateManifest asks Claude to plan the file structure and group files by dependency level.
func (p *ChatProcessor) generateManifest(ctx context.Context, plan *models.ArchitectPlan, chatHistory []models.ChatMessage) (*models.ProjectManifest, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Project: %s (type: %s)\n\nTables:\n", plan.ProjectName, plan.ProjectType)
	for _, t := range plan.Tables {
		fmt.Fprintf(&sb, "- %s (slug: %s): ", t.Label, t.Slug)
		for i, f := range t.Fields {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(f.Slug)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nUI Structure:\n" + plan.UIStructure)

	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{
		{Type: "text", Text: sb.String()},
	})

	manifest, err := callWithTool[models.ProjectManifest](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.PlannerModel,
			MaxTokens:  8000, // manifest is a compact JSON structure, 32K is wasteful
			System:     helper.PromptManifestGenerator,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitManifest},
			ToolChoice: helper.ForcedTool(helper.ToolEmitManifest.Name),
		},
		timeoutPlanner,
		"Generating file manifest",
	)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	return manifest, nil
}

// generateFoundation generates Group 0 (shared) files using the full PromptAdminPanelGenerator.
// It receives the full manifest so App.tsx can include routes to all feature pages.
func (p *ChatProcessor) generateFoundation(
	ctx context.Context,
	clarified string,
	imageURLs []string,
	chatHistory []models.ChatMessage,
	apiConfig string,
	foundationGroup models.ManifestGroup,
	manifest *models.ProjectManifest,
) (*models.GeneratedProject, error) {
	var sb strings.Builder

	// Foundation instruction FIRST — before anything else so Claude reads it before processing context.
	sb.WriteString("====================================\n")
	sb.WriteString("⚠ CHUNKED GENERATION — FOUNDATION PHASE ONLY ⚠\n")
	sb.WriteString("====================================\n")
	sb.WriteString("You are generating ONLY the foundation (Group 0) files listed below.\n")
	sb.WriteString("Feature page implementations will be generated separately in parallel agents.\n")
	sb.WriteString("EMIT ONLY the files in 'YOUR FILES TO GENERATE'. No exceptions.\n\n")

	sb.WriteString("YOUR FILES TO GENERATE (Group 0 — Foundation):\n")
	for _, f := range foundationGroup.Files {
		fmt.Fprintf(&sb, "  %s  (exports: [%s])\n", f.Path, strings.Join(f.Exports, ", "))
	}

	// Collect page paths from feature groups so App.tsx has complete routing.
	sb.WriteString("\nFEATURE PAGES — add routes to App.tsx but DO NOT implement them:\n")
	for _, g := range manifest.Groups {
		if g.ID == 0 {
			continue
		}
		for _, f := range g.Files {
			if strings.Contains(f.Path, "/pages/") || strings.HasSuffix(f.Path, "Page.tsx") {
				fmt.Fprintf(&sb, "  %s  → component: [%s]\n", f.Path, strings.Join(f.Exports, ", "))
			}
		}
	}

	sb.WriteString("\nFULL PROJECT MANIFEST (for routing and types reference):\n")
	sb.WriteString(buildManifestSummary(manifest))

	sb.WriteString("\n====================================\n")
	sb.WriteString("PROJECT REQUEST\n")
	sb.WriteString("====================================\n")
	sb.WriteString(clarified)
	sb.WriteString("\n\n")
	sb.WriteString(apiConfig)

	// Inject template context (same as single-call path).
	contextFiles := GetTemplateContext("admin_panel")
	if len(contextFiles) > 0 {
		sb.WriteString("\n====================================\n")
		sb.WriteString("PRE-BUILT UTILITIES — MANDATORY USAGE\n")
		sb.WriteString("====================================\n")
		sb.WriteString("The following files ALREADY EXIST in the project. You MUST import from them.\n")
		sb.WriteString("NEVER re-implement these utilities. NEVER output these files in your response.\n\n")
		sb.WriteString("REQUIRED IMPORTS (use exactly these paths):\n")
		sb.WriteString("  import { useApiQuery, useApiMutation } from '@/hooks/useApi'\n")
		sb.WriteString("  import { extractList, extractSingle, extractCount } from '@/lib/apiUtils'\n")
		sb.WriteString("  import { cn, formatDate, formatCurrency, getInitials } from '@/lib/utils'\n")
		sb.WriteString("  import { AppProviders } from '@/components/shared/AppProviders' (wrap root in App.tsx)\n\n")
		sb.WriteString("FILE CONTENTS FOR REFERENCE:\n")
		for _, f := range contextFiles {
			fmt.Fprintf(&sb, "\n### %s\n```typescript\n%s\n```\n", f.Path, f.Content)
		}
	}

	sb.WriteString("\n====================================\n")
	sb.WriteString("REMINDER: Emit ONLY the Group 0 files listed at the top. DO NOT generate feature pages.\n")
	sb.WriteString("====================================\n")

	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(sb.String(), imageURLs))

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptChunkedCoder,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating foundation (Group 0)",
	)
	if err != nil {
		return nil, fmt.Errorf("foundation: %w", err)
	}
	if len(project.Files) == 0 {
		return nil, fmt.Errorf("foundation: empty response")
	}
	return project, nil
}

// generateUIKit generates Group 1 (UI Kit) sequentially after foundation.
// Feature groups depend on ui/* components, so this must complete before parallel feature generation.
func (p *ChatProcessor) generateUIKit(
	ctx context.Context,
	uiKitGroup models.ManifestGroup,
	foundationCtx string,
	apiConfig string,
) (*models.GeneratedProject, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "CHUNKED GENERATION — UI Kit (Group 1)\n\n")

	sb.WriteString("YOUR FILES TO IMPLEMENT (emit ONLY these):\n")
	for _, f := range uiKitGroup.Files {
		fmt.Fprintf(&sb, "  %s  (exports: [%s])\n", f.Path, strings.Join(f.Exports, ", "))
	}

	sb.WriteString("\n")
	sb.WriteString(foundationCtx)
	sb.WriteString("\n====================================\n")
	sb.WriteString("REMINDER: Emit ONLY the ui/* files listed above. No page logic, no API calls.\n")
	sb.WriteString("====================================\n")

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptUIKitCoder,
			Messages:   []models.ChatMessage{{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: sb.String()}}}},
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating UI Kit (Group 1)",
	)
	if err != nil {
		return nil, fmt.Errorf("ui kit: %w", err)
	}
	if len(project.Files) == 0 {
		return nil, fmt.Errorf("ui kit: empty response")
	}
	return project, nil
}

// generateChunk generates one feature group in isolation.
// It receives full foundation file contents so all imports are satisfied.
func (p *ChatProcessor) generateChunk(
	ctx context.Context,
	group models.ManifestGroup,
	foundationCtx string,
	manifestSummary string,
	apiConfig string,
) (*models.GeneratedProject, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "CHUNKED GENERATION — Feature Group %d: %s\n\n", group.ID, group.Name)

	sb.WriteString("YOUR FILES TO IMPLEMENT (emit ONLY these):\n")
	for _, f := range group.Files {
		fmt.Fprintf(&sb, "  %s  (exports: [%s])\n", f.Path, strings.Join(f.Exports, ", "))
	}

	sb.WriteString("\n")
	sb.WriteString(apiConfig)
	sb.WriteString("\n")
	sb.WriteString(foundationCtx)
	sb.WriteString("\n====================================\n")
	sb.WriteString("FULL PROJECT MANIFEST (import reference)\n")
	sb.WriteString("====================================\n")
	sb.WriteString(manifestSummary)

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     helper.PromptChunkedCoder,
			Messages:   []models.ChatMessage{{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: sb.String()}}}},
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		fmt.Sprintf("Generating feature chunk: %s", group.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("chunk %s: %w", group.Name, err)
	}
	return project, nil
}

// buildFoundationContext formats generated foundation files for injection into feature chunk prompts.
func buildFoundationContext(files []models.ProjectFile) string {
	var sb strings.Builder
	sb.WriteString("====================================\n")
	sb.WriteString("FOUNDATION FILES (already generated — import freely, NEVER re-emit)\n")
	sb.WriteString("====================================\n")
	for _, f := range files {
		lang := "typescript"
		if strings.HasSuffix(f.Path, ".css") {
			lang = "css"
		} else if strings.HasSuffix(f.Path, ".json") {
			lang = "json"
		}
		fmt.Fprintf(&sb, "\n### %s\n```%s\n%s\n```\n", f.Path, lang, f.Content)
	}
	return sb.String()
}

// buildManifestSummary formats the manifest as readable text for context injection.
func buildManifestSummary(manifest *models.ProjectManifest) string {
	var sb strings.Builder
	for _, g := range manifest.Groups {
		fmt.Fprintf(&sb, "Group %d (%s):\n", g.ID, g.Name)
		for _, f := range g.Files {
			fmt.Fprintf(&sb, "  %s → [%s]\n", f.Path, strings.Join(f.Exports, ", "))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// filterToGroup keeps only files whose paths are explicitly listed in the manifest group.
// This prevents Claude from emitting files outside its assigned scope — leaked files
// (e.g. foundation generating feature pages) would otherwise overwrite correct implementations.
func filterToGroup(files []models.ProjectFile, group models.ManifestGroup) []models.ProjectFile {
	allowed := make(map[string]struct{}, len(group.Files))
	for _, f := range group.Files {
		allowed[f.Path] = struct{}{}
	}
	out := files[:0]
	for _, f := range files {
		if _, ok := allowed[f.Path]; ok {
			out = append(out, f)
		}
	}
	return out
}

// mergeChunks combines foundation + feature chunks. Foundation files always win on dedup.
func mergeChunks(foundation *models.GeneratedProject, chunks []*models.GeneratedProject) *models.GeneratedProject {
	seen := make(map[string]struct{}, len(foundation.Files)*3)
	allFiles := make([]models.ProjectFile, 0, len(foundation.Files)*3)
	env := make(map[string]any, len(foundation.Env))

	for _, f := range foundation.Files {
		seen[f.Path] = struct{}{}
		allFiles = append(allFiles, f)
	}
	maps.Copy(env, foundation.Env)

	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		for k, v := range chunk.Env {
			if _, exists := env[k]; !exists {
				env[k] = v
			}
		}
		for _, f := range chunk.Files {
			if _, exists := seen[f.Path]; !exists {
				seen[f.Path] = struct{}{}
				allFiles = append(allFiles, f)
			}
		}
	}

	return &models.GeneratedProject{
		ProjectName: foundation.ProjectName,
		Files:       allFiles,
		FileGraph:   foundation.FileGraph,
		Env:         env,
	}
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

func (p *ChatProcessor) callAnthropicWithTracking(_ context.Context, req models.AnthropicRequest, timeout time.Duration, description string) (string, error) {
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
