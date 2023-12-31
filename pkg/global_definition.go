package gmakec

import (
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/exp/slices"
)

type GlobalDefinition struct {
	Description  string               `yaml:"description"` // unused atm
	Version      string               `yaml:"version"`
	Compilers    []CompilerDefinition `yaml:"compilers"`
	Actions      []ActionDefinition   `yaml:"actions"`
	Hooks        []HookDefinition     `yaml:"hooks"`
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

func (this *GlobalDefinition) sanitizeTargets(definitionContext *DefinitionContext) error {
	t := make([]TargetDefinition, 0)

	for index, targetDef := range this.Targets {
		if len(targetDef.Platform) > 0 && runtime.GOOS != targetDef.Platform {
			continue
		}

		if len(targetDef.Output) == 0 {
			return fmt.Errorf(
				"Target of definition path `%s` and index %d has no output!",
				definitionContext.DefinitionPath,
				index,
			)
		}

		t = append(t, targetDef)
	}

	if len(t) == 0 {
		return fmt.Errorf("No targets left to build after sanitizing targets!\n")
	}

	this.Targets = t
	return nil
}

func (this *GlobalDefinition) sanitizeCompilers() error {
	for index, compilerDef := range this.Compilers {
		if len(compilerDef.Path) == 0 {
			return fmt.Errorf(
				"Global compiler definition of name `%s` (index %d) need to have the field `path` set!",
				compilerDef.Name, index,
			)
		}
		if len(compilerDef.Name) == 0 {
			return fmt.Errorf(
				"Global compiler definition with path `%s` (index %d) need to have the field `name` set!",
				compilerDef.Path, index,
			)
		}
	}

	return nil
}

func (this *GlobalDefinition) sanitize(definitionContext *DefinitionContext) error {
	if err := this.sanitizeVersion(); err != nil {
		return err
	}

	if err := this.sanitizeTargets(definitionContext); err != nil {
		return err
	}

	if err := this.sanitizeCompilers(); err != nil {
		return err
	}

	return nil
}

func (this *GlobalDefinition) generateDependencyGraphs(targets ...string) [][]int {
	graphs := [][]int{}

	for index, targetDef := range this.Targets {
		if len(targets) == 0 || slices.Contains(targets, targetDef.Name) {
			graphs = append(graphs, this.Targets[index].dependencyGraph(index, &this.Targets))
		}
	}

	return graphs
}
