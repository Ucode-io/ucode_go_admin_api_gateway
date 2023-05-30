package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CompanyServiceI interface {
	Company() company_service.CompanyServiceClient
	Project() company_service.ProjectServiceClient
	Environment() company_service.EnvironmentServiceClient
	Resource() company_service.ResourceServiceClient
	ServiceResource() company_service.MicroserviceResourceClient
	Redirect() company_service.RedirectUrlServiceClient
}

type companyServiceClient struct {
	companyService     company_service.CompanyServiceClient
	projectService     company_service.ProjectServiceClient
	environmentService company_service.EnvironmentServiceClient
	resourceService    company_service.ResourceServiceClient
	serviceResource    company_service.MicroserviceResourceClient
	redirectService    company_service.RedirectUrlServiceClient
}

func NewCompanyServiceClient(ctx context.Context, cfg config.Config) (CompanyServiceI, error) {

	connCompanyService, err := grpc.DialContext(
		ctx,
		cfg.CompanyServiceHost+cfg.CompanyServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &companyServiceClient{
		companyService:     company_service.NewCompanyServiceClient(connCompanyService),
		projectService:     company_service.NewProjectServiceClient(connCompanyService),
		environmentService: company_service.NewEnvironmentServiceClient(connCompanyService),
		resourceService:    company_service.NewResourceServiceClient(connCompanyService),
		serviceResource:    company_service.NewMicroserviceResourceClient(connCompanyService),
		redirectService:    company_service.NewRedirectUrlServiceClient(connCompanyService),
	}, nil
}

func (g *companyServiceClient) Company() company_service.CompanyServiceClient {
	return g.companyService
}

func (g *companyServiceClient) Project() company_service.ProjectServiceClient {
	return g.projectService
}

func (g *companyServiceClient) Environment() company_service.EnvironmentServiceClient {
	return g.environmentService
}

func (g *companyServiceClient) Resource() company_service.ResourceServiceClient {
	return g.resourceService
}

func (g *companyServiceClient) ServiceResource() company_service.MicroserviceResourceClient {
	return g.serviceResource
}

func (g *companyServiceClient) Redirect() company_service.RedirectUrlServiceClient {
	return g.redirectService
}
