package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/pos_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PosServiceI interface {
	OfflineAppointment() pos_service.OfflineAppointmentServiceClient
	BookedAppointment() pos_service.BookedAppointmentServiceClient
}

type posServiceClient struct {
	offlineAppointmentService pos_service.OfflineAppointmentServiceClient
	bookedAppointmentService  pos_service.BookedAppointmentServiceClient
}

func NewPosServiceClient(ctx context.Context, cfg config.Config) (PosServiceI, error) {

	connPosService, err := grpc.DialContext(
		ctx,
		cfg.PosServiceHost+cfg.PosGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &posServiceClient{
		offlineAppointmentService: pos_service.NewOfflineAppointmentServiceClient(connPosService),
		bookedAppointmentService:  pos_service.NewBookedAppointmentServiceClient(connPosService),
	}, nil
}

func (g *posServiceClient) OfflineAppointment() pos_service.OfflineAppointmentServiceClient {
	return g.offlineAppointmentService
}

func (g *posServiceClient) BookedAppointment() pos_service.BookedAppointmentServiceClient {
	return g.bookedAppointmentService
}
