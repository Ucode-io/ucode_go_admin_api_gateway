package redis

import (
	"context"
	"fmt"
	"time"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/go-redis/redis/v8"
)

type Storage struct {
	db *redis.Client
}

func NewRedis(cfg config.Config) storage.RedisStorageI {
	return Storage{
		db: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.GetRequestRedisHost, cfg.GetRequestRedisPort),
			Password: cfg.GetRequestRedisPassword,
			DB:       cfg.GetRequestRedisDatabase,
		}),
	}
}

func (s Storage) SetX(ctx context.Context, key string, value string, duration time.Duration) error {
	return s.db.SetEX(ctx, key, value, duration).Err()
}

func (s Storage) Get(ctx context.Context, key string) (string, error) {
	return s.db.Get(ctx, key).Result()
}
