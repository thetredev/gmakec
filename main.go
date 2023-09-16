package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/structs"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"
)

const CONFIGURE_DIR string = ".gmakec"
const GLOBAL_DEFINITION_YAML string = "gmakec.yaml"

type CompilerDefinition struct {
	Name  string   `yaml:"name"`
	Ref   string   `yaml:"ref"`
	Path  string   `yaml:"path"`
	Flags []string `yaml:"flags"`
}

func (compilerDef *CompilerDefinition) FindRef(globalCompilerDefs *[]CompilerDefinition) *CompilerDefinition {
	for _, globalCompilerDef := range *globalCompilerDefs {
		if globalCompilerDef.Name == compilerDef.Ref {
			return &globalCompilerDef
		}
	}

	return nil
}

func (compilerDef *CompilerDefinition) WithRef(globalCompilerDefs *[]CompilerDefinition) (*CompilerDefinition, error) {
	if len(compilerDef.Ref) == 0 {
		if len(compilerDef.Path) == 0 {
			return nil, fmt.Errorf("Non-ref compiler definition of name `%s` need to have the field `path` set!", compilerDef.Name)
		}

		path, err := exec.LookPath(compilerDef.Path)

		if err != nil {
			return nil, fmt.Errorf("Non-ref compiler path `%s` not found!", compilerDef.Path)
		}

		compilerDef.Path = path
		return compilerDef, nil
	}

	compilerRef := compilerDef.FindRef(globalCompilerDefs)

	if compilerRef == nil {
		return nil, fmt.Errorf("Could not find compiler ref: %s\n", compilerDef.Ref)
	}

	return &CompilerDefinition{
		Name:  compilerRef.Name,
		Path:  compilerRef.Path,
		Flags: append(compilerRef.Flags, compilerDef.Flags...),
	}, nil
}

type LinkDefinition struct {
	Path string `yaml:"path"`
	Link string `yaml:"link"`
}

type TargetDefinition struct {
	Name         string             `yaml:"name"`
	Compiler     CompilerDefinition `yaml:"compiler"`
	Sources      []string           `yaml:"sources"`
	Includes     []string           `yaml:"includes"`
	Links        []LinkDefinition   `yaml:"links"`
	Output       string             `yaml:"output"`
	Dependencies []string           `yaml:"dependencies"`
}

func (targetDef *TargetDefinition) FieldStringValue(fieldName string) (string, error) {
	fields := structs.Fields(targetDef)

	for _, field := range fields {
		tag := field.Tag("yaml")

		if tag == fieldName {
			return fmt.Sprintf("%s", field.Value()), nil
		}
	}

	return "", fmt.Errorf("Could not find field `%s`", fieldName)
}

func (targetDef *TargetDefinition) DependencyGraph(index int, targetDefs *[]TargetDefinition) []int {
	dependencyGraph := []int{index}

	if len(targetDef.Dependencies) > 0 {
		for i, otherTarget := range *targetDefs {
			if slices.Contains(targetDef.Dependencies, otherTarget.Name) {
				dependencyGraph = append(dependencyGraph, i)
			}
		}
	}

	return dependencyGraph
}

type TargetGroup struct {
	Targets []int
}

func (targetGroup *TargetGroup) Configure(globalDef *GlobalDefinition) ([]string, error) {
	buildCommands := []string{}

	for i := len(targetGroup.Targets) - 1; i >= 0; i-- {
		index := targetGroup.Targets[i]
		targetDef := globalDef.Targets[index]

		// merge compiler flags
		compilerDef, err := targetDef.Compiler.WithRef(&globalDef.Compilers)

		if err != nil {
			return nil, err
		}

		buildCommand := []string{compilerDef.Path}
		buildCommand = append(buildCommand, compilerDef.Flags...)

		for _, include := range targetDef.Includes {
			buildCommand = append(buildCommand, "-I")
			buildCommand = append(buildCommand, include)
		}

		for _, link := range targetDef.Links {
			if len(link.Path) > 0 {
				buildCommand = append(buildCommand, "-L")
				linkPath := link.Path

				if strings.Contains(linkPath, ":") {
					linkPath, err = globalDef.RefTargetStringValue(linkPath, &targetDef)
				}

				buildCommand = append(buildCommand, filepath.Dir(linkPath))
			}

			buildCommand = append(buildCommand, link.Link)
		}

		buildCommand = append(buildCommand, "-o")
		buildCommand = append(buildCommand, targetDef.Output)

		if err := os.MkdirAll(filepath.Dir(targetDef.Output), os.ModePerm); err != nil {
			return nil, err
		}

		for _, source := range targetDef.Sources {
			if strings.Contains(source, "*") {
				globbed, err := filepath.Glob(source)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, globbed...)
			} else if strings.Contains(source, ":") {
				refStringValue, err := globalDef.RefTargetStringValue(source, &targetDef)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, refStringValue)
			} else {
				buildCommand = append(buildCommand, source)
			}
		}

		buildCommands = append(buildCommands, strings.Join(buildCommand, " "))
	}

	return buildCommands, nil
}

