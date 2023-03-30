package code_server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/services"

	"google.golang.org/protobuf/types/known/emptypb"
)

func CreateCodeServer(functionName string, cfg config.Config, id string) (string, error) {

	// command := fmt.Sprintf("--username udevs --password %s code-server https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable", cfg.GitlabIntegrationToken)
	cmd := exec.Command("helm", "repo", "add", "--username", "udevs", "--password", cfg.GitlabIntegrationToken, "code-server", "https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Println("----exec command-----", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Println("err 0::", err)
		return "", errors.New("error while adding repo:" + stderr.String())
	}

	log.Println("----exec command-----", cmd.String())
	cmd = exec.Command("helm", "repo", "update")
	err = cmd.Run()
	if err != nil {
		fmt.Println("err 1::", err)
		log.Println("error exec command:", cmd.String())
		return "", errors.New("error while repo update helm::" + stderr.String())
	}

	hostName := fmt.Sprintf("--set=ingress.hosts[0].host=%s.u-code.io", id)
	hostNameTls := fmt.Sprintf("--set=ingress.tls[0].hosts[0]=%s.u-code.io", id)
	// secretName := fmt.Sprintf("--set=ingress.tls[0].secretName=%s-tls", id)
	secretName := "--set=ingress.tls[0].secretName=ucode-wildcard"

	path := "--set=ingress.hosts[0].paths[0]=/"

	log.Println("----exec command-----", cmd.String())

	cmd = exec.Command("helm", "install", functionName, "code-server/code-server", "-n", "test", hostName, hostNameTls, secretName, path)

	// helm install newnewnew-sdfsdfsdf-doupdate code-server/code-server -n test --set=ingress.hosts[0].host=7a759e6b-d8d5-4a3a-8427-9da68b0983f5.u-code.io --set=ingress.tls[0].hosts[0]=7a759e6b-d8d5-4a3a-8427-9da68b0983f5.u-code.io --set=ingress.tls[0].secretName=ucode-wildcard --set=ingress.hosts[0].paths[0]=/
	err = cmd.Run()
	isErr := !strings.HasPrefix(stderr.String(), "WARNING:")
	if err != nil && isErr {
		fmt.Println("err 2::", err)
		log.Println("error exec command:", cmd.String())
		return "", errors.New("error while install code server::" + stderr.String())
	}
	// var out bytes.Buffer

	// log.Println("----exec command-----", cmd.String())
	// cmd = exec.Command("kubectl", "get", "secret", "--namespace", "test", functionName+"-code-server", "-o", "jsonpath=\"{.data.password}\"")

	// cmd.Stdout = &out

	// err = cmd.Run()
	// if err != nil {
	// 	log.Println("err 3::", err)
	// 	return "", errors.New("error running get password command::" + stderr.String())
	// }

	// s := strings.ReplaceAll(out.String(), `"`, "")
	// str, err := base64.StdEncoding.DecodeString(s)
	// if err != nil {
	// 	log.Println("err 4::", err)
	// 	return "", errors.New("error while base64 to string::" + stderr.String())
	// }
	// pass := string(str)

	return "", nil
}

func DeleteCodeServer(ctx context.Context, srvs services.ServiceManagerI, cfg config.Config) error {

	functions, err := srvs.FunctionService().FunctionService().GetListByRequestTime(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	var ids = make([]string, 0, functions.GetCount())
	for _, function := range functions.GetFunctions() {
		var stdout bytes.Buffer
		cmd := exec.Command("helm", "uninstall", function.Path, "-n", "test")
		err = cmd.Run()
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout
		if err != nil {
			log.Println(stdout.String())
			log.Println("err "+function.GetPath(), ":: ", err)
			log.Println("error while uninstalling " + function.GetPath() + " error: " + stderr.String())
			continue
		}
		ids = append(ids, function.GetId())
		fmt.Println(function.GetPath())
	}
	if len(ids) > 0 {
		_, err = srvs.FunctionService().FunctionService().UpdateManyByRequestTime(context.Background(), &new_function_service.UpdateManyUrlAndPassword{
			Ids: ids,
		})
		if err != nil {
			return err
		}
	}
	fmt.Println("finish")

	return nil
}

func DeleteCodeServerByPath(path string, cfg config.Config) error {

	var stdout bytes.Buffer
	cmd := exec.Command("helm", "uninstall", path, "-n", "test")
	err := cmd.Run()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err != nil {
		log.Println(stdout.String())
		log.Println("err "+path, ":: ", err)
		log.Println("error while uninstalling " + path + " error: " + stderr.String())
		return err
	}

	return nil
}
