package models

type CreateWebhook struct {
	Username      string `json:"username" binding:"required"`
	RepoName      string `json:"repo_name" binding:"required"`
	Branch        string `json:"branch"`
	FrameworkType string `json:"framework_type"`
	GithubToken   string `json:"github_token"`
	FunctionType  string `json:"type"`
	Name          string `json:"name"`
}

type GithubLogin struct {
	Code string `json:"code"`
}

type PipelineLogRequest struct {
	RepoId string `json:"repo_id"`
}

type PipelineLogResponse struct {
	JobName string `json:"job_name"`
	Log     string `json:"log"`
}

type Job struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
