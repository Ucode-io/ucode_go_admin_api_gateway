package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

const publishChunkSize = 30

const pushRetryCount = 3

const pushRetryDelay = 2 * time.Second

// runMicrofrontendEdit fetches the current files from u-gen, asks the AI to edit
// them, then pushes the result back to u-gen. No McpProject is touched.
func (p *ChatProcessor) runMicrofrontendEdit(ctx context.Context, clarified, fileGraphJSON string, chatHistory []models.ChatMessage, imageURLs []string, existingFiles []models.GitlabFileChange) (*models.ParsedClaudeResponse, error) {
	log.Printf("[MICROFE EDIT] planning changes for microfrontend id=%s", p.microFrontendId)

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
	if err = p.pushMicrofrontendChangesChunked(ctx, edited.Project.Files); err != nil {
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
		"/v2/functions/micro-frontend/files?repo_id=" + p.microFrontendRepoId

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

// pushMicrofrontendChanges отправляет файлы в push-changes одним запросом.
// Используется для небольших правок (edit flow).
func (p *ChatProcessor) pushMicrofrontendChanges(ctx context.Context, generatedFiles []models.ProjectFile) error {
	repoIDInt := 0
	fmt.Sscanf(p.microFrontendRepoId, "%d", &repoIDInt)
	if repoIDInt == 0 {
		return fmt.Errorf("invalid microfrontend_repo_id: %q", p.microFrontendRepoId)
	}

	files := make([]models.GitlabFileChange, 0, len(generatedFiles))
	for _, f := range generatedFiles {
		files = append(files, models.GitlabFileChange{
			FilePath: f.Path,
			Content:  f.Content,
		})
	}

	return p.doPushChanges(ctx, repoIDInt, files)
}

// и отправляет каждый с retry. Используется при публикации (121+ файл).
func (p *ChatProcessor) pushMicrofrontendChangesChunked(ctx context.Context, generatedFiles []models.ProjectFile) error {
	repoIDInt := 0
	fmt.Sscanf(p.microFrontendRepoId, "%d", &repoIDInt)
	if repoIDInt == 0 {
		return fmt.Errorf("invalid microfrontend_repo_id: %q", p.microFrontendRepoId)
	}

	files := make([]models.GitlabFileChange, 0, len(generatedFiles))
	for _, f := range generatedFiles {
		files = append(files, models.GitlabFileChange{
			FilePath: f.Path,
			Content:  f.Content,
		})
	}

	total := len(files)
	chunks := splitIntoChunks(files, publishChunkSize)
	log.Printf("[MICROFE PUSH] pushing %d files in %d chunk(s) of max %d", total, len(chunks), publishChunkSize)

	for i, chunk := range chunks {
		log.Printf("[MICROFE PUSH] chunk %d/%d: %d files", i+1, len(chunks), len(chunk))

		var lastErr error
		for attempt := 1; attempt <= pushRetryCount; attempt++ {
			lastErr = p.doPushChanges(ctx, repoIDInt, chunk)
			if lastErr == nil {
				log.Printf("[MICROFE PUSH] ✅ chunk %d/%d done", i+1, len(chunks))
				break
			}
			log.Printf("[MICROFE PUSH] chunk %d/%d attempt %d/%d failed: %v", i+1, len(chunks), attempt, pushRetryCount, lastErr)
			if attempt < pushRetryCount {
				time.Sleep(pushRetryDelay)
			}
		}

		if lastErr != nil {
			return fmt.Errorf("push chunk %d/%d failed after %d attempts: %w", i+1, len(chunks), pushRetryCount, lastErr)
		}
	}

	log.Printf("[MICROFE PUSH] ✅ all %d files pushed successfully", total)
	return nil
}

// doPushChanges выполняет один HTTP-запрос к push-changes endpoint.
func (p *ChatProcessor) doPushChanges(ctx context.Context, repoIDInt int, files []models.GitlabFileChange) error {
	if len(files) == 0 {
		return fmt.Errorf("doPushChanges: no files provided")
	}

	type pushReq struct {
		RepoID        int                       `json:"repo_id"`
		Files         []models.GitlabFileChange `json:"files"`
		CommitMessage string                    `json:"commit_message"`
	}

	bodyBytes, err := json.Marshal(pushReq{
		RepoID:        repoIDInt,
		Files:         files,
		CommitMessage: p.userMessage,
	})
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

// splitIntoChunks делит слайс на части размером не более chunkSize.
func splitIntoChunks(files []models.GitlabFileChange, chunkSize int) [][]models.GitlabFileChange {
	if chunkSize <= 0 {
		chunkSize = publishChunkSize
	}
	var chunks [][]models.GitlabFileChange
	for len(files) > 0 {
		end := chunkSize
		if end > len(files) {
			end = len(files)
		}
		chunks = append(chunks, files[:end])
		files = files[end:]
	}
	return chunks
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

// publishToMicrofrontend публикует сгенерированный проект в microfrontend.
//
// Стратегия (исправленная):
//  1. publish-ai — создаём GitLab репо, передаём ВСЕ файлы сразу.
//     Endpoint требует минимум один коммит с реальными файлами для инициализации ветки u-gen.
//  2. Если publish-ai вернул ошибку "no files provided" или 5xx — повторяем с retry.
//
// Фаза 2 (push-changes) больше НЕ используется при публикации, т.к. publish-ai
// сам делает первый коммит. push-changes используется только для последующих правок (edit flow).
func (p *ChatProcessor) publishToMicrofrontend(ctx context.Context, projectName, path string, generated *models.ParsedClaudeResponse, projectData *models.ProjectData) error {
	if generated == nil || generated.Project == nil || len(generated.Project.Files) == 0 {
		return fmt.Errorf("no generated files to publish")
	}

	// Build the sanitized file list once.
	allFiles := make([]models.GitlabFileChange, 0, len(generated.Project.Files))
	for _, f := range generated.Project.Files {
		cleanPath := strings.TrimLeft(f.Path, "/")
		if cleanPath == "" {
			continue
		}
		allFiles = append(allFiles, models.GitlabFileChange{
			FilePath: cleanPath,
			Content:  sanitizeFileContent(f.Content),
		})
	}

	if len(allFiles) == 0 {
		return fmt.Errorf("no valid files to publish after sanitization")
	}

	safeName := slugify(projectName)
	if safeName == "" {
		safeName = "ai-project"
	}
	safePath := slugify(path)
	if safePath == "" {
		safePath = "app"
	}

	// ── Phase 1: создаём GitLab репо с ПЕРВЫМ чанком файлов ──────────────────
	// publish-ai требует реальные файлы для инициализации ветки u-gen.
	// Передаём первые publishChunkSize файлов (включая package.json если есть).
	// Остальные файлы пушим через push-changes в Phase 2.

	// Сортируем: package.json всегда первым чтобы repo корректно инициализировался.
	initFiles := buildInitFiles(allFiles, publishChunkSize)

	log.Printf("[MICROFRONTEND] publish-ai: name=%q path=%q project_id=%s env_id=%s init_files=%d total_files=%d",
		safeName, safePath, projectData.UcodeProjectId, projectData.EnvironmentId, len(initFiles), len(allFiles))

	createBody := models.PublishAiMicroFrontendRequest{
		ProjectId:     projectData.UcodeProjectId,
		EnvironmentId: projectData.EnvironmentId,
		Name:          safeName,
		Path:          safePath,
		FrameworkType: "REACT",
		Files:         initFiles,
	}

	createBytes, err := json.Marshal(createBody)
	if err != nil {
		return fmt.Errorf("marshal create request: %w", err)
	}

	createURL := p.baseConf.GoFunctionServiceHost + p.baseConf.GoFunctionServiceHTTPPort + "/v2/functions/micro-frontend/publish-ai"

	var createResult models.PublishAiMicroFrontendResponse

	// Retry publish-ai до 3 раз — иногда сервис даёт 500 на первом запросе.
	for attempt := 1; attempt <= pushRetryCount; attempt++ {
		createResult, err = p.doPublishAI(ctx, createURL, createBytes)
		if err == nil {
			break
		}
		log.Printf("[MICROFRONTEND] publish-ai attempt %d/%d failed: %v", attempt, pushRetryCount, err)
		if attempt < pushRetryCount {
			time.Sleep(pushRetryDelay)
		}
	}
	if err != nil {
		return fmt.Errorf("microfrontend publish failed: %w", err)
	}

	p.microFrontendId = createResult.Data.ID
	p.microFrontendRepoId = createResult.Data.RepoId
	log.Printf("[MICROFRONTEND] repo created: id=%s repo_id=%s url=%s", createResult.Data.ID, createResult.Data.RepoId, createResult.Data.Url)

	// ── Phase 2: пушим оставшиеся файлы через push-changes (если они есть) ───
	// Файлы которые уже были в initFiles — пропускаем (они уже в репо).
	initSet := make(map[string]struct{}, len(initFiles))
	for _, f := range initFiles {
		initSet[f.FilePath] = struct{}{}
	}
	remainingFiles := make([]models.ProjectFile, 0, len(allFiles))
	for _, f := range allFiles {
		if _, alreadyPushed := initSet[f.FilePath]; !alreadyPushed {
			remainingFiles = append(remainingFiles, models.ProjectFile{
				Path:    f.FilePath,
				Content: f.Content,
			})
		}
	}

	if len(remainingFiles) > 0 {
		log.Printf("[MICROFRONTEND] push-changes: pushing remaining %d files in chunks", len(remainingFiles))
		if pushErr := p.pushMicrofrontendChangesChunked(ctx, remainingFiles); pushErr != nil {
			// НЕ фейлим весь деплой — репо уже создан, частичный результат лучше нуля.
			log.Printf("[MICROFRONTEND] ⚠️ push-changes partial failure (repo exists, %d files missing): %v", len(remainingFiles), pushErr)
		}
	}

	log.Printf("[MICROFRONTEND] ✅ published: id=%s url=%s", p.microFrontendId, createResult.Data.Url)
	return nil
}

// doPublishAI выполняет один HTTP-запрос к publish-ai endpoint.
func (p *ChatProcessor) doPublishAI(ctx context.Context, url string, bodyBytes []byte) (models.PublishAiMicroFrontendResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("build create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", p.authToken)

	client := &http.Client{Timeout: timeoutPublishMicrofrontend}
	createResp, err := client.Do(httpReq)
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("publish-ai call failed: %w", err)
	}
	defer createResp.Body.Close()

	createRespBytes, err := io.ReadAll(createResp.Body)
	if err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("read publish-ai response: %w", err)
	}
	if createResp.StatusCode >= 400 {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("publish-ai returned %d: %s", createResp.StatusCode, string(createRespBytes))
	}

	var result models.PublishAiMicroFrontendResponse
	if err = json.Unmarshal(createRespBytes, &result); err != nil {
		return models.PublishAiMicroFrontendResponse{}, fmt.Errorf("parse publish-ai response: %w", err)
	}
	return result, nil
}

// buildInitFiles возвращает первые n файлов для phase 1, гарантируя что package.json идёт первым.
func buildInitFiles(allFiles []models.GitlabFileChange, n int) []models.GitlabFileChange {
	if n <= 0 || n >= len(allFiles) {
		return allFiles
	}

	result := make([]models.GitlabFileChange, 0, n)

	// Сначала ищем package.json — он нужен для инициализации React-репо.
	pkgIdx := -1
	for i, f := range allFiles {
		if f.FilePath == "package.json" {
			pkgIdx = i
			break
		}
	}

	if pkgIdx >= 0 {
		result = append(result, allFiles[pkgIdx])
	}

	// Добавляем остальные файлы до лимита n.
	for i, f := range allFiles {
		if len(result) >= n {
			break
		}
		if i == pkgIdx {
			continue // уже добавили
		}
		result = append(result, f)
	}

	return result
}

// sanitizeFileContent removes characters that can cause JSON parse failures
// in downstream services: null bytes, BOM, and other C0 control characters
// except standard whitespace (tab=0x09, newline=0x0A, carriage-return=0x0D).
func sanitizeFileContent(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			sb.WriteRune(r)
		case r == '\uFEFF': // BOM — strip silently
		case r < 0x20: // other C0 control characters — strip
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
