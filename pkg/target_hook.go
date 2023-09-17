package gmakec

import (
	"log"
	"os/exec"
	"strings"
)

type TargetHook struct {
	Shell   string `yaml:"shell"`
	Step    string `yaml:"step"`
	Command string `yaml:"command"`
}

func (this *TargetHook) doExecute(command *exec.Cmd, workingDir string) {
	if err := executeCommand(command, workingDir); err != nil {
		log.Fatalf("ERROR: %s\n", err.Error())
	}
}

func (this *TargetHook) executeWithShell(shellString string, workingDir string) {
	commandPrefix := strings.Split(shellString, " ")
	var command *exec.Cmd

	if len(commandPrefix) > 1 {
		command = exec.Command(commandPrefix[0], commandPrefix[1])
	} else {
		command = exec.Command(commandPrefix[0])
	}

	command.Args = append(command.Args, this.Command)
	this.doExecute(command, workingDir)
}

func (this *TargetHook) executeWithoutShell(workingDir string) {
	args := strings.Split(this.Command, " ")
	command := exec.Command(args[0])

	if len(args) > 1 {
		command.Args = append(command.Args, args[1:]...)
	}

	this.doExecute(command, workingDir)
}

func (this *TargetHook) execute(workingDir string) {
	shellString := this.Shell

	if shellString == "none" {
		shellString = ""
	}

	if len(shellString) > 0 {
		this.executeWithShell(shellString, workingDir)
	} else {
		this.executeWithoutShell(workingDir)
	}
}
