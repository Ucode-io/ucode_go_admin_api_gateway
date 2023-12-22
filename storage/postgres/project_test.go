package postgres

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/project_service"
)

func TestProjectCreate(t *testing.T) {

	conf := config.Config{}
	// conf.Postgres.Host = "161.35.26.178"
	// conf.Postgres.Port = 30032
	// conf.Postgres.Username = "admin_api_gateway"
	// conf.Postgres.Database = "admin_api_gateway"
	// conf.Postgres.Password = "aoM0zohB"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	pg, err := NewPostgres(ctx, conf)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	defer pg.CloseDB()

	_, err = pg.Project().Create(ctx, &project_service.CreateProjectRequest{
		Name:                     "medion",
		Namespace:                fmt.Sprintf("%s_%d", "medion", rand.Intn(10000)),
		ObjectBuilderServiceHost: "0.0.0.0",
		ObjectBuilderServicePort: "80",
		AuthServiceHost:          "0.0.0.1",
		AuthServicePort:          "80",
		AnalyticsServiceHost:     "0.0.0.2",
		AnalyticsServicePort:     "80",
	})
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
}

func TestGetProjectList(t *testing.T) {

	conf := config.Config{}
	// conf.Postgres.Host = "161.35.26.178"
	// conf.Postgres.Port = 30032
	// conf.Postgres.Username = "admin_api_gateway"
	// conf.Postgres.Database = "admin_api_gateway"
	// conf.Postgres.Password = "aoM0zohB"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	pg, err := NewPostgres(ctx, conf)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	defer pg.CloseDB()

	res, err := pg.Project().GetList(ctx, &project_service.GetAllProjectsRequest{})
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	fmt.Println(res.Count)
}

func TestDelete(t *testing.T) {
	conf := config.Config{}
	// conf.Postgres.Host = "161.35.26.178"
	// conf.Postgres.Port = 30032
	// conf.Postgres.Username = "admin_api_gateway"
	// conf.Postgres.Database = "admin_api_gateway"
	// conf.Postgres.Password = "aoM0zohB"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	pg, err := NewPostgres(ctx, conf)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	defer pg.CloseDB()

	name, rows, err := pg.Project().Delete(ctx, &project_service.ProjectPrimaryKey{Id: "71fda691-82db-4eda-a07f-187cdc34007f"})
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	fmt.Println(name, rows)
}
