package gmakec

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"
)

const CONFIGURE_DIR string = ".gmakec"

type DefinitionContext struct {
	DefinitionPath string
	Definition     *GlobalDefinition
	ConfigureDir   string
}

func NewDefinitionContext(path string) (*DefinitionContext, error) {
	yamlFile, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	globalDef := GlobalDefinition{}
	err = yaml.Unmarshal(yamlFile, &globalDef)

	if err != nil {
		return nil, err
	}

	definitionPath := filepath.Dir(path)

	defContext := &DefinitionContext{
		DefinitionPath: definitionPath,
		Definition:     &globalDef,
		ConfigureDir:   fmt.Sprintf("%s/%s", definitionPath, CONFIGURE_DIR),
	}

	if err = defContext.Definition.sanitize(defContext); err != nil {
		return nil, err
	}

	return defContext, nil
}

func (this *DefinitionContext) isConfigured(expectedFileCount int) (bool, error) {
	entries, err := os.ReadDir(this.ConfigureDir)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return len(entries) == expectedFileCount, nil
}

func (this *DefinitionContext) Configure(definitionContexts *[]*DefinitionContext) error {
	graphs := this.Definition.generateDependencyGraphs()
	targetGroupMatrix := generateTargetGroupMatrix(graphs)

	alreadyConfigured, err := this.isConfigured(len(targetGroupMatrix))

	if err != nil {
		return err
	}

	if alreadyConfigured {
		return nil
	}

	if err = os.RemoveAll(this.ConfigureDir); err != nil {
		fmt.Printf("WARNING: could not remove directory %s: %s", this.ConfigureDir, err.Error())
	}

	if err = os.MkdirAll(this.ConfigureDir, os.ModePerm); err != nil {
		return err
	}

	for index, targetGroupIndices := range targetGroupMatrix {
		targetGroup := &TargetGroup{
			Targets: targetGroupIndices,
		}

		buildCommands, err := targetGroup.configure(this, definitionContexts)

		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("%s/%d", this.ConfigureDir, index)
		file, err := os.Create(filePath)

		if err != nil {
			return err
		}

		defer file.Close()

		for _, buildCommand := range buildCommands {
			_, err = file.WriteString(fmt.Sprintf("%s\n", buildCommand))

			if err != nil {
				return err
			}
		}
	}

	return err
}

func (this *DefinitionContext) Build(verbose bool) error {
	var wg sync.WaitGroup

	err := filepath.Walk(this.ConfigureDir, func(name string, info os.FileInfo, err error) error {
		if name == this.ConfigureDir && info != nil && info.IsDir() {
			return nil
		}

		bytes, err := os.ReadFile(name)

		if err != nil {
			return err
		}

		wg.Add(1)

		go func(lines []string) {
			defer wg.Done()

			for _, line := range lines {
				if len(line) == 0 {
					continue
				}

				shellCommand := strings.Split(line, " ")
				targetIndex, err := strconv.Atoi(shellCommand[0])

				if err != nil {
					log.Fatal(err)
				}

				targetDef := this.Definition.Targets[targetIndex]

				if err := targetDef.executeHooks("preBuild", this.DefinitionPath); err != nil {
					log.Fatal(err)
				}

				outputDir := filepath.Dir(fmt.Sprintf("%s/%s", this.DefinitionPath, targetDef.Output))

				if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
					log.Fatal(err)
				}

				if verbose {
					shellCommand = slices.Insert(shellCommand, 2, "-v")
				}

				command := exec.Command(shellCommand[1], shellCommand[2:]...)

				if err := executeCommand(command, this.DefinitionPath); err != nil {
					log.Fatal(err)
				}

				if err := targetDef.executeHooks("postBuild", this.DefinitionPath); err != nil {
					log.Fatal(err)
				}
			}
		}(strings.Split(string(bytes), "\n"))
		return nil
	})

	wg.Wait()
	return err
}
