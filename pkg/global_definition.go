package gmakec

type GlobalDefinition struct {
	Description string               `yaml:"description"` // unused atm
	Version     string               `yaml:"version"`     // unused atm
	Compilers   []CompilerDefinition `yaml:"compilers"`
	Targets     []TargetDefinition   `yaml:"targets"`
	Imports     []string             `yaml:"imports"`
}

func (globalDef *GlobalDefinition) GenerateDependencyGraphs() [][]int {
	graphs := [][]int{}

	for index := range globalDef.Targets {
		graphs = append(graphs, globalDef.Targets[index].DependencyGraph(index, &globalDef.Targets))
	}

	return graphs
}
