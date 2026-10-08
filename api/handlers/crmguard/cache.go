package crmguard

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/storage"
)

type scopedCache struct {
	storage.RedisStorageI
	cfg config.CRMNativeConfig
}

func Cache(base storage.RedisStorageI, cfg config.CRMNativeConfig) storage.RedisStorageI {
	if !cfg.Enabled {
		return base
	}
	return scopedCache{RedisStorageI: base, cfg: cfg}
}
func (s scopedCache) Get(ctx context.Context, key, project, node string) (string, error) {
	decoded := key
	if bytes, err := base64.StdEncoding.DecodeString(key); err == nil {
		decoded = string(bytes)
	}
	// Legacy handlers sometimes drop context. Native data cache keys contain the
	// resource/environment UUID; this prevents serving old actorless entries.
	if Scoped(ctx) || (project == s.cfg.Project && s.cfg.ResourceEnvironment != "" && strings.Contains(decoded, s.cfg.ResourceEnvironment)) {
		return "", errors.New("scoped cache read disabled")
	}
	return s.RedisStorageI.Get(ctx, key, project, node)
}
