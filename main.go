package main

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

	"github.com/urfave/cli/v2"

	gmakec "github.com/thetredev/gmakec/pkg"
)

const CONFIGURE_DIR string = ".gmakec"
const GLOBAL_DEFINITION_YAML string = "gmakec.yaml"

func parseYaml() (*gmakec.GlobalDefinition, error) {
	yamlFile, err := os.ReadFile(GLOBAL_DEFINITION_YAML)

	if err != nil {
		return nil, err
	}

	globalDef := gmakec.GlobalDefinition{}
	err = yaml.Unmarshal(yamlFile, &globalDef)

	if err != nil {
		return nil, err
	}

	return &globalDef, nil
}

func sanitize(globalDef *gmakec.GlobalDefinition) error {
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

func _configure() (*gmakec.GlobalDefinition, error) {
	globalDef, err := parseYaml()

	if err != nil {
		return nil, err
	}

	if err = sanitize(globalDef); err != nil {
		return nil, err
	}

	graphs := globalDef.GenerateDependencyGraphs()
	targetGroupMatrix := generateTargetGroupMatrix(graphs)

	alreadyConfigured, err := isConfigured(len(targetGroupMatrix))

	if err != nil {
		return nil, err
	}

	if alreadyConfigured {
		return nil, nil
	}

	if err = os.RemoveAll(CONFIGURE_DIR); err != nil {
		fmt.Printf("WARNING: could not remove directory %s: %s", CONFIGURE_DIR, err.Error())
	}

	if err = os.MkdirAll(CONFIGURE_DIR, os.ModePerm); err != nil {
		return nil, err
	}

	for index, targetGroupIndices := range targetGroupMatrix {
		targetGroup := &gmakec.TargetGroup{
			Targets: targetGroupIndices,
		}

		buildCommands, err := targetGroup.Configure(globalDef)

		for _, targetIndex := range targetGroupIndices {
			globalDef.Targets[targetIndex].ExecuteHooks("postConfigure")
		}

		if err != nil {
			return nil, err
		}

		filePath := fmt.Sprintf("%s/%d", CONFIGURE_DIR, index)
		file, err := os.Create(filePath)

		if err != nil {
			return nil, err
		}

		defer file.Close()

		for _, buildCommand := range buildCommands {
			_, err = file.WriteString(fmt.Sprintf("%s\n", buildCommand))

			if err != nil {
				return nil, err
			}
		}
	}

	return globalDef, nil
}

func configure(context *cli.Context) error {
	_, err := _configure()
	return err
}

// only GCC for now
func build(context *cli.Context) error {
	globalDef, err := _configure()

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
				targetIndex, err := strconv.Atoi(shellCommand[0])

				if err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				targetDef := globalDef.Targets[targetIndex]
				targetDef.ExecuteHooks("preBuild")

				command := exec.Command(shellCommand[1], shellCommand[2:]...)
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				if err = command.Run(); err != nil {
					log.Fatalf("ERROR: %s\n", err.Error())
				}

				targetDef.ExecuteHooks("postBuild")
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

	if _, err = _configure(); err != nil {
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
