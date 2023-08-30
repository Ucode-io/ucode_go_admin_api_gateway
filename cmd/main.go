package main

import (
	"context"
	"time"
	"ucode/ucode_go_api_gateway/api"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/crons"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage/redis"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	var loggerLevel = new(string)
	*loggerLevel = logger.LevelDebug

	switch cfg.Environment {
	case config.DebugMode:
		*loggerLevel = logger.LevelDebug
		gin.SetMode(gin.DebugMode)
	case config.TestMode:
		*loggerLevel = logger.LevelDebug
		gin.SetMode(gin.TestMode)
	default:
		*loggerLevel = logger.LevelInfo
		gin.SetMode(gin.ReleaseMode)
	}

	log := logger.NewLogger("ucode/ucode_go_api_gateway", *loggerLevel)
	defer func() {
		err := logger.Cleanup(log)
		if err != nil {
			return
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	newRedis := redis.NewRedis(cfg)

	authSrvc, err := services.NewAuthGrpcClient(ctx, cfg)
	if err != nil {
		log.Error("[ucode] error while establishing auth grpc conn", logger.Error(err))
		return
	}

	grpcSvcs, err := services.NewGrpcClients(ctx, cfg)
	if err != nil {
		log.Error("[ucode] error while establishing grpc conn", logger.Error(err))
		return
	}

	serviceNodes := services.NewServiceNodes()
	serviceNodes.Add(grpcSvcs, cfg.UcodeNamespace)

	r := gin.New()

	r.Use(gin.Logger(), gin.Recovery())

	h := handlers.NewHandler(cfg, log, serviceNodes, grpcSvcs, authSrvc, newRedis)

	api.SetUpAPI(r, h, cfg)
	cronjobs := crons.ExecuteCron()
	for _, cronjob := range cronjobs {
		go func(ctx context.Context, cronjob crons.Cronjob) {
			for {
				// tc := time.NewTicker(cronjob.Interval)
				select {
				case <-time.After(cronjob.Interval):
					err := cronjob.Function(ctx, grpcSvcs, cfg)
					if err != nil {
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx, cronjob)

	}

	log.Info("server is running...")
	if err := r.Run(cfg.HTTPPort); err != nil {
		return
	}
}
