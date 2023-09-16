package gmakec

import "strings"

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
