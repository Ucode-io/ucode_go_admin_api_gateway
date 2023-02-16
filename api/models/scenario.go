package models

import (
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"
)

type DAGStep struct {
	Id              string                 `json:"id"`
	Slug            string                 `json:"slug"`
	ParentId        string                 `json:"parent_id"`
	DagId           string                 `json:"dag_id"`
	Type            string                 `json:"type"`
	ConnectInfo     pb.ConnectInfo         `json:"connect_info"`
	RequestInfo     map[string]interface{} `json:"request_info"`
	ConditionAction map[string]interface{} `json:"condition_action"`
	IsParallel      bool                   `json:"is_parallel"`
}
