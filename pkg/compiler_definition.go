package gmakec

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CompilerDefinition struct {
	Name   string                   `yaml:"name"`
	Ref    string                   `yaml:"ref"`
	Path   string                   `yaml:"path"`
	Flags  []string                 `yaml:"flags"`
	Find   []CompilerFindDefinition `yaml:"find"`
	Object *Compiler
}

func (this *CompilerDefinition) findRef(refCompilerDefinitions *[]CompilerDefinition) *CompilerDefinition {
	for _, refCompilerDefinition := range *refCompilerDefinitions {
		if refCompilerDefinition.Name == this.Ref {
			return &refCompilerDefinition
		}
	}

	return nil
}

func (this *CompilerDefinition) withRef(refCompilerDefinitions *[]CompilerDefinition) (*CompilerDefinition, error) {
	if len(this.Ref) == 0 {
		if len(this.Path) == 0 {
			return nil, fmt.Errorf("Non-ref compiler definition of name `%s` need to have the field `path` set!", this.Name)
		}

		path, err := exec.LookPath(this.Path)

		if err != nil {
			return nil, fmt.Errorf("Non-ref compiler path `%s` not found!", this.Path)
		}

		object, err := findCompilerByPath(path)

		if err != nil {
			return nil, err
		}

		this.Path = object.Path
		this.Object = object
		return this, nil
	}

	compilerRef := this.findRef(refCompilerDefinitions)

	if compilerRef == nil {
		return nil, fmt.Errorf("Could not find compiler ref: %s\n", this.Ref)
	}

	object, err := findCompilerByPath(compilerRef.Path)

	if err != nil {
		return nil, err
	}

	compilerRef.Object = object
	compilerRef.Flags = append(compilerRef.Flags, this.Flags...)
	compilerRef.Find = append(compilerRef.Find, this.Find...)
	return compilerRef, nil
}

func (this *CompilerDefinition) sanitize(refCompilerDefinitions *[]CompilerDefinition) (*CompilerDefinition, error) {
	var err error
	this, err = this.withRef(refCompilerDefinitions)

	if err != nil {
		return nil, err
	}

	notFoundIndices := []int{}

	for index, find := range this.Find {
		paths := []string{}

		if len(find.Paths) > 0 {
			for _, findPath := range find.Paths {
				// TODO: this is not portable at all, POSIX only. But it works for now.
				paths = append(paths, os.ExpandEnv(strings.ReplaceAll(findPath, "~", "${HOME}")))
			}
		}

		paths = append(paths, strings.Split(os.Getenv("PATH"), ":")...)
		// TODO: add the ones defined in this project, if we will ever do that

		if find.Type == "filesystem" {
			for _, file := range find.Names {
				found := false

				for _, filePath := range paths {
					fullPath := filepath.Join(filePath, file)
					_, err := os.Stat(fullPath)

					if err != nil {
						if os.IsNotExist(err) {
							continue
						}

						return nil, err
					}

					found = true
					this.Find[index].Results = append(this.Find[index].Results, CompilerFindResult{
						Type: find.Type,
						File: file,
						Path: fullPath,
					})

					break
				}

				if !found {
					notFoundIndices = append(notFoundIndices, index)
				}
			}
		}

		// TODO: implement search for libraries
	}

	if len(notFoundIndices) > 0 {
		for _, index := range notFoundIndices {
			find := this.Find[index]
			log.Printf("ERROR: Could not find %s object of names %v and paths %v\n", find.Type, find.Names, find.Paths)
		}

		return nil, fmt.Errorf("Exiting...")
	}

	return this, nil
}
