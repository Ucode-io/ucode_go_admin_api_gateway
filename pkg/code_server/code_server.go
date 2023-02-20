package code_server

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"ucode/ucode_go_api_gateway/config"
)

func CreateCodeServer(functionName string, cfg config.Config, id string) (string, error) {

	// command := fmt.Sprintf("--username udevs --password %s code-server https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable", cfg.GitlabIntegrationToken)
	cmd := exec.Command("helm", "repo", "add", "--username", "udevs", "--password", cfg.GitlabIntegrationToken, "code-server", "https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable")
	err := cmd.Run()
	if err != nil {
		return "", errors.New("error while adding repo:" + err.Error())
	}
	fmt.Println("test repo add")

	cmd = exec.Command("helm", "repo", "update")
	err = cmd.Run()
	if err != nil {
		return "", errors.New("error while repo update helm::" + err.Error())
	}

	hostName := fmt.Sprintf("--set=ingress.hosts[0].host=%s.u-code.io", id)
	hostNameTls := fmt.Sprintf("--set=ingress.tls[0].hosts[0]=%s.u-code.io", id)
	secretName := fmt.Sprintf("--set=ingress.tls[0].secretName=%s-tls", id)
	path := "--set=ingress.hosts[0].paths[0]=/"

	cmd = exec.Command("helm", "install", functionName, "code-server/code-server", "-n", "test", hostName, hostNameTls, secretName, path)
	err = cmd.Run()
	if err != nil {
		return "", errors.New("error while install code server::" + err.Error())
	}
	fmt.Println("test helm install code server")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd = exec.Command("kubectl", "get", "secret", "--namespace", "test", functionName+"-code-server", "-o", "jsonpath=\"{.data.password}\"")
	if err != nil {
		return "", errors.New("error while get password 0::" + err.Error())
	}
	cmd.Stdout = &out
	fmt.Println("aa::", cmd.Stderr)
	cmd.Stderr = &stderr
	err = cmd.Run()
	fmt.Println("ss::", cmd.Stdout)
	fmt.Println("ff::", out.String())
	if err != nil {
		return "", errors.New("error running get password command::" + stderr.String())
	}

	s := strings.ReplaceAll(out.String(), `"`, "")
	str, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", errors.New("error while base64 to string::" + stderr.String())
	}
	pass := fmt.Sprintf("%s", str)
	fmt.Println("pass:", pass)

	return pass, nil
}

// func DeleteCodeServer(cfg config.Config) (string, error) {


// 	cmd := exec.Command("helm", "uninstall", functionName)
// 	err := cmd.Run()
// 	if err != nil {
// 		return "", errors.New("error while repo update helm::" + err.Error())
// 	}

// 	hostName := fmt.Sprintf("--set=ingress.hosts[0].host=%s.u-code.io", id)
// 	hostNameTls := fmt.Sprintf("--set=ingress.tls[0].hosts[0]=%s.u-code.io", id)
// 	secretName := fmt.Sprintf("--set=ingress.tls[0].secretName=%s-tls", id)
// 	path := "--set=ingress.hosts[0].paths[0]=/"

// 	cmd = exec.Command("helm", "install", functionName, "code-server/code-server", "-n", "test", hostName, hostNameTls, secretName, path)
// 	err = cmd.Run()
// 	if err != nil {
// 		return "", errors.New("error while install code server::" + err.Error())
// 	}
// 	fmt.Println("test helm install code server")
// 	var out bytes.Buffer
// 	var stderr bytes.Buffer
// 	cmd = exec.Command("kubectl", "get", "secret", "--namespace", "test", functionName+"-code-server", "-o", "jsonpath=\"{.data.password}\"")
// 	if err != nil {
// 		return "", errors.New("error while get password 0::" + err.Error())
// 	}
// 	cmd.Stdout = &out
// 	fmt.Println("aa::", cmd.Stderr)
// 	cmd.Stderr = &stderr
// 	err = cmd.Run()
// 	fmt.Println("ss::", cmd.Stdout)
// 	fmt.Println("ff::", out.String())
// 	if err != nil {
// 		return "", errors.New("error running get password command::" + stderr.String())
// 	}

// 	s := strings.ReplaceAll(out.String(), `"`, "")
// 	str, err := base64.StdEncoding.DecodeString(s)
// 	if err != nil {
// 		return "", errors.New("error while base64 to string::" + stderr.String())
// 	}
// 	pass := fmt.Sprintf("%s", str)
// 	fmt.Println("pass:", pass)

// 	return pass, nil
// }
