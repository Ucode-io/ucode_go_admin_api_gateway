package main

import (
	"context"
	"fmt"
	"time"

	"ucode/ucode_go_api_gateway/api"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/api/handlers/api_call_limits"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/pkg/vault"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage/redis"

	"github.com/gin-gonic/gin"
	go_redis "github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaeger_config "github.com/uber/jaeger-client-go/config"
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

	jaegerCfg := &jaeger_config.Configuration{
		ServiceName: baseConf.ServiceName,
		Sampler: &jaeger_config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaeger_config.ReporterConfig{
			LogSpans:           false,
			LocalAgentHostPort: baseConf.JaegerHostPort,
		},
	}

	log := logger.NewLogger("ucode/ucode_go_api_gateway", *loggerLevel)
	defer func() {
		err := logger.Cleanup(log)
		if err != nil {
			return
		}
	}()

	tracer, closer, err := jaegerCfg.NewTracer(jaeger_config.Logger(jaeger.StdLogger))
	if err != nil {
		log.Error("ERROR: cannot init Jaeger", logger.Error(err))
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uConf := config.Load()

	// auth connection
	authSrvc, err := services.NewAuthGrpcClient(ctx, baseConf)
	if err != nil {
		log.Error("[ucode] error while establishing auth grpc conn-", logger.Error(err))
		return
	}

	// company connection
	compSrvc, err := services.NewCompanyServiceClient(ctx, uConf)
	if err != nil {
		log.Error("[ucode] error while establishing company grpc conn", logger.Error(err))
		return
	}

	serviceNodes := services.NewServiceNodes()
	// u-code grpc services
	grpcSvcs, err := services.NewGrpcClients(ctx, uConf)
	if err != nil {
		log.Error("Error adding grpc client with base config. NewGrpcClients", logger.Error(err))
		return
	}

	err = serviceNodes.Add(grpcSvcs, baseConf.UcodeNamespace)
	if err != nil {
		log.Error("Error adding grpc client to serviceNode. ServiceNode!!", logger.Error(err))
		return
	}

	log.Info(" --- U-code services --- added to serviceNodes")

	// pooling grpc services of enterprice projects
	projectServiceNodes, mapProjectConfs := helper.EnterPriceProjectsGrpcSvcs(ctx, compSrvc, serviceNodes, log)

	if projectServiceNodes == nil {
		projectServiceNodes = serviceNodes
	}

	if mapProjectConfs == nil {
		mapProjectConfs = make(map[string]config.Config)
	}

	mapProjectConfs[baseConf.UcodeNamespace] = uConf

	newRedis := redis.NewRedis(mapProjectConfs, log)

	centralRedis := go_redis.NewClient(&go_redis.Options{
		Addr:     fmt.Sprintf("%s:%s", uConf.GetRequestRedisHost, uConf.GetRequestRedisPort),
		Password: uConf.GetRequestRedisPassword,
		DB:       uConf.GetRequestRedisDatabase,
	})

	err = centralRedis.Ping(ctx).Err()
	if err != nil {
		log.Error("error connecting to central redis", logger.Error(err), logger.String("host", uConf.GetRequestRedisHost), logger.String("port", uConf.GetRequestRedisPort))
	} else {
		log.Info("successfully connected to central redis", logger.String("host", uConf.GetRequestRedisHost), logger.String("port", uConf.GetRequestRedisPort))
	}

	trackerCfg := api_call_limits.LoadTrackerConfig(uConf)
	if trackerCfg.MetricsFlushInterval == 0 {
		trackerCfg.MetricsFlushInterval = 10 * time.Second
	}
	tracker := api_call_limits.NewTracker(centralRedis, trackerCfg)
	go tracker.Start(ctx)

	consumerCfg := api_call_limits.ConsumerConfig{
		DbFlushInterval: 1 * time.Minute,
	}
	consumer := api_call_limits.NewMetricsConsumer(centralRedis, compSrvc, consumerCfg)
	go consumer.Start(ctx)

	cache, err := caching.NewExpiringLRUCache(config.LRU_CACHE_SIZE)
	if err != nil {
		log.Error("Error adding caching.", logger.Error(err))
	}

	r := gin.New()

	r.Use(gin.Logger(), gin.Recovery())

	limiter := util.NewApiKeyRateLimiter(newRedis, config.RATE_LIMITER_RPS_LIMIT, config.RATE_LIMITER_RPS_LIMIT)

	var vaultClient vault.VaultClient
	if baseConf.VaultAddress != "" {
		vaultClient, err = vault.New(ctx, vault.Config{
			Address:   baseConf.VaultAddress,
			RoleID:    baseConf.VaultRoleID,
			SecretID:  baseConf.VaultSecretID,
			MountPath: baseConf.VaultMountPath,
		})
		if err != nil {
			log.Error("[ucode] error while initializing vault client", logger.Error(err))
		}
	}

	h := handlers.NewHandler(baseConf, mapProjectConfs, log, projectServiceNodes, compSrvc, authSrvc, newRedis, cache, limiter, vaultClient)

	api.SetUpAPI(r, h, baseConf, tracer, tracker)

	log.Info("server is running...")
	if err := r.Run(baseConf.HTTPPort); err != nil {
		return
	}
}
