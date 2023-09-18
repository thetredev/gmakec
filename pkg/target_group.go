package gmakec

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yargevad/filepathx"
)

type TargetGroup struct {
	Targets []int
}

func (this *TargetGroup) configure(
	definitionContext *DefinitionContext, definitionContexts *[]*DefinitionContext,
) ([]string, error) {
	buildCommands := []string{}

	for i := len(this.Targets) - 1; i >= 0; i-- {
		targetIndex := this.Targets[i]
		targetDef := definitionContext.Definition.Targets[targetIndex]

		if err := targetDef.mergeHookRefs(targetIndex, definitionContext); err != nil {
			return nil, err
		}

		if err := targetDef.executeHooks("preConfigure", definitionContext.DefinitionPath); err != nil {
			return nil, err
		}

		// merge compiler flags
		compilerDef, err := targetDef.Compiler.sanitize(&definitionContext.Definition.Compilers)

		if err != nil {
			return nil, err
		}

		for _, configureFile := range targetDef.ConfigureFiles {
			if err := configureFile.Execute(definitionContext); err != nil {
				return nil, err
			}
		}

		buildCommand := []string{
			fmt.Sprintf("%d", targetIndex),
			compilerDef.Object.Path,
		}

		buildCommand = append(buildCommand, compilerDef.Flags...)

		for _, define := range targetDef.Defines {
			buildCommand = append(buildCommand, compilerDef.Object.DefineFlag)
			buildCommand = append(buildCommand, define)
		}

		for _, include := range targetDef.Includes {
			includeStrings := []string{}

			if strings.Contains(include, ":") {
				refStringArrayValue, err := findRefTargetStringArrayValue(include, &targetDef, definitionContexts)

				if err != nil {
					return nil, err
				}

				includeStrings = append(includeStrings, refStringArrayValue...)
			} else {
				includeStrings = append(includeStrings, include)
			}

			for _, includeString := range includeStrings {
				if strings.Contains(includeString, "*") {
					globbed, err := filepathx.Glob(includeString)

					if err != nil {
						return nil, err
					}

					for _, match := range globbed {
						f, _ := os.Stat(match)

						if f.IsDir() {
							buildCommand = append(buildCommand, compilerDef.Object.IncludeSearchFlag)
							buildCommand = append(buildCommand, match)
						}
					}
				} else {
					buildCommand = append(buildCommand, compilerDef.Object.IncludeSearchFlag)
					buildCommand = append(buildCommand, includeString)
				}
			}
		}

		for _, link := range targetDef.Links {
			if len(link.Path) > 0 {
				buildCommand = append(buildCommand, compilerDef.Object.LinkSearchFlag)
				linkPath := link.Path

				if strings.Contains(linkPath, ":") {
					linkPath, err = findRefTargetStringValue(linkPath, &targetDef, definitionContexts)
				}

				buildCommand = append(buildCommand, filepath.Dir(linkPath))
			}

			buildCommand = append(buildCommand, link.Link)
		}

		buildCommand = append(buildCommand, compilerDef.Object.OutputFlag)
		buildCommand = append(buildCommand, targetDef.Output)

		for _, source := range targetDef.Sources {
			if len(source.Platform) > 0 && runtime.GOOS != source.Platform {
				continue
			}

			if strings.Contains(source.Path, "*") {
				globbed, err := filepathx.Glob(source.Path)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, globbed...)
			} else if strings.Contains(source.Path, ":") {
				refStringValue, err := findRefTargetStringValue(source.Path, &targetDef, definitionContexts)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, refStringValue)
			} else {
				buildCommand = append(buildCommand, source.Path)
			}
		}

		buildCommands = append(buildCommands, strings.Join(buildCommand, " "))

		if err = targetDef.executeHooks("postConfigure", definitionContext.DefinitionPath); err != nil {
			return nil, err
		}
	}

	return buildCommands, nil
}
