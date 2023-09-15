package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

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
	Compiler CompilerDefinition `yaml:"compiler"`
	Sources  []string           `yaml:"sources"`
	Includes []string           `yaml:"includes"`
	Links    []LinkDefinition   `yaml:"links"`
	Output   string             `yaml:"output"`
}

type GlobalDefinition struct {
	Description string               `yaml:"description"` // unused atm
	Version     string               `yaml:"version"`     // unused atm
	Compilers   []CompilerDefinition `yaml:"compilers"`
	Targets     []TargetDefinition   `yaml:"targets"`
}

// only GCC for now
func build(cCtx *cli.Context) error {
	yamlFile, err := os.ReadFile("gomakec.yaml")

	if err != nil {
		return err
	}

	c := &GlobalDefinition{}
	err = yaml.Unmarshal(yamlFile, &c)

	if err != nil {
		return err
	}

	for index, compilerDef := range c.Compilers {
		if len(compilerDef.Path) == 0 {
			return fmt.Errorf("Global compiler definition of name `%s` need to have the field `path` set!", compilerDef.Name)
		}
		if len(compilerDef.Name) == 0 {
			return fmt.Errorf("Global compiler definition with path `%s` (index %d) need to have the field `name` set!", compilerDef.Path, index)
		}
	}

	for _, targetDef := range c.Targets {
		// merge compiler flags
		var compilerDef *CompilerDefinition
		compilerDef, err = targetDef.Compiler.WithRef(&c.Compilers)

		if err != nil {
			return err
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
				buildCommand = append(buildCommand, link.Path)
			}

			buildCommand = append(buildCommand, link.Link)
		}

		buildCommand = append(buildCommand, "-o")
		buildCommand = append(buildCommand, targetDef.Output)

		if err := os.MkdirAll(filepath.Dir(targetDef.Output), os.ModePerm); err != nil {
			return err
		}

		for _, source := range targetDef.Sources {
			if strings.Contains(source, "*") {
				globbed, err := filepath.Glob(source)

				if err != nil {
					return err
				}

				buildCommand = append(buildCommand, globbed...)
			} else {
				buildCommand = append(buildCommand, source)
			}
		}

		command := exec.Command(buildCommand[0], buildCommand[1:]...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		command.Run()
	}

	return nil
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "build",
				Usage:  "build the project",
				Action: build,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
