package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/scenario_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ScenarioServiceI interface {
	DagService() scenario_service.DAGServiceClient
	DagStepService() scenario_service.DAGStepServiceClient
	RunService() scenario_service.RunServiceClient
	CategoryService() scenario_service.CategoryServiceClient
}

type scenarioServiceClient struct {
	dagServiceClient      scenario_service.DAGServiceClient
	dagStepServiceClient  scenario_service.DAGStepServiceClient
	runServiceClient      scenario_service.RunServiceClient
	categoryServiceClient scenario_service.CategoryServiceClient
}

func NewScenarioServiceClient(ctx context.Context, cfg config.Config) (ScenarioServiceI, error) {

	connScenarioService, err := grpc.DialContext(
		ctx,
		cfg.ScenarioServiceHost+cfg.ScenarioGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &scenarioServiceClient{
		dagServiceClient:      scenario_service.NewDAGServiceClient(connScenarioService),
		dagStepServiceClient:  scenario_service.NewDAGStepServiceClient(connScenarioService),
		runServiceClient:      scenario_service.NewRunServiceClient(connScenarioService),
		categoryServiceClient: scenario_service.NewCategoryServiceClient(connScenarioService),
	}, nil
}

func (g *scenarioServiceClient) DagService() scenario_service.DAGServiceClient {
	return g.dagServiceClient
}

func (g *scenarioServiceClient) DagStepService() scenario_service.DAGStepServiceClient {
	return g.dagStepServiceClient
}

func (g *scenarioServiceClient) RunService() scenario_service.RunServiceClient {
	return g.runServiceClient
}

func (g *scenarioServiceClient) CategoryService() scenario_service.CategoryServiceClient {
	return g.categoryServiceClient
}
