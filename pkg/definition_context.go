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

	if err = defContext.Sanitize(); err != nil {
		return nil, err
	}

	return defContext, nil
}

func (defContext *DefinitionContext) Sanitize() error {
	for index, compilerDef := range defContext.Definition.Compilers {
		if len(compilerDef.Path) == 0 {
			return fmt.Errorf("Global compiler definition of name `%s` (index %d) need to have the field `path` set!", compilerDef.Name, index)
		}
		if len(compilerDef.Name) == 0 {
			return fmt.Errorf("Global compiler definition with path `%s` (index %d) need to have the field `name` set!", compilerDef.Path, index)
		}
	}

	return nil
}

func generateTargetGroupMatrix(graphs [][]int) [][]int {
	targetGroupMatrix := [][]int{}

	for i := len(graphs) - 1; i >= 0; i-- {
		graph := graphs[i]
		mergedGraph := graph

		for _, graphIndex := range graph {
			for j, outerGraph := range graphs {
				if i == j {
					break
				}

				found := false

				for _, outerGraphIndex := range outerGraph {
					toAdd := -1

					if graphIndex == outerGraphIndex {
						found = true
						toAdd = graphIndex
					} else if found {
						toAdd = outerGraphIndex
					}

					if toAdd > -1 && !slices.Contains(mergedGraph, toAdd) {
						mergedGraph = append(mergedGraph, toAdd)
					}
				}
			}
		}

		if slices.Compare(graph, mergedGraph) != 0 {
			targetGroupMatrix = append(targetGroupMatrix, mergedGraph)
		} else {
			isRemainder := false

			for _, sortedGraphItem := range targetGroupMatrix {
				for _, mergedGraphIndex := range mergedGraph {
					if slices.Contains(sortedGraphItem, mergedGraphIndex) {
						isRemainder = true
						break
					}
				}

				if isRemainder {
					break
				}
			}

			if !isRemainder {
				targetGroupMatrix = append(targetGroupMatrix, mergedGraph)
			}
		}
	}

	return targetGroupMatrix
}

func FindRefTarget(targetName string, defContexts *[]*DefinitionContext) (*DefinitionContext, *TargetDefinition) {
	for _, defContext := range *defContexts {
		for index := range defContext.Definition.Targets {
			if defContext.Definition.Targets[index].Name == targetName {
				return defContext, &defContext.Definition.Targets[index]
			}
		}
	}

	return nil, nil
}

func FindRefData(refString string, defContexts *[]*DefinitionContext) (string, *DefinitionContext, *TargetDefinition, error) {
	ref := strings.Split(refString, ":")

	if len(ref) < 2 {
		return "", nil, nil, fmt.Errorf("Incorrect format for target reference: `%s`!", refString)
	}

	refContext, refTarget := FindRefTarget(ref[0], defContexts)

	if refContext == nil || refTarget == nil {
		return "", nil, nil, fmt.Errorf("Could not find referenced target of name `%s`!", ref[0])
	}

	return ref[1], refContext, refTarget, nil
}

func FindRefTargetStringValue(
	refString string, targetDef *TargetDefinition, defContexts *[]*DefinitionContext,
) (string, error) {
	fieldName, refContext, refTarget, err := FindRefData(refString, defContexts)

	if err != nil {
		return "", nil
	}

	fieldValue, err := refTarget.FieldStringValue(fieldName, refContext)

	if err != nil {
		return "", err
	}

	return fieldValue, nil
}

func FindRefTargetStringArrayValue(
	refString string, targetDef *TargetDefinition, defContexts *[]*DefinitionContext,
) ([]string, error) {
	fieldName, refContext, refTarget, err := FindRefData(refString, defContexts)

	if err != nil {
		return nil, nil
	}

	fieldValue, err := refTarget.FieldStringArrayValue(fieldName, refContext)

	if err != nil {
		return nil, err
	}

	return fieldValue, nil
}

func (defContext *DefinitionContext) IsConfigured(expectedFileCount int) (bool, error) {
	entries, err := os.ReadDir(defContext.ConfigureDir)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return len(entries) == expectedFileCount, nil
}

func (defContext *DefinitionContext) Configure(defContexts *[]*DefinitionContext) error {
	graphs := defContext.Definition.GenerateDependencyGraphs()
	targetGroupMatrix := generateTargetGroupMatrix(graphs)

	alreadyConfigured, err := defContext.IsConfigured(len(targetGroupMatrix))

	if err != nil {
		return err
	}

	if alreadyConfigured {
		return nil
	}

	if err = os.RemoveAll(defContext.ConfigureDir); err != nil {
		fmt.Printf("WARNING: could not remove directory %s: %s", defContext.ConfigureDir, err.Error())
	}

	if err = os.MkdirAll(defContext.ConfigureDir, os.ModePerm); err != nil {
		return err
	}

	for index, targetGroupIndices := range targetGroupMatrix {
		targetGroup := &TargetGroup{
			Targets: targetGroupIndices,
		}

		buildCommands, err := targetGroup.Configure(defContext, defContexts)

		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("%s/%d", defContext.ConfigureDir, index)
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

func (defContext *DefinitionContext) Build(verbose bool) error {
	var wg sync.WaitGroup

	err := filepath.Walk(defContext.ConfigureDir, func(name string, info os.FileInfo, err error) error {
		if name == defContext.ConfigureDir && info != nil && info.IsDir() {
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
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				targetDef := defContext.Definition.Targets[targetIndex]
				targetDef.ExecuteHooks("preBuild", defContext.DefinitionPath)

				outputDir := filepath.Dir(fmt.Sprintf("%s/%s", defContext.DefinitionPath, targetDef.Output))

				if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				if verbose {
					shellCommand = slices.Insert(shellCommand, 2, "-v")
				}

				command := exec.Command(shellCommand[1], shellCommand[2:]...)
				command.Dir = defContext.DefinitionPath
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				if err = command.Run(); err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				targetDef.ExecuteHooks("postBuild", defContext.DefinitionPath)
			}
		}(strings.Split(string(bytes), "\n"))
		return nil
	})

	wg.Wait()
	return err
}
