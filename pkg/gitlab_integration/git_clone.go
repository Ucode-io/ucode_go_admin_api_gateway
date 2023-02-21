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
	cmd := exec.Command("git", "-c", "core.sshCommand=ssh -i /key/ssh-privatekey", "clone", path)
	fmt.Println("path clone:::", cfg.PathToClone)
	cmd.Dir = cfg.PathToClone
	fmt.Println("test clone")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New("could not clone repo into given path::" + stderr.String())
	}
	fmt.Println("test clone 22")
	fmt.Println("Output::")

	return nil
}
