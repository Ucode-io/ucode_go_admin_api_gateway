package crons

import (
	"context"
	"time"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/code_server"
	"ucode/ucode_go_api_gateway/services"
)

type Cronjob struct {
	Interval time.Duration
	Function func(context.Context, services.ServiceManagerI, config.Config) error
}

func ExecuteCron() []Cronjob {
	cronjobs := make([]Cronjob, 0)
	getResult := Cronjob{
		Interval: time.Duration(time.Minute) * 30,
		Function: code_server.DeleteCodeServer,
	}
	cronjobs = append(cronjobs, getResult)
	return cronjobs
}
