package postgres

import (
	"context"
	"errors"
	pb "ucode/ucode_go_api_gateway/genproto/project_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type projectRepo struct {
	db *pgxpool.Pool
}

func NewProjectRepo(db *pgxpool.Pool) storage.ProjectRepoI {
	return &projectRepo{
		db: db,
	}
}

func (r *projectRepo) Create(ctx context.Context, entity *pb.CreateProjectRequest) (pKey *pb.ProjectPrimaryKey, err error) {
	query := `INSERT INTO "project" (
		id,
		name,
		namespace,
		object_builder_service_host,
		object_builder_service_port,
		auth_service_host,
		auth_service_port,
		analytics_service_host,
		analytics_service_port
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9
	)`

	uuid, err := uuid.NewRandom()
	if err != nil {
		return pKey, err
	}

	_, err = r.db.Exec(ctx, query,
		uuid.String(),
		entity.Name,
		entity.Namespace,
		entity.ObjectBuilderServiceHost,
		entity.ObjectBuilderServicePort,
		entity.AuthServiceHost,
		entity.AuthServicePort,
		entity.AnalyticsServiceHost,
		entity.AuthServicePort,
	)

	pKey = &pb.ProjectPrimaryKey{
		Id: uuid.String(),
	}

	return pKey, err
}

func (r *projectRepo) GetByPK(ctx context.Context, pKey *pb.ProjectPrimaryKey) (res *pb.Project, err error) {
	// res = &pb.Project{}
	// query := `SELECT
	// 	id,
	// 	project_id,
	// 	name,
	// 	subdomain
	// FROM
	// 	"project"
	// WHERE
	// 	id = $1`

	// err = r.db.QueryRow(ctx, query, pKey.Id).Scan(
	// 	&res.Id,
	// 	&res.ProjectId,
	// 	&res.Name,
	// 	&res.Subdomain,
	// )

	// if err != nil {
	// 	return res, err
	// }

	// return res, nil
	return nil, errors.New("error not implemented")
}

func (r *projectRepo) GetList(ctx context.Context, queryParam *pb.GetAllProjectsRequest) (res *pb.GetAllProjectsResponse, err error) {
	res = &pb.GetAllProjectsResponse{}
	var arr []interface{}
	params := make(map[string]interface{})
	query := `SELECT
		id,
		name,
		namespace,
		object_builder_service_host,
		object_builder_service_port,
		auth_service_host,
		auth_service_port,
		analytics_service_host,
		analytics_service_port
	FROM
		"project"`
	filter := " WHERE deleted_at is null"
	order := " ORDER BY created_at"
	arrangement := " DESC"
	offset := ""
	limit := ""

	if len(queryParam.Search) > 0 {
		params["search"] = queryParam.Search
		filter += " AND ((name || namespace) ILIKE ('%' || :search || '%'))"
	}

	if queryParam.Offset > 0 {
		params["offset"] = queryParam.Offset
		offset = " OFFSET :offset"
	}

	if queryParam.Limit > 0 {
		params["limit"] = queryParam.Limit
		limit = " LIMIT :limit"
	}

	cQ := `SELECT count(1) FROM "project"` + filter

	cQ, arr = helper.ReplaceQueryParams(cQ, params)
	err = r.db.QueryRow(ctx, cQ, arr...).Scan(&res.Count)

	if err != nil {
		return res, err
	}

	q := query + filter + order + arrangement + offset + limit

	q, arr = helper.ReplaceQueryParams(q, params)
	rows, err := r.db.Query(ctx, q, arr...)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		obj := &pb.Project{}
		err = rows.Scan(
			&obj.Id,
			&obj.Name,
			&obj.Namespace,
			&obj.ObjectBuilderServiceHost,
			&obj.ObjectBuilderServicePort,
			&obj.AuthServiceHost,
			&obj.AuthServicePort,
			&obj.AnalyticsServiceHost,
			&obj.AnalyticsServicePort,
		)
		if err != nil {
			return res, err
		}
		res.Projects = append(res.Projects, obj)
	}

	return res, nil
}

func (r *projectRepo) Update(ctx context.Context, entity *pb.UpdateProjectRequest) (rowsAffected int64, err error) {
	// query := `UPDATE "project" SET
	// 	name = :name,
	// 	subdomain = :subdomain,
	// 	updated_at = now()
	// WHERE
	// 	id = :id`

	// params := map[string]interface{}{
	// 	"id":        entity.Id,
	// 	"name":      entity.Name,
	// 	"subdomain": entity.Subdomain,
	// }

	// query, arr := helper.ReplaceQueryParams(query, params)
	// result, err := r.db.Exec(ctx, query, arr...)
	// if err != nil {
	// 	return 0, err
	// }

	// rowsAffected = result.RowsAffected()

	// return rowsAffected, err
	return 0, errors.New("error: not implemented")
}

func (r *projectRepo) Delete(ctx context.Context, pKey *pb.ProjectPrimaryKey) (rowsAffected int64, err error) {
	query := `UPDATE "project" SET deleted_at = now() WHERE id = $1`
	result, err := r.db.Exec(ctx, query, pKey.Id)
	if err != nil {
		return result.RowsAffected(), err
	}

	return result.RowsAffected(), nil
}
