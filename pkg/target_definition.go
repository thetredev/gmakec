package gmakec

import (
	"fmt"

	"github.com/fatih/structs"
	"golang.org/x/exp/slices"
)

type TargetDefinition struct {
	Name           string             `yaml:"name"`
	Platform       string             `yaml:"platform"`
	Compiler       CompilerDefinition `yaml:"compiler"`
	ConfigureFiles []ConfigureFile    `yaml:"configure_files"`
	Defines        []string           `yaml:"defines"`
	Sources        []SourceDefinition `yaml:"sources"`
	Includes       []string           `yaml:"includes"`
	Links          []LinkDefinition   `yaml:"links"`
	Output         string             `yaml:"output"`
	Dependencies   []string           `yaml:"dependencies"`
	Hooks          []HookDefinition   `yaml:"hooks"`
}

func (this *TargetDefinition) mergeHookRefs(targetIndex int, definitionContext *DefinitionContext) error {
	for index := range this.Hooks {
		hook, err := this.Hooks[index].withRef(definitionContext)

		if err != nil {
			return err
		}

		this.Hooks[index] = *hook
	}

	for index, hook := range this.Hooks {
		if len(hook.Step) == 0 {
			return fmt.Errorf("Hook with index %d of target with index %d does not define a step!\n", index, targetIndex)
		}

		if len(hook.Actions) == 0 {
			return fmt.Errorf(
				"Hook with index %d (step: %s) of target with index %d does not define any actions!\n",
				index, hook.Step, targetIndex,
			)
		}
	}

	return nil
}

func (this *TargetDefinition) executeHooks(step string, workingDir string) error {
	for _, targetHook := range this.Hooks {
		if targetHook.Step == step {
			for index := range targetHook.Actions {
				message := fmt.Sprintf("[%s] Executing hook", step)

				if len(targetHook.Actions[index].Description) > 0 {
					message = fmt.Sprintf("%s: %s", message, targetHook.Actions[index].Description)
				}

				fmt.Printf("%s...\n", message)
				ok, err := targetHook.Actions[index].execute(workingDir)

				if err != nil {
					return fmt.Errorf("ERROR: Could not execute hook for step `%s`: %s\n", step, err.Error())
				}

				if ok {
					for _, successHandle := range targetHook.Actions[index].Output.OnSuccess {
						successHandle.handle(this, step)
					}
				} else {
					for _, failureHandle := range targetHook.Actions[index].Output.OnFailure.Handle {
						failureHandle.handle(this, step)
					}

					if !targetHook.Actions[index].Output.OnFailure.Continue {
						return fmt.Errorf("ERROR: Execution of hook step `%s` failed!\n", step)
					}
				}
			}
		}
	}

	return nil
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
