package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
)

type RequestForm struct {
	Method  string
	RawUrl  string
	Params  map[string]string
	Headers map[string]string
	Body    io.Reader
}

func DoRequest(url, token string, method string, body interface{}) (responseModel models.GitlabIntegrationResponse, err error) {
	data, err := json.Marshal(&body)
	if err != nil {
		return
	}
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	url += "?access_token=" + token
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	emptyMap := make(map[string]interface{})
	err = json.Unmarshal(respByte, &emptyMap)
	responseModel.Message = emptyMap
	responseModel.Code = resp.StatusCode

	return
}

func DoRequestV2(ctx context.Context, reqData RequestForm) (models.GitlabIntegrationResponse, error) {
	message := make(map[string]interface{})

	request, err := url.Parse(reqData.RawUrl)
	if err != nil {
		return models.GitlabIntegrationResponse{}, err
	}

	v := request.Query()
	if reqData.Params != nil {
		for key, value := range reqData.Params {
			v.Set(key, value)
		}
	}
	request.RawQuery = v.Encode()

	req, err := http.NewRequest(reqData.Method, request.String(), reqData.Body)
	if err != nil {
		return models.GitlabIntegrationResponse{}, err
	}

	if reqData.Headers != nil {
		for key, value := range reqData.Headers {
			req.Header.Set(key, value)
		}
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return models.GitlabIntegrationResponse{}, err
	}
	defer res.Body.Close()

	bodyByte, err := io.ReadAll(res.Body)
	if err != nil {
		return models.GitlabIntegrationResponse{}, err
	}

	err = json.Unmarshal(bodyByte, &message)
	if err != nil {
		return models.GitlabIntegrationResponse{}, err
	}

	return models.GitlabIntegrationResponse{
		Code:    res.StatusCode,
		Message: message,
	}, nil
}

func MakeGitLabRequest(method, url string, payload map[string]interface{}, token string) (map[string]interface{}, error) {
	reqBody := new(bytes.Buffer)
	if payload != nil {
		json.NewEncoder(reqBody).Encode(payload)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func MakeRequest(method, url, token string, payload map[string]interface{}) (map[string]interface{}, error) {
	reqBody := new(bytes.Buffer)
	if payload != nil {
		json.NewEncoder(reqBody).Encode(payload)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func MakeRequestV1(method, url, token string, payload map[string]interface{}) ([]byte, error) {
	var reqBody = new(bytes.Buffer)

	if payload != nil {
		json.NewEncoder(reqBody).Encode(payload)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}
