package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"strings"
	"sync"
	"time"
	"ucode/ucode_go_api_gateway/api/handlers/helper/chat_prompts"

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
			System:     chat_prompts.PromptArchitect,
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

// generateCode routes to the correct generation strategy based on project type.
//
//	admin_panel → chunked (250K+ output, parallel CRUD features)
//	web         → chunked website (parallel pages, Sonnet model)
//	landing     → single call (Sonnet model, focused landing prompt)
func (p *ChatProcessor) generateCode(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	switch plan.ProjectType {
	case "admin_panel":
		log.Println("GENERATION CODE: generateCodeChunked")
		return p.generateCodeChunkedAdminPanel(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	case "web":
		log.Println("GENERATION CODE: generateCodeChunkedWebsite")
		return p.generateCodeChunkedWebsite(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	default:
		log.Println("GENERATION CODE: generateCodeSingle (landing)")
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}
}

// generateCodeSingle is the original single-call path for landing pages and websites.
func (p *ChatProcessor) generateCodeSingle(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	prompt := clarified + "\n\n" + buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)
	if p.baseConf.UnsplashAccessKey != "" {
		p.emitter().Emit(SSEEvent{Type: EvProgress, Icon: "image", Message: "Подбираю изображения для проекта...", Percent: 17})
		pool := helper.FetchImagePool(ctx, p.baseConf.UnsplashAccessKey, plan)
		if pool.Err != nil {
			p.emitter().Emit(SSEEvent{Type: EvProgress, Icon: "alert-triangle", Message: "Unsplash: не удалось подобрать фото", Value: pool.Err.Error()})
		} else {
			prompt += "\n\n" + pool.Block
			p.emitter().Emit(SSEEvent{Type: EvProgress, Icon: "image", Message: fmt.Sprintf("Подобрано %d фото по запросу: %s", pool.Count, strings.Join(pool.Keywords, ", ")), Percent: 18})
		}
	}

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

	p.emitter().Emit(SSEEvent{Type: EvProgress, Icon: "code-2", Message: "Генерирую исходный код проекта...", Percent: 18})

	var project *models.GeneratedProject
	if err := withHeartbeat(ctx, p.emitter(),
		[]string{
			"Генерирую React компоненты...",
			"Создаю страницы и формы...",
			"Пишу бизнес-логику...",
			"Настраиваю роутинг и навигацию...",
			"Подключаю API хуки к таблицам...",
			"Настраиваю CSS стили и темы...",
			"Создаю layout и sidebar...",
			"Генерирую CRUD операции...",
			"Подключаю валидацию форм...",
			"Финализирую код проекта...",
		},
		18, 82, 360*time.Second,
		func() error {
			var e error
			coderModel := p.baseConf.LandingCoderModel
			systemPrompt := chat_prompts.PromptLandingGenerator
			if plan.ProjectType == "web" {
				systemPrompt = chat_prompts.PromptWebsiteGenerator
			}
			project, e = callWithTool[models.GeneratedProject](
				p, ctx,
				models.AnthropicToolRequest{
					Model:      coderModel,
					MaxTokens:  p.baseConf.CoderMaxTokens,
					System:     systemPrompt,
					Messages:   messages,
					Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
					ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
				},
				timeoutCoder,
				"Generating project code",
			)
			return e
		},
	); err != nil {
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

	// Always force-inject .env with correct credentials — Claude may guess wrong values.
	project.Files = injectEnvFile(project.Files, p.baseConf.UcodeBaseUrl, apiKey)

	// ── POST-GENERATION VALIDATION + REPAIR ──
	p.emitter().Emit(SSEEvent{Type: EvProgress, Icon: "shield-check", Message: "Проверяю импорты и зависимости...", Percent: 83})
	validationErrors := validateGeneratedProject(project.Files, project.Env)
	errorCount, _ := logValidationResults(validationErrors)

	if errorCount > 0 {
		p.emitter().Emit(SSEEvent{Type: EvRepair, Icon: "wrench", Message: "Исправляю найденные проблемы", Value: fmt.Sprintf("%d ошибок", errorCount), Percent: 84})
		log.Printf("[generate] 🔧 attempting Haiku repair for %d broken files...", errorCount)
		repaired := p.repairBrokenFiles(ctx, project.Files, validationErrors)
		if len(repaired) > 0 {
			applyRepairs(project.Files, repaired)
			postErrors := validateGeneratedProject(project.Files, project.Env)
			postCount, _ := logValidationResults(postErrors)
			log.Printf("[generate] post-repair: %d errors remaining (was %d)", postCount, errorCount)
		}
	}

	log.Printf("[generate] done: %d files (type=%s, %d validation errors)", len(project.Files), plan.ProjectType, errorCount)
	return &models.ParsedClaudeResponse{Project: project}, nil
}

func (p *ChatProcessor) generateCodeChunkedAdminPanel(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	log.Printf("[chunked] starting chunked generation for admin_panel: %s", plan.ProjectName)

	emit := p.emitter()

	// Phase 1: manifest — reuse eager manifest from buildNewProject if available, otherwise generate.
	var manifest *models.ProjectManifest
	if p.prebuiltManifest != nil {
		manifest = p.prebuiltManifest
		p.prebuiltManifest = nil
		log.Printf("[chunked] using prebuilt manifest: %d groups", len(manifest.Groups))
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "list-tree", Message: "Структура файлов готова", Percent: 23})
	} else {
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "list-tree", Message: "Планирую структуру файлов и зависимости...", Percent: 16})
		if err := withHeartbeat(ctx, emit,
			[]string{
				"Планирую структуру файлов...",
				"Определяю зависимости между модулями...",
				"Разбиваю проект на фичи...",
				"Рассчитываю порядок генерации...",
				"Строю граф зависимостей...",
			},
			16, 23, 60*time.Second,
			func() error { var e error; manifest, e = p.generateManifest(ctx, plan, chatHistory); return e },
		); err != nil || len(manifest.Groups) < 2 {
			log.Printf("[chunked] manifest failed or insufficient groups (%v) — falling back to single call", err)
			return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
		}
	}

	// Split into foundation (group 0), UI kit (group 1), and feature groups (2..N).
	var foundationGroup models.ManifestGroup
	var uiKitGroup models.ManifestGroup
	var featureGroups []models.ManifestGroup

	for _, g := range manifest.Groups {
		switch g.ID {
		case 0:
			foundationGroup = g
		case 1:
			uiKitGroup = g
		default:
			featureGroups = append(featureGroups, g)
		}
	}

	if len(foundationGroup.Files) == 0 || len(featureGroups) == 0 {
		log.Printf("[chunked] manifest missing foundation or features — falling back to single call")
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	featureNames := make([]string, 0, len(featureGroups))
	for _, g := range featureGroups {
		featureNames = append(featureNames, g.Name)
	}
	totalFiles := 0
	for _, g := range manifest.Groups {
		totalFiles += len(g.Files)
	}
	emit.Emit(SSEEvent{
		Type:    EvManifest,
		Icon:    "git-branch",
		Percent: 23,
		Message: "Структура проекта спланирована",
		Value:   fmt.Sprintf("%d файлов · %d фич", totalFiles, len(featureGroups)),
		Data: ManifestEventData{
			TotalFiles:   totalFiles,
			GroupCount:   len(manifest.Groups),
			FeatureNames: featureNames,
		},
	})
	log.Printf("[chunked] manifest: foundation=%d files, uikit=%d files, feature_groups=%d", len(foundationGroup.Files), len(uiKitGroup.Files), len(featureGroups))

	time.Sleep(1000 * time.Millisecond)

	// Phase 2: Foundation + UI Kit in parallel.
	// ui/* components only import cn() (pre-built template) and CSS variables by name.
	// shared/* components import type names from @/types — available from manifest exports.
	// No actual Foundation file contents are needed by UIKit at generation time.

	apiConfig := buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)
	if p.baseConf.UnsplashAccessKey != "" {
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "image", Message: "Подбираю изображения для проекта...", Percent: 24})
		pool := helper.FetchImagePool(ctx, p.baseConf.UnsplashAccessKey, plan)
		if pool.Err != nil {
			emit.Emit(SSEEvent{Type: EvProgress, Icon: "alert-triangle", Message: "Unsplash: не удалось подобрать фото", Value: pool.Err.Error()})
		} else {
			apiConfig += "\n\n" + pool.Block
			emit.Emit(SSEEvent{Type: EvProgress, Icon: "image", Message: fmt.Sprintf("Подобрано %d фото по запросу: %s", pool.Count, strings.Join(pool.Keywords, ", ")), Percent: 25})
		}
	}

	emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    "layers",
		Message: "Генерирую фундамент и UI Kit параллельно",
		Value:   fmt.Sprintf("%d + %d файлов", len(foundationGroup.Files), len(uiKitGroup.Files)),
		Percent: 24,
	})

	var (
		foundation *models.GeneratedProject
		uiKit      *models.GeneratedProject
		foundErr   error
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		for attempt := 1; attempt <= 2; attempt++ {
			foundation, e = p.generateFoundation(ctx, clarified, imageURLs, chatHistory, apiConfig, foundationGroup, manifest)
			if e == nil {
				break
			}
			if attempt < 2 && !errors.Is(e, helper.ErrMaxTokens) {
				log.Printf("[chunked] foundation attempt %d failed (%v) — retrying", attempt, e)
			}
		}
		if e != nil {
			foundErr = e
			return
		}
		foundation.Files = filterToGroup(foundation.Files, foundationGroup)
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "check-circle", Message: "Foundation готов", Value: fmt.Sprintf("%d файлов", len(foundation.Files)), Percent: 38})
		log.Printf("[chunked] foundation done: %d files (after filter)", len(foundation.Files))
	}()

	if len(uiKitGroup.Files) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stub := buildFoundationStub(foundationGroup)
			var e error
			uiKit, e = p.generateUIKit(ctx, uiKitGroup, stub, apiConfig, p.baseConf.CoderModel)
			if e != nil {
				log.Printf("[chunked] UI kit parallel failed (%v) — continuing without it", e)
				return
			}
			uiKit.Files = filterToGroup(uiKit.Files, uiKitGroup)
			emit.Emit(SSEEvent{Type: EvProgress, Icon: "check-circle", Message: "UI Kit готов", Value: fmt.Sprintf("%d компонентов", len(uiKit.Files)), Percent: 50})
			log.Printf("[chunked] UI kit done: %d files (after filter)", len(uiKit.Files))
		}()
	}

	// Single heartbeat covers both parallel phases — messages interleave naturally.
	if err := withHeartbeat(ctx, emit,
		[]string{
			"Генерирую layout, sidebar и навигацию...",
			"Создаю UI компоненты и дизайн-систему...",
			"Генерирую TypeScript типы и интерфейсы...",
			"Создаю DataTable, FormModal и PageHeader...",
			"Пишу API хуки и конфигурацию axios...",
			"Создаю Button, Input, Card, Dialog...",
			"Формирую глобальные стили и CSS переменные...",
			"Настраиваю роутинг и App.tsx...",
		},
		24, 50, 180*time.Second,
		func() error {
			wg.Wait()
			return foundErr
		},
	); err != nil {
		log.Printf("[chunked] foundation failed (%v) — falling back to single call", err)
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	// Phase 3: feature groups — parallel goroutines, each gets foundation + UI kit context.
	allSharedFiles := make([]models.ProjectFile, 0, len(foundation.Files)+len(uiKitGroup.Files))
	allSharedFiles = append(allSharedFiles, foundation.Files...)
	if uiKit != nil {
		allSharedFiles = append(allSharedFiles, uiKit.Files...)
	}
	foundationCtx := buildFoundationContext(allSharedFiles)
	manifestSummary := buildManifestSummary(manifest)

	var uiKitAPISummary string
	if uiKit != nil {
		uiKitAPISummary = buildUIKitAPISummary(uiKit.Files)
	}

	totalChunks := len(featureGroups)
	time.Sleep(1000 * time.Millisecond)
	emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    "zap",
		Percent: 51,
		Message: "Запускаю параллельную генерацию фич",
		Value:   fmt.Sprintf("%d фич одновременно", totalChunks),
	})

	results := make(chan chunkResult, totalChunks)

	// Heartbeat goroutine for the parallel chunk phase — emits every 12 s so the
	// client sees activity even if no chunk completes for a while.
	stopChunkHB := make(chan struct{})
	go func() {
		ticker := time.NewTicker(12 * time.Second)
		defer ticker.Stop()
		tick := 0
		for {
			select {
			case <-ticker.C:
				tick++
				pct := 51 + tick*2
				if pct > 76 {
					pct = 76
				}
				emit.Emit(SSEEvent{Type: EvProgress, Icon: "cpu", Message: "Генерирую фичи параллельно...", Percent: pct})
			case <-stopChunkHB:
				return
			}
		}
	}()
	defer close(stopChunkHB) // guaranteed cleanup regardless of early-return path

	for i, group := range featureGroups {
		g := group
		startDelay := time.Duration(i) * 120 * time.Millisecond // stagger chunk starts so events don't burst
		go func() {
			if startDelay > 0 {
				time.Sleep(startDelay)
			}
			emit.Emit(SSEEvent{
				Type:    EvChunkStart,
				Icon:    "package",
				Message: "Генерирую фичу",
				Value:   g.Name,
				Data:    map[string]any{"feature": g.Name},
			})
			proj, chunkErr := p.generateChunkAdminPanel(ctx, g, foundationCtx, manifestSummary, apiConfig, uiKitAPISummary)
			results <- chunkResult{group: g, project: proj, err: chunkErr}
		}()
	}

	var successChunks []*models.GeneratedProject
	var failedCount int
	completedChunks := 0
	for range featureGroups {
		res := <-results
		completedChunks++
		chunkPct := 51 + completedChunks*25/totalChunks
		if res.err != nil {
			log.Printf("[chunked] feature chunk %q error: %v", res.group.Name, res.err)
			failedCount++
			if stub := buildStubChunk(res.group); len(stub.Files) > 0 {
				successChunks = append(successChunks, stub)
				log.Printf("[chunked] injected %d stub file(s) for failed chunk %q", len(stub.Files), res.group.Name)
			}
		} else {
			res.project.Files = filterToGroup(res.project.Files, res.group)
			log.Printf("[chunked] ✅ chunk %q: %d files", res.group.Name, len(res.project.Files))
			emit.Emit(SSEEvent{
				Type:    EvChunkDone,
				Icon:    "check-circle",
				Percent: chunkPct,
				Message: fmt.Sprintf("Фича готова (%d/%d)", completedChunks, totalChunks),
				Value:   res.group.Name,
				Data: ChunkDoneData{
					Feature: res.group.Name,
					Index:   completedChunks,
					Total:   totalChunks,
					Files:   res.project.Files,
				},
			})
			successChunks = append(successChunks, res.project)
		}
	}
	// stopChunkHB closed by deferred call above

	if failedCount == totalChunks {
		return nil, fmt.Errorf("chunked: all %d feature chunks failed", totalChunks)
	}
	if failedCount > 0 {
		log.Printf("[chunked] WARNING: %d/%d feature chunks failed — deploying partial project", failedCount, totalChunks)
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

	merged.Files = injectEnvFile(merged.Files, p.baseConf.UcodeBaseUrl, apiKey)

	// ── POST-GENERATION VALIDATION + REPAIR ──
	emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    "shield-check",
		Percent: 80,
		Message: "Проверяю качество кода",
		Value:   fmt.Sprintf("%d файлов", len(merged.Files)),
	})
	validationErrors := validateGeneratedProject(merged.Files, merged.Env)
	errorCount, _ := logValidationResults(validationErrors)

	if errorCount > 0 {
		emit.Emit(SSEEvent{
			Type:    EvRepair,
			Icon:    "wrench",
			Percent: 82,
			Message: "Автоматически исправляю проблемы",
			Value:   fmt.Sprintf("%d ошибок", errorCount),
		})
		log.Printf("[chunked] 🔧 attempting Haiku repair for %d broken files...", errorCount)
		repaired := p.repairBrokenFiles(ctx, merged.Files, validationErrors)
		if len(repaired) > 0 {
			applyRepairs(merged.Files, repaired)
			postErrors := validateGeneratedProject(merged.Files, merged.Env)
			postCount, _ := logValidationResults(postErrors)
			log.Printf("[chunked] post-repair: %d errors remaining (was %d)", postCount, errorCount)
		}
	}

	log.Printf("[chunked] done: %d total files (%d feature groups, %d failed, %d validation errors)", len(merged.Files), totalChunks, failedCount, errorCount)
	return &models.ParsedClaudeResponse{Project: merged}, nil
}