type GlobalDefinition struct {
	Description string               `yaml:"description"` // unused atm
	Version     string               `yaml:"version"`     // unused atm
	Compilers   []CompilerDefinition `yaml:"compilers"`
	Targets     []TargetDefinition   `yaml:"targets"`
}

func (globalDef *GlobalDefinition) GenerateDependencyGraphs() [][]int {
	graphs := [][]int{}

	for index := range globalDef.Targets {
		graphs = append(graphs, globalDef.Targets[index].DependencyGraph(index, &globalDef.Targets))
	}

	return graphs
}

func (globalDef *GlobalDefinition) FindRefTarget(targetName string) *TargetDefinition {
	for index := range globalDef.Targets {
		if globalDef.Targets[index].Name == targetName {
			return &globalDef.Targets[index]
		}
	}

	return nil
}

func (globalDef *GlobalDefinition) RefTargetStringValue(refString string, targetDef *TargetDefinition) (string, error) {
	ref := strings.Split(refString, ":")
	refTarget := globalDef.FindRefTarget(ref[0])
	refFieldValue, err := refTarget.FieldStringValue(ref[1])

	if err != nil {
		return "", err
	}

	return refFieldValue, nil
}

func parseYaml() (*GlobalDefinition, error) {
	yamlFile, err := os.ReadFile(GLOBAL_DEFINITION_YAML)

	if err != nil {
		return nil, err
	}

	globalDef := GlobalDefinition{}
	err = yaml.Unmarshal(yamlFile, &globalDef)

	if err != nil {
		return nil, err
	}

	return &globalDef, nil
}

func sanitize(globalDef *GlobalDefinition) error {
	for index, compilerDef := range globalDef.Compilers {
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

func isConfigured(expectedFileCount int) (bool, error) {
	entries, err := os.ReadDir(CONFIGURE_DIR)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return len(entries) == expectedFileCount, nil
}

func configure(context *cli.Context) error {
	globalDef, err := parseYaml()

	if err != nil {
		return err
	}

	if err = sanitize(globalDef); err != nil {
		return err
	}

	graphs := globalDef.GenerateDependencyGraphs()
	targetGroupMatrix := generateTargetGroupMatrix(graphs)

	alreadyConfigured, err := isConfigured(len(targetGroupMatrix))

	if err != nil {
		return err
	}

	if alreadyConfigured {
		return nil
	}

	if err = os.RemoveAll(CONFIGURE_DIR); err != nil {
		fmt.Printf("WARNING: could not remove directory %s: %s", CONFIGURE_DIR, err.Error())
	}

	if err = os.MkdirAll(CONFIGURE_DIR, os.ModePerm); err != nil {
		return err
	}

	for index, targetGroupIndices := range targetGroupMatrix {
		targetGroup := &TargetGroup{targetGroupIndices}
		buildCommands, err := targetGroup.Configure(globalDef)

		if err != nil {
			return err
		}

		filePath := fmt.Sprintf("%s/%d", CONFIGURE_DIR, index)
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

	return nil
}

// only GCC for now
func build(context *cli.Context) error {
	err := configure(context)

	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	err = filepath.Walk(CONFIGURE_DIR, func(name string, info os.FileInfo, err error) error {
		if name == CONFIGURE_DIR && info != nil && info.IsDir() {
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

				command := exec.Command(shellCommand[0], shellCommand[1:]...)
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				if err = command.Run(); err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}
			}
		}(strings.Split(string(bytes), "\n"))
		return nil
	})

	wg.Wait()
	return nil
}

func clean(context *cli.Context) error {
	globalDef, err := parseYaml()

	if err != nil {
		return err
	}

	for _, targetDef := range globalDef.Targets {
		if err = os.RemoveAll(targetDef.Output); err != nil {
			fmt.Printf("WARNING: could not remove directory %s: %s", targetDef.Output, err.Error())
		}
	}

	if err = os.RemoveAll(CONFIGURE_DIR); err != nil {
		fmt.Printf("WARNING: could not remove directory %s: %s", CONFIGURE_DIR, err.Error())
	}

	return nil
}

func rebuild(context *cli.Context) error {
	var err error

	if err = clean(context); err != nil {
		return err
	}

	if err = build(context); err != nil {
		return err
	}

	return nil
}

func reconfigure(context *cli.Context) error {
	var err error

	if err = clean(context); err != nil {
		return err
	}

	if err = configure(context); err != nil {
		return err
	}

	return nil
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "configure",
				Usage:  "configure the project",
				Action: configure,
			},
			{
				Name:   "build",
				Usage:  "build the project",
				Action: build,
			},
			{
				Name:   "clean",
				Usage:  "rm -rf the output files",
				Action: clean,
			},
			{
				Name:   "reconfigure",
				Usage:  "Shorthand for clean + configure",
				Action: reconfigure,
			},
			{
				Name:   "rebuild",
				Usage:  "Shorthand for clean + build",
				Action: rebuild,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
