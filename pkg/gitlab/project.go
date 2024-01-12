package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/helper"
)

func CreateProjectFork(projectName string, data IntegrationData) (response models.GitlabIntegrationResponse, err error) {
	// create repo in given group by existing project in gitlab

	projectId := data.GitlabProjectId
	strProjectId := strconv.Itoa(projectId)
	// fmt.Println("config::::::", data.GitlabIntegrationUrl, data.GitlabIntegrationToken)

	resp, err := DoRequest(data.GitlabIntegrationUrl+"/api/v4/projects/"+strProjectId+"/fork", data.GitlabIntegrationToken, "POST", models.CreateProject{
		NamespaceID:          data.GitlabGroupId,
		Name:                 projectName,
		Path:                 projectName,
		InitializeWithReadme: true,
		DefaultBranch:        "master",
		Visibility:           "private",
	})

	// fmt.Println("res:::::::::::", resp)

	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}

	return resp, err
}

func DeleteForkedProject(repoName string, cfg config.BaseConfig) (response models.GitlabIntegrationResponse, err error) {

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

func UpdateProject(cfg IntegrationData, data map[string]interface{}) (response models.GitlabIntegrationResponse, err error) {
	// create repo in given group by existing project in gitlab

	projectId := cfg.GitlabProjectId
	strProjectId := strconv.Itoa(projectId)
	// fmt.Println("config::::::", cfg.GitlabIntegrationUrl, cfg.GitlabIntegrationToken)

	resp, err := DoRequest(cfg.GitlabIntegrationUrl+"/api/v4/projects/"+strProjectId, cfg.GitlabIntegrationToken, "PUT", data)

	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}

	return resp, err
}

func CreateProjectVariable(cfg IntegrationData, data map[string]interface{}) (response models.GitlabIntegrationResponse, err error) {
	// create repo in given group by existing project in gitlab

	projectId := cfg.GitlabProjectId
	strProjectId := strconv.Itoa(projectId)
	// fmt.Println("config::::::", cfg.GitlabIntegrationUrl, cfg.GitlabIntegrationToken)

	resp, err := DoRequest(cfg.GitlabIntegrationUrl+"/api/v4/projects/"+strProjectId+"/variables", cfg.GitlabIntegrationToken, "POST", data)

	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}

	return resp, err
}

func CreateProjectVariableV2(cfg IntegrationData, data interface{}) (response models.GitlabIntegrationResponse, err error) {

	url := fmt.Sprintf("%s/api/v4/projects/%d/variables", cfg.GitlabIntegrationUrl, cfg.GitlabProjectId)

	ctx, finish := context.WithTimeout(context.Background(), 10*time.Second)
	defer finish()

	dataJson, err := json.Marshal(data)
	if err != nil {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	}

	resp, err := DoRequestV2(ctx, RequestForm{
		Method: "POST",
		RawUrl: url,
		Headers: map[string]string{
			"PRIVATE-TOKEN": cfg.GitlabIntegrationToken,
		},
		Body: bytes.NewBuffer(dataJson),
	})

	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}

	return resp, err
}

func ImportFromGithub(cfg ImportData) (response ImportResponse, err error) {
	gitlabBodyJSON, err := json.Marshal(cfg)
	if err != nil {
		return ImportResponse{}, errors.New("failed to marshal JSON")
	}

	gitlabUrl := "https://gitlab.udevs.io/api/v4/import/github"
	req, err := http.NewRequest("POST", gitlabUrl, bytes.NewBuffer(gitlabBodyJSON))
	if err != nil {
		return ImportResponse{}, errors.New("failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", cfg.GitlabToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ImportResponse{}, errors.New("failed to send request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ImportResponse{}, errors.New("failed to read response body")
	}

	var importResponse ImportResponse
	err = json.Unmarshal(respBody, &importResponse)
	if err != nil {
		return ImportResponse{}, errors.New("failed to unmarshal response body")
	}
	return importResponse, nil
}

func AddFilesToRepo(gitlabToken string, path string, gitlabRepoId int, branch string) error {
	//localFolderPath := "/go/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway/github_integration"
	localFolderPath := "github_integration"

	files, err := helper.ListFiles(localFolderPath)
	if err != nil {
		return errors.New("error listing files")
	}

	var actions []map[string]interface{}

	for _, file := range files {
		if file == ".gitlab-ci.yml" {
			continue
		}
		filePath := filepath.Join(localFolderPath, file)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return errors.New("failed to read file")
		}

		action := map[string]interface{}{
			"action":    "create",
			"file_path": file,
			"content":   string(fileContent),
		}

		actions = append(actions, action)
	}

	commitURL := fmt.Sprintf("%s/projects/%v/repository/commits", "https://gitlab.udevs.io/api/v4", gitlabRepoId)
	commitPayload := map[string]interface{}{
		"branch":         branch,
		"commit_message": "Added devops files",
		"actions":        actions,
	}

	_, err = MakeGitLabRequest("POST", commitURL, commitPayload, gitlabToken)
	if err != nil {
		return errors.New("failed to make GitLab request")
	}

	return nil
}

func AddCiFile(gitlabToken, path string, gitlabRepoId int, branch string) error {
	//localFolderPath := "/go/src/gitlab.udevs.io/ucode/ucode_go_admin_api_gateway/github_integration"
	localFolderPath := "github_integration"
	filePath := fmt.Sprintf("%v/.gitlab-ci.yml", localFolderPath)

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return errors.New("failed to read file")
	}

	commitURL := fmt.Sprintf("%s/projects/%v/repository/commits", "https://gitlab.udevs.io/api/v4", gitlabRepoId)
	commitPayload := map[string]interface{}{
		"branch":         branch,
		"commit_message": "Added ci file",
		"actions": []map[string]interface{}{
			{
				"action":    "create",
				"file_path": ".gitlab-ci.yml",
				"content":   string(fileContent),
			},
		},
	}

	_, err = MakeGitLabRequest("POST", commitURL, commitPayload, gitlabToken)
	if err != nil {
		return errors.New("failed to make GitLab request")
	}

	return nil
}

type Pipeline struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

func GetLatestPipeline(token, branchName string, projectID int) (*Pipeline, error) {
	apiURL := fmt.Sprintf("%s/projects/%v/pipelines?ref=%s&order_by=id&sort=desc&per_page=1", "https://gitlab.udevs.io/api/v4", projectID, branchName)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pipeline. Status code: %d", resp.StatusCode)
	}

	var pipelines []Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipelines); err != nil {
		return nil, err
	}

	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipelines found for the specified branch")
	}

	return &pipelines[0], nil
}

func DeleteRepository(token string, projectID int) error {
	apiURL := fmt.Sprintf("%s/projects/%v", "https://gitlab.udevs.io/api/v4", projectID)

	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
