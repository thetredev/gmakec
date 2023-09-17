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

func (this *CompilerDefinition) FindRef(refCompilerDefs *[]CompilerDefinition) *CompilerDefinition {
	for _, refCompilerDef := range *refCompilerDefs {
		if refCompilerDef.Name == this.Ref {
			return &refCompilerDef
		}
	}

	return nil
}

func (this *CompilerDefinition) WithRef(refCompilerDefs *[]CompilerDefinition) (*CompilerDefinition, error) {
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

	compilerRef := this.FindRef(refCompilerDefs)

	if compilerRef == nil {
		return nil, fmt.Errorf("Could not find compiler ref: %s\n", this.Ref)
	}

	return &CompilerDefinition{
		Name:  compilerRef.Name,
		Path:  compilerRef.Path,
		Flags: append(compilerRef.Flags, this.Flags...),
	}, nil
}
