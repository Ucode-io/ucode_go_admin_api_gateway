package testing

import (
	"log"
	"os"
	"testing"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	cfg                config.Config
	connCompanyService *grpc.ClientConn
	companyClient      company_service.CompanyServiceClient
	projectClient      company_service.ProjectServiceClient
)

func TestMain(m *testing.M) {
	var err error
	cfg = config.Load()

	connCompanyService, err = grpc.Dial(
		cfg.CompanyServiceHost+cfg.CompanyServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Fatalf("did not connect to server %s", err)
	}

	companyClient = company_service.NewCompanyServiceClient(connCompanyService)
	projectClient = company_service.NewProjectServiceClient(connCompanyService)

	os.Exit(m.Run())

}
