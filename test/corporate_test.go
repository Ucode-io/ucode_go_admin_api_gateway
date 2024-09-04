package test

import (
	"testing"
	"ucode/ucode_go_api_gateway/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMain(m *testing.M) {

	cfg := config.Load()

	// Set up a connection to the server

	connCorporateService, _ := grpc.Dial(
		cfg.CorporateServiceHost+cfg.CorporateGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	defer connCorporateService.Close()

	//branchClient := corporate_service.NewBranchServiceClient(connCorporateService)

	// run downstream
	//branchChannel := make(chan interface{})

}
