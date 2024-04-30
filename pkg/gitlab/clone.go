package gitlab

import (
	"bytes"
	"errors"
	"os/exec"
	"ucode/ucode_go_api_gateway/config"
)

func CloneForkToPath(path string, cfg config.BaseConfig) error {

	//git -c core.sshCommand="ssh -i /key/ssh-privatekey” clone git@gitlab.udevs.io:ucode/ucode_go_admin_api_gateway.git
	// path = strings.TrimPrefix(path, "https://")
	// command := fmt.Sprintf("https://oauth:%s@%s", cfg.GitlabIntegrationToken, path)
	// cmd := exec.Command("git", "-c", "core.sshCommand=\"ssh -i /key/ssh-privatekey\"", "clone", path)
	// sshCommand := "GIT_SSH_COMMAND='ssh -i /key/ssh-privatekey -o IdentitiesOnly=yes'"
	cmd := exec.Command("git", "clone", path) //path ssh url
	cmd.Dir = cfg.PathToClone
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New("could not clone repo into given path::" + stderr.String())
	}

	return nil
}

func CloneForkToPathV2(path string, cfg config.Config) error {

	//git -c core.sshCommand="ssh -i /key/ssh-privatekey” clone git@gitlab.udevs.io:ucode/ucode_go_admin_api_gateway.git
	// path = strings.TrimPrefix(path, "https://")
	// command := fmt.Sprintf("https://oauth:%s@%s", cfg.GitlabIntegrationToken, path)
	// cmd := exec.Command("git", "-c", "core.sshCommand=\"ssh -i /key/ssh-privatekey\"", "clone", path)
	// sshCommand := "GIT_SSH_COMMAND='ssh -i /key/ssh-privatekey -o IdentitiesOnly=yes'"
	cmd := exec.Command("git", "clone", path) //path ssh url
	cmd.Dir = cfg.PathToClone
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New("could not clone repo into given path::" + stderr.String())
	}

	return nil
}

func DeletedClonedRepoByPath(path string, cfg config.BaseConfig) error {

	//git -c core.sshCommand="ssh -i /key/ssh-privatekey” clone git@gitlab.udevs.io:ucode/ucode_go_admin_api_gateway.git
	// path = strings.TrimPrefix(path, "https://")
	// command := fmt.Sprintf("https://oauth:%s@%s", cfg.GitlabIntegrationToken, path)
	// cmd := exec.Command("git", "-c", "core.sshCommand=\"ssh -i /key/ssh-privatekey\"", "clone", path)
	// sshCommand := "GIT_SSH_COMMAND='ssh -i /key/ssh-privatekey -o IdentitiesOnly=yes'"
	cmd := exec.Command("rm", "-rf", path) //path ssh url
	cmd.Dir = cfg.PathToClone
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New("could not delete cloned repo into given path::" + stderr.String())
	}

	return nil
}
