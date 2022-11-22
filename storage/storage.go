package storage

import (
	"context"
	"errors"
	pb "ucode/ucode_go_api_gateway/genproto/project_service"
)

var ErrorTheSameId = errors.New("cannot use the same uuid for 'id' and 'parent_id' fields")
var ErrorProjectId = errors.New("not valid 'project_id'")

type StorageI interface {
	CloseDB()
	Project() ProjectRepoI
}

type ProjectRepoI interface {
	Create(ctx context.Context, entity *pb.CreateProjectRequest) (pKey *pb.ProjectPrimaryKey, err error)
	GetList(ctx context.Context, queryParam *pb.GetAllProjectsRequest) (res *pb.GetAllProjectsResponse, err error)
	GetByPK(ctx context.Context, pKey *pb.ProjectPrimaryKey) (res *pb.Project, err error)
	Update(ctx context.Context, entity *pb.UpdateProjectRequest) (rowsAffected int64, err error)
	Delete(ctx context.Context, pKey *pb.ProjectPrimaryKey) (namespace string, rowsAffected int64, err error)
}
