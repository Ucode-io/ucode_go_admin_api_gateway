package gitlab_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
)

func DoRequest(url, token string, method string, body interface{}) (responseModel models.GitlabIntegrationResponse, err error) {
	data, err := json.Marshal(&body)
	if err != nil {
		return
	}
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	url += "?access_token=" + token
	fmt.Println(url)
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
