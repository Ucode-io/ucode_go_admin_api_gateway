package gitlab_integration

import (
	"errors"
	"fmt"
	"strconv"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
)

func CreateProjectFork(cfg config.Config, projectName string) (response models.GitlabIntegrationResponse, err error) {
	// this is create project by group_id in gitlab

	projectId := cfg.GitlabProjectId
	strProjectId := strconv.Itoa(projectId)
	fmt.Println("config::::::", cfg.GitlabIntegrationURL, cfg.GitlabIntegrationToken)

	resp, err := DoRequest(cfg.GitlabIntegrationURL+"/api/v4/projects/"+strProjectId+"/fork", cfg.GitlabIntegrationToken, "POST", models.CreateProject{
		NamespaceID:          cfg.GitlabGroupId,
		Name:                 projectName,
		Path:                 projectName,
		InitializeWithReadme: true,
		DefaultBranch:        "master",
		Visibility:           "private",
	})

	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}

	return resp, err
}

func DeleteForkedProject(repoName string, cfg config.Config) (response models.GitlabIntegrationResponse, err error) {

	resp, err := DoRequest(cfg.GitlabIntegrationURL+"/api/v4/projects/ucode_functions_group%2"+"F"+repoName, cfg.GitlabIntegrationToken, "DELETE", nil)
	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}
	return models.GitlabIntegrationResponse{
		Code:    200,
		Message: map[string]interface{}{"message": "Successfully deleted"},
	}, nil
}
