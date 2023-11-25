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
	"ucode/ucode_go_api_gateway/genproto/company_service"
	pb "ucode/ucode_go_api_gateway/genproto/new_function_service"
	"ucode/ucode_go_api_gateway/services"
)

type CodeServer struct {
	Id             string
	Name           string
	HelmRepoAdd    string
	HelmRepoUpdate string
	HelmInstall    string
	HelmUninstall  string
}

func CreateCodeServer(functionName string, cfg config.BaseConfig, id string) (string, error) {

	// command := fmt.Sprintf("--username udevs --password %s code-server https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable", cfg.GitlabIntegrationToken)
	cmd := exec.Command("helm", "repo", "add", "--username", "udevs", "--password", cfg.GitlabIntegrationToken, "code-server", "https://gitlab.udevs.io/api/v4/projects/1512/packages/helm/stable")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// log.Println("----exec command-----", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Println("err 0::", err)
		return "", errors.New("error while adding repo:" + stderr.String())
	}

	cmd = exec.Command("helm", "repo", "update")
	// log.Println("----exec command-----", cmd.String())
	err = cmd.Run()
	if err != nil {
		log.Println("error exec command:", cmd.String())
		return "", errors.New("error while repo update helm::" + stderr.String())
	}

	hostName := fmt.Sprintf("--set=ingress.hosts[0].host=%s.u-code.io", id)
	hostNameTls := fmt.Sprintf("--set=ingress.tls[0].hosts[0]=%s.u-code.io", id)
	// secretName := fmt.Sprintf("--set=ingress.tls[0].secretName=%s-tls", id)
	secretName := "--set=ingress.tls[0].secretName=ucode-wildcard"

	path := "--set=ingress.hosts[0].paths[0]=/"

	cmd = exec.Command("helm", "install", functionName, "code-server/code-server", "-n", "test", hostName, hostNameTls, secretName, path)
	// log.Println("----exec command-----", cmd.String())

	// helm install newnewnew-sdfsdfsdf-doupdate code-server/code-server -n test --set=ingress.hosts[0].host=7a759e6b-d8d5-4a3a-8427-9da68b0983f5.u-code.io --set=ingress.tls[0].hosts[0]=7a759e6b-d8d5-4a3a-8427-9da68b0983f5.u-code.io --set=ingress.tls[0].secretName=ucode-wildcard --set=ingress.hosts[0].paths[0]=/
	err = cmd.Run()
	isErr := !strings.HasPrefix(stderr.String(), "WARNING:")
	if err != nil && isErr {
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

	// fmt.Println("finish")

	return "", nil
}

func CreateCodeServerV2(data CodeServer) error {
	var (
		stderr bytes.Buffer
		stdout bytes.Buffer
	)

	if len(data.HelmRepoAdd) < 2 || len(data.HelmRepoUpdate) < 2 || len(data.HelmInstall) < 2 {
		err := errors.New("invalid helm command")
		return err
	}
	helmRepoAdd := strings.Split(strings.ReplaceAll(data.HelmRepoAdd, "  ", " "), " ")
	helmRepoUpdate := strings.Split(strings.ReplaceAll(data.HelmRepoUpdate, "  ", " "), " ")

	cmd := exec.Command(helmRepoAdd[0], helmRepoAdd[1:]...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	// log.Println("----exec command----- [helmRepoAdd]", cmd.String())

	err := cmd.Run()
	if err != nil {
		log.Println("error [helmRepoAdd]", err)
		return errors.New("error while adding repo:" + stderr.String())
	}

	cmd = exec.Command(helmRepoUpdate[0], helmRepoUpdate[1:]...)

	// log.Println("----exec command----- [helmRepoUpdate]", cmd.String())

	err = cmd.Run()
	if err != nil {
		// log.Println("error [helmRepoUpdate]", err)
		return errors.New("error while update repo:" + stderr.String())
	}

	data.HelmInstall = strings.ReplaceAll(data.HelmInstall, "{id}", data.Id)
	data.HelmInstall = strings.ReplaceAll(data.HelmInstall, "{name}", data.Name)

	helmInstall := strings.Split(strings.ReplaceAll(data.HelmInstall, "  ", " "), " ")

	cmd = exec.Command(helmInstall[0], helmInstall[1:]...)

	// log.Println("----exec command----- [helmInstall]", cmd.String())

	err = cmd.Run()
	isErr := !strings.HasPrefix(stderr.String(), "WARNING:")
	if err != nil && isErr {
		log.Println("error [helmInstall]", err)
		return errors.New("error while install code server:" + stderr.String())
	}

	// log.Println("finish [code-server]")

	return nil
}

func DeleteCodeServer(ctx context.Context, srvs services.ServiceManagerI, cfg config.BaseConfig, comp services.CompanyServiceI) error {
	log.Println("!!!---DeleteCodeServer--->")
	var (
		allFunctions = make([]*pb.Function, 0)
		ids          = make([]string, 0)
	)

	req := &company_service.GetListResourceEnvironmentReq{}
	resEnvsIds, err := comp.Resource().GetListResourceEnvironment(ctx, req)
	if err != nil {
		log.Println("error while getting resource environments")
		return err
	}

	if len(resEnvsIds.GetData()) == 0 {
		log.Println("no resource environments")
		return nil
	}

	fmt.Println("test length resource::::", len(resEnvsIds.GetData()))
	for _, v := range resEnvsIds.GetData() {
		functions, err := srvs.FunctionService().FunctionService().GetListByRequestTime(context.Background(), &pb.GetListByRequestTimeRequest{
			ProjectId: v.GetId(),
			Type:      "FUNCTION",
		})
		if err != nil {
			log.Println("error while getting functions for project id: "+v.GetProjectId(), err.Error())
			continue
		}

		allFunctions = append(allFunctions, functions.GetFunctions()...)
	}

	if err != nil {
		return err
	}
	for _, function := range allFunctions {
		log.Println("uninstalling func " + function.GetPath())
		var stdout bytes.Buffer

		cmd := exec.Command("helm", "uninstall", function.Path, "-n", "test")
		log.Println(" --- exec command --- ", cmd.String())
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
		log.Println("successfully uninstalled " + function.GetPath())
		ids = append(ids, function.GetId())
	}
	for _, v := range resEnvsIds.GetData() {
		if len(ids) > 0 {
			_, err = srvs.FunctionService().FunctionService().UpdateManyByRequestTime(context.Background(), &pb.UpdateManyUrlAndPassword{
				Ids:       ids,
				ProjectId: v.GetId(),
			})
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("finish")

	return nil
}

func DeleteCodeServerByPath(path string, cfg config.BaseConfig) error {

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
