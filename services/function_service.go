package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	function_service "ucode/ucode_go_api_gateway/genproto/new_function_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FunctionServiceI interface {
	FunctionService() function_service.FunctionServiceV2Client
	CustomEventService() function_service.CustomEventServiceV2Client
	FunctionFolderService() function_service.FunctionFolderServiceClient
}

type functionServiceClient struct {
	functionService       function_service.FunctionServiceV2Client
	customEventService    function_service.CustomEventServiceV2Client
	functionFolderService function_service.FunctionFolderServiceClient
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
		functionService:       function_service.NewFunctionServiceV2Client(connFunctionService),
		customEventService:    function_service.NewCustomEventServiceV2Client(connFunctionService),
		functionFolderService: function_service.NewFunctionFolderServiceClient(connFunctionService),
	}, nil
}

func (g *functionServiceClient) FunctionService() function_service.FunctionServiceV2Client {
	return g.functionService
}

func (g *functionServiceClient) CustomEventService() function_service.CustomEventServiceV2Client {
	return g.customEventService
}
func (g *functionServiceClient) FunctionFolderService() function_service.FunctionFolderServiceClient {
	return g.functionFolderService
}
