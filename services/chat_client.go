package services

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/chat_ucode"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatServiceI interface {
	Chat() chat_ucode.ChatServiceClient
}

type chatServiceClient struct {
	chat chat_ucode.ChatServiceClient
}

func NewChatServiceClient(ctx context.Context, cfg config.Config) (ChatServiceI, error) {

	connChatService, err := grpc.DialContext(
		ctx,
		cfg.ChatServiceGrpcHost+cfg.ChatServiceGrpcPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &chatServiceClient{
		chat: chat_ucode.NewChatServiceClient(connChatService),
	}, nil
}

func (g *chatServiceClient) Chat() chat_ucode.ChatServiceClient {
	return g.chat
}
