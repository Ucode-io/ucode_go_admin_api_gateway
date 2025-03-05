package models

type (
	GithubLogin struct {
		Code string `json:"code"`
	}

	PipelineLogRequest struct {
		RepoId string `json:"repo_id"`
	}

	PipelineLogResponse struct {
		JobName string `json:"job_name"`
		Log     string `json:"log"`
	}

	Job struct {
		Id     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
)
