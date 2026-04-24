package v1

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	as "ucode/ucode_go_api_gateway/genproto/auth_service"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	helperFunc "ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"
)

const (
	timeoutHaiku     = 180 * time.Second
	timeoutArchitect = 900 * time.Second
	timeoutInspector = 300 * time.Second
	timeoutPlanner   = 300 * time.Second
	timeoutCoder     = 900 * time.Second

	// uGenBranch is the branch where all AI-generated code is pushed.
	// master is only for pipeline triggers (created when the microfrontend is first forked).
	uGenBranch = "u-gen"

	timeoutPublishMicrofrontend = 5 * time.Minute
)

type ChatProcessor struct {
	h                 *HandlerV1
	service           services.ServiceManagerI
	baseConf          config.BaseConfig
	chatId            string
	mcpProjectId      string
	resourceEnvId     string
	ucodeProjectId    string
	mcpUcodeProjectId string

	userId       string
	clientTypeId string
	roleId       string
	authToken    string // forwarded to the function service for microfrontend creation

	microFrontendId     string // populated after PublishAiGeneratedMicroFrontend succeeds, or from request
	microFrontendRepoId string // GitLab numeric project Id — stored from publish response or from request
	newProject          bool   // true → provision a new ucode project; false → create microfrontend in current project

	schemaCache    []models.TableSchema
	schemaCachedAt time.Time
}

func newChatProcessor(h *HandlerV1, service services.ServiceManagerI, baseConf config.BaseConfig, chatId, mcpProjectId, resourceEnvId, ucodeProjectId string, userId, clientTypeId, roleId, authToken string) *ChatProcessor {
	return &ChatProcessor{
		h:              h,
		service:        service,
		baseConf:       baseConf,
		chatId:         chatId,
		mcpProjectId:   mcpProjectId,
		resourceEnvId:  resourceEnvId,
		ucodeProjectId: ucodeProjectId,
		userId:         userId,
		clientTypeId:   clientTypeId,
		roleId:         roleId,
		authToken:      authToken,
	}
}

// ============================================================================
// NEW PROJECT BUILD
// ============================================================================

func (p *ChatProcessor) buildNewProject(ctx context.Context, clarified string, chatHistory []models.ChatMessage, imageURLs []string, estimatedName string) (*models.ParsedClaudeResponse, error) {
	plan, err := p.callArchitect(ctx, clarified, imageURLs, chatHistory, "")
	if err != nil {
		return nil, fmt.Errorf("architect phase failed: %w", err)
	}

	if plan.ProjectName == "" {
		plan.ProjectName = "AI Project"
		if estimatedName != "" {
			plan.ProjectName = estimatedName
		}
	}

	log.Printf("[new-project] architect done: name=%q type=%q design=%s", plan.ProjectName, plan.ProjectType, plan.Design.DesignInspiration)

	projectData, err := p.provisionBackend(ctx, plan.ProjectName, p.mcpProjectId)
	if err != nil {
		return nil, fmt.Errorf("backend provisioning failed: %w", err)
	}

	p.mcpProjectId = projectData.McpProjectId

	go func(bPlan *models.ArchitectPlan, resourceEnvId, ucodeProjectId, userId, envId string) {
		if err := createBackendFromPlan(context.Background(), bPlan, resourceEnvId, ucodeProjectId, userId, envId, p.service); err != nil {
			log.Printf("[new-project] async table creation failed: %v", err)
		}
	}(plan, projectData.ResourceEnvId, projectData.UcodeProjectId, p.userId, projectData.EnvironmentId)

	generated, err := p.generateCode(ctx, clarified, imageURLs, chatHistory, plan, projectData.ApiKey)
	if err != nil {
		return nil, err
	}

	if err = p.publishToMicrofrontend(ctx, plan.ProjectName, uniqueMFEPath(), generated, projectData); err != nil {
		return nil, fmt.Errorf("microfrontend publish failed: %w", err)
	}

	log.Printf("[new-project] done — mfe_id=%s", p.microFrontendId)
	return &models.ParsedClaudeResponse{Description: generated.Description}, nil
}

