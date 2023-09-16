package gmakec

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type TargetHook struct {
	Step    string `yaml:"step"`
	Command string `yaml:"command"`
}

func (targetHook *TargetHook) Execute() {
	shell, err := parseUserShell()

	if err != nil {
		log.Fatalf("ERROR: %s\n", err.Error())
	}

	command := exec.Command(shell, "-c", targetHook.Command)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err = command.Run(); err != nil {
		log.Fatalf("ERROR: %s\n", err.Error())
	}
}

func parseUserShell() (string, error) {
	currentUser, err := user.Current()

	if err != nil {
		return "", err
	}

	passwd, err := os.Open("/etc/passwd")

	if err != nil {
		return "", err
	}

	defer passwd.Close()

	fileScanner := bufio.NewScanner(passwd)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		text := fileScanner.Text()

		if strings.HasPrefix(text, currentUser.Username) {
			ent := strings.Split(text, ":")
			return ent[len(ent)-1], nil
		}
	}

	panic("This line is never supposed to be reached on Linux! PANIC!!!")
}
