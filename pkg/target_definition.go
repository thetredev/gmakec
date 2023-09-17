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
	Hooks        []TargetHook       `yaml:"hooks"`
}

func (targetDef *TargetDefinition) ExecuteHooks(step string, workingDir string) {
	for _, targetHook := range targetDef.Hooks {
		if targetHook.Step == step {
			targetHook.Execute(workingDir)
		}
	}
}

func (targetDef *TargetDefinition) FindField(fieldName string) *structs.Field {
	fields := structs.Fields(targetDef)

	for _, field := range fields {
		tag := field.Tag("yaml")

		if tag == fieldName {
			return field
		}
	}

	return nil
}

func (targetDef *TargetDefinition) FieldStringValue(fieldName string, defContext *DefinitionContext) (string, error) {
	field := targetDef.FindField(fieldName)

	if field == nil {
		return "", fmt.Errorf("Could not find field `%s`", fieldName)
	}

	return fmt.Sprintf("%s/%s", defContext.DefinitionPath, field.Value().(string)), nil
}

func (targetDef *TargetDefinition) FieldStringArrayValue(fieldName string, defContext *DefinitionContext) ([]string, error) {
	field := targetDef.FindField(fieldName)

	if field == nil {
		return nil, fmt.Errorf("Could not find field `%s`", fieldName)
	}

	result := []string{}

	for _, value := range field.Value().([]string) {
		result = append(result, fmt.Sprintf("%s/%s", defContext.DefinitionPath, value))
	}

	return result, nil
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
