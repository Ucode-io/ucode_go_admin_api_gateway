package code_server

import (
	"errors"
	"fmt"
	"os/exec"
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

	cmd = exec.Command("helm", "install", functionName, "code-server/code-server", "-n", "test", hostName, hostNameTls, secretName)
	err = cmd.Run()
	if err != nil {
		return "", errors.New("error while install code server::" + err.Error())
	}
	fmt.Println("test helm install code server")

	output, err := exec.Command("kubectl", "get", "secret", " --namespace", "test", functionName, "-o", "jsonpath=\"{.data.password}\"", "|", "base64", "-d").Output()
	if err != nil {
		return "", errors.New("error while get password 0::" + err.Error())
	}
	// output, err := cmd.Output()
	// if err != nil {
	// 	return "", errors.New("error while get password 1::" + err.Error())
	// }

	fmt.Println("Finish", string(output))

	return string(output), nil
}