func (p *ChatProcessor) buildMicrofrontendForCurrentProject(ctx context.Context, clarified string, chatHistory []models.ChatMessage, imageURLs []string, estimatedName string) (*models.ParsedClaudeResponse, error) {
	// Fetch existing project schema so the architect knows which tables/APIs are already available.
	var schemaCtx string
	schema, err := p.getProjectSchemaCached(ctx, p.resourceEnvId)
	if err != nil {
		log.Printf("[mfe-current] could not fetch project schema (non-fatal): %v", err)
	} else if len(schema) > 0 {
		schemaLines := strings.Builder{}
		for _, t := range schema {
			schemaLines.WriteString(fmt.Sprintf("- table: %s (slug: %s)\n", t.Label, t.Slug))
			for _, f := range t.Fields {
				schemaLines.WriteString(fmt.Sprintf("  * %s (%s)\n", f.Slug, f.Type))
			}
		}
		schemaCtx = schemaLines.String()
	}

	plan, err := p.callArchitect(ctx, clarified, imageURLs, chatHistory, schemaCtx)
	if err != nil {
		return nil, fmt.Errorf("architect phase failed: %w", err)
	}

	if plan.ProjectName == "" {
		plan.ProjectName = "AI Project"
		if estimatedName != "" {
			plan.ProjectName = estimatedName
		}
	}

	log.Printf("[mfe-current] architect done: name=%q type=%q design=%s", plan.ProjectName, plan.ProjectType, plan.Design.DesignInspiration)

	projectData, err := p.getExistingProjectData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing project data: %w", err)
	}

	// Create any NEW tables the architect defined that don't yet exist in the project.
	if len(plan.Tables) > 0 {
		existingSlugs := make(map[string]bool, len(schema))
		for _, t := range schema {
			existingSlugs[t.Slug] = true
		}
		newTables := make([]models.TablePlan, 0)
		for _, t := range plan.Tables {
			if !existingSlugs[t.Slug] {
				newTables = append(newTables, t)
			}
		}
		if len(newTables) > 0 {
			log.Printf("[mfe-current] architect defined %d new table(s) — provisioning async", len(newTables))
			newPlan := &models.ArchitectPlan{
				ProjectName: plan.ProjectName,
				ProjectType: plan.ProjectType,
				Tables:      newTables,
			}
			go func(bPlan *models.ArchitectPlan, resourceEnvId, ucodeProjectId, userId, envId string) {
				if err := createBackendFromPlan(context.Background(), bPlan, resourceEnvId, ucodeProjectId, userId, envId, p.service); err != nil {
					log.Printf("[mfe-current] async table creation failed: %v", err)
				}
			}(newPlan, projectData.ResourceEnvId, projectData.UcodeProjectId, p.userId, projectData.EnvironmentId)
		}
	}

	generated, err := p.generateCode(ctx, clarified, imageURLs, chatHistory, plan, projectData.ApiKey)
	if err != nil {
		return nil, err
	}

	if err = p.publishToMicrofrontend(ctx, plan.ProjectName, uniqueMFEPath(), generated, projectData); err != nil {
		return nil, fmt.Errorf("microfrontend publish failed: %w", err)
	}

	log.Printf("[mfe-current] done — mfe_id=%s", p.microFrontendId)
	return &models.ParsedClaudeResponse{Description: generated.Description}, nil
}

