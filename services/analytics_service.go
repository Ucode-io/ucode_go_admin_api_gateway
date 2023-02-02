package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/analytics_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AnalyticsServiceI interface {
	Query() analytics_service.QueryServiceClient
}

type analyticsServiceClient struct {
	queryService analytics_service.QueryServiceClient
}

func NewAnalyticsServiceClient(ctx context.Context, cfg config.Config) (AnalyticsServiceI, error) {

	connAnalyticsService, err := grpc.DialContext(
		ctx,
		cfg.AnalyticsServiceHost+cfg.AnalyticsGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &analyticsServiceClient{
		queryService: analytics_service.NewQueryServiceClient(connAnalyticsService),
	}, nil
}

func (g *analyticsServiceClient) Query() analytics_service.QueryServiceClient {
	return g.queryService
}
