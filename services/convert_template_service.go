package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/convert_template"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ConvertTemplateServiceI interface {
	ConvertTemplateService() convert_template.ConvertTemplateServiceClient
	PingTemplateService()            convert_template.ConvertPingServiceClient
}

type convertTemplateServiceClient struct {
	convertTemplateService convert_template.ConvertTemplateServiceClient
	pingTemplateService    convert_template.ConvertPingServiceClient
}

func NewConvertTemplateServiceClient(ctx context.Context, cfg config.Config) (ConvertTemplateServiceI, error) {

	connConvertTemplateService, err := grpc.DialContext(
		ctx,
		cfg.ConvertTemplateServiceGrpcHost+cfg.ConvertTemplateServiceGrpcPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	return &convertTemplateServiceClient{
		convertTemplateService: convert_template.NewConvertTemplateServiceClient(connConvertTemplateService),
		pingTemplateService:    convert_template.NewConvertPingServiceClient(connConvertTemplateService),
	}, nil
}

func (g *convertTemplateServiceClient) ConvertTemplateService() convert_template.ConvertTemplateServiceClient {
	return g.convertTemplateService
}

func (g *convertTemplateServiceClient) PingTemplateService() convert_template.ConvertPingServiceClient {
	return g.pingTemplateService
}