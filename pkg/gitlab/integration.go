package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
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
	// fmt.Println(url, string(data))
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

	respByte, err := ioutil.ReadAll(resp.Body)
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
