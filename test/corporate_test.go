package test

import (
	"log"
	"testing"
	"ucode/ucode_go_api_gateway/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMain(m *testing.M) {

	cfg := config.Load()

	// Set up a connection to the server

	connCorporateService, err := grpc.Dial(
		cfg.CorporateServiceHost+cfg.CorporateGRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Fatalf("did not connect to server %s", err)
	}

	defer connCorporateService.Close()

	//branchClient := corporate_service.NewBranchServiceClient(connCorporateService)

	// run downstream
	//branchChannel := make(chan interface{})

	//fmt.Println(branchClient, branchChannel)

}
