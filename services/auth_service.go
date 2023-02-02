package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/auth_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthServiceI interface {
	Client() auth_service.ClientServiceClient
	Session() auth_service.SessionServiceClient
	Integration() auth_service.IntegrationServiceClient
	Permission() auth_service.PermissionServiceClient
	User() auth_service.UserServiceClient
	Email() auth_service.EmailOtpServiceClient
	ApiKey() auth_service.ApiKeysClient
}

type authServiceClient struct {
	clientService         auth_service.ClientServiceClient
	sessionService        auth_service.SessionServiceClient
	integrationService    auth_service.IntegrationServiceClient
	clientServiceAuth     auth_service.ClientServiceClient
	permissionServiceAuth auth_service.PermissionServiceClient
	userService           auth_service.UserServiceClient
	sessionServiceAuth    auth_service.SessionServiceClient
	emailServie           auth_service.EmailOtpServiceClient
	apiKeyService         auth_service.ApiKeysClient
}

func NewAuthServiceClient(ctx context.Context, cfg config.Config) (AuthServiceI, error) {

	connAuthService, err := grpc.DialContext(
		ctx,
		cfg.AuthServiceHost+cfg.AuthGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &authServiceClient{
		clientService:         auth_service.NewClientServiceClient(connAuthService),
		sessionService:        auth_service.NewSessionServiceClient(connAuthService),
		clientServiceAuth:     auth_service.NewClientServiceClient(connAuthService),
		permissionServiceAuth: auth_service.NewPermissionServiceClient(connAuthService),
		userService:           auth_service.NewUserServiceClient(connAuthService),
		sessionServiceAuth:    auth_service.NewSessionServiceClient(connAuthService),
		integrationService:    auth_service.NewIntegrationServiceClient(connAuthService),
		emailServie:           auth_service.NewEmailOtpServiceClient(connAuthService),
		apiKeyService:         auth_service.NewApiKeysClient(connAuthService),
	}, nil
}

func (g *authServiceClient) Client() auth_service.ClientServiceClient {
	return g.clientService
}

func (g *authServiceClient) Session() auth_service.SessionServiceClient {
	return g.sessionService
}

func (g *authServiceClient) Permission() auth_service.PermissionServiceClient {
	return g.permissionServiceAuth
}

func (g *authServiceClient) User() auth_service.UserServiceClient {
	return g.userService
}

func (g *authServiceClient) Integration() auth_service.IntegrationServiceClient {
	return g.integrationService
}

func (g *authServiceClient) Email() auth_service.EmailOtpServiceClient {
	return g.emailServie
}

func (g *authServiceClient) ApiKey() auth_service.ApiKeysClient {
	return g.apiKeyService
}
