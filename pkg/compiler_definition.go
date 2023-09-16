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

func (compilerDef *CompilerDefinition) FindRef(globalCompilerDefs *[]CompilerDefinition) *CompilerDefinition {
	for _, globalCompilerDef := range *globalCompilerDefs {
		if globalCompilerDef.Name == compilerDef.Ref {
			return &globalCompilerDef
		}
	}

	return nil
}

func (compilerDef *CompilerDefinition) WithRef(globalCompilerDefs *[]CompilerDefinition) (*CompilerDefinition, error) {
	if len(compilerDef.Ref) == 0 {
		if len(compilerDef.Path) == 0 {
			return nil, fmt.Errorf("Non-ref compiler definition of name `%s` need to have the field `path` set!", compilerDef.Name)
		}

		path, err := exec.LookPath(compilerDef.Path)

		if err != nil {
			return nil, fmt.Errorf("Non-ref compiler path `%s` not found!", compilerDef.Path)
		}

		compilerDef.Path = path
		return compilerDef, nil
	}

	compilerRef := compilerDef.FindRef(globalCompilerDefs)

	if compilerRef == nil {
		return nil, fmt.Errorf("Could not find compiler ref: %s\n", compilerDef.Ref)
	}

	return &CompilerDefinition{
		Name:  compilerRef.Name,
		Path:  compilerRef.Path,
		Flags: append(compilerRef.Flags, compilerDef.Flags...),
	}, nil
}
