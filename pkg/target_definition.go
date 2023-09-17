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

func (this *TargetDefinition) ExecuteHooks(step string, workingDir string) {
	for _, targetHook := range this.Hooks {
		if targetHook.Step == step {
			targetHook.Execute(workingDir)
		}
	}
}

func (this *TargetDefinition) findField(fieldName string) *structs.Field {
	fields := structs.Fields(this)

	for _, field := range fields {
		tag := field.Tag("yaml")

		if tag == fieldName {
			return field
		}
	}

	return nil
}

func (this *TargetDefinition) FieldStringValue(fieldName string, defContext *DefinitionContext) (string, error) {
	field := this.findField(fieldName)

	if field == nil {
		return "", fmt.Errorf("Could not find field `%s`", fieldName)
	}

	return fmt.Sprintf("%s/%s", defContext.DefinitionPath, field.Value().(string)), nil
}

func (this *TargetDefinition) FieldStringArrayValue(fieldName string, defContext *DefinitionContext) ([]string, error) {
	field := this.findField(fieldName)

	if field == nil {
		return nil, fmt.Errorf("Could not find field `%s`", fieldName)
	}

	result := []string{}

	for _, value := range field.Value().([]string) {
		result = append(result, fmt.Sprintf("%s/%s", defContext.DefinitionPath, value))
	}

	return result, nil
}

func (this *TargetDefinition) DependencyGraph(index int, targetDefs *[]TargetDefinition) []int {
	dependencyGraph := []int{index}

	if len(this.Dependencies) > 0 {
		for i, otherTarget := range *targetDefs {
			if slices.Contains(this.Dependencies, otherTarget.Name) {
				dependencyGraph = append(dependencyGraph, i)
			}
		}
	}

	return dependencyGraph
}
