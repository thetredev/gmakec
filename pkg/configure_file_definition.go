package gmakec

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type ConfigureFile struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

func (this *ConfigureFile) configureVariable(key string, definitionContext *DefinitionContext) (string, error) {
	// TODO: make this "cmake complete" and probably more fleixble...
	switch {
	case key == "PROJECT_VERSION":
		return definitionContext.Definition.Version, nil
	case key == "PROJECT_VERSION_MAJOR":
		return definitionContext.Definition.VersionMajor, nil
	case key == "PROJECT_VERSION_MINOR":
		return definitionContext.Definition.VersionMinor, nil
	case key == "PROJECT_VERSION_PATCH":
		return definitionContext.Definition.VersionPatch, nil
	case key == "PROJECT_VERSION_TWEAK":
		return definitionContext.Definition.VersionTweak, nil
	}

	return key, fmt.Errorf("WARNING: Could not find key `%s` to configure file `%s`!\n", key, this.Source)
}

func (this *ConfigureFile) Execute(definitionContext *DefinitionContext) error {
	source, err := os.Open(this.Source)

	if err != nil {
		return err
	}

	defer source.Close()

	if err = os.RemoveAll(this.Destination); err != nil {
		log.Printf("WARNING: Could not remove destination file `%s`!\n", this.Destination)
	}

	destination, err := os.Create(this.Destination)

	if err != nil {
		return err
	}

	defer destination.Close()

	fileScanner := bufio.NewScanner(source)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()

		if strings.HasPrefix(line, "#define") && strings.Count(line, "@") == 2 {
			start := strings.Index(line, "@") + 1
			end := strings.Index(line[start:], "@")

			targetString := line[start : start+end]
			value, err := this.configureVariable(targetString, definitionContext)

			if err != nil {
				log.Printf(err.Error())
			}

			line = strings.ReplaceAll(line, fmt.Sprintf("@%s@", targetString), value)
		}

		destination.WriteString(fmt.Sprintf("%s\n", line))
	}

	return nil
}
