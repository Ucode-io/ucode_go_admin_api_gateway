package v1

import (
	"testing"

	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

func TestFacebookPageHasOtherConnections(t *testing.T) {
	connected := func(id string) *pb.ProjectResource {
		return &pb.ProjectResource{Id: id, Settings: &pb.Settings{FacebookLeads: &pb.FacebookLeadsCredentials{Status: config.FacebookStatusActive}}}
	}
	if !facebookPageHasOtherConnections([]*pb.ProjectResource{connected("armada"), connected("udevs")}, "armada") {
		t.Fatal("disconnecting Armada must preserve the page subscription used by Udevs")
	}
	if facebookPageHasOtherConnections([]*pb.ProjectResource{connected("armada")}, "armada") {
		t.Fatal("a sole page connection should be unsubscribed")
	}
}
