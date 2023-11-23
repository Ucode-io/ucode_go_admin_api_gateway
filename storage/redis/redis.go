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
	pool map[string]*redis.Client
}

func NewRedis(cfg map[string]config.Config) storage.RedisStorageI {
	redisPool := make(map[string]*redis.Client)

	for k, v := range cfg {
		redisPool[k] = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", v.GetRequestRedisHost, v.GetRequestRedisPort),
			Password: v.GetRequestRedisPassword,
			DB:       v.GetRequestRedisDatabase,
		})
	}

	return Storage{
		pool: redisPool,
	}
}

func (s Storage) SetX(ctx context.Context, key string, value string, duration time.Duration, projectId string) error {
	return s.pool[projectId].SetEX(ctx, key, value, duration).Err()
}

func (s Storage) Get(ctx context.Context, key string, projectId string) (string, error) {
	return s.pool[projectId].Get(ctx, key).Result()
}