func (p *ChatProcessor) generateManifest(ctx context.Context, plan *models.ArchitectPlan, chatHistory []models.ChatMessage) (*models.ProjectManifest, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Project: %s (type: %s)\n\n", plan.ProjectName, plan.ProjectType)
	if len(plan.Tables) > 0 {
		sb.WriteString("Tables:\n")
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
		sb.WriteString("\n")
	}
	sb.WriteString("UI Structure:\n" + plan.UIStructure)

	systemPrompt := chat_prompts.PromptManifestGenerator
	if plan.ProjectType == "web" {
		systemPrompt = chat_prompts.PromptWebsiteManifestGenerator
	}

	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{
		{Type: "text", Text: sb.String()},
	})

	manifest, err := callWithTool[models.ProjectManifest](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.PlannerModel,
			MaxTokens:  8000,
			System:     systemPrompt,
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

	// Mandate comprehensive types — the single most impactful foundation instruction.
	// Feature chunks import ALL entity types from '@/types'. If types are missing or wrong here,
	// every feature chunk that uses them will produce TypeScript errors.
	sb.WriteString("\n====================================\n")
	sb.WriteString("CRITICAL — src/types.ts MUST contain ALL entity interfaces\n")
	sb.WriteString("====================================\n")
	sb.WriteString("src/types.ts is the SINGLE SOURCE OF TRUTH for all entity types.\n")
	sb.WriteString("Feature chunks import entity types ONLY from '@/types' — never from feature files.\n")
	sb.WriteString("For EVERY table in the project, generate a TypeScript interface with ALL fields.\n")
	sb.WriteString("Use exact field slugs from the table schema below as property names.\n")
	sb.WriteString("Every entity interface must include: guid (string), created_at? (string), and all domain fields.\n")
	sb.WriteString("Also include: PaginationParams, SelectOption<T>, FormState.\n")
	sb.WriteString("⚠ DO NOT include NavItem or TableColumn — they are PRE-BUILT in src/types/common.ts.\n")
	sb.WriteString("  Layout files that need NavItem MUST import from '@/types/common', never from '@/types'.\n\n")

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
			System:     chat_prompts.PromptAdminPanelGenerator,
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

func (p *ChatProcessor) generateUIKit(
	ctx context.Context,
	uiKitGroup models.ManifestGroup,
	foundationCtx string,
	apiConfig string,
	coderModel string,
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
			Model:      coderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     chat_prompts.PromptUIKitCoder,
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

func (p *ChatProcessor) generateChunkAdminPanel(
	ctx context.Context,
	group models.ManifestGroup,
	foundationCtx string,
	manifestSummary string,
	apiConfig string,
	uiKitAPISummary string,
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

	// Inject template hook files (useApi.ts, apiUtils.ts) so Claude sees the EXACT
	// TypeScript signatures — this prevents the callback-based hallucination pattern.
	sb.WriteString(buildTemplateHooksContext())

	sb.WriteString(foundationCtx)

	// Inject UI Kit API reference so the chunk uses exact component names and props.
	if uiKitAPISummary != "" {
		sb.WriteString("\n")
		sb.WriteString(uiKitAPISummary)
	}

	sb.WriteString("\n====================================\n")
	sb.WriteString("FULL PROJECT MANIFEST (import reference)\n")
	sb.WriteString("====================================\n")
	sb.WriteString(manifestSummary)

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.CoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     chat_prompts.PromptChunkedCoderAdminPanel,
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

// buildStubChunk creates minimal placeholder files for a failed chunk so that
// App.tsx route imports resolve and the virtual FS build doesn't crash.
// Only page files (src/pages/*.tsx) get stubs — hook/util files are optional imports
// and don't block the build if absent.
func buildStubChunk(group models.ManifestGroup) *models.GeneratedProject {
	files := make([]models.ProjectFile, 0, len(group.Files))
	for _, f := range group.Files {
		if strings.HasSuffix(f.Path, ".tsx") && strings.Contains(f.Path, "/pages/") {
			// Extract component name from path: src/pages/LeavesPage.tsx → LeavesPage
			base := f.Path[strings.LastIndex(f.Path, "/")+1:]
			compName := strings.TrimSuffix(base, ".tsx")
			files = append(files, models.ProjectFile{
				Path: f.Path,
				Content: fmt.Sprintf(`import React from 'react'

export default function %s() {
  return (
    <div className="flex items-center justify-center h-64">
      <p className="text-muted-foreground">This section is temporarily unavailable.</p>
    </div>
  )
}
`, compName),
			})
		}
	}
	return &models.GeneratedProject{Files: files}
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

// buildFoundationStub creates a compact context for the UI Kit phase describing what Foundation
// (Group 0) will produce. Used instead of actual Foundation file contents when Foundation and
// UI Kit are generated in parallel. Provides manifest exports + standard CSS variable names —
// everything UI Kit needs to avoid re-implementing Foundation files.
func buildFoundationStub(foundationGroup models.ManifestGroup) string {
	var sb strings.Builder
	sb.WriteString("====================================\n")
	sb.WriteString("FOUNDATION (Group 0) — ALREADY GENERATED (import freely, NEVER re-emit)\n")
	sb.WriteString("====================================\n")
	sb.WriteString("These files exist in the project. Import from them by path. Never re-implement or re-emit.\n\n")
	for _, f := range foundationGroup.Files {
		if len(f.Exports) > 0 {
			fmt.Fprintf(&sb, "  %-52s → exports: [%s]\n", f.Path, strings.Join(f.Exports, ", "))
		} else {
			fmt.Fprintf(&sb, "  %s\n", f.Path)
		}
	}
	sb.WriteString("\nCSS VARIABLES (defined in src/index.css — use in Tailwind arbitrary values):\n")
	sb.WriteString("  --background  --foreground  --card  --card-foreground\n")
	sb.WriteString("  --primary     --primary-foreground\n")
	sb.WriteString("  --secondary   --secondary-foreground\n")
	sb.WriteString("  --muted       --muted-foreground\n")
	sb.WriteString("  --accent      --accent-foreground\n")
	sb.WriteString("  --destructive --border  --input  --ring  --radius\n")
	sb.WriteString("\nEntity interfaces are exported from src/types.ts — import what you need.\n")
	sb.WriteString("Pre-built utilities: cn() from @/lib/utils, useApiQuery/useApiMutation from @/hooks/useApi.\n")
	return sb.String()
}

// buildTemplateHooksContext injects useApi.ts and apiUtils.ts source into chunk prompts.
// Feature chunks don't get the full template context (only foundation does), so without
// this they hallucinate callback-based hook signatures that don't exist in the template.
func buildTemplateHooksContext() string {
	critical := map[string]bool{
		"src/hooks/useApi.ts":     true,
		"src/hooks/useAppForm.ts": true,
		"src/lib/apiUtils.ts":     true,
	}
	var sb strings.Builder
	sb.WriteString("\n====================================\n")
	sb.WriteString("TEMPLATE HOOK SOURCE — READ SIGNATURES CAREFULLY\n")
	sb.WriteString("====================================\n")
	sb.WriteString("These files ALREADY EXIST in the project. Import from them using the paths below.\n")
	sb.WriteString("Use EXACTLY these function signatures — do not invent alternative forms.\n\n")
	for _, f := range GetTemplateContext("admin_panel") {
		if critical[f.Path] {
			fmt.Fprintf(&sb, "### %s\n```typescript\n%s\n```\n\n", f.Path, f.Content)
		}
	}
	return sb.String()
}

// injectEnvFile always writes a correct .env into the file list, overriding whatever
// Claude may have generated. We have the real values in Go — no need to trust the AI here.
func injectEnvFile(files []models.ProjectFile, baseURL, apiKey string) []models.ProjectFile {
	envBaseURLKey := "VITE_API_BASE_URL"
	envAPIKeyKey := "VITE_X_API_KEY"
	for _, f := range GetTemplateContext("admin_panel") {
		if strings.Contains(f.Path, "config/axios") || strings.Contains(f.Path, "config/env") {
			if strings.Contains(f.Content, "VITE_BASE_URL") && !strings.Contains(f.Content, "VITE_API_BASE_URL") {
				envBaseURLKey = "VITE_BASE_URL"
			}
			if strings.Contains(f.Content, "VITE_API_KEY") && !strings.Contains(f.Content, "VITE_X_API_KEY") {
				envAPIKeyKey = "VITE_API_KEY"
			}
		}
	}

	content := fmt.Sprintf("%s=%s\n%s=%s\n", envBaseURLKey, baseURL, envAPIKeyKey, apiKey)
	for i, f := range files {
		if f.Path == ".env" || f.Path == ".env.production" {
			files[i].Content = content
			return files
		}
	}
	return append(files, models.ProjectFile{Path: ".env", Content: content})
}

func (p *ChatProcessor) inspectCode(ctx context.Context, userQuestion, filesContext string, chatHistory []models.ChatMessage, imageURLs []string) (string, error) {
	content := chat_prompts.BuildInspectorMessage(userQuestion, filesContext)
	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(content, imageURLs))

	response, err := p.callAnthropicWithTracking(
		ctx,
		models.AnthropicRequest{
			Model:     p.baseConf.InspectorModel,
			MaxTokens: p.baseConf.InspectorMaxTokens,
			System:    chat_prompts.PromptInspector,
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
	content := chat_prompts.BuildPlannerMessage(clarified, fileGraphJSON, hasImages)
	messages := buildMessagesWithHistory(chatHistory, []models.ContentBlock{{Type: "text", Text: content}})

	result, err := callWithTool[models.SonnetPlanResult](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.PlannerModel,
			MaxTokens:  p.baseConf.PlannerMaxTokens,
			System:     chat_prompts.PromptPlanner,
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
		systemPrompt = chat_prompts.PromptCodeEditor
		planJSON, _ := json.Marshal(plan)
		content := chat_prompts.BuildCodeEditorMessage(clarified, string(planJSON), filesContext, len(imageURLs) > 0)
		contentBlocks = buildContentBlocksWithImages(content, imageURLs)
	} else {
		log.Printf("[CODE] planned files not found in project, falling back to free generation")
		systemPrompt = chat_prompts.PromptAdminPanelGenerator
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

func (p *ChatProcessor) generateCodeChunkedWebsite(ctx context.Context, clarified string, imageURLs []string, chatHistory []models.ChatMessage, plan *models.ArchitectPlan, apiKey string) (*models.ParsedClaudeResponse, error) {
	log.Printf("[chunked-web] starting chunked website generation: %s", plan.ProjectName)
	emit := p.emitter()

	// Phase 1: manifest — reuse eager manifest if available.
	var manifest *models.ProjectManifest
	if p.prebuiltManifest != nil {
		manifest = p.prebuiltManifest
		p.prebuiltManifest = nil
		log.Printf("[chunked-web] using prebuilt manifest: %d groups", len(manifest.Groups))
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "list-tree", Message: "Структура страниц готова", Percent: 23})
	} else {
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "list-tree", Message: "Планирую структуру страниц...", Percent: 16})
		if err := withHeartbeat(ctx, emit,
			[]string{"Планирую страницы сайта...", "Строю структуру компонентов...", "Определяю зависимости..."},
			16, 23, 60*time.Second,
			func() error { var e error; manifest, e = p.generateManifest(ctx, plan, chatHistory); return e },
		); err != nil || len(manifest.Groups) < 2 {
			log.Printf("[chunked-web] manifest failed (%v) — falling back to single call", err)
			return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
		}
	}

	var foundationGroup, uiKitGroup models.ManifestGroup
	var pageGroups []models.ManifestGroup
	for _, g := range manifest.Groups {
		switch g.ID {
		case 0:
			foundationGroup = g
		case 1:
			uiKitGroup = g
		default:
			pageGroups = append(pageGroups, g)
		}
	}
	if len(foundationGroup.Files) == 0 || len(pageGroups) == 0 {
		log.Printf("[chunked-web] manifest missing foundation or pages — falling back to single call")
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	pageNames := make([]string, 0, len(pageGroups))
	for _, g := range pageGroups {
		pageNames = append(pageNames, g.Name)
	}
	totalFiles := 0
	for _, g := range manifest.Groups {
		totalFiles += len(g.Files)
	}
	emit.Emit(SSEEvent{
		Type:    EvManifest,
		Icon:    "git-branch",
		Percent: 23,
		Message: "Структура сайта спланирована",
		Value:   fmt.Sprintf("%d файлов · %d страниц", totalFiles, len(pageGroups)),
		Data: ManifestEventData{
			TotalFiles:   totalFiles,
			GroupCount:   len(manifest.Groups),
			FeatureNames: pageNames,
		},
	})

	// Build shared API config block (contains design tokens + image pool).
	apiConfig := buildAPIConfigBlock(p.baseConf.UcodeBaseUrl, apiKey, plan)
	if p.baseConf.UnsplashAccessKey != "" {
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "image", Message: "Подбираю изображения...", Percent: 24})
		pool := helper.FetchImagePool(ctx, p.baseConf.UnsplashAccessKey, plan)
		if pool.Err == nil {
			apiConfig += "\n\n" + pool.Block
			emit.Emit(SSEEvent{Type: EvProgress, Icon: "image", Message: fmt.Sprintf("Подобрано %d фото: %s", pool.Count, strings.Join(pool.Keywords, ", ")), Percent: 25})
		}
	}

	// Phase 2: Foundation + UI Kit in parallel (Sonnet for speed).
	time.Sleep(500 * time.Millisecond)
	emit.Emit(SSEEvent{Type: EvProgress, Icon: "layers", Message: "Генерирую Layout, Navbar, Footer и UI Kit параллельно...", Percent: 25})

	var (
		foundation *models.GeneratedProject
		uiKit      *models.GeneratedProject
		foundErr   error
	)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var e error
		for attempt := 1; attempt <= 2; attempt++ {
			foundation, e = p.generateWebsiteFoundation(ctx, clarified, imageURLs, chatHistory, apiConfig, foundationGroup, manifest)
			if e == nil {
				break
			}
			if attempt < 2 && !errors.Is(e, helper.ErrMaxTokens) {
				log.Printf("[chunked-web] foundation attempt %d failed (%v) — retrying", attempt, e)
			}
		}
		if e != nil {
			foundErr = e
			return
		}
		foundation.Files = filterToGroup(foundation.Files, foundationGroup)
		emit.Emit(SSEEvent{Type: EvProgress, Icon: "check-circle", Message: "Foundation готов", Value: fmt.Sprintf("%d файлов", len(foundation.Files)), Percent: 38})
	}()

	if len(uiKitGroup.Files) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stub := buildFoundationStub(foundationGroup)
			var e error
			uiKit, e = p.generateUIKit(ctx, uiKitGroup, stub, apiConfig, p.baseConf.LandingCoderModel)
			if e != nil {
				log.Printf("[chunked-web] UI kit failed (%v) — continuing without it", e)
				return
			}
			uiKit.Files = filterToGroup(uiKit.Files, uiKitGroup)
			emit.Emit(SSEEvent{Type: EvProgress, Icon: "check-circle", Message: "UI Kit готов", Value: fmt.Sprintf("%d компонентов", len(uiKit.Files)), Percent: 48})
		}()
	}

	if err := withHeartbeat(ctx, emit,
		[]string{
			"Генерирую Layout и навигацию...",
			"Создаю Navbar и Footer...",
			"Настраиваю роутинг в App.tsx...",
			"Создаю UI компоненты...",
			"Формирую глобальные стили и CSS переменные...",
		},
		25, 50, 180*time.Second,
		func() error { wg.Wait(); return foundErr },
	); err != nil {
		log.Printf("[chunked-web] foundation failed (%v) — falling back to single call", err)
		return p.generateCodeSingle(ctx, clarified, imageURLs, chatHistory, plan, apiKey)
	}

	// Phase 3: Pages in parallel (each gets foundation context).
	allSharedFiles := make([]models.ProjectFile, 0, len(foundation.Files))
	allSharedFiles = append(allSharedFiles, foundation.Files...)
	if uiKit != nil {
		allSharedFiles = append(allSharedFiles, uiKit.Files...)
	}
	foundationCtx := buildFoundationContext(allSharedFiles)
	manifestSummary := buildManifestSummary(manifest)

	totalPages := len(pageGroups)
	time.Sleep(500 * time.Millisecond)
	emit.Emit(SSEEvent{
		Type:    EvProgress,
		Icon:    "zap",
		Percent: 51,
		Message: "Генерирую все страницы параллельно",
		Value:   fmt.Sprintf("%d страниц одновременно", totalPages),
	})

	pageResults := make(chan chunkResult, totalPages)

	stopHB := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		tick := 0
		for {
			select {
			case <-ticker.C:
				tick++
				pct := 51 + tick*2
				if pct > 76 {
					pct = 76
				}
				emit.Emit(SSEEvent{Type: EvProgress, Icon: "cpu", Message: "Генерирую страницы параллельно...", Percent: pct})
			case <-stopHB:
				return
			}
		}
	}()
	defer close(stopHB)

	for i, group := range pageGroups {
		g := group
		startDelay := time.Duration(i) * 100 * time.Millisecond
		go func() {
			if startDelay > 0 {
				time.Sleep(startDelay)
			}
			emit.Emit(SSEEvent{Type: EvChunkStart, Icon: "file-text", Message: "Генерирую страницу", Value: g.Name, Data: map[string]any{"feature": g.Name}})
			proj, chunkErr := p.generateWebsitePage(ctx, g, foundationCtx, manifestSummary, apiConfig)
			pageResults <- chunkResult{group: g, project: proj, err: chunkErr}
		}()
	}

	var successChunks []*models.GeneratedProject
	var failedCount int
	for i := range pageGroups {
		res := <-pageResults
		completed := i + 1
		pct := 51 + completed*24/totalPages
		if res.err != nil {
			log.Printf("[chunked-web] page %q error: %v", res.group.Name, res.err)
			failedCount++
			if stub := buildStubChunk(res.group); len(stub.Files) > 0 {
				successChunks = append(successChunks, stub)
			}
		} else {
			res.project.Files = filterToGroup(res.project.Files, res.group)
			log.Printf("[chunked-web] ✅ page %q: %d files", res.group.Name, len(res.project.Files))
			emit.Emit(SSEEvent{
				Type:    EvChunkDone,
				Icon:    "check-circle",
				Percent: pct,
				Message: fmt.Sprintf("Страница готова (%d/%d)", completed, totalPages),
				Value:   res.group.Name,
				Data:    ChunkDoneData{Feature: res.group.Name, Index: completed, Total: totalPages, Files: res.project.Files},
			})
			successChunks = append(successChunks, res.project)
		}
	}

	if failedCount == totalPages {
		return nil, fmt.Errorf("chunked-web: all %d page chunks failed", totalPages)
	}

	var allChunks []*models.GeneratedProject
	if uiKit != nil {
		allChunks = append(allChunks, uiKit)
	}
	allChunks = append(allChunks, successChunks...)
	merged := mergeChunks(foundation, allChunks)

	scaffoldFiles := GetTemplateScaffold("web")
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

	merged.Files = injectEnvFile(merged.Files, p.baseConf.UcodeBaseUrl, apiKey)

	emit.Emit(SSEEvent{Type: EvProgress, Icon: "shield-check", Percent: 80, Message: "Проверяю качество кода", Value: fmt.Sprintf("%d файлов", len(merged.Files))})
	validationErrors := validateGeneratedProject(merged.Files, merged.Env)
	errorCount, _ := logValidationResults(validationErrors)

	if errorCount > 0 {
		emit.Emit(SSEEvent{Type: EvRepair, Icon: "wrench", Percent: 82, Message: "Автоматически исправляю проблемы", Value: fmt.Sprintf("%d ошибок", errorCount)})
		repaired := p.repairBrokenFiles(ctx, merged.Files, validationErrors)
		if len(repaired) > 0 {
			applyRepairs(merged.Files, repaired)
			postErrors := validateGeneratedProject(merged.Files, merged.Env)
			postCount, _ := logValidationResults(postErrors)
			log.Printf("[chunked-web] post-repair: %d errors remaining (was %d)", postCount, errorCount)
		}
	}

	log.Printf("[chunked-web] done: %d total files (%d pages, %d failed, %d validation errors)", len(merged.Files), totalPages, failedCount, errorCount)
	return &models.ParsedClaudeResponse{Project: merged}, nil
}

