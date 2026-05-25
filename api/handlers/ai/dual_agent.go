package ai

import (
	"context"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
)

// DualAgent routes AI calls between two sub-agents:
//   - interactive: routing, planning, editing, inspect, database — latency-sensitive
//   - generation: architect, manifest, code generation, repair — quality-sensitive
//
// Used for free-tier users: interactive=Gemini (fast/free), generation=Claude (quality).
type DualAgent struct {
	interactive Agent
	generation  Agent
}

func NewDualAgent(interactive, generation Agent) Agent {
	return &DualAgent{interactive: interactive, generation: generation}
}

func (d *DualAgent) RouteRequest(ctx context.Context, in models.RouterInput) (*models.HaikuRoutingResult, error) {
	return d.interactive.RouteRequest(ctx, in)
}

func (d *DualAgent) PlanChanges(ctx context.Context, in models.PlannerInput) (*models.SonnetPlanResult, error) {
	return d.interactive.PlanChanges(ctx, in)
}

func (d *DualAgent) EditCode(ctx context.Context, in models.EditorInput) (*models.GeneratedProject, error) {
	return d.interactive.EditCode(ctx, in)
}

func (d *DualAgent) InspectCode(ctx context.Context, in models.InspectorInput) (string, error) {
	return d.interactive.InspectCode(ctx, in)
}

func (d *DualAgent) DatabaseQuery(ctx context.Context, in models.DatabaseQueryInput) (*models.DatabaseActionRequest, error) {
	return d.interactive.DatabaseQuery(ctx, in)
}

func (d *DualAgent) VisualEdit(ctx context.Context, in models.VisualEditInput) ([]models.ProjectFile, string, error) {
	return d.interactive.VisualEdit(ctx, in)
}

func (d *DualAgent) ArchitectProject(ctx context.Context, in models.ArchitectInput) (*models.ArchitectPlan, error) {
	return d.generation.ArchitectProject(ctx, in)
}

func (d *DualAgent) GenerateManifest(ctx context.Context, in models.ManifestInput) (*models.ProjectManifest, error) {
	return d.generation.GenerateManifest(ctx, in)
}

func (d *DualAgent) GenerateCode(ctx context.Context, agentCfg config.AgentConfig, systemPrompt, userPrompt string, images []string, history []models.ChatMessage, desc string) (*models.GeneratedProject, error) {
	return d.generation.GenerateCode(ctx, agentCfg, systemPrompt, userPrompt, images, history, desc)
}

func (d *DualAgent) GenerateCodeNoHistory(ctx context.Context, agentCfg config.AgentConfig, systemPrompt, userPrompt, desc string) (*models.GeneratedProject, error) {
	return d.generation.GenerateCodeNoHistory(ctx, agentCfg, systemPrompt, userPrompt, desc)
}

func (d *DualAgent) RepairFile(ctx context.Context, in models.RepairFileInput) (models.ProjectFile, error) {
	return d.generation.RepairFile(ctx, in)
}
