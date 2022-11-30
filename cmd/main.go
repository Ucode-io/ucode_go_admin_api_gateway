package main

import (
	"context"
	"fmt"
	"sync"
	"time"
	"ucode/ucode_go_api_gateway/api"
	"ucode/ucode_go_api_gateway/api/handlers"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/project_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage/postgres"

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

	grpcSvcs, err := services.NewGrpcClients(cfg)
	if err != nil {
		log.Error("[ucode] error while establishing grpc conn", logger.Error(err))
		return
	}

	projectsService, err := services.NewProjectGrpcsClient(
		&services.ProjectServices{
			Services: map[string]services.ServiceManagerI{},
			Mu:       sync.Mutex{}},
		grpcSvcs,
		"test.admin.u-code.io",
	)
	if err != nil {
		log.Error("[ucode] error while adding grpc client", logger.Error(err))
		return
	}

	log.Info("success established grpc conn to ucode")

	pgStore, err := postgres.NewPostgres(context.Background(), cfg)
	if err != nil {
		log.Panic("postgres.NewPostgres", logger.Error(err))
		return
	}
	defer pgStore.CloseDB()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	projects, err := pgStore.Project().GetList(ctx, &project_service.GetAllProjectsRequest{})
	if err != nil {
		log.Error("error while getting projects", logger.Error(err))
		return
	}

	for _, project := range projects.GetProjects() {

		conf := config.Config{}

		conf.ObjectBuilderServiceHost = project.ObjectBuilderServiceHost
		conf.ObjectBuilderGRPCPort = project.ObjectBuilderServicePort

		conf.AuthServiceHost = project.AuthServiceHost
		conf.AuthGRPCPort = project.AuthServicePort

		conf.AnalyticsServiceHost = project.AnalyticsServiceHost
		conf.AnalyticsGRPCPort = project.AnalyticsServicePort

		grpcServices, err := services.NewGrpcClients(conf)
		if err != nil {
			log.Error(fmt.Sprintf("[%s] error while establishing grpc conn", project.Namespace), logger.Error(err))
		}

		_, err = services.NewProjectGrpcsClient(projectsService, grpcServices, project.Namespace)
		if err != nil {
			log.Error(fmt.Sprintf("[%s] error while adding grpc client", project.Namespace), logger.Error(err))
		}

		log.Info("success established grpc conn to" + project.Namespace)
	}

	r := gin.New()

	r.Use(gin.Logger(), gin.Recovery())

	h := handlers.NewHandler(cfg, log, projectsService, pgStore, grpcSvcs)

	api.SetUpAPI(r, h, cfg)

	log.Info("server is running...")
	if err := r.Run(cfg.HTTPPort); err != nil {
		return
	}

	// rProjects := gin.New()

	// rProjects.Use(gin.Logger(), gin.Recovery())
	// rProjects.UseH2C = true

	// hProjects := handlers.NewProjectsHandler(cfg, log, projectsService, pgStore)

	// api.SetUpProjectAPIs(rProjects, hProjects, cfg)

	// log.Info("server is running...")
	// if err := rProjects.Run(cfg.HTTPPort); err != nil {
	// 	log.Error("error while running", logger.Error(err))
	// 	return
	// }
}
