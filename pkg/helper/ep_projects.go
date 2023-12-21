package helper

import (
	"context"
	"strconv"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"

	"google.golang.org/protobuf/types/known/emptypb"
)

func EnterPriceProjectsGrpcSvcs(ctx context.Context, compSrvc services.CompanyServiceI, serviceNodes services.ServiceNodesI, log logger.LoggerI) (services.ServiceNodesI, map[string]config.Config) {

	epProjects, err := compSrvc.Project().GetProjectConfigList(
		ctx,
		&emptypb.Empty{},
	)
	if err != nil {
		log.Error("Error getting enter prise project. GetList", logger.Error(err))
	}

	if epProjects != nil {
		mapProjectConf := map[string]config.Config{}

		for _, v := range epProjects.Configs {
			num, err := strconv.Atoi(v.REDIS_DATABASE)
			if err != nil {
				log.Error("Error:", logger.Error(err))
			}

			// v.OBJECT_BUILDER_GRPC_PORT = ":9102"
			// v.OBJECT_BUILDER_SERVICE_HOST = "localhost"

			// v.REDIS_HOST = "localhost"
			// v.REDIS_PORT = ":6379"
			// v.REDIS_PASSWORD = ""
			// num = 0

			// v.FUNCTION_GRPC_PORT = ":8100"
			// v.FUNCTION_SERVICE_HOST = "localhost"

			projectConf := config.Config{
				ConvertTemplateServiceGrpcPort: v.CONVERT_TEMPLATE_GRPC_PORT,
				ConvertTemplateServiceGrpcHost: v.CONVERT_TEMPLATE_SERVICE_HOST,
				AnalyticsGRPCPort:              v.ANALYTICS_GRPC_PORT,
				AnalyticsServiceHost:           v.ANALYTICS_SERVICE_HOST,
				ApiReferenceServicePort:        v.API_REF_GRPC_PORT,
				ApiReferenceServiceHost:        v.API_REF_SERVICE_HOST,
				ChatServiceGrpcPort:            v.CHAT_GRPC_PORT,
				ChatServiceGrpcHost:            v.CHAT_SERVICE_HOST,
				FunctionServicePort:            v.FUNCTION_GRPC_PORT,
				FunctionServiceHost:            v.FUNCTION_SERVICE_HOST,
				NotificationGRPCPort:           v.NOTIFICATION_GRPC_PORT,
				NotificationServiceHost:        v.NOTIFICATION_SERVICE_HOST,
				ObjectBuilderGRPCPort:          v.OBJECT_BUILDER_GRPC_PORT,
				ObjectBuilderServiceHost:       v.OBJECT_BUILDER_SERVICE_HOST,
				HighObjectBuilderGRPCPort:      v.OBJECT_BUILDER_HIGH_GRPC_PORT,
				HighObjectBuilderServiceHost:   v.OBJECT_BUILDER_SERVICE_HIGHT_HOST,
				QueryServicePort:               v.QUERY_GRPC_PORT,
				// QueryServiceHost:               v.QUERY_SERVICE_HOST,
				// ScenarioGRPCPort:               v.SCENARIO_GRPC_PORT,
				// ScenarioServiceHost:            v.SCENARIO_SERVICE_HOST,
				// SmsGRPCPort:                    v.SMS_GRPC_PORT,
				// SmsServiceHost:                 v.SMS_SERVICE_HOST,
				TemplateGRPCPort:        v.TEMPLATE_GRPC_PORT,
				TemplateServiceHost:     v.TEMPLATE_SERVICE_HOST,
				VersioningGRPCPort:      v.VERSIONING_GRPC_PORT,
				VersioningServiceHost:   v.VERSIONING_SERVICE_HOST,
				GetRequestRedisHost:     v.REDIS_HOST,
				GetRequestRedisPort:     v.REDIS_PORT,
				GetRequestRedisDatabase: num,
				GetRequestRedisPassword: v.REDIS_PASSWORD,
			}

			grpcSvcs, err := services.NewGrpcClients(ctx, projectConf)
			if err != nil {
				log.Error("Error connecting grpc client "+v.ProjectId, logger.Error(err))
			}

			err = serviceNodes.Add(grpcSvcs, v.ProjectId)
			if err != nil {
				log.Error("Error adding to grpc pooling enter prise project. ServiceNode ! "+v.ProjectId, logger.Error(err))
			}

			log.Info(" --- " + v.ProjectId + " --- added to serviceNodes")

			mapProjectConf[v.ProjectId] = projectConf
		}

		return serviceNodes, mapProjectConf
	} else {
		return nil, nil
	}

}
