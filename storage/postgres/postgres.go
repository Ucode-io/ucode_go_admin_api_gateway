package postgres

import (
	"context"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Store struct {
	db      *pgxpool.Pool
	project storage.ProjectRepoI
}

func NewPostgres(ctx context.Context, cfg config.Config) (storage.StorageI, error) {
	// config, err := pgxpool.ParseConfig(
	// 	fmt.Sprintf(
	// 		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
	// 		cfg.Postgres.Username,
	// 		cfg.Postgres.Password,
	// 		cfg.Postgres.Host,
	// 		cfg.Postgres.Port,
	// 		cfg.Postgres.Database,
	// 	))

	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}

	config.MaxConns = 4

	pool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &Store{db: pool}, err
}

func (s *Store) CloseDB() {
	s.db.Close()
}

func (s *Store) Project() storage.ProjectRepoI {
	if s.project == nil {
		s.project = NewProjectRepo(s.db)
	}

	return s.project
}
