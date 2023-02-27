package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/integration_service_v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type IntegrationServicePayzeI interface {
	IntegrationPayzeService() integration_service_v2.PayzeServiceClient
}

type PayzeServiceClient struct {
	integrationPayzeService integration_service_v2.PayzeServiceClient
}

func NewPayzeServiceClient(ctx context.Context, cfg config.Config) (IntegrationServicePayzeI, error) {

	connIntegrationPayzeService, err := grpc.DialContext(
		ctx,
		cfg.IntegrationPayzeServiceHost+cfg.IntegrationPayzeServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &PayzeServiceClient{
		integrationPayzeService: integration_service_v2.NewPayzeServiceClient(connIntegrationPayzeService),
	}, nil
}

func (g *PayzeServiceClient) IntegrationPayzeService() integration_service_v2.PayzeServiceClient {
	return g.integrationPayzeService
}
