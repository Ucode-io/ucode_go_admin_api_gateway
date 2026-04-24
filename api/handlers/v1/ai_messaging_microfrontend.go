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
	if err = p.pushMicrofrontendChanges(ctx, edited.Project.Files); err != nil {
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

// pushMicrofrontendChanges sends AI-generated files to the function service which
// commits them to the u-gen branch of the microfrontend's repo.
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

	type pushReq struct {
		RepoID int                       `json:"repo_id"`
		Files  []models.GitlabFileChange `json:"files"`
	}

	bodyBytes, err := json.Marshal(pushReq{RepoID: repoIDInt, Files: files})
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

// publishToMicrofrontend creates the microfrontend repo and pushes AI-generated
// files using a two-phase approach:
//  1. publish-ai — creates the GitLab repo with a minimal init file
//  2. push-changes — pushes all content files (same endpoint used for edits)
//
// This avoids hitting payload-size or file-count limits on the publish-ai endpoint.
func (p *ChatProcessor) publishToMicrofrontend(ctx context.Context, projectName, path string, generated *models.ParsedClaudeResponse, projectData *models.ProjectData) error {
	if generated == nil || generated.Project == nil || len(generated.Project.Files) == 0 {
		return fmt.Errorf("no generated files to publish")
	}

	// Build the sanitized file list once — used in both phases.
	allFiles := make([]models.ProjectFile, 0, len(generated.Project.Files))
	for _, f := range generated.Project.Files {
		cleanPath := strings.TrimLeft(f.Path, "/")
		if cleanPath == "" {
			continue
		}
		allFiles = append(allFiles, models.ProjectFile{
			Path:    cleanPath,
			Content: sanitizeFileContent(f.Content),
		})
	}

	safeName := slugify(projectName)
	if safeName == "" {
		safeName = "ai-project"
	}
	safePath := slugify(path)
	if safePath == "" {
		safePath = "app"
	}

	// ── Phase 1: Create the GitLab repo with a single small init file ─────────
	// We pass only package.json (always present for React projects) so the
	// publish-ai endpoint can initialize the repo without a large payload.
	var initFile models.GitlabFileChange
	for _, f := range allFiles {
		if f.Path == "package.json" {
			initFile = models.GitlabFileChange{FilePath: f.Path, Content: f.Content}
			break
		}
	}
	if initFile.FilePath == "" && len(allFiles) > 0 {
		// Fallback: use the first (usually smallest) file.
		initFile = models.GitlabFileChange{FilePath: allFiles[0].Path, Content: allFiles[0].Content}
	}

	createBody := models.PublishAiMicroFrontendRequest{
		ProjectId:     projectData.UcodeProjectId,
		EnvironmentId: projectData.EnvironmentId,
		Name:          safeName,
		Path:          safePath,
		FrameworkType: "REACT",
		Files:         []models.GitlabFileChange{initFile},
	}

	createBytes, err := json.Marshal(createBody)
	if err != nil {
		return fmt.Errorf("marshal create request: %w", err)
	}

	log.Printf("[MICROFRONTEND] publish-ai (phase 1): name=%q path=%q project_id=%s env_id=%s init_file=%s total_files=%d",
		safeName, safePath, projectData.UcodeProjectId, projectData.EnvironmentId, initFile.FilePath, len(allFiles))

	createURL := p.baseConf.GoFunctionServiceHost + p.baseConf.GoFunctionServiceHTTPPort + "/v2/functions/micro-frontend/publish-ai"
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(createBytes))
	if err != nil {
		return fmt.Errorf("build create request: %w", err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", p.authToken)

	client := &http.Client{Timeout: timeoutPublishMicrofrontend}
	createResp, err := client.Do(createReq)
	if err != nil {
		return fmt.Errorf("publish-ai call failed: %w", err)
	}
	defer createResp.Body.Close()

	createRespBytes, err := io.ReadAll(createResp.Body)
	if err != nil {
		return fmt.Errorf("read publish-ai response: %w", err)
	}
	if createResp.StatusCode >= 400 {
		return fmt.Errorf("publish-ai returned %d: %s", createResp.StatusCode, string(createRespBytes))
	}

	var createResult models.PublishAiMicroFrontendResponse
	if err = json.Unmarshal(createRespBytes, &createResult); err != nil {
		return fmt.Errorf("parse publish-ai response: %w", err)
	}

	p.microFrontendId = createResult.Data.ID
	p.microFrontendRepoId = createResult.Data.RepoId
	log.Printf("[MICROFRONTEND] repo created: id=%s repo_id=%s", createResult.Data.ID, createResult.Data.RepoId)

	// ── Phase 2: Push all files via push-changes (proven endpoint) ────────────
	log.Printf("[MICROFRONTEND] push-changes (phase 2): pushing %d files to repo_id=%s", len(allFiles), p.microFrontendRepoId)
	if err = p.pushMicrofrontendChanges(ctx, allFiles); err != nil {
		return fmt.Errorf("push-changes after publish-ai failed: %w", err)
	}

	log.Printf("[MICROFRONTEND] ✅ published: id=%s url=%s", p.microFrontendId, createResult.Data.Url)
	return nil
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
