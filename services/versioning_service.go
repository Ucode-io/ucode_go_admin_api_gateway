package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/versioning_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type VersioningServiceI interface {
	Commit() versioning_service.CommitServiceClient
	Release() versioning_service.ReleaseServiceClient
}

type versioningServiceClient struct {
	commitService  versioning_service.CommitServiceClient
	releaseService versioning_service.ReleaseServiceClient
}

func NewVersioningServiceClient(ctx context.Context, cfg config.Config) (VersioningServiceI, error) {

	connVersioningService, err := grpc.DialContext(
		ctx,
		cfg.VersioningServiceHost+cfg.VersioningGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &versioningServiceClient{
		commitService:  versioning_service.NewCommitServiceClient(connVersioningService),
		releaseService: versioning_service.NewReleaseServiceClient(connVersioningService),
	}, nil
}

func (g *versioningServiceClient) Commit() versioning_service.CommitServiceClient {
	return g.commitService
}

func (g *versioningServiceClient) Release() versioning_service.ReleaseServiceClient {
	return g.releaseService
}
