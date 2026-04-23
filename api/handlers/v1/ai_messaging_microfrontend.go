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

// publishToMicrofrontend calls the function service to create a microfrontend
// and push all AI-generated files to the u-gen branch.
// It stores the resulting microfrontend ID on the processor for the response.
func (p *ChatProcessor) publishToMicrofrontend(ctx context.Context, projectName, path string, generated *models.ParsedClaudeResponse, projectData *models.ProjectData) error {
	if generated == nil || generated.Project == nil || len(generated.Project.Files) == 0 {
		return fmt.Errorf("no generated files to publish")
	}

	// Convert ProjectFile list to the format the function service expects
	files := make([]models.GitlabFileChange, 0, len(generated.Project.Files))
	for _, f := range generated.Project.Files {
		files = append(files, models.GitlabFileChange{
			FilePath: f.Path,
			Content:  f.Content,
		})
	}

	reqBody := models.PublishAiMicroFrontendRequest{
		ProjectId:     projectData.UcodeProjectId,
		EnvironmentId: projectData.EnvironmentId,
		Name:          projectName,
		Path:          path,
		FrameworkType: "REACT",
		Files:         files,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseConf.GoFunctionServiceHost + p.baseConf.GoFunctionServiceHTTPPort + "/v2/functions/micro-frontend/publish-ai"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", p.authToken)

	client := &http.Client{Timeout: timeoutPublishMicrofrontend}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("function service call failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read function service response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("function service returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var result models.PublishAiMicroFrontendResponse
	if err = json.Unmarshal(respBytes, &result); err != nil {
		return fmt.Errorf("parse function service response: %w", err)
	}

	p.microFrontendId = result.Data.ID
	p.microFrontendRepoId = result.Data.RepoId
	log.Printf("[MICROFRONTEND] created id=%s repo_id=%s url=%s", result.Data.ID, result.Data.RepoId, result.Data.Url)
	return nil
}
