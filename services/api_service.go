package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/api_reference_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ApiReferenceServiceI interface {
	ApiReference() api_reference_service.ApiReferenceServiceClient
	Category() api_reference_service.CategoryServiceClient
}

type apiReferenceClient struct {
	apiReferenceService api_reference_service.ApiReferenceServiceClient
	categoryService     api_reference_service.CategoryServiceClient
}

func NewApiReferenceServiceClient(ctx context.Context, cfg config.Config) (ApiReferenceServiceI, error) {

	connApiReferenceService, err := grpc.DialContext(
		ctx,
		cfg.ApiReferenceServiceHost+cfg.ApiReferenceServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	return &apiReferenceClient{
		apiReferenceService: api_reference_service.NewApiReferenceServiceClient(connApiReferenceService),
		categoryService:     api_reference_service.NewCategoryServiceClient(connApiReferenceService),
	}, nil
}

func (g *apiReferenceClient) ApiReference() api_reference_service.ApiReferenceServiceClient {
	return g.apiReferenceService
}

func (g *apiReferenceClient) Category() api_reference_service.CategoryServiceClient {
	return g.categoryService
}