// generateWebsiteFoundation generates Group 0 files (Layout, Navbar, Footer, App.tsx, index.css)
// for a multi-page website using the website generator prompt and Sonnet model.
func (p *ChatProcessor) generateWebsiteFoundation(
	ctx context.Context,
	clarified string,
	imageURLs []string,
	chatHistory []models.ChatMessage,
	apiConfig string,
	foundationGroup models.ManifestGroup,
	manifest *models.ProjectManifest,
) (*models.GeneratedProject, error) {
	var sb strings.Builder

	sb.WriteString("====================================\n")
	sb.WriteString("⚠ CHUNKED GENERATION — FOUNDATION PHASE ONLY ⚠\n")
	sb.WriteString("====================================\n")
	sb.WriteString("You are generating ONLY the foundation (Group 0) files listed below.\n")
	sb.WriteString("Individual page implementations will be generated separately in parallel.\n")
	sb.WriteString("EMIT ONLY the files in 'YOUR FILES TO GENERATE'. No exceptions.\n\n")

	sb.WriteString("YOUR FILES TO GENERATE (Group 0 — Foundation):\n")
	for _, f := range foundationGroup.Files {
		fmt.Fprintf(&sb, "  %s  (exports: [%s])\n", f.Path, strings.Join(f.Exports, ", "))
	}

	sb.WriteString("\nPAGES — add routes to App.tsx but DO NOT implement their content:\n")
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

	sb.WriteString("\nFULL MANIFEST (for routing reference):\n")
	sb.WriteString(buildManifestSummary(manifest))

	sb.WriteString("\n====================================\n")
	sb.WriteString("PROJECT REQUEST\n")
	sb.WriteString("====================================\n")
	sb.WriteString(clarified)
	sb.WriteString("\n\n")
	sb.WriteString(apiConfig)

	sb.WriteString("\n====================================\n")
	sb.WriteString("REMINDER: Emit ONLY the Group 0 files listed above. DO NOT generate page content.\n")
	sb.WriteString("====================================\n")

	messages := buildMessagesWithHistory(chatHistory, buildContentBlocksWithImages(sb.String(), imageURLs))

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.LandingCoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     chat_prompts.PromptWebsiteGenerator,
			Messages:   messages,
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		"Generating website foundation (Group 0)",
	)
	if err != nil {
		return nil, fmt.Errorf("website foundation: %w", err)
	}
	if len(project.Files) == 0 {
		return nil, fmt.Errorf("website foundation: empty response")
	}
	return project, nil
}

