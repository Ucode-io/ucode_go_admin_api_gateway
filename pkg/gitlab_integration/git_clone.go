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
	cmd.Dir = cfg.PathToClone
	err := cmd.Run()
	if err != nil {
		return errors.New("could not clone repo into given path::" + err.Error())
	}
	fmt.Println("Output::")

	return nil
}
