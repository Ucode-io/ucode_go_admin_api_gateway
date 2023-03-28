package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/notification_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NotificationServiceI interface {
	Notification() notification_service.NotificationServiceClient
	Category() notification_service.CategoryServiceClient
}

type notificationServiceClient struct {
	notificationService notification_service.NotificationServiceClient
	categoryService     notification_service.CategoryServiceClient
}

func NewNotificationServiceClient(ctx context.Context, cfg config.Config) (NotificationServiceI, error) {

	connNotificationService, err := grpc.DialContext(
		ctx,
		cfg.NotificationServiceHost+cfg.NotificationGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &notificationServiceClient{
		notificationService: notification_service.NewNotificationServiceClient(connNotificationService),
		categoryService:     notification_service.NewCategoryServiceClient(connNotificationService),
	}, nil
}

func (g *notificationServiceClient) Notification() notification_service.NotificationServiceClient {
	return g.notificationService
}

func (g *notificationServiceClient) Category() notification_service.CategoryServiceClient {
	return g.categoryService
}
