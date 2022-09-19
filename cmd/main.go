package main

import (
	"ucode/ucode_go_admin_api_gateway/api"
	"ucode/ucode_go_admin_api_gateway/api/handlers"
	"ucode/ucode_go_admin_api_gateway/config"
	"ucode/ucode_go_admin_api_gateway/pkg/logger"
	"ucode/ucode_go_admin_api_gateway/services"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	grpcSvcs, err := services.NewGrpcClients(cfg)
	if err != nil {
		panic(err)
	}

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

	log := logger.NewLogger("ucode_go_admin_api_gateway", *loggerLevel)
	defer func() {
		err := logger.Cleanup(log)
		if err != nil {
			return
		}
	}()

	r := gin.New()

	r.Use(gin.Logger(), gin.Recovery())

	h := handlers.NewHandler(cfg, log, grpcSvcs)

	api.SetUpAPI(r, h, cfg)

	if err := r.Run(cfg.HTTPPort); err != nil {
		return
	}
}
