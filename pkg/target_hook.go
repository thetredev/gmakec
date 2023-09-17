package gmakec

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

type TargetHook struct {
	Shell   string `yaml:"shell"`
	Step    string `yaml:"step"`
	Command string `yaml:"command"`
}

func (this *TargetHook) execute(command *exec.Cmd, workingDir string) {
	command.Dir = workingDir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
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
	this.execute(command, workingDir)
}

func (this *TargetHook) executeWithoutShell(workingDir string) {
	args := strings.Split(this.Command, " ")
	this.execute(exec.Command(args[0], args[1:]...), workingDir)
}

func (this *TargetHook) Execute(workingDir string) {
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
