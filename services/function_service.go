package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/function_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FunctionServiceI interface {
	FunctionService() function_service.FunctionServiceClient
	CustomEventService() function_service.CustomEventServiceClient
}

type functionServiceClient struct {
	functionService function_service.FunctionServiceClient
	categoryService function_service.CustomEventServiceClient
}

func NewFunctionServiceClient(ctx context.Context, cfg config.Config) (FunctionServiceI, error) {

	connFunctionService, err := grpc.DialContext(
		ctx,
		cfg.FunctionServiceHost+cfg.FunctionServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	return &functionServiceClient{
		functionService: function_service.NewFunctionServiceClient(connFunctionService),
		categoryService: function_service.NewCustomEventServiceClient(connFunctionService),
	}, nil
}

func (g *functionServiceClient) FunctionService() function_service.FunctionServiceClient {
	return g.functionService
}

func (g *functionServiceClient) CustomEventService() function_service.CustomEventServiceClient {
	return g.categoryService
}
