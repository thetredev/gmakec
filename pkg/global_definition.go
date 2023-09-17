package gmakec

type GlobalDefinition struct {
	Description string               `yaml:"description"` // unused atm
	Version     string               `yaml:"version"`     // unused atm
	Compilers   []CompilerDefinition `yaml:"compilers"`
	Targets     []TargetDefinition   `yaml:"targets"`
	Imports     []string             `yaml:"imports"`
}

func (this *GlobalDefinition) generateDependencyGraphs() [][]int {
	graphs := [][]int{}

	for index := range this.Targets {
		graphs = append(graphs, this.Targets[index].dependencyGraph(index, &this.Targets))
	}

	return graphs
}
