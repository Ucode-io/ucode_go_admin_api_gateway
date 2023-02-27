package gitlab_integration

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"ucode/ucode_go_api_gateway/config"
)

func CloneForkToPath(path string, cfg config.Config) error {

	//git -c core.sshCommand="ssh -i /key/ssh-privatekey” clone git@gitlab.udevs.io:ucode/ucode_go_admin_api_gateway.git
	// path = strings.TrimPrefix(path, "https://")
	// command := fmt.Sprintf("https://oauth:%s@%s", cfg.GitlabIntegrationToken, path)
	fmt.Println("ssh url::", path)
	// cmd := exec.Command("git", "-c", "core.sshCommand=\"ssh -i /key/ssh-privatekey\"", "clone", path)
	// sshCommand := "GIT_SSH_COMMAND='ssh -i /key/ssh-privatekey -o IdentitiesOnly=yes'"
	cmd := exec.Command("git", "clone", path) //path ssh url
	fmt.Println("path clone:::", cfg.PathToClone)
	cmd.Dir = cfg.PathToClone
	fmt.Println("test clone")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("err:::", err)
		return errors.New("could not clone repo into given path::" + stderr.String())
	}
	fmt.Println("test clone 22")
	fmt.Println("Output::")

	return nil
}

func DeletedClonedRepoByPath(path string, cfg config.Config) error {

	//git -c core.sshCommand="ssh -i /key/ssh-privatekey” clone git@gitlab.udevs.io:ucode/ucode_go_admin_api_gateway.git
	// path = strings.TrimPrefix(path, "https://")
	// command := fmt.Sprintf("https://oauth:%s@%s", cfg.GitlabIntegrationToken, path)
	fmt.Println("ssh url::", path)
	// cmd := exec.Command("git", "-c", "core.sshCommand=\"ssh -i /key/ssh-privatekey\"", "clone", path)
	// sshCommand := "GIT_SSH_COMMAND='ssh -i /key/ssh-privatekey -o IdentitiesOnly=yes'"
	cmd := exec.Command("rm", "-rf", path) //path ssh url
	fmt.Println("deleted cloned repo:::", cfg.PathToClone)
	cmd.Dir = cfg.PathToClone
	fmt.Println("delete cloned repo")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("err:::", err)
		return errors.New("could not delete cloned repo into given path::" + stderr.String())
	}
	fmt.Println("test delete clone 22")
	fmt.Println("Output::")

	return nil
}
