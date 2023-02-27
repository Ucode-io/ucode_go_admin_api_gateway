package services

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"ucode/ucode_go_api_gateway/config"
	tmp "ucode/ucode_go_api_gateway/genproto/query_service"
)

type QueryServiceI interface {
	Query() tmp.QueryServiceClient
	Folder() tmp.FolderServiceClient
	Log() tmp.LogServiceClient
}

type QueryServiceClient struct {
	queryService  tmp.QueryServiceClient
	folderService tmp.FolderServiceClient
	logService    tmp.LogServiceClient
}

func NewQueryServiceClient(ctx context.Context, cfg config.Config) (QueryServiceI, error) {

	connQueryService, err := grpc.DialContext(
		ctx,
		cfg.QueryServiceHost+cfg.QueryServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &QueryServiceClient{
		queryService:  tmp.NewQueryServiceClient(connQueryService),
		folderService: tmp.NewFolderServiceClient(connQueryService),
		logService:    tmp.NewLogServiceClient(connQueryService),
	}, nil
}

func (g *QueryServiceClient) Query() tmp.QueryServiceClient {
	return g.queryService
}

func (g *QueryServiceClient) Folder() tmp.FolderServiceClient {
	return g.folderService
}

func (g *QueryServiceClient) Log() tmp.LogServiceClient {
	return g.logService
}
