package gitlab_integration

import (
	"errors"
	"fmt"
	"os/exec"
	"ucode/ucode_go_api_gateway/config"
)

func CloneForkToPath(path string, cfg config.Config) error {

	cmd := exec.Command("git clone %s?%access_token=%s  ./%s", path, cfg.GitlabIntegrationToken, cfg.PathToClone)

	out, err := cmd.Output()
	if err != nil {
		return errors.New("could not clone repo into given path::" + err.Error())
	}
	fmt.Println("Output::", string(out))

	return nil
}
