package gmakec

import (
	"fmt"

	"github.com/fatih/structs"
	"golang.org/x/exp/slices"
)

type TargetDefinition struct {
	Name           string             `yaml:"name"`
	Compiler       CompilerDefinition `yaml:"compiler"`
	ConfigureFiles []ConfigureFile    `yaml:"configure_files"`
	Defines        []string           `yaml:"defines"`
	Sources        []SourceDefinition `yaml:"sources"`
	Includes       []string           `yaml:"includes"`
	Links          []LinkDefinition   `yaml:"links"`
	Output         string             `yaml:"output"`
	Dependencies   []string           `yaml:"dependencies"`
	Hooks          []TargetHook       `yaml:"hooks"`
}

func (this *TargetDefinition) executeHooks(step string, workingDir string) {
	for _, targetHook := range this.Hooks {
		if targetHook.Step == step {
			targetHook.execute(workingDir)
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

func (this *TargetDefinition) fieldStringValue(fieldName string, definitionContext *DefinitionContext) (string, error) {
	field := this.findField(fieldName)

	if field == nil {
		return "", fmt.Errorf("Could not find field `%s`", fieldName)
	}

	return fmt.Sprintf("%s/%s", definitionContext.DefinitionPath, field.Value().(string)), nil
}

func (this *TargetDefinition) fieldStringArrayValue(
	fieldName string, definitionContext *DefinitionContext,
) ([]string, error) {
	field := this.findField(fieldName)

	if field == nil {
		return nil, fmt.Errorf("Could not find field `%s`", fieldName)
	}

	result := []string{}

	for _, value := range field.Value().([]string) {
		result = append(result, fmt.Sprintf("%s/%s", definitionContext.DefinitionPath, value))
	}

	return result, nil
}

func (this *TargetDefinition) dependencyGraph(index int, targetDefinitions *[]TargetDefinition) []int {
	dependencyGraph := []int{index}

	if len(this.Dependencies) > 0 {
		for i, otherTarget := range *targetDefinitions {
			if slices.Contains(this.Dependencies, otherTarget.Name) {
				dependencyGraph = append(dependencyGraph, i)
			}
		}
	}

	return dependencyGraph
}
