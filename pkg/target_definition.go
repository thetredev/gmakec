package gmakec

import (
	"fmt"

	"github.com/fatih/structs"
	"golang.org/x/exp/slices"
)

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
