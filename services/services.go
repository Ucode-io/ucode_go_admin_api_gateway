package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
)

type ServiceManagerI interface {
	BuilderService() BuilderServiceI
	HighBuilderService() BuilderServiceI
	AuthService() AuthServiceI
	CompanyService() CompanyServiceI
	SmsService() SmsServiceI
	FunctionService() FunctionServiceI
	TemplateService() TemplateServiceI
	GetBuilderServiceByType(nodeType string) BuilderServiceI
	GoObjectBuilderService() GoBuilderServiceI
}

type grpcClients struct {
	builderService         BuilderServiceI
	highBuilderService     BuilderServiceI
	authService            AuthServiceI
	companyService         CompanyServiceI
	smsService             SmsServiceI
	functionService        FunctionServiceI
	templateService        TemplateServiceI
	goObjectBuilderService GoBuilderServiceI
}

func NewGrpcClients(ctx context.Context, cfg config.Config) (ServiceManagerI, error) {
	builderServiceClient, err := NewBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	highBuilderServiceClient, err := NewHighBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	authServiceClient, err := NewAuthServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	companyServiceClient, err := NewCompanyServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	smsServiceClient, err := NewSmsServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	functionServiceClient, err := NewFunctionServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	templateServiceClient, err := NewTemplateServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	goObjectBuilderServiceClient, err := NewGoBuilderServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return grpcClients{
		authService:            authServiceClient,
		builderService:         builderServiceClient,
		highBuilderService:     highBuilderServiceClient,
		smsService:             smsServiceClient,
		companyService:         companyServiceClient,
		functionService:        functionServiceClient,
		templateService:        templateServiceClient,
		goObjectBuilderService: goObjectBuilderServiceClient,
	}, nil
}

func (g grpcClients) GetBuilderServiceByType(nodeType string) BuilderServiceI {
	switch nodeType {
	case config.LOW_NODE_TYPE:
		return g.builderService
	case config.HIGH_NODE_TYPE:
		return g.highBuilderService
	}

	return g.builderService
}

func (g grpcClients) BuilderService() BuilderServiceI {
	return g.builderService
}

func (g grpcClients) GoObjectBuilderService() GoBuilderServiceI {
	return g.goObjectBuilderService
}

func (g grpcClients) HighBuilderService() BuilderServiceI {
	return g.highBuilderService
}

func (g grpcClients) AuthService() AuthServiceI {
	return g.authService
}

func (g grpcClients) CompanyService() CompanyServiceI {
	return g.companyService
}

func (g grpcClients) SmsService() SmsServiceI {
	return g.smsService
}

func (g grpcClients) FunctionService() FunctionServiceI {
	return g.functionService
}
func (g grpcClients) TemplateService() TemplateServiceI {
	return g.templateService
}
