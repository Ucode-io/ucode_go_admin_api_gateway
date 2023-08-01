package gitlab

import (
	"errors"
	"strconv"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
)

func CreatePipeline(cfg IntegrationData, data map[string]interface{}) (response models.GitlabIntegrationResponse, err error) {

	projectId := cfg.GitlabProjectId
	strProjectId := strconv.Itoa(projectId)
	// fmt.Println("config::::::", cfg.GitlabIntegrationUrl, cfg.GitlabIntegrationToken)

	resp, err := DoRequest(cfg.GitlabIntegrationUrl+"/api/v4/projects/"+strProjectId+"/pipeline", cfg.GitlabIntegrationToken+"&"+"ref=staging", "POST", data)

	// fmt.Println("resp::::::::::::::::::::", resp)
	if resp.Code >= 400 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.BadRequest.Description)
	} else if resp.Code >= 500 {
		return models.GitlabIntegrationResponse{}, errors.New(status_http.InternalServerError.Description)
	}

	return resp, err
}
