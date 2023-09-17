package gmakec

import (
	"os/exec"
	"strings"
)

type TargetHook struct {
	Shell   string `yaml:"shell"`
	Step    string `yaml:"step"`
	Command string `yaml:"command"`
}

func (this *TargetHook) executeWithShell(shellString string, workingDir string) error {
	commandPrefix := strings.Split(shellString, " ")
	var command *exec.Cmd

	if len(commandPrefix) > 1 {
		command = exec.Command(commandPrefix[0], commandPrefix[1])
	} else {
		command = exec.Command(commandPrefix[0])
	}

	command.Args = append(command.Args, this.Command)
	return executeCommand(command, workingDir)
}

func (this *TargetHook) executeWithoutShell(workingDir string) error {
	args := strings.Split(this.Command, " ")
	command := exec.Command(args[0])

	if len(args) > 1 {
		command.Args = append(command.Args, args[1:]...)
	}

	return executeCommand(command, workingDir)
}

func (this *TargetHook) execute(workingDir string) error {
	shellString := this.Shell

	if shellString == "none" {
		shellString = ""
	}

	if len(shellString) > 0 {
		return this.executeWithShell(shellString, workingDir)
	}

	return this.executeWithoutShell(workingDir)
}
