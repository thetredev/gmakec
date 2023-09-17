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

	if err = defContext.sanitize(); err != nil {
		return nil, err
	}

	return defContext, nil
}

func (this *DefinitionContext) sanitize() error {
	for index, compilerDef := range this.Definition.Compilers {
		if len(compilerDef.Path) == 0 {
			return fmt.Errorf(
				"Global compiler definition of name `%s` (index %d) need to have the field `path` set!",
				compilerDef.Name, index,
			)
		}
		if len(compilerDef.Name) == 0 {
			return fmt.Errorf(
				"Global compiler definition with path `%s` (index %d) need to have the field `name` set!",
				compilerDef.Path, index,
			)
		}
	}

	return nil
}

// Don't ask me why this works.
// This "algorithm" came about when I was trying to create the dependency graph.
// Maybe there's some maths formulae or algorithms which would improve this code.
// However, it works at the moment and is actually not slow (at least on my machine).
//
// Basically it figures out which targets to build first and puts them in a "matrix".
// Example:
//
//	Target index 0: no dependencies
//	Target index 1: dependency on target 0
//	Target index 2: no dependencies
//	Target index 3: dependency on target 1
//
// Would result in: [[0, 1, 3] [2]]
// Targets will be built in exactly the order of the two target groups (inner arrays).
// The target groups will also be built in parallel.
//
// I'm always open for suggestions. :)
// ~ thetredev
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

func FindRefTarget(targetName string, refDefinitionContexts *[]*DefinitionContext) (*DefinitionContext, *TargetDefinition) {
	for _, defContext := range *refDefinitionContexts {
		for index := range defContext.Definition.Targets {
			if defContext.Definition.Targets[index].Name == targetName {
				return defContext, &defContext.Definition.Targets[index]
			}
		}
	}

	return nil, nil
}

func FindRefData(refString string, definitionContexts *[]*DefinitionContext) (string, *DefinitionContext, *TargetDefinition, error) {
	ref := strings.Split(refString, ":")

	if len(ref) < 2 {
		return "", nil, nil, fmt.Errorf("Incorrect format for target reference: `%s`!", refString)
	}

	refContext, refTarget := FindRefTarget(ref[0], definitionContexts)

	if refContext == nil || refTarget == nil {
		return "", nil, nil, fmt.Errorf("Could not find referenced target of name `%s`!", ref[0])
	}

	return ref[1], refContext, refTarget, nil
}

func FindRefTargetStringValue(
	refString string, targetDefinition *TargetDefinition, definitionContexts *[]*DefinitionContext,
) (string, error) {
	fieldName, refContext, refTarget, err := FindRefData(refString, definitionContexts)

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
	refString string, targetDefinition *TargetDefinition, definitionContexts *[]*DefinitionContext,
) ([]string, error) {
	fieldName, refContext, refTarget, err := FindRefData(refString, definitionContexts)

	if err != nil {
		return nil, nil
	}

	fieldValue, err := refTarget.FieldStringArrayValue(fieldName, refContext)

	if err != nil {
		return nil, err
	}

	return fieldValue, nil
}

func (this *DefinitionContext) IsConfigured(expectedFileCount int) (bool, error) {
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
	graphs := this.Definition.GenerateDependencyGraphs()
	targetGroupMatrix := generateTargetGroupMatrix(graphs)

	alreadyConfigured, err := this.IsConfigured(len(targetGroupMatrix))

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

		buildCommands, err := targetGroup.Configure(this, definitionContexts)

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
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				targetDef := this.Definition.Targets[targetIndex]
				targetDef.ExecuteHooks("preBuild", this.DefinitionPath)

				outputDir := filepath.Dir(fmt.Sprintf("%s/%s", this.DefinitionPath, targetDef.Output))

				if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				if verbose {
					shellCommand = slices.Insert(shellCommand, 2, "-v")
				}

				command := exec.Command(shellCommand[1], shellCommand[2:]...)

				if err := ExecuteCommand(command, this.DefinitionPath); err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				targetDef.ExecuteHooks("postBuild", this.DefinitionPath)
			}
		}(strings.Split(string(bytes), "\n"))
		return nil
	})

	wg.Wait()
	return err
}
