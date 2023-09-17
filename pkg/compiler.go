package gmakec

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type Compiler struct {
	Name              string
	Path              string
	IncludeSearchFlag string
	LinkSearchFlag    string
	OutputFlag        string
}

var compilers []*Compiler

func fromCompilerTemplate(compilerTemplate *Compiler, name string) *Compiler {
	cc := *compilerTemplate
	cc.Name = name

	return &cc
}

func InitCompilers() {
	compilerTemplate := &Compiler{
		IncludeSearchFlag: "-I",
		LinkSearchFlag:    "-L",
		OutputFlag:        "-o",
	}

	compilers = make([]*Compiler, 0)
	compilers = []*Compiler{
		fromCompilerTemplate(compilerTemplate, "gcc"),
		fromCompilerTemplate(compilerTemplate, "g++"),
		fromCompilerTemplate(compilerTemplate, "clang"),
		fromCompilerTemplate(compilerTemplate, "clang++"),
	}
}

func findCompilerByPath(lookedPath string) (*Compiler, error) {
	lookedPath, err := exec.LookPath(lookedPath)

	if err != nil {
		return nil, fmt.Errorf("Compiler of path `%s` not found!", lookedPath)
	}

	absolutePath, err := filepath.Abs(lookedPath)

	if err != nil {
		return nil, fmt.Errorf("Absolute path for compiler with path `%s` not found!", lookedPath)
	}

	name := filepath.Base(absolutePath)

	for index := range compilers {
		if compilers[index].Name == name {
			compilers[index].Path = absolutePath
			return compilers[index], nil
		}
	}

	return nil, fmt.Errorf("Compiler of path `%s` is not supported (yet)!", absolutePath)
}
