package gitlab_integration

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"ucode/ucode_go_api_gateway/config"
)

func CloneForkToPath(path string, cfg config.Config) error {
	path = strings.TrimPrefix(path, "https://")
	command := fmt.Sprintf("https://oauth:%s@%s", cfg.GitlabIntegrationToken, path)
	cmd := exec.Command("git", "clone", command)
	fmt.Println("path clone:::", cfg.PathToClone)
	cmd.Dir = cfg.PathToClone
	fmt.Println("test clone")
	err := cmd.Run()
	if err != nil {
		return errors.New("could not clone repo into given path::" + err.Error())
	}
	fmt.Println("test clone 22")
	fmt.Println("Output::")

	return nil
}
