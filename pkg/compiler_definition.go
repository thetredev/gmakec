package gmakec

import (
	"fmt"
	"os/exec"
)

type CompilerDefinition struct {
	Name  string   `yaml:"name"`
	Ref   string   `yaml:"ref"`
	Path  string   `yaml:"path"`
	Flags []string `yaml:"flags"`
}

func (this *CompilerDefinition) findRef(refCompilerDefinitions *[]CompilerDefinition) *CompilerDefinition {
	for _, refCompilerDefinition := range *refCompilerDefinitions {
		if refCompilerDefinition.Name == this.Ref {
			return &refCompilerDefinition
		}
	}

	return nil
}

func (this *CompilerDefinition) WithRef(refCompilerDefinitions *[]CompilerDefinition) (*CompilerDefinition, error) {
	if len(this.Ref) == 0 {
		if len(this.Path) == 0 {
			return nil, fmt.Errorf("Non-ref compiler definition of name `%s` need to have the field `path` set!", this.Name)
		}

		path, err := exec.LookPath(this.Path)

		if err != nil {
			return nil, fmt.Errorf("Non-ref compiler path `%s` not found!", this.Path)
		}

		this.Path = path
		return this, nil
	}

	compilerRef := this.findRef(refCompilerDefinitions)

	if compilerRef == nil {
		return nil, fmt.Errorf("Could not find compiler ref: %s\n", this.Ref)
	}

	return &CompilerDefinition{
		Name:  compilerRef.Name,
		Path:  compilerRef.Path,
		Flags: append(compilerRef.Flags, this.Flags...),
	}, nil
}
