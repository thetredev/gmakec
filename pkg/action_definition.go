package gmakec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/exp/slices"
)

type ActionDefinition struct {
	Ref         string                 `yaml:"ref"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Shell       string                 `yaml:"shell"`
	Environment []string               `yaml:"environment"`
	Command     string                 `yaml:"command"`
	Output      ActionOutputDefinition `yaml:"output"`
}

func (this *ActionDefinition) commandWithShell(shellString string) *exec.Cmd {
	commandPrefix := strings.Split(shellString, " ")
	var command *exec.Cmd

	if len(commandPrefix) > 1 {
		command = exec.Command(commandPrefix[0], commandPrefix[1])
	} else {
		command = exec.Command(commandPrefix[0])
	}

	command.Args = append(command.Args, this.Command)
	return command
}

func (this *ActionDefinition) commandWithoutShell() *exec.Cmd {
	args := strings.Split(this.Command, " ")
	command := exec.Command(args[0])

	if len(args) > 1 {
		command.Args = append(command.Args, args[1:]...)
	}

	return command
}

func (this *ActionDefinition) execute(workingDir string) (bool, error) {
	if len(this.Command) == 0 {
		return false, fmt.Errorf("Command of action `%s` does not have a command attached to it!\n", this.Name)
	}

	shellString := this.Shell

	if shellString == "none" {
		shellString = ""
	}

	var command *exec.Cmd

	if len(shellString) > 0 {
		command = this.commandWithShell(shellString)
	} else {
		command = this.commandWithoutShell()
	}

	command.Dir = workingDir

	for _, environmentVariable := range this.Environment {
		command.Env = append(command.Env, os.ExpandEnv(environmentVariable))
	}

	captureStdout := slices.Contains(this.Output.Capture, "stdout")

	if captureStdout {
		command.Stdout = &this.Output.Stdout
	}

	captureStderr := slices.Contains(this.Output.Capture, "stderr")

	if captureStderr {
		command.Stderr = &this.Output.Stderr
	}

	command.Run()
	this.sanitizeOutput()

	return command.ProcessState.ExitCode() == 0, nil
}

func (this *ActionDefinition) sanitizedName() string {
	name := strings.ReplaceAll(this.Name, "-", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "?", "_")
	// TODO: catch all other cases

	if name == ".." {
		name = "__"
	}

	if name == "." {
		name = "_"
	}

	return name
}

func sanitizeActionText(text string, pattern string, replaceText string) string {
	if len(replaceText) == 0 {
		replaceText = "<none>\n"
	}

	return strings.ReplaceAll(text, pattern, replaceText)
}

type ActionDumpDefinition struct {
	File string `yaml:"file"`
	Text string `yaml:"text"`
	Mode string `yaml:"mode"`
}

func (this *ActionDumpDefinition) sanitize(actionDefinition *ActionDefinition) {
	this.File = sanitizeActionText(this.File, "<name:sanitized>", actionDefinition.sanitizedName())

	this.Text = sanitizeActionText(this.Text, "<capture:stdout>", actionDefinition.Output.Stdout.String())
	this.Text = sanitizeActionText(this.Text, "<capture:stderr>", actionDefinition.Output.Stderr.String())
}

type ActionHandleDefinition struct {
	Define string               `yaml:"define"`
	Print  string               `yaml:"print"`
	Dump   ActionDumpDefinition `yaml:"dump"`
}

func (this *ActionHandleDefinition) sanitize(actionOutputDefinition *ActionOutputDefinition) {
	stdout := strings.TrimSpace(strings.TrimSuffix(actionOutputDefinition.Stdout.String(), "\n"))
	stderr := strings.TrimSpace(strings.TrimSuffix(actionOutputDefinition.Stderr.String(), "\n"))

	this.Define = sanitizeActionText(this.Define, "<capture:stdout>", stdout)
	this.Define = sanitizeActionText(this.Define, "<capture:stderr>", stderr)

	this.Print = sanitizeActionText(this.Print, "<capture:stdout>", stdout)
	this.Print = sanitizeActionText(this.Print, "<capture:stderr>", stderr)
}

func (this *ActionHandleDefinition) handle(targetDefinition *TargetDefinition, step string) error {
	if len(this.Define) > 0 {
		targetDefinition.Defines = append(targetDefinition.Defines, this.Define)
	}

	if len(this.Print) > 0 {
		fmt.Printf("[%s:print] %s\n", step, this.Print)
	}

	if len(this.Dump.File) > 0 {
		if len(this.Dump.Text) == 0 {
			return fmt.Errorf(
				"ERROR: Could not execute hook for step `%s`: dump:text definition is empty!\n",
				step,
			)
		}

		if len(this.Dump.Mode) == 0 {
			this.Dump.Mode = "create"
		}

		flag := os.O_RDWR

		switch {
		case this.Dump.Mode == "create":
			flag |= os.O_CREATE
		case this.Dump.Mode == "append":
			flag |= os.O_APPEND
		case this.Dump.Mode == "delete":
		default:
			return fmt.Errorf(
				"ERROR: Unsupported dump:mode definition for hook step `%s`: `%s`\n",
				step, this.Dump.Mode,
			)
		}

		file, err := os.OpenFile(this.Dump.File, flag, os.ModePerm)

		if err != nil {
			return fmt.Errorf(
				"ERROR: Could not open file `%s` with mode `%s` for hook step `%s`: %s\n",
				this.Dump.File, this.Dump.Mode, step, err.Error(),
			)
		}

		defer file.Close()
		_, err = file.WriteString(this.Dump.Text)

		if err != nil {
			return fmt.Errorf(
				"ERROR: Could not write dump:text to file `%s` for hook step `%s`: %s\n",
				this.Dump.File, step, err.Error(),
			)
		}
	}

	return nil
}

type ActionFailureDefinition struct {
	Continue bool                     `yaml:"continue"`
	Handle   []ActionHandleDefinition `yaml:"handle"`
}

type ActionOutputDefinition struct {
	Capture   []string                 `yaml:"capture"`
	OnSuccess []ActionHandleDefinition `yaml:"on_success"`
	OnFailure ActionFailureDefinition  `yaml:"on_failure"`
	Stdout    bytes.Buffer
	Stderr    bytes.Buffer
}

func (this *ActionDefinition) findRef(refActionDefinitions *[]ActionDefinition) *ActionDefinition {
	for _, refActionDefinition := range *refActionDefinitions {
		if refActionDefinition.Name == this.Ref {
			return &refActionDefinition
		}
	}

	return nil
}

func (this *ActionDefinition) withRef(definitionContext *DefinitionContext) (*ActionDefinition, error) {
	if len(this.Ref) == 0 {
		return this, nil
	}

	actionRef := this.findRef(&definitionContext.Definition.Actions)

	if actionRef == nil {
		return nil, fmt.Errorf("Could not find referenced action `%s`!", this.Ref)
	}

	return actionRef, nil
}

func (this *ActionDefinition) sanitizeOutput() {
	for index := range this.Output.OnSuccess {
		this.Output.OnSuccess[index].sanitize(&this.Output)
		this.Output.OnSuccess[index].Dump.sanitize(this)
	}

	for index := range this.Output.OnFailure.Handle {
		this.Output.OnFailure.Handle[index].sanitize(&this.Output)
		this.Output.OnFailure.Handle[index].Dump.sanitize(this)
	}
}
