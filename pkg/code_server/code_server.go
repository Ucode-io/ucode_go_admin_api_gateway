package code_server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/services"
)

func CreateCodeServer(functionName string, cfg config.Config, id string) (string, error) {

	// command := fmt.Sprintf("--username udevs --password %s code-server https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable", cfg.GitlabIntegrationToken)
	cmd := exec.Command("helm", "repo", "add", "--username", "udevs", "--password", cfg.GitlabIntegrationToken, "code-server", "https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("err 0::", err)
		return "", errors.New("error while adding repo:" + stderr.String())
	}
	fmt.Println("test repo add")

	cmd = exec.Command("helm", "repo", "update")
	err = cmd.Run()
	if err != nil {
		fmt.Println("err 1::", err)
		return "", errors.New("error while repo update helm::" + stderr.String())
	}

	hostName := fmt.Sprintf("--set=ingress.hosts[0].host=%s.u-code.io", id)
	hostNameTls := fmt.Sprintf("--set=ingress.tls[0].hosts[0]=%s.u-code.io", id)
	secretName := fmt.Sprintf("--set=ingress.tls[0].secretName=%s-tls", id)
	path := "--set=ingress.hosts[0].paths[0]=/"

	cmd = exec.Command("helm", "install", functionName, "code-server/code-server", "-n", "test", hostName, hostNameTls, secretName, path)
	err = cmd.Run()
	if err != nil {
		fmt.Println("err 2::", err)
		return "", errors.New("error while install code server::" + stderr.String())
	}
	fmt.Println("test helm install code server")
	var out bytes.Buffer
	cmd = exec.Command("kubectl", "get", "secret", "--namespace", "test", functionName+"-code-server", "-o", "jsonpath=\"{.data.password}\"")
	if err != nil {
		fmt.Println("err 3::", err)
		return "", errors.New("error while get password 0::" + stderr.String())
	}

	cmd.Stdout = &out
	fmt.Println("aa::", cmd.Stderr)

	err = cmd.Run()
	fmt.Println("ss::", cmd.Stdout)
	fmt.Println("ff::", out.String())
	if err != nil {
		fmt.Println("err 4::", err)
		return "", errors.New("error running get password command::" + stderr.String())
	}

	s := strings.ReplaceAll(out.String(), `"`, "")
	str, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		fmt.Println("err 5::", err)
		return "", errors.New("error while base64 to string::" + stderr.String())
	}
	pass := fmt.Sprintf("%s", str)
	fmt.Println("pass:", pass)

	return pass, nil
}

func DeleteCodeServer(ctx context.Context, srvs services.ServiceManagerI, cfg config.Config) error {

	functions, err := srvs.FunctionService().FunctionService().GetList(context.Background(), &new_function_service.GetAllFunctionsRequest{})
	if err != nil {
		return err
	}
	for _, function := range functions.GetFunctions() {
		cmd := exec.Command("helm", "uninstall", function.Path)
		err = cmd.Run()
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err != nil {
			return errors.New("error while uninstalling " + function.GetPath() + "error: " + stderr.String())
		}
	}

	return nil
}
