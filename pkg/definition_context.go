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
		ConfigureDir:   filepath.Join(definitionPath, CONFIGURE_DIR),
	}

	if err = defContext.Definition.sanitize(defContext); err != nil {
		return nil, err
	}

	return defContext, nil
}

func (this *DefinitionContext) Configure(definitionContexts *[]*DefinitionContext) error {
	graphs := this.Definition.generateDependencyGraphs()
	targetGroupMatrix := generateTargetGroupMatrix(graphs)

	RemovePath(this.ConfigureDir)
	if err := os.MkdirAll(this.ConfigureDir, os.ModePerm); err != nil {
		return err
	}

	for index, targetGroupIndices := range targetGroupMatrix {
		targetGroup := &TargetGroup{
			Targets: targetGroupIndices,
		}

		targets, err := targetGroup.configure(this, definitionContexts)

		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("%s/%d", this.ConfigureDir, index)
		file, err := os.Create(filePath)

		if err != nil {
			return err
		}

		defer file.Close()

		for _, target := range targets {
			contents, err := target.buildCommand()

			if err != nil {
				return err
			}

			_, err = file.WriteString(fmt.Sprintf("%s\n", contents))

			if err != nil {
				return err
			}
		}
	}

	return nil
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

				if shellCommand[1] == "skip" {
					fmt.Printf("[build] Skipping target %d of target group %s\n", targetIndex, filepath.Base(name))
					continue
				}

				targetDef := this.Definition.Targets[targetIndex]

				if err := targetDef.executeHooks("pre-build", this.DefinitionPath); err != nil {
					log.Fatal(err)
				}

				outputDir := filepath.Dir(filepath.Join(this.DefinitionPath, targetDef.Output))

				if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
					log.Fatal(err)
				}

				if verbose {
					shellCommand = slices.Insert(shellCommand, 3, "-v")
				}

				command := exec.Command(shellCommand[2], shellCommand[3:]...)

				if err := executeCommand(command, this.DefinitionPath); err != nil {
					log.Fatal(err)
				}

				if err := targetDef.executeHooks("post-build", this.DefinitionPath); err != nil {
					log.Fatal(err)
				}
			}
		}(strings.Split(string(bytes), "\n"))
		return nil
	})

	wg.Wait()
	return err
}