func (p *ChatProcessor) getExistingProjectData(ctx context.Context) (*models.ProjectData, error) {
	ucodeProjectId := p.ucodeProjectId

	if p.mcpProjectId != "" {
		mcpProject, err := p.service.GoObjectBuilderService().McpProject().GetMcpProjectFiles(ctx, &pbo.McpProjectId{
			ResourceEnvId: p.resourceEnvId,
			Id:            p.mcpProjectId,
		})
		if err == nil && mcpProject != nil && mcpProject.GetUcodeProjectId() != "" {
			ucodeProjectId = mcpProject.GetUcodeProjectId()
			log.Printf("[GET EXISTING PROJECT] using ucode_project_id=%s from MCP project", ucodeProjectId)
		}
	}

	envList, err := p.h.companyServices.Environment().GetList(ctx, &pb.GetEnvironmentListRequest{
		ProjectId: ucodeProjectId,
		Limit:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("get environment list: %w", err)
	}

	envs := envList.GetEnvironments()
	if len(envs) == 0 {
		return nil, fmt.Errorf("no environment found for project %s", ucodeProjectId)
	}
	env := envs[0]

	apiKeys, err := p.h.authService.ApiKey().GetList(ctx, &as.GetListReq{
		EnvironmentId: env.GetId(),
		ProjectId:     ucodeProjectId,
		Limit:         1,
	})
	if err != nil {
		return nil, fmt.Errorf("get api keys: %w", err)
	}

	var apiKey string

	if keys := apiKeys.GetData(); len(keys) > 0 {
		apiKey = keys[0].GetAppId()
	}

	return &models.ProjectData{
		UcodeProjectId: ucodeProjectId,
		EnvironmentId:  env.GetId(),
		ResourceEnvId:  env.GetResourceEnvironmentId(),
		ApiKey:         apiKey,
	}, nil
}

func (p *ChatProcessor) provisionBackend(ctx context.Context, projectName string, existingMcpId string) (*models.ProjectData, error) {
	currentProject, err := p.h.companyServices.Project().GetById(
		ctx, &pb.GetProjectByIdRequest{
			ProjectId: p.ucodeProjectId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get current project info: %w", err)
	}

	backendProject, err := p.h.companyServices.Project().Create(
		ctx, &pb.CreateProjectRequest{
			Title:        projectName,
			CompanyId:    currentProject.GetCompanyId(),
			K8SNamespace: currentProject.GetK8SNamespace(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create backend project: %w", err)
	}

	env, err := p.h.companyServices.Environment().CreateV2(
		ctx, &pb.CreateEnvironmentRequest{
			CompanyId:    currentProject.GetCompanyId(),
			ProjectId:    backendProject.GetProjectId(),
			UserId:       p.userId,
			ClientTypeId: p.clientTypeId,
			RoleId:       p.roleId,
			Name:         "Production",
			DisplayColor: "#00FF00",
			Description:  "Production Environment",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}

	// CreateV2 provisions the resource synchronously but returns the Environment
	// object created BEFORE resource provisioning — so ResourceEnvironmentId is empty.
	// Re-fetch to get the fully populated record.
	env, err = p.h.companyServices.Environment().GetById(ctx, &pb.EnvironmentPrimaryKey{Id: env.GetId()})
	if err != nil {
		return nil, fmt.Errorf("re-fetch environment after CreateV2: %w", err)
	}
	resourceEnvId := env.GetResourceEnvironmentId()
	if resourceEnvId == "" {
		return nil, fmt.Errorf("ResourceEnvironmentId empty after provisioning env %s", env.GetId())
	}
	log.Printf("[provisionBackend] project=%s env=%s resourceEnv=%s", backendProject.GetProjectId(), env.GetId(), resourceEnvId)

	apiKeys, err := p.h.authService.ApiKey().GetList(
		ctx, &as.GetListReq{
			EnvironmentId: env.GetId(),
			ProjectId:     backendProject.GetProjectId(),
			Limit:         1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fetch api keys: %w", err)
	}

	var apiKey string
	if len(apiKeys.GetData()) > 0 {
		apiKey = apiKeys.GetData()[0].GetAppId()
	}

	mcpProjectId := existingMcpId
	if mcpProjectId != "" {
		_, err = p.service.GoObjectBuilderService().McpProject().UpdateMcpProject(
			ctx, &pbo.McpProject{
				ResourceEnvId:  p.resourceEnvId,
				Id:             mcpProjectId,
				Title:          projectName,
				Description:    "Provisioned by AI architect",
				UcodeProjectId: backendProject.GetProjectId(),
				ApiKey:         apiKey,
				EnvironmentId:  env.GetId(),
				Status:         "ready",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("update MCP project: %w", err)
		}
	} else {
		project, err := p.service.GoObjectBuilderService().McpProject().CreateMcpProject(
			ctx, &pbo.CreateMcpProjectReqeust{
				ResourceEnvId:  p.resourceEnvId,
				Title:          projectName,
				Description:    "Generated by AI Architect",
				UcodeProjectId: backendProject.GetProjectId(),
				ApiKey:         apiKey,
				EnvironmentId:  env.GetId(),
				Status:         "ready",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("create MCP project link: %w", err)
		}
		mcpProjectId = project.GetId()
	}

	p.mcpUcodeProjectId = backendProject.GetProjectId()

	return &models.ProjectData{
		UcodeProjectId: backendProject.GetProjectId(),
		McpProjectId:   mcpProjectId,
		ApiKey:         apiKey,
		EnvironmentId:  env.GetId(),
		ResourceEnvId:  resourceEnvId,
	}, nil
}

// ============================================================================
// DATA ACCESS HELPERS
// ============================================================================

func (p *ChatProcessor) saveMessage(ctx context.Context, role, content string, images []string) (*pbo.Message, error) {
	return p.service.GoObjectBuilderService().AiChat().CreateMessage(ctx, &pbo.CreateMessageRequest{
		ChatId:        p.chatId,
		Role:          role,
		Content:       content,
		Images:        images,
		ResourceEnvId: p.resourceEnvId,
	})
}

func (p *ChatProcessor) getChatHistory(ctx context.Context) ([]models.ChatMessage, error) {
	messages, err := p.service.GoObjectBuilderService().AiChat().GetMessages(ctx, &pbo.GetMessagesRequest{
		ResourceEnvId: p.resourceEnvId,
		ChatId:        p.chatId,
	})
	if err != nil {
		return nil, fmt.Errorf("get chat history: %w", err)
	}

	msgList := messages.GetMessages()
	if len(msgList) > 10 {
		msgList = msgList[len(msgList)-10:]
	}

	result := make([]models.ChatMessage, 0, len(msgList))
	for _, msg := range msgList {
		text := msg.GetContent()
		// Strip embedded plan JSON — the AI only needs the marker + description for state detection.
		if strings.HasPrefix(text, "[DIAGRAMS_GENERATED] ") {
			if idx := strings.Index(text, "\n"); idx != -1 {
				text = text[:idx]
			}
		}
		result = append(result, models.ChatMessage{
			Role:    msg.GetRole(),
			Content: []models.ContentBlock{{Type: "text", Text: text}},
		})
	}
	return result, nil
}

func (p *ChatProcessor) saveProject(ctx context.Context, req *models.ParsedClaudeResponse) (*pbo.McpProject, error) {
	if req == nil || req.Project == nil {
		return nil, fmt.Errorf("invalid project data")
	}

	projectEnv, err := helperFunc.ConvertMapToStruct(req.Project.Env)
	if err != nil {
		return nil, fmt.Errorf("convert project env: %w", err)
	}

	var projectFiles []*pbo.McpProjectFiles
	for _, file := range req.Project.Files {
		var fileGraph map[string]any
		if val, ok := req.Project.FileGraph[file.Path].(map[string]any); ok {
			fileGraph = val
		}
		fileGraphStruct, _ := helperFunc.ConvertMapToStruct(fileGraph)
		projectFiles = append(projectFiles, &pbo.McpProjectFiles{
			Path:      file.Path,
			Content:   file.Content,
			FileGraph: fileGraphStruct,
		})
	}

	return p.service.GoObjectBuilderService().McpProject().UpdateMcpProject(ctx, &pbo.McpProject{
		Id:            p.mcpProjectId,
		ResourceEnvId: p.resourceEnvId,
		Title:         truncateString(req.Project.ProjectName, 255),
		Description:   truncateString(req.Description, 255),
		ProjectFiles:  projectFiles,
		ProjectEnv:    projectEnv,
	})
}

// ============================================================================
// UTILITIES
// ============================================================================

func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// buildAPIConfigBlock generates the API configuration + design tokens injected into the coder prompt.
func buildAPIConfigBlock(baseURL, apiKey string, plan *models.ArchitectPlan) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"\n====================================\nAPI CONFIGURATION FOR FRONTEND\n====================================\nVITE_API_BASE_URL: %s\nVITE_X_API_KEY: %s\n\nTables to use:\n",
		baseURL, apiKey,
	))
	for _, t := range plan.Tables {
		sb.WriteString(fmt.Sprintf("- Table: %s, slug: %s\n", t.Label, t.Slug))
		for _, f := range t.Fields {
			sb.WriteString(fmt.Sprintf("  * field: %s, type: %s\n", f.Slug, f.Type))
		}
	}
	sb.WriteString("\nUse this UI Structure provided by the Architect:\n" + plan.UIStructure + "\n")

	// Inject design tokens so the coder doesn't have to invent a design system.
	d := plan.Design
	if d.PrimaryColor != "" {
		sb.WriteString("\n====================================\nDESIGN TOKENS\n====================================\n")
		sb.WriteString(fmt.Sprintf("design_inspiration: %s\n", d.DesignInspiration))
		sb.WriteString(fmt.Sprintf("font_family (heading): %s\n", d.FontFamily))
		sb.WriteString(fmt.Sprintf("body_font: %s\n", d.BodyFont))
		sb.WriteString(fmt.Sprintf("border_radius: %s\n", d.BorderRadius))
		sb.WriteString(fmt.Sprintf("background_color: %s  (HSL: %s)\n", d.BackgroundColor, d.BackgroundHSL))
		sb.WriteString(fmt.Sprintf("surface_color: %s  (HSL: %s)\n", d.SurfaceColor, d.SurfaceHSL))
		sb.WriteString(fmt.Sprintf("primary_color: %s  (HSL: %s)\n", d.PrimaryColor, d.PrimaryHSL))
		sb.WriteString(fmt.Sprintf("accent_color: %s  (HSL: %s)\n", d.AccentColor, d.AccentHSL))
		sb.WriteString(fmt.Sprintf("text_color: %s\n", d.TextColor))
		sb.WriteString(fmt.Sprintf("text_muted_color: %s\n", d.TextMutedColor))
		sb.WriteString(fmt.Sprintf("border_color: %s\n", d.BorderColor))
		sb.WriteString(fmt.Sprintf("sidebar_background: %s  (HSL: %s)\n", d.SidebarBackground, d.SidebarBackgroundHSL))
		sb.WriteString(fmt.Sprintf("sidebar_foreground: %s\n", d.SidebarForeground))
		sb.WriteString(fmt.Sprintf("sidebar_style: %s\n", d.SidebarStyle))
	}

	return sb.String()
}

func buildContentBlocksWithImages(textContent string, imageURLs []string) []models.ContentBlock {
	blocks := make([]models.ContentBlock, 0, len(imageURLs)+1)
	for _, imageURL := range imageURLs {
		if strings.TrimSpace(imageURL) != "" {
			blocks = append(blocks, models.ContentBlock{
				Type:   "image",
				Source: &models.ImageSource{Type: "url", URL: imageURL},
			})
		}
	}
	blocks = append(blocks, models.ContentBlock{Type: "text", Text: textContent})
	return blocks
}

func buildMessagesWithHistory(history []models.ChatMessage, contentBlocks []models.ContentBlock) []models.ChatMessage {
	messages := make([]models.ChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, models.ChatMessage{
		Role:    "user",
		Content: contentBlocks,
	})
	return messages
}

func buildHistoryText(history []models.ChatMessage) string {
	if len(history) == 0 {
		return ""
	}
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	var sb strings.Builder
	for _, msg := range history[start:] {
		var text string
		for _, block := range msg.Content {
			if block.Type == "text" {
				text += block.Text
			}
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", strings.ToUpper(msg.Role), text))
	}
	return sb.String()
}

// uniqueMFEPath returns a short unique GitLab path for a new microfrontend:
// "app-XXXXXX" where XXXXXX is 6 random lowercase hex chars.
// This prevents GitLab project name collisions on retries.
// Format: 10 chars → functionPath in function service stays ≤ 20 chars.
func uniqueMFEPath() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("app-%x", time.Now().UnixNano()&0xFFFFFF)
	}
	return fmt.Sprintf("app-%x", b)
}

// slugify converts a project name to a lowercase hyphen-separated slug
// valid for use as a GitLab path (only [a-z0-9-]).
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) == 0 {
		s = "ai-project"
	}
	return s
}
