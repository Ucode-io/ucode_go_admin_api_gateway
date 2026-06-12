package v1

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
)

// runCreateAgent handles the create_agent intent: it designs a reusable end-user
// agent from the builder's natural-language description, validates the requested
// permissions against the real project schema, persists the agent, and returns a
// builder-facing confirmation.
func (p *ChatProcessor) runCreateAgent(ctx context.Context, clarified string, chatHistory []models.ChatMessage) (*models.ParsedClaudeResponse, error) {

	resourceEnvId, err := p.resolveBuilderResourceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve builder resource ID: %w", err)
	}

	schema, err := p.getProjectSchemaCached(ctx, resourceEnvId)
	if err != nil {
		return nil, fmt.Errorf("schema fetch failed: %w", err)
	}
	if len(schema) == 0 {
		return &models.ParsedClaudeResponse{
			Description: "В этом проекте пока нет таблиц. Сначала создайте таблицы, чтобы агент мог с ними работать.",
		}, nil
	}

	spec, err := p.agent.BuildAgentSpec(ctx, models.AgentSpecInput{
		Description: clarified,
		SchemaText:  formatSchemaForSQL(schema),
		History:     chatHistory,
	})
	if err != nil {
		return nil, fmt.Errorf("build agent spec: %w", err)
	}

	perms := buildValidatedPermissions(spec.Permissions, schema)
	if len(perms) == 0 {
		return &models.ParsedClaudeResponse{
			Description: "Не удалось определить, к каким таблицам агенту нужен доступ. Пожалуйста, уточните, что именно должен делать агент.",
		}, nil
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		name = "AI Agent"
	}

	agent, err := p.service.GoObjectBuilderService().Agent().CreateAgent(ctx, &nb.CreateAgentRequest{
		ResourceEnvId: resourceEnvId,
		ProjectId:     resourceEnvId,
		Name:          name,
		Description:   spec.Description,
		Instruction:   spec.Instruction,
		Model:         p.agentCfgs().Coder.Model,
		MaxSteps:      defaultAgentMaxSteps,
		Enabled:       true,
		Permissions:   perms,
	})
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	reply := strings.TrimSpace(spec.Reply)
	if reply == "" {
		reply = fmt.Sprintf("Агент «%s» создан.", agent.GetName())
	}

	return &models.ParsedClaudeResponse{
		Description: reply + "\n\n" + formatAgentSummary(agent),
	}, nil
}

// buildValidatedPermissions converts the model's proposed permissions into proto
// rules, keeping only rules that reference a real table slug, are not duplicated,
// and grant at least one operation. This is the trust boundary for the LLM output.
func buildValidatedPermissions(specPerms []models.AgentSpecPermission, schema []models.TableSchema) []*nb.AgentPermission {
	known := make(map[string]bool, len(schema))
	for _, t := range schema {
		known[t.Slug] = true
	}

	seen := make(map[string]bool, len(specPerms))
	perms := make([]*nb.AgentPermission, 0, len(specPerms))
	for _, sp := range specPerms {
		slug := strings.TrimSpace(sp.TableSlug)
		if slug == "" || !known[slug] || seen[slug] {
			continue
		}
		if !sp.CanCreate && !sp.CanRead && !sp.CanUpdate && !sp.CanDelete && !sp.CanList {
			continue
		}
		seen[slug] = true
		perms = append(perms, &nb.AgentPermission{
			TableSlug: slug,
			CanCreate: sp.CanCreate,
			CanRead:   sp.CanRead,
			CanUpdate: sp.CanUpdate,
			CanDelete: sp.CanDelete,
			CanList:   sp.CanList,
		})
	}
	return perms
}

// formatAgentSummary renders a short markdown summary of the created agent for the
// builder: its name, description, and the per-table operations it was granted.
func formatAgentSummary(agent *nb.Agent) string {
	var b strings.Builder

	b.WriteString("**")
	b.WriteString(agent.GetName())
	b.WriteString("**")
	if d := strings.TrimSpace(agent.GetDescription()); d != "" {
		b.WriteString(" — ")
		b.WriteString(d)
	}

	perms := agent.GetPermissions()
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].GetTableSlug() < perms[j].GetTableSlug()
	})

	b.WriteString("\n\nДоступ к данным:\n")
	for _, perm := range perms {
		fmt.Fprintf(&b, "- %s: %s\n", perm.GetTableSlug(), strings.Join(allowedOps(perm), ", "))
	}

	return b.String()
}