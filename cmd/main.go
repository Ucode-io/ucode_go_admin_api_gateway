package main

import (
	"context"
	"time"
	"ucode/ucode_go_api_gateway/api"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/crons"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage/redis"

	"github.com/gin-gonic/gin"
)

func main() {

	baseConf := config.BaseLoad()

	var loggerLevel = new(string)
	*loggerLevel = logger.LevelDebug

	switch baseConf.Environment {
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

	// auth connection
	authSrvc, err := services.NewAuthGrpcClient(ctx, baseConf)
	if err != nil {
		log.Error("[ucode] error while establishing auth grpc conn", logger.Error(err))
		return
	}

	// company connection
	compSrvc, err := services.NewCompanyServiceClient(ctx, baseConf)
	if err != nil {
		log.Error("[ucode] error while establishing company grpc conn", logger.Error(err))
		return
	}

	serviceNodes := services.NewServiceNodes()
	// u-code grpc services
	uConf := config.Load()
	grpcSvcs, err := services.NewGrpcClients(ctx, uConf)
	if err != nil {
		log.Error("Error adding grpc client with base config. NewGrpcClients", logger.Error(err))
		return
	}

	err = serviceNodes.Add(grpcSvcs, baseConf.UcodeNamespace)
	if err != nil {
		log.Error("Error adding grpc client to serviceNode. ServiceNode", logger.Error(err))
		return
	}
	log.Info(" --- U-code services --- added to serviceNodes")

	// pooling grpc services of enterprice projects
	projectServiceNodes, mapProjectConfs := helper.EnterPriceProjectsGrpcSvcs(ctx, compSrvc, serviceNodes, log)
	mapProjectConfs[baseConf.UcodeNamespace] = uConf

	newRedis := redis.NewRedis(mapProjectConfs)

	r := gin.New()

	r.Use(gin.Logger(), gin.Recovery())

	h := handlers.NewHandler(baseConf, mapProjectConfs, log, projectServiceNodes, compSrvc, authSrvc, newRedis)

	api.SetUpAPI(r, h, baseConf)
	cronjobs := crons.ExecuteCron()
	for _, cronjob := range cronjobs {
		go func(ctx context.Context, cronjob crons.Cronjob) {
			for {
				select {
				case <-time.After(cronjob.Interval):
					// err := cronjob.Function(ctx, grpcSvcs, baseConf, compSrvc)
					// if err != nil {
					// }
				case <-ctx.Done():
					return
				}
			}
		}(ctx, cronjob)
	}

	log.Info("server is running...")
	if err := r.Run(baseConf.HTTPPort); err != nil {
		return
	}
}
