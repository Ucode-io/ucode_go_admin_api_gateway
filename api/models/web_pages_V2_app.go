package models

import "google.golang.org/protobuf/types/known/structpb"

type WebPageAppPages struct {
	DefaultPage string `json:"default_page"`
	LoginPage   string `json:"login_page"`
}

type CreateAppReqModel struct {
	ProjectId string          `json:"project_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Logo      string          `json:"logo"`
	SubDomain string          `json:"sub_domain"`
	Pages     WebPageAppPages `json:"pages"`
	Routes    structpb.Struct `json:"routes"`
}

type UpdateAppReqModel struct {
	Id        string          `json:"id,omitempty"`
	ProjectId string          `json:"project_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Logo      string          `json:"logo"`
	SubDomain string          `json:"sub_domain"`
	Pages     WebPageAppPages `json:"pages"`
	Routes    structpb.Struct `json:"routes"`
}
