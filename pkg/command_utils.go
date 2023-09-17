package gmakec

import (
	"os"
	"os/exec"
)

func ExecuteCommand(command *exec.Cmd, workingDir string) error {
	command.Dir = workingDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}
