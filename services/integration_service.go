package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	integration_service "ucode/ucode_go_api_gateway/genproto/integration_service_v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type IntegrationServiceI interface {
	TransactionService() integration_service.TransactionServiceClient
	PayzeService() integration_service.PayzeServiceClient
}

type integrationServiceClient struct {
	transactionServiceClient integration_service.TransactionServiceClient
	payzeServiceClient       integration_service.PayzeServiceClient
}

func NewIntegrationServiceClient(ctx context.Context, cfg config.Config) (IntegrationServiceI, error) {

	connIntegrationService, err := grpc.DialContext(
		ctx,
		cfg.IntegrationServiceHost+cfg.IntegrationGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &integrationServiceClient{
		transactionServiceClient: integration_service.NewTransactionServiceClient(connIntegrationService),
		payzeServiceClient:       integration_service.NewPayzeServiceClient(connIntegrationService),
	}, nil
}

func (g *integrationServiceClient) TransactionService() integration_service.TransactionServiceClient {
	return g.transactionServiceClient
}

func (g *integrationServiceClient) PayzeService() integration_service.PayzeServiceClient {
	return g.payzeServiceClient
}
