package gmakec

import (
	"fmt"
	"strings"
)

type GlobalDefinition struct {
	Description  string               `yaml:"description"` // unused atm
	Version      string               `yaml:"version"`
	Compilers    []CompilerDefinition `yaml:"compilers"`
	Targets      []TargetDefinition   `yaml:"targets"`
	Imports      []string             `yaml:"imports"`
	VersionMajor string
	VersionMinor string
	VersionPatch string
	VersionTweak string
}

func (this *GlobalDefinition) sanitizeVersion() error {
	if !strings.Contains(this.Version, ".") {
		return fmt.Errorf(
			"WARNING: Version string `%s` not available in semver format, skipping semver sanitize step...\n",
			this.Version,
		)
	} else {
		semver := strings.Split(this.Version, ".")

		if len(semver) < 3 {
			return fmt.Errorf(
				"WARNING: Version string `%s` not available in semver format, skipping semver sanitize step...\n",
				this.Version,
			)
		} else {
			this.VersionMajor = semver[0]
			this.VersionMinor = semver[1]
			this.VersionPatch = semver[2]

			if len(semver) > 3 {
				this.VersionTweak = semver[3]
			}
		}
	}

	return nil
}

func (this *GlobalDefinition) generateDependencyGraphs() [][]int {
	graphs := [][]int{}

	for index := range this.Targets {
		graphs = append(graphs, this.Targets[index].dependencyGraph(index, &this.Targets))
	}

	return graphs
}
