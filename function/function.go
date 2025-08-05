package function

import (
	"errors"
	"fmt"
	"net/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/util"
)

type HandlerFunc func(string, string, models.NewInvokeFunctionRequest) (string, error)

var FuncHandlers = map[string]HandlerFunc{
	"FUNCTION": ExecOpenFaaS,
	"KNATIVE":  ExecKnative,
	"WORKFLOW": ExecWorkflow,
}

func ExecOpenFaaS(path, name string, req models.NewInvokeFunctionRequest) (string, error) {
	url := fmt.Sprintf("%s%s", config.OpenFaaSBaseUrl, path)
	resp, err := util.DoRequest(url, http.MethodPost, req)
	if err != nil {
		return name, err
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		return name, errors.New(errStr)
	}

	return "", nil
}

func ExecKnative(path, name string, req models.NewInvokeFunctionRequest) (string, error) {
	url := fmt.Sprintf("http://%s.%s", path, config.KnativeBaseUrl)
	resp, err := util.DoRequest(url, http.MethodPost, req)
	if err != nil {
		return name, err
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		return name, errors.New(errStr)
	}

	return "", nil
}

func ExecWorkflow(path, name string, req models.NewInvokeFunctionRequest) (string, error) {
	url := fmt.Sprintf("%s/webhook/%s", req.AutomationURL, path)
	resp, err := util.DoRequest(url, http.MethodPost, req)
	if err != nil {
		return name, err
	} else if resp.Status == "error" {
		var errStr = resp.Status
		if resp.Data != nil && resp.Data["message"] != nil {
			errStr = resp.Data["message"].(string)
		}
		return name, errors.New(errStr)
	}

	return "", nil
}
