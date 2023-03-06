package models

import (
	pb "ucode/ucode_go_api_gateway/genproto/scenario_service"
)

type DAGStepConfig struct {
	Id              string                 `json:"id"`
	Slug            string                 `json:"slug"`
	ParentId        string                 `json:"parent_id"`
	DagId           string                 `json:"dag_id"`
	Type            string                 `json:"type"`
	ConnectInfo     *pb.ConnectInfo        `json:"connect_info"`
	RequestInfo     map[string]interface{} `json:"request_info"`
	ConditionAction map[string]interface{} `json:"condition_action"`
	IsParallel      bool                   `json:"is_parallel"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	CallbackType    string                 `json:"callback_type"`
}

type DAGStep struct {
	Config      DAGStepConfig          `json:"config"`
	UiComponent map[string]interface{} `json:"ui_component"`
}

type RunScenarioRequest struct {
	DagId     string                 `json:"dag_id"`
	Header    map[string]string      `json:"header"`
	Body      map[string]interface{} `json:"body"`
	Url       string                 `json:"url"`
	DagStepId string                 `json:"dag_step_id"`
	Method    string                 `json:"method"`
}

type DAG struct {
	Id         string                 `json:"id"`
	Title      string                 `json:"title"`
	Slug       string                 `json:"slug"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	CategoryId string                 `json:"category_id"`
	Attributes map[string]interface{} `json:"attributes"`
	CommitInfo *pb.CommitInfo         `json:"commit_info"`
}

type CreateScenarioRequest struct {
	Dag   DAG        `json:"dag"`
	Steps []*DAGStep `json:"steps"`
}

type GetAllDAGStepResponse struct {
	DAG   DAG        `json:"dag"`
	Steps []*DAGStep `json:"steps"`
}
