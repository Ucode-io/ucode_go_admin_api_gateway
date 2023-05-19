package helper

import (
	"context"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/services"
)

type MatchingData struct {
	ProjectId string
	EnvId     string
	Path      string
}

func FindUrlTo(s services.ServiceManagerI, data MatchingData) (string, error) {
	res, err := s.CompanyService().Redirect().GetList(context.Background(), &pb.GetListRedirectUrlReq{
		ProjectId: data.ProjectId,
		EnvId:     data.EnvId,
		Offset:    0,
		Limit:     100,
	})
	if err != nil {
		return "", err
	}

	for _, v := range res.GetRedirectUrls() {
		if v.From == data.Path {
			return v.To, nil
		}
	}
	return data.Path, nil
}

// something/{id}/{id} regex ^something/([^/]+)$
// get-list/{id}
// something/abcd/abcd
