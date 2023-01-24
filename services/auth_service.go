package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"

	"ucode/ucode_go_api_gateway/genproto/auth_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthServiceManagerI interface {
	ClientService() auth_service.ClientServiceClient
	SessionService() auth_service.SessionServiceClient
	IntegrationService() auth_service.IntegrationServiceClient
	ClientServiceAuth() auth_service.ClientServiceClient
	PermissionServiceAuth() auth_service.PermissionServiceClient
	UserService() auth_service.UserServiceClient
	SessionServiceAuth() auth_service.SessionServiceClient
	EmailServie() auth_service.EmailOtpServiceClient
	CompanyService() auth_service.CompanyServiceClient
	ApiKeyService() auth_service.ApiKeysClient
}

type authGrpcClients struct {
	clientService         auth_service.ClientServiceClient
	sessionService        auth_service.SessionServiceClient
	integrationService    auth_service.IntegrationServiceClient
	clientServiceAuth     auth_service.ClientServiceClient
	permissionServiceAuth auth_service.PermissionServiceClient
	userService           auth_service.UserServiceClient
	sessionServiceAuth    auth_service.SessionServiceClient
	emailServie           auth_service.EmailOtpServiceClient
	authCompanyService    auth_service.CompanyServiceClient
	apikeyService         auth_service.ApiKeysClient
}

func NewAuthGrpcClient(ctx context.Context, cfg config.Config) (AuthServiceManagerI, error) {

	connAuthService, err := grpc.DialContext(
		ctx,
		cfg.AuthServiceHost+cfg.AuthGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &authGrpcClients{
		clientService:         auth_service.NewClientServiceClient(connAuthService),
		sessionService:        auth_service.NewSessionServiceClient(connAuthService),
		clientServiceAuth:     auth_service.NewClientServiceClient(connAuthService),
		permissionServiceAuth: auth_service.NewPermissionServiceClient(connAuthService),
		userService:           auth_service.NewUserServiceClient(connAuthService),
		sessionServiceAuth:    auth_service.NewSessionServiceClient(connAuthService),
		integrationService:    auth_service.NewIntegrationServiceClient(connAuthService),
		emailServie:           auth_service.NewEmailOtpServiceClient(connAuthService),
		authCompanyService:    auth_service.NewCompanyServiceClient(connAuthService),
		apikeyService:         auth_service.NewApiKeysClient(connAuthService),
	}, nil
}

func (g *authGrpcClients) ClientService() auth_service.ClientServiceClient {
	return g.clientService
}

func (g *authGrpcClients) SessionService() auth_service.SessionServiceClient {
	return g.sessionService
}

// auth functions

func (g *authGrpcClients) ClientServiceAuth() auth_service.ClientServiceClient {
	return g.clientServiceAuth
}

func (g *authGrpcClients) PermissionServiceAuth() auth_service.PermissionServiceClient {
	return g.permissionServiceAuth
}

func (g *authGrpcClients) UserService() auth_service.UserServiceClient {
	return g.userService
}

func (g *authGrpcClients) SessionServiceAuth() auth_service.SessionServiceClient {
	return g.sessionServiceAuth
}

func (g *authGrpcClients) IntegrationService() auth_service.IntegrationServiceClient {
	return g.integrationService
}

func (g *authGrpcClients) EmailServie() auth_service.EmailOtpServiceClient {
	return g.emailServie
}

func (g *authGrpcClients) CompanyService() auth_service.CompanyServiceClient {
	return g.authCompanyService
}

func (g *authGrpcClients) ApiKeyService() auth_service.ApiKeysClient {
	return g.apikeyService
}
