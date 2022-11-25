package postgres

import (
	"context"
	"testing"
	"time"
	"ucode/ucode_go_api_gateway/config"
)

func TestPostgresConn(t *testing.T) {

	conf := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	pg, err := NewPostgres(ctx, conf)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	defer pg.CloseDB()
}
