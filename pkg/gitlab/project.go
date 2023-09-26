package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
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
