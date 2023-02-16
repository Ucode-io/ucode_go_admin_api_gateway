package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	tmp "ucode/ucode_go_api_gateway/genproto/template_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TemplateServiceI interface {
	Template() tmp.TemplateFolderServiceClient
	Note() tmp.NoteFolderServiceClient
	Share() tmp.ShareServiceClient
	UserPermission() tmp.UserPermissionServiceClient
}

type templateServiceClient struct {
	templateService tmp.TemplateFolderServiceClient
	noteService     tmp.NoteFolderServiceClient
	shareService    tmp.ShareServiceClient
	userPermission  tmp.UserPermissionServiceClient
}

func NewTemplateServiceClient(ctx context.Context, cfg config.Config) (TemplateServiceI, error) {

	connTemplateService, err := grpc.DialContext(
		ctx,
		cfg.TemplateServiceHost+cfg.TemplateGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &templateServiceClient{
		templateService: tmp.NewTemplateFolderServiceClient(connTemplateService),
		noteService:     tmp.NewNoteFolderServiceClient(connTemplateService),
		shareService:    tmp.NewShareServiceClient(connTemplateService),
		userPermission:  tmp.NewUserPermissionServiceClient(connTemplateService),
	}, nil
}

func (g *templateServiceClient) Template() tmp.TemplateFolderServiceClient {
	return g.templateService
}

func (g *templateServiceClient) Note() tmp.NoteFolderServiceClient {
	return g.noteService
}

func (g *templateServiceClient) Share() tmp.ShareServiceClient {
	return g.shareService
}

func (g *templateServiceClient) UserPermission() tmp.UserPermissionServiceClient {
	return g.userPermission
}
