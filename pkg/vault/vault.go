package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
)

type VaultClient interface {
	Get(ctx context.Context, secretPath string) (map[string]any, error)
	Put(ctx context.Context, secretPath string, data map[string]any) error
}

type Config struct {
	Address   string
	RoleID    string
	SecretID  string
	MountPath string
}

type vaultClient struct {
	client    *vaultapi.Client
	roleID    string
	secretID  string
	mountPath string
}

func New(ctx context.Context, cfg Config) (VaultClient, error) {
	if cfg.Address == "" {
		return nil, errors.New("vault address is required")
	}

	apiCfg := vaultapi.DefaultConfig()
	apiCfg.Address = strings.TrimRight(cfg.Address, "/")

	client, err := vaultapi.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize vault client: %w", err)
	}

	vc := &vaultClient{
		client:    client,
		roleID:    cfg.RoleID,
		secretID:  cfg.SecretID,
		mountPath: cfg.MountPath,
	}

	if err := vc.login(ctx); err != nil {
		return nil, fmt.Errorf("vault login failed: %w", err)
	}

	return vc, nil
}

func (v *vaultClient) login(ctx context.Context) error {
	appRoleAuth, err := auth.NewAppRoleAuth(
		v.roleID,
		&auth.SecretID{FromString: v.secretID},
		auth.WithMountPath(v.mountPath),
	)
	if err != nil {
		return err
	}

	authInfo, err := v.client.Auth().Login(ctx, appRoleAuth)
	if err != nil {
		return err
	}
	if authInfo == nil {
		return errors.New("no auth info returned from vault")
	}

	return nil
}

func (v *vaultClient) Get(ctx context.Context, secretPath string) (map[string]any, error) {
	secret, err := v.client.KVv2("secret").Get(ctx, secretPath)
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (v *vaultClient) Put(ctx context.Context, secretPath string, data map[string]any) error {
	_, err := v.client.KVv2("secret").Put(ctx, secretPath, data)
	return err
}
