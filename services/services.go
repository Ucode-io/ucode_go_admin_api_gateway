package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
)

type ServiceManagerI interface {
	BuilderService() BuilderServiceI
	AuthService() AuthServiceI
	CompanyService() CompanyServiceI
	AnalyticsService() AnalyticsServiceI
	ApiReferenceService() ApiReferenceServiceI
	SmsService() SmsServiceI
	PosService() PosServiceI
}

type grpcClients struct {
	builderService      BuilderServiceI
	authService         AuthServiceI
	companyService      CompanyServiceI
	analyticsService    AnalyticsServiceI
	apiReferenceService ApiReferenceServiceI
	smsService          SmsServiceI
	posService          PosServiceI
}

func NewGrpcClients(ctx context.Context, cfg config.Config) (ServiceManagerI, error) {
	builderServiceClient, err := NewBuilderServiceClient(ctx, cfg)
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

	analyticsServiceClient, err := NewAnalyticsServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	apiReferenceClient, err := NewApiReferenceServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	smsServiceClient, err := NewSmsServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	posServiceClient, err := NewPosServiceClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return grpcClients{
		apiReferenceService: apiReferenceClient,
		analyticsService:    analyticsServiceClient,
		authService:         authServiceClient,
		builderService:      builderServiceClient,
		posService:          posServiceClient,
		smsService:          smsServiceClient,
		companyService:      companyServiceClient,
	}, nil
}

func (g grpcClients) BuilderService() BuilderServiceI {
	return g.builderService
}

func (g grpcClients) AuthService() AuthServiceI {
	return g.authService
}

func (g grpcClients) CompanyService() CompanyServiceI {
	return g.companyService
}

func (g grpcClients) AnalyticsService() AnalyticsServiceI {
	return g.analyticsService
}

func (g grpcClients) ApiReferenceService() ApiReferenceServiceI {
	return g.apiReferenceService
}

func (g grpcClients) SmsService() SmsServiceI {
	return g.smsService
}

func (g grpcClients) PosService() PosServiceI {
	return g.posService
}
