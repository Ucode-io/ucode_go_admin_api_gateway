package github

type ListWebhookRequest struct {
	Username    string `json:"username"`
	RepoName    string `json:"repo_name"`
	GithubToken string `json:"github_token"`
	ProjectUrl  string `json:"project_url"`
}

type CreateWebhookRequest struct {
	Username      string `json:"username" binding:"required"`
	RepoName      string `json:"repo_name" binding:"required"`
	WebhookSecret string `json:"secret"`
	GithubToken   string `json:"github_token"`
	FrameworkType string `json:"framework_type"`
	Branch        string `json:"branch"`
	FunctionType  string `json:"type"`
	ProjectUrl    string `json:"project_url"`
	Name          string `json:"name"`
}

type WebhookPayload struct {
	Name   string   `json:"name"`
	Active bool     `json:"active"`
	Events []string `json:"events"`
	Config Config   `json:"config"`
}

type Config struct {
	URL           string `json:"url"`
	ContentType   string `json:"content_type"`
	Secret        string `json:"secret"`
	FrameworkType string `json:"framework_type"`
	Branch        string `json:"branch"`
	FunctionType  string `json:"type"`
	Name          string `json:"name"`
}