// generateWebsitePage generates one page file with full foundation context injected.
func (p *ChatProcessor) generateWebsitePage(
	ctx context.Context,
	group models.ManifestGroup,
	foundationCtx string,
	manifestSummary string,
	apiConfig string,
) (*models.GeneratedProject, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "CHUNKED GENERATION — Website Page: %s\n\n", group.Name)

	sb.WriteString("YOUR FILE TO IMPLEMENT (emit ONLY this file):\n")
	for _, f := range group.Files {
		fmt.Fprintf(&sb, "  %s  (exports: [%s])\n", f.Path, strings.Join(f.Exports, ", "))
	}

	sb.WriteString("\n")
	sb.WriteString(apiConfig)
	sb.WriteString("\n")
	sb.WriteString(foundationCtx)

	sb.WriteString("\n====================================\n")
	sb.WriteString("PROJECT MANIFEST (for import reference)\n")
	sb.WriteString("====================================\n")
	sb.WriteString(manifestSummary)

	project, err := callWithTool[models.GeneratedProject](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.LandingCoderModel,
			MaxTokens:  p.baseConf.CoderMaxTokens,
			System:     chat_prompts.PromptWebsitePageCoder,
			Messages:   []models.ChatMessage{{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: sb.String()}}}},
			Tools:      []models.ClaudeFunctionTool{helper.ToolEmitProject},
			ToolChoice: helper.ForcedTool(helper.ToolEmitProject.Name),
		},
		timeoutCoder,
		fmt.Sprintf("Generating website page: %s", group.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("website page %s: %w", group.Name, err)
	}
	return project, nil
}
