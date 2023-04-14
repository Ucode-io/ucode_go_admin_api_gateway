package services

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"ucode/ucode_go_api_gateway/config"
	tmp "ucode/ucode_go_api_gateway/genproto/web_page_service"
)

type WebPageServiceI interface {
	Folder() tmp.FolderServiceClient
	WebPage() tmp.WebPageServiceClient
	App() tmp.AppServiceClient
}

type WebPageServiceClient struct {
	folderService  tmp.FolderServiceClient
	webPageService tmp.WebPageServiceClient
	appService     tmp.AppServiceClient
}

func NewWebPageServiceClient(ctx context.Context, cfg config.Config) (WebPageServiceI, error) {

	connWebPageService, err := grpc.DialContext(
		ctx,
		cfg.WebPageServiceHost+cfg.WebPageServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &WebPageServiceClient{
		folderService:  tmp.NewFolderServiceClient(connWebPageService),
		webPageService: tmp.NewWebPageServiceClient(connWebPageService),
		appService:     tmp.NewAppServiceClient(connWebPageService),
	}, nil
}

func (g *WebPageServiceClient) Folder() tmp.FolderServiceClient {
	return g.folderService
}

func (g *WebPageServiceClient) WebPage() tmp.WebPageServiceClient {
	return g.webPageService
}

func (g *WebPageServiceClient) App() tmp.AppServiceClient {
	return g.appService
}
