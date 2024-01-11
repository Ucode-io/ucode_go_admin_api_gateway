package gitlab

type IntegrationData struct {
	GitlabProjectId        int
	GitlabIntegrationUrl   string
	GitlabIntegrationToken string
	GitlabGroupId          int
}

type ImportData struct {
	PersonalAccessToken string `json:"personal_access_token"`
	RepoId              string `json:"repo_id"`
	TargetNamespace     string `json:"target_namespace"`
	NewName             string `json:"new_name"`
	GitlabToken         string `json:"gitlab_token"`
}

type ImportResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	FullName string `json:"full_name"`
	RefsURL  string `json:"refs_url"`
}
